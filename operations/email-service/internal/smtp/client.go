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
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

// Attachment represents a file attached to an email.
type Attachment struct {
	ContentName string
	ContentType string
	Data        []byte
}

// Message encapsulates the email data to be sent.
type Message struct {
	To          []string
	CC          []string
	BCC         []string
	ReplyTo     []string
	From        string
	Subject     string
	HTMLBody    string
	Attachments []Attachment
}

// Config holds the SMTP server credentials.
type Config struct {
	Hostname string
	Username string
	Password string
	Port string
}

// Client is a reusable SMTP sender.
type Client struct {
	cfg Config
}

// New creates a new Client using the provided Config.
func New(cfg Config) *Client {
	if cfg.Port == "" {
		cfg.Port = "587"
	}
	return &Client{cfg: cfg}
}

// Ping verifies that the SMTP server is reachable and that the configured
// credentials are accepted. It dials the server, upgrades to TLS via
// STARTTLS, performs AUTH, then sends QUIT without touching the message
// pipeline. It returns nil when the server is healthy.
func (c *Client) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	addr := net.JoinHostPort(c.cfg.Hostname, c.cfg.Port)

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial SMTP server: %w", err)
	}

	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}
	stop := context.AfterFunc(ctx, func() {
		conn.SetDeadline(time.Now())
	})

	smtpClient, err := smtp.NewClient(conn, c.cfg.Hostname)
	if err != nil {
		stop()
		conn.Close()
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer stop()
	defer smtpClient.Close()

	tlsCfg := &tls.Config{
		ServerName: c.cfg.Hostname,
		MinVersion: tls.VersionTLS12,
	}
	if err = smtpClient.StartTLS(tlsCfg); err != nil {
		return fmt.Errorf("STARTTLS: %w", err)
	}

	auth := smtp.PlainAuth("", c.cfg.Username, c.cfg.Password, c.cfg.Hostname)
	if err = smtpClient.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth: %w", err)
	}

	return smtpClient.Quit()
}

// SendEmail builds a multipart/mixed MIME message and sends it via STARTTLS.
func (c *Client) SendEmail(ctx context.Context, msg *Message) error {
	raw, err := buildMIMEMessage(msg)
	if err != nil {
		return fmt.Errorf("build MIME message: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	addr := net.JoinHostPort(c.cfg.Hostname, c.cfg.Port)

	// Dial plain TCP first — STARTTLS upgrades the connection.
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial SMTP server: %w", err)
	}

	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}
	stop := context.AfterFunc(ctx, func() {
		conn.SetDeadline(time.Now())
	})

	smtpClient, err := smtp.NewClient(conn, c.cfg.Hostname)
	if err != nil {
		stop()
		conn.Close()
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer stop()
	defer smtpClient.Close()

	// Upgrade to TLS via STARTTLS.
	tlsCfg := &tls.Config{
		ServerName: c.cfg.Hostname,
		MinVersion: tls.VersionTLS12,
	}
	if err = smtpClient.StartTLS(tlsCfg); err != nil {
		return fmt.Errorf("STARTTLS: %w", err)
	}

	// Authenticate.
	auth := smtp.PlainAuth("", c.cfg.Username, c.cfg.Password, c.cfg.Hostname)
	if err = smtpClient.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth: %w", err)
	}

	// Parse From address for envelope.
	envelopeFrom := msg.From
	if parsed, err := mail.ParseAddress(msg.From); err == nil {
		envelopeFrom = parsed.Address
	}

	// Set envelope sender.
	if err = smtpClient.Mail(envelopeFrom); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}

	// Add all recipients (To + CC + BCC).
	allRecipients := append(append(msg.To, msg.CC...), msg.BCC...)
	for _, rcpt := range allRecipients {
		envelopeRcpt := rcpt
		if parsed, err := mail.ParseAddress(rcpt); err == nil {
			envelopeRcpt = parsed.Address
		}
		if err = smtpClient.Rcpt(envelopeRcpt); err != nil {
			return fmt.Errorf("RCPT TO <%s>: %w", rcpt, err)
		}
	}

	// Stream the message body.
	wc, err := smtpClient.Data()
	if err != nil {
		return fmt.Errorf("DATA command: %w", err)
	}
	if _, err = wc.Write(raw); err != nil {
		wc.Close()
		return fmt.Errorf("write message body: %w", err)
	}
	return wc.Close()
}

