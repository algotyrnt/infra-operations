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
	"log/slog"
	"net/http"
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
	client Mailer
}

// NewEmailHandler creates an EmailHandler backed by the given Mailer.
func NewEmailHandler(client Mailer) *EmailHandler {
	return &EmailHandler{client: client}
}

// SendEmail handles POST /send-email.
// It parses the JSON request body and dispatches the email using the Mailer interface.
func (h *EmailHandler) SendEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ResponseMessage{Message: "method not allowed"})
		return
	}

	var req EmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("failed to decode request body", "error", err)
		writeJSON(w, http.StatusBadRequest, ResponseMessage{Message: "invalid request body"})
		return
	}

	if len(req.To) == 0 {
		msg := "There should be at least one recipient!"
		slog.Error(msg)
		writeJSON(w, http.StatusBadRequest, ResponseMessage{Message: msg})
		return
	}

	if strings.TrimSpace(req.From) == "" {
		msg := "The 'From' email address cannot be empty"
		slog.Error(msg)
		writeJSON(w, http.StatusBadRequest, ResponseMessage{Message: msg})
		return
	}

	if strings.TrimSpace(req.Subject) == "" {
		msg := "The 'Subject' cannot be empty"
		slog.Error(msg)
		writeJSON(w, http.StatusBadRequest, ResponseMessage{Message: msg})
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(req.Template)
	if err != nil {
		msg := "An error occurred while decoding the email template!"
		slog.Error(msg, "error", err)
		writeJSON(w, http.StatusBadRequest, ResponseMessage{Message: msg})
		return
	}
	htmlBody := string(decoded)

	outMsg := &smtpclient.Message{
		To:       req.To,
		CC:       req.CC,
		BCC:      req.BCC,
		ReplyTo:  req.ReplyTo,
		From:     req.From,
		Subject:  req.Subject,
		HTMLBody: htmlBody,
	}

	for _, att := range req.Attachments {
		if err := smtpclient.ValidateMIMEType(att.ContentType); err != nil {
			msgStr := "Attachment content type is not supported"
			slog.Error(msgStr, "contentType", att.ContentType, "error", err)
			writeJSON(w, http.StatusBadRequest, ResponseMessage{Message: msgStr})
			return
		}

		outMsg.Attachments = append(outMsg.Attachments, smtpclient.Attachment{
			ContentName: att.ContentName,
			ContentType: att.ContentType,
			Data:        att.Attachment,
		})
	}

	if err := h.client.SendEmail(r.Context(), outMsg); err != nil {
		msgStr := "An error occurred while sending the email!"
		slog.Error(msgStr, "error", err)
		writeJSON(w, http.StatusInternalServerError, ResponseMessage{Message: msgStr})
		return
	}

	slog.Info("email sent successfully", "subject", req.Subject)
	writeJSON(w, http.StatusOK, ResponseMessage{Message: "Email sent successfully"})
}

// writeJSON encodes v as JSON and writes it with the given HTTP status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to write JSON response", "error", err)
	}
}
