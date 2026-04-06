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
	"net/http"
	"net/http/httptest"
	"testing"

	smtpclient "github.com/wso2-open-operations/infra-operations/operations/email-service/internal/smtp"
)

type mockMailer struct {
	err error
}

func (m *mockMailer) SendEmail(ctx context.Context, msg *smtpclient.Message) error {
	return m.err
}

func (m *mockMailer) Ping(ctx context.Context) error {
	return m.err
}

// newTestHandler returns an EmailHandler using a mock Mailer.
func newTestHandler(err error) *EmailHandler {
	return NewEmailHandler(&mockMailer{err: err})
}

func doPost(t *testing.T, h *EmailHandler, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/send-email", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.SendEmail(rr, req)
	return rr
}

func decodeResponse(t *testing.T, rr *httptest.ResponseRecorder) ResponseMessage {
	t.Helper()
	var msg ResponseMessage
	if err := json.NewDecoder(rr.Body).Decode(&msg); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return msg
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

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	msg := decodeResponse(t, rr)
	want := "The 'From' email address cannot be empty"
	if msg.Message != want {
		t.Errorf("expected %q, got %q", want, msg.Message)
	}
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

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	msg := decodeResponse(t, rr)
	want := "There should be at least one recipient!"
	if msg.Message != want {
		t.Errorf("expected %q, got %q", want, msg.Message)
	}
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

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	msg := decodeResponse(t, rr)
	want := "An error occurred while decoding the email template!"
	if msg.Message != want {
		t.Errorf("expected %q, got %q", want, msg.Message)
	}
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

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	msg := decodeResponse(t, rr)
	want := "Attachment content type is not supported"
	if msg.Message != want {
		t.Errorf("expected %q, got %q", want, msg.Message)
	}
}

// TestInvalidBody tests that a malformed JSON body returns 400.
func TestInvalidBody(t *testing.T) {
	h := newTestHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/send-email", bytes.NewBufferString("NOT JSON"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.SendEmail(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// TestMethodNotAllowed ensures non-POST methods are rejected.
func TestMethodNotAllowed(t *testing.T) {
	h := newTestHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/send-email", nil)
	rr := httptest.NewRecorder()
	h.SendEmail(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
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

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	msg := decodeResponse(t, rr)
	want := "Email sent successfully"
	if msg.Message != want {
		t.Errorf("expected %q, got %q", want, msg.Message)
	}
}