// buildMIMEMessage constructs a multipart/mixed MIME email message.
func buildMIMEMessage(msg *Message) ([]byte, error) {
	var buf bytes.Buffer
	mixedWriter := multipart.NewWriter(&buf)

	writeHeader := func(key, value string) {
		fmt.Fprintf(&buf, "%s: %s\r\n", key, value)
	}

	// We write headers into a scratch buffer before the multipart boundary so
	// that we can know the boundary string from mixedWriter first.
	var headerBuf bytes.Buffer
	writeHeaderTo := func(key, value string) {
		fmt.Fprintf(&headerBuf, "%s: %s\r\n", key, value)
	}
	_ = writeHeader // suppress "declared and not used" — headers go to headerBuf

	writeHeaderTo("MIME-Version", "1.0")
	writeHeaderTo("Date", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 -0700"))
	writeHeaderTo("From", msg.From)
	writeHeaderTo("To", strings.Join(msg.To, ", "))

	if len(msg.CC) > 0 {
		writeHeaderTo("Cc", strings.Join(msg.CC, ", "))
	}
	if len(msg.ReplyTo) > 0 {
		writeHeaderTo("Reply-To", strings.Join(msg.ReplyTo, ", "))
	}

	writeHeaderTo("Subject", mime.QEncoding.Encode("UTF-8", msg.Subject))
	writeHeaderTo("Content-Type", fmt.Sprintf(`multipart/mixed; boundary="%s"`, mixedWriter.Boundary()))

	// Reassemble: headers → blank line → multipart body.
	var final bytes.Buffer
	final.Write(headerBuf.Bytes())
	final.WriteString("\r\n")

	htmlPartHeader := textproto.MIMEHeader{}
	htmlPartHeader.Set("Content-Type", `text/html; charset="UTF-8"`)
	htmlPartHeader.Set("Content-Transfer-Encoding", "quoted-printable")

	htmlPart, err := mixedWriter.CreatePart(htmlPartHeader)
	if err != nil {
		return nil, err
	}
	qpWriter := quotedprintable.NewWriter(htmlPart)
	if _, err = qpWriter.Write([]byte(msg.HTMLBody)); err != nil {
		return nil, err
	}
	if err = qpWriter.Close(); err != nil {
		return nil, err
	}

	for _, att := range msg.Attachments {
		attHeader := textproto.MIMEHeader{}
		attHeader.Set("Content-Type", att.ContentType)
		attHeader.Set("Content-Transfer-Encoding", "base64")
		attHeader.Set("Content-Disposition",
			fmt.Sprintf(`attachment; filename="%s"`, att.ContentName))

		attPart, err := mixedWriter.CreatePart(attHeader)
		if err != nil {
			return nil, err
		}

		encoded := base64.StdEncoding.EncodeToString(att.Data)
		// RFC 2045 §6.8: wrap base64 at 76-character lines.
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			attPart.Write([]byte(encoded[i:end] + "\r\n")) //nolint:errcheck
		}
	}

	if err = mixedWriter.Close(); err != nil {
		return nil, err
	}

	final.Write(buf.Bytes())
	return final.Bytes(), nil
}

// ValidateMIMEType returns an error if contentType is not a valid MIME type.
// A valid MIME type must have the form "type/subtype" (e.g. "application/pdf").
// Go's mime.ParseMediaType is lenient, so we enforce the slash explicitly.
func ValidateMIMEType(contentType string) error {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return fmt.Errorf("invalid MIME type %q: %w", contentType, err)
	}
	// mediaType must contain exactly one "/" separating type and subtype.
	if !strings.Contains(mediaType, "/") {
		return fmt.Errorf("invalid MIME type %q: missing subtype", contentType)
	}
	return nil
}
