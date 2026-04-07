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
package smtpclient

import (
	"strings"
	"testing"
)

// TestNew_DefaultPort verifies that the SMTP client defaults to port 587.
func TestNew_DefaultPort(t *testing.T) {
	c := New(Config{Hostname: "smtp.example.com", Username: "user", Password: "pass"})
	if c.cfg.Port != "587" {
		t.Errorf("expected default port 587, got %q", c.cfg.Port)
	}
}

// TestNew_CustomPort verifies that custom SMTP ports are correctly set.
func TestNew_CustomPort(t *testing.T) {
	c := New(Config{Hostname: "smtp.example.com", Port: "465"})
	if c.cfg.Port != "465" {
		t.Errorf("expected port 465, got %q", c.cfg.Port)
	}
}

// TestValidateMIMEType ensures that only properly formatted MIME types are accepted.
func TestValidateMIMEType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		wantErr     bool
	}{
		{"valid pdf", "application/pdf", false},
		{"valid html with params", "text/html; charset=utf-8", false},
		{"valid octet-stream", "application/octet-stream", false},
		{"valid jpeg", "image/jpeg", false},
		{"valid plain text", "text/plain", false},
		{"missing slash — bare word", "applicationpdf", true},
		{"type without subtype", "application", true},
		{"type with trailing slash", "image/", true},
		{"subtype with leading slash", "/jpeg", true},
		{"multiple slashes", "application/pdf/extra", true},
		{"dot separator instead of slash", "application.pdf", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMIMEType(tt.contentType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMIMEType(%q) error = %v, wantErr %v", tt.contentType, err, tt.wantErr)
			}
		})
	}
}

// TestBuildMIMEMessage_HTMLOnly tests MIME generation for a simple HTML email.
func TestBuildMIMEMessage_HTMLOnly(t *testing.T) {
	msg := &Message{
		To:       []string{"to@example.com"},
		From:     "sender@example.com",
		Subject:  "Test Subject",
		HTMLBody: "<h1>Hello</h1>",
	}
	raw, err := buildMIMEMessage(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(raw)

	for _, want := range []string{
		"MIME-Version: " + MIME_VERSION,
		"From: sender@example.com",
		"To: to@example.com",
		MIME_TYPE_MULTIPART_MIXED,
		MIME_TYPE_TEXT_HTML,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

// TestBuildMIMEMessage_CCAndReplyTo ensures CC and Reply-To headers are correctly included.
func TestBuildMIMEMessage_CCAndReplyTo(t *testing.T) {
	msg := &Message{
		To:       []string{"to@example.com"},
		CC:       []string{"cc@example.com"},
		ReplyTo:  []string{"reply@example.com"},
		From:     "sender@example.com",
		Subject:  "CC Test",
		HTMLBody: "<p>Hello</p>",
	}
	raw, err := buildMIMEMessage(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(raw)

	if !strings.Contains(s, "Cc: cc@example.com") {
		t.Error("missing Cc header")
	}
	if !strings.Contains(s, "Reply-To: reply@example.com") {
		t.Error("missing Reply-To header")
	}
}

// TestBuildMIMEMessage_NoCCOrReplyTo ensures optional headers are absent when empty.
func TestBuildMIMEMessage_NoCCOrReplyTo(t *testing.T) {
	// Cc and Reply-To headers must be absent when the fields are empty.
	msg := &Message{
		To:       []string{"to@example.com"},
		From:     "sender@example.com",
		Subject:  "No CC",
		HTMLBody: "<p>body</p>",
	}
	raw, err := buildMIMEMessage(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(raw)

	if strings.Contains(s, "\r\nCc:") {
		t.Error("Cc header should not be present when CC is empty")
	}
	if strings.Contains(s, "\r\nReply-To:") {
		t.Error("Reply-To header should not be present when ReplyTo is empty")
	}
}

// TestBuildMIMEMessage_WithAttachment validates MIME structure for emails with attachments.
func TestBuildMIMEMessage_WithAttachment(t *testing.T) {
	msg := &Message{
		To:       []string{"to@example.com"},
		From:     "sender@example.com",
		Subject:  "With Attachment",
		HTMLBody: "<p>See attached</p>",
		Attachments: []Attachment{
			{ContentName: "doc.pdf", ContentType: "application/pdf", Data: []byte{1, 2, 3, 4, 5}},
		},
	}
	raw, err := buildMIMEMessage(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(raw)

	if !strings.Contains(s, "application/pdf") {
		t.Error("missing attachment MIME type")
	}
	if !strings.Contains(s, "doc.pdf") {
		t.Error("missing attachment filename in Content-Disposition")
	}
	if !strings.Contains(s, MIME_ENCODING_BASE64) {
		t.Error("attachment must be base64 encoded")
	}
}

// TestBuildMIMEMessage_NonASCIIFilename ensures non-ASCII attachment names are handled.
func TestBuildMIMEMessage_NonASCIIFilename(t *testing.T) {
	// Non-ASCII filenames must be encoded rather than causing an error.
	msg := &Message{
		To:       []string{"to@example.com"},
		From:     "sender@example.com",
		Subject:  "Résumé",
		HTMLBody: "<p>See attached</p>",
		Attachments: []Attachment{
			{ContentName: "résumé.pdf", ContentType: "application/pdf", Data: []byte("data")},
		},
	}
	if _, err := buildMIMEMessage(msg); err != nil {
		t.Fatalf("unexpected error for non-ASCII filename: %v", err)
	}
}

// TestBuildMIMEMessage_SubjectQEncoded ensures non-ASCII subjects are RFC-2047 encoded.
func TestBuildMIMEMessage_SubjectQEncoded(t *testing.T) {
	// Non-ASCII subjects must be Q-encoded per RFC 2047.
	msg := &Message{
		To:       []string{"to@example.com"},
		From:     "sender@example.com",
		Subject:  "こんにちは",
		HTMLBody: "<p>Hello</p>",
	}
	raw, err := buildMIMEMessage(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(raw)

	// Q-encoded subjects begin with =?<charset>?
	if !strings.Contains(s, "=?"+MIME_CHARSET_UTF8+"?") {
		t.Error("non-ASCII subject should be Q-encoded")
	}
}

// TestBuildMIMEMessage_MultipleRecipients ensures the To header is correctly comma-separated.
func TestBuildMIMEMessage_MultipleRecipients(t *testing.T) {
	msg := &Message{
		To:       []string{"a@example.com", "b@example.com"},
		From:     "sender@example.com",
		Subject:  "Multi",
		HTMLBody: "<p>Hi</p>",
	}
	raw, err := buildMIMEMessage(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(raw)

	if !strings.Contains(s, "a@example.com, b@example.com") {
		t.Error("multiple To recipients should be comma-separated")
	}
}
