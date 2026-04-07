// Copyright (c) 2026 WSO2 LLC. (https://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	smtpclient "github.com/wso2-open-operations/infra-operations/operations/email-service/internal/smtp"
)

// Mailer defines the dependencies for the email service handlers.
type Mailer interface {
	SendEmail(ctx context.Context, msg *smtpclient.Message) error
	Ping(ctx context.Context) error
}

// EmailHandler holds the dependencies for the email HTTP handler.
type EmailHandler struct {
	client             Mailer
	maxRequestBodySize int64
}

// NewEmailHandler creates an EmailHandler backed by the given Mailer.
func NewEmailHandler(client Mailer, maxRequestBodySize int64) *EmailHandler {
	return &EmailHandler{
		client:             client,
		maxRequestBodySize: maxRequestBodySize,
	}
}

// SendEmail handles POST /send-email.
// It parses the JSON request body and dispatches the email using the Mailer interface.
func (h *EmailHandler) SendEmail(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.maxRequestBodySize)

	var req EmailRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			slog.Error(ERR_REQUEST_BODY_TOO_LARGE, "error", err, "limit", h.maxRequestBodySize)
			writeJSON(w, http.StatusRequestEntityTooLarge, ResponseMessage{Message: ERR_REQUEST_BODY_TOO_LARGE})
			return
		}

		slog.Error("failed to decode request body", "error", err)
		writeJSON(w, http.StatusBadRequest, ResponseMessage{Message: ERR_INVALID_REQUEST_BODY})
		return
	}

	// Check for trailing JSON data.
	if err := dec.Decode(&struct{}{}); err != nil && !errors.Is(err, io.EOF) {
		slog.Error("failed to decode request body", "error", "trailing JSON data")
		writeJSON(w, http.StatusBadRequest, ResponseMessage{Message: ERR_INVALID_REQUEST_BODY})
		return
	}

	if len(req.To) == 0 {
		writeJSON(w, http.StatusBadRequest, ResponseMessage{Message: ERR_RECIPIENTS_REQUIRED})
		return
	}

	if strings.TrimSpace(req.From) == "" {
		writeJSON(w, http.StatusBadRequest, ResponseMessage{Message: ERR_FROM_REQUIRED})
		return
	}

	if strings.TrimSpace(req.Subject) == "" {
		writeJSON(w, http.StatusBadRequest, ResponseMessage{Message: ERR_SUBJECT_REQUIRED})
		return
	}

	// Validate addresses to prevent CR/LF injection and ensure proper format.
	if err := validateAddress(req.From); err != nil {
		slog.Warn(ERR_INVALID_FROM, "address", req.From, "error", err)
		writeJSON(w, http.StatusBadRequest, ResponseMessage{Message: ERR_INVALID_FROM})
		return
	}
	for _, addr := range req.To {
		if err := validateAddress(addr); err != nil {
			slog.Warn(ERR_INVALID_TO, "address", addr, "error", err)
			writeJSON(w, http.StatusBadRequest, ResponseMessage{Message: ERR_INVALID_TO})
			return
		}
	}
	for _, addr := range req.CC {
		if err := validateAddress(addr); err != nil {
			slog.Warn(ERR_INVALID_CC, "address", addr, "error", err)
			writeJSON(w, http.StatusBadRequest, ResponseMessage{Message: ERR_INVALID_CC})
			return
		}
	}
	for _, addr := range req.BCC {
		if err := validateAddress(addr); err != nil {
			slog.Warn(ERR_INVALID_BCC, "address", addr, "error", err)
			writeJSON(w, http.StatusBadRequest, ResponseMessage{Message: ERR_INVALID_BCC})
			return
		}
	}
	for _, addr := range req.ReplyTo {
		if err := validateAddress(addr); err != nil {
			slog.Warn(ERR_INVALID_REPLY_TO, "address", addr, "error", err)
			writeJSON(w, http.StatusBadRequest, ResponseMessage{Message: ERR_INVALID_REPLY_TO})
			return
		}
	}

	decoded, err := base64.StdEncoding.DecodeString(req.Template)
	if err != nil {
		slog.Warn(ERR_TEMPLATE_DECODE_ERR, "error", err)
		writeJSON(w, http.StatusBadRequest, ResponseMessage{Message: ERR_TEMPLATE_DECODE_ERR})
		return
	}
	htmlBody := string(decoded)

	outMsg := &smtpclient.Message{
		To:       append([]string(nil), req.To...),
		CC:       append([]string(nil), req.CC...),
		BCC:      append([]string(nil), req.BCC...),
		ReplyTo:  append([]string(nil), req.ReplyTo...),
		From:     req.From,
		Subject:  req.Subject,
		HTMLBody: htmlBody,
	}

	for _, att := range req.Attachments {
		if err := smtpclient.ValidateMIMEType(att.ContentType); err != nil {
			slog.Warn(ERR_INVALID_CONTENT_TYPE, "contentType", att.ContentType, "error", err)
			writeJSON(w, http.StatusBadRequest, ResponseMessage{Message: ERR_INVALID_CONTENT_TYPE})
			return
		}

		outMsg.Attachments = append(outMsg.Attachments, smtpclient.Attachment{
			ContentName: att.ContentName,
			ContentType: att.ContentType,
			Data:        att.Attachment,
		})
	}

	if err := h.client.SendEmail(r.Context(), outMsg); err != nil {
		slog.Error(ERR_EMAIL_SEND_ERR, "error", err)
		writeJSON(w, http.StatusInternalServerError, ResponseMessage{Message: ERR_EMAIL_SEND_ERR})
		return
	}

	slog.Info(MSG_EMAIL_SENT_SUCCESS)
	writeJSON(w, http.StatusOK, ResponseMessage{Message: MSG_EMAIL_SENT_SUCCESS})
}

// writeJSON encodes v as JSON and writes it with the given HTTP status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set(HEADER_CONTENT_TYPE, CONTENT_TYPE_JSON)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to write JSON response", "error", err)
	}
}

// validateAddress returns an error if the email address is invalid or contains
// characters that could lead to header injection (CR/LF).
func validateAddress(addr string) error {
	if strings.ContainsAny(addr, "\r\n") {
		return errors.New("address contains CR/LF characters")
	}
	if strings.TrimSpace(addr) != addr {
		return errors.New("address contains leading or trailing whitespace")
	}
	_, err := mail.ParseAddress(addr)
	return err
}
