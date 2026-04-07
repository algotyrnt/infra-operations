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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	smtpclient "github.com/wso2-open-operations/infra-operations/operations/email-service/internal/smtp"
)

type mockMailer struct {
	err error
}

// SendEmail records the mock call and returns the configured error.
func (m *mockMailer) SendEmail(ctx context.Context, msg *smtpclient.Message) error {
	return m.err
}

// Ping returns the configured error to simulate SMTP server availability.
func (m *mockMailer) Ping(ctx context.Context) error {
	return m.err
}

// newTestHandler returns an EmailHandler using a mock Mailer with a default large limit.
func newTestHandler(err error) *EmailHandler {
	return NewEmailHandler(&mockMailer{err: err}, 10*1024*1024)
}

// doPost is a helper that executes a POST /send-email request and returns the recorder.
func doPost(t *testing.T, h *EmailHandler, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/send-email", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.SendEmail(rr, req)
	return rr
}

// decodeResponse parses the recorder body into a ResponseMessage.
func decodeResponse(t *testing.T, rr *httptest.ResponseRecorder) ResponseMessage {
	t.Helper()
	var msg ResponseMessage
	if err := json.NewDecoder(rr.Body).Decode(&msg); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return msg
}

// assertResponse checks the HTTP status code and, optionally, the JSON message body.
func assertResponse(t *testing.T, rr *httptest.ResponseRecorder, wantCode int, wantMsg string) {
	t.Helper()
	if rr.Code != wantCode {
		t.Errorf("status: got %d, want %d", rr.Code, wantCode)
	}
	if wantMsg != "" {
		msg := decodeResponse(t, rr)
		if msg.Message != wantMsg {
			t.Errorf("message: got %q, want %q", msg.Message, wantMsg)
		}
	}
}

// TestEmptyFromField tests when the from field is empty.
func TestEmptyFromField(t *testing.T) {
	h := newTestHandler(nil)
	rr := doPost(t, h, map[string]any{
		"to":       []string{"test@example.com"},
		"from":     "",
		"subject":  "test subject",
		"template": base64.StdEncoding.EncodeToString([]byte("<h1>Hello</h1>")),
	})
	assertResponse(t, rr, http.StatusBadRequest, ERR_FROM_REQUIRED)
}

// TestEmptyRecipients tests when recipients are empty.
func TestEmptyRecipients(t *testing.T) {
	h := newTestHandler(nil)
	rr := doPost(t, h, map[string]any{
		"to":       []string{},
		"from":     "sender@example.com",
		"subject":  "test subject",
		"template": base64.StdEncoding.EncodeToString([]byte("<h1>Hello</h1>")),
	})
	assertResponse(t, rr, http.StatusBadRequest, ERR_RECIPIENTS_REQUIRED)
}

// TestInvalidTemplate tests when the template is invalid.
// "A" is a single character — invalid base64 padding.
func TestInvalidTemplate(t *testing.T) {
	h := newTestHandler(nil)
	rr := doPost(t, h, map[string]any{
		"to":       []string{"test@example.com"},
		"from":     "sender@example.com",
		"subject":  "test subject",
		"template": "A", // invalid base64
	})
	assertResponse(t, rr, http.StatusBadRequest, ERR_TEMPLATE_DECODE_ERR)
}

// TestEmptySubject tests that a blank subject is rejected.
func TestEmptySubject(t *testing.T) {
	h := newTestHandler(nil)
	rr := doPost(t, h, map[string]any{
		"to":       []string{"test@example.com"},
		"from":     "sender@example.com",
		"subject":  "   ", // whitespace-only
		"template": base64.StdEncoding.EncodeToString([]byte("<h1>Hello</h1>")),
	})
	assertResponse(t, rr, http.StatusBadRequest, ERR_SUBJECT_REQUIRED)
}

// TestInvalidContentType tests when the content type is invalid.
// "application.pdf" is not a valid MIME type (missing slash).
func TestInvalidContentType(t *testing.T) {
	h := newTestHandler(nil)
	rr := doPost(t, h, map[string]any{
		"to":       []string{"test@example.com"},
		"from":     "sender@example.com",
		"subject":  "test subject",
		"template": base64.StdEncoding.EncodeToString([]byte("<h1>Hello</h1>")),
		"attachments": []map[string]any{
			{
				"contentName": "test.pdf",
				"contentType": "application.pdf", // invalid — missing "/"
				"attachment":  base64.StdEncoding.EncodeToString([]byte{44, 33, 55, 22, 1}),
			},
		},
	})
	assertResponse(t, rr, http.StatusBadRequest, ERR_INVALID_CONTENT_TYPE)
}

// TestInvalidBody tests that a malformed JSON body returns 400.
func TestInvalidBody(t *testing.T) {
	h := newTestHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/send-email", bytes.NewBufferString("NOT JSON"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.SendEmail(rr, req)
	assertResponse(t, rr, http.StatusBadRequest, "")
}

// TestHappyPath confirms that a valid request passes all
// validations and successfully simulates sending via the mock.
func TestHappyPath(t *testing.T) {
	h := newTestHandler(nil)
	rr := doPost(t, h, map[string]any{
		"to":       []string{"recipient@example.com"},
		"cc":       []string{"cc@example.com"},
		"bcc":      []string{"bcc@example.com"},
		"replyTo":  []string{"reply@example.com"},
		"from":     "sender@example.com",
		"subject":  "integration test",
		"template": base64.StdEncoding.EncodeToString([]byte("<h1>Hello</h1>")),
		"attachments": []map[string]any{
			{
				"contentName": "test.pdf",
				"contentType": "application/pdf",
				"attachment":  base64.StdEncoding.EncodeToString([]byte{12, 55, 33, 77, 34, 98, 21}),
			},
		},
	})
	assertResponse(t, rr, http.StatusOK, MSG_EMAIL_SENT_SUCCESS)
}

// TestSMTPError confirms that a mailer error returns 500 with a consistent message.
func TestSMTPError(t *testing.T) {
	h := newTestHandler(errors.New("connection refused"))
	rr := doPost(t, h, map[string]any{
		"to":       []string{"recipient@example.com"},
		"from":     "sender@example.com",
		"subject":  "test",
		"template": base64.StdEncoding.EncodeToString([]byte("<p>Hi</p>")),
	})
	assertResponse(t, rr, http.StatusInternalServerError, ERR_EMAIL_SEND_ERR)
}

// TestMaxBodySize ensures that large request bodies are rejected.
func TestMaxBodySize(t *testing.T) {
	h := NewEmailHandler(&mockMailer{err: nil}, 10) // 10-byte limit
	req := httptest.NewRequest(http.MethodPost, "/send-email", bytes.NewBufferString(`{"to":["a@b.com"]}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.SendEmail(rr, req)
	assertResponse(t, rr, http.StatusRequestEntityTooLarge, ERR_REQUEST_BODY_TOO_LARGE)
}
