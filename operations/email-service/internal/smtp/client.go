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
	Port     string
}

// Client is a reusable SMTP sender.
type Client struct {
	cfg Config
}

// New creates a new Client using the provided Config.
func New(cfg Config) *Client {
	if cfg.Port == "" {
		cfg.Port = PORT_STARTTLS
	}
	return &Client{cfg: cfg}
}

// dialAndAuth dials the SMTP server, upgrades to TLS via STARTTLS, and
// authenticates. It returns an smtp.Client and a cleanup function.
func (c *Client) dialAndAuth(ctx context.Context) (*smtp.Client, func(), error) {
	addr := net.JoinHostPort(c.cfg.Hostname, c.cfg.Port)

	var conn net.Conn
	var err error

	tlsCfg := &tls.Config{
		ServerName: c.cfg.Hostname,
		MinVersion: tls.VersionTLS12,
	}

	if c.cfg.Port == PORT_SMTPS {
		// Immediate TLS dial for port 465.
		var d tls.Dialer
		d.Config = tlsCfg
		conn, err = d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, nil, fmt.Errorf(ERR_FMT_TLS_DIAL, err)
		}
	} else {
		// Regular TCP dial followed by STARTTLS (standard for 587).
		var d net.Dialer
		conn, err = d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, nil, fmt.Errorf(ERR_FMT_DIAL, err)
		}
	}

	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			conn.Close()
			return nil, nil, fmt.Errorf("set connection deadline: %w", err)
		}
	}
	stop := context.AfterFunc(ctx, func() {
		conn.SetDeadline(time.Now())
	})

	sc, err := smtp.NewClient(conn, c.cfg.Hostname)
	if err != nil {
		stop()
		conn.Close()
		return nil, nil, fmt.Errorf(ERR_FMT_NEW_CLIENT, err)
	}

	cleanup := func() {
		if stop != nil {
			stop()
		}
		sc.Close()
	}

	// Upgrade to TLS via STARTTLS if not already on an encrypted connection.
	if c.cfg.Port != PORT_SMTPS {
		if err = sc.StartTLS(tlsCfg); err != nil {
			cleanup()
			return nil, nil, fmt.Errorf(ERR_FMT_STARTTLS, err)
		}
	}

	auth := smtp.PlainAuth("", c.cfg.Username, c.cfg.Password, c.cfg.Hostname)
	if err = sc.Auth(auth); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf(ERR_FMT_AUTH, err)
	}

	return sc, cleanup, nil
}

// Ping verifies that the SMTP server is reachable and credentials are accepted.
// It performs a full handshake and returns nil if the server is healthy.
func (c *Client) Ping(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DEFAULT_PING_TIMEOUT)
		defer cancel()
	}

	sc, cleanup, err := c.dialAndAuth(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	return sc.Quit()
}

// SendEmail builds a multipart/mixed MIME message and sends it via STARTTLS.
func (c *Client) SendEmail(ctx context.Context, msg *Message) error {
	raw, err := buildMIMEMessage(msg)
	if err != nil {
		return fmt.Errorf(ERR_FMT_BUILD_MIME, err)
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DEFAULT_SEND_TIMEOUT)
		defer cancel()
	}

	sc, cleanup, err := c.dialAndAuth(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	// Parse From address for envelope.
	envelopeFrom := msg.From
	if parsed, err := mail.ParseAddress(msg.From); err == nil {
		envelopeFrom = parsed.Address
	}

	// Set envelope sender.
	if err = sc.Mail(envelopeFrom); err != nil {
		return fmt.Errorf(ERR_FMT_MAIL_FROM, err)
	}

	// Add all recipients (To + CC + BCC).
	allRecipients := make([]string, 0, len(msg.To)+len(msg.CC)+len(msg.BCC))
	allRecipients = append(allRecipients, msg.To...)
	allRecipients = append(allRecipients, msg.CC...)
	allRecipients = append(allRecipients, msg.BCC...)
	for _, rcpt := range allRecipients {
		envelopeRcpt := rcpt
		if parsed, err := mail.ParseAddress(rcpt); err == nil {
			envelopeRcpt = parsed.Address
		}
		if err = sc.Rcpt(envelopeRcpt); err != nil {
			return fmt.Errorf(ERR_FMT_RCPT_TO, err)
		}
	}

	// Stream the message body.
	wc, err := sc.Data()
	if err != nil {
		return fmt.Errorf(ERR_FMT_DATA, err)
	}
	if _, err = wc.Write(raw); err != nil {
		wc.Close()
		return fmt.Errorf(ERR_FMT_WRITE_BODY, err)
	}
	return wc.Close()
}

// buildMIMEMessage constructs a multipart/mixed MIME email message.
func buildMIMEMessage(msg *Message) ([]byte, error) {
	var buf bytes.Buffer

	mixedWriter := multipart.NewWriter(&buf)

	writeHeader := func(key, value string) {
		fmt.Fprintf(&buf, "%s: %s%s", key, value, CRLF)
	}

	writeHeader(HEADER_MIME_VERSION, MIME_VERSION)
	writeHeader(HEADER_DATE, time.Now().UTC().Format(time.RFC1123Z))
	writeHeader(HEADER_FROM, msg.From)
	writeHeader(HEADER_TO, strings.Join(msg.To, ", "))

	if len(msg.CC) > 0 {
		writeHeader(HEADER_CC, strings.Join(msg.CC, ", "))
	}
	if len(msg.ReplyTo) > 0 {
		writeHeader(HEADER_REPLY_TO, strings.Join(msg.ReplyTo, ", "))
	}

	writeHeader(HEADER_SUBJECT, mime.QEncoding.Encode(MIME_CHARSET_UTF8, msg.Subject))
	writeHeader(HEADER_CONTENT_TYPE, fmt.Sprintf(`%s; boundary="%s"`, MIME_TYPE_MULTIPART_MIXED, mixedWriter.Boundary()))

	// Blank line separates headers from body.
	buf.WriteString(CRLF)

	htmlPartHeader := textproto.MIMEHeader{}
	htmlPartHeader.Set(HEADER_CONTENT_TYPE, MIME_TYPE_TEXT_HTML+`; charset="`+MIME_CHARSET_UTF8+`"`)
	htmlPartHeader.Set(HEADER_CONTENT_TRANSFER_ENCODING, MIME_ENCODING_QP)

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
		contentType := mime.FormatMediaType(att.ContentType, nil)
		if contentType == "" {
			return nil, fmt.Errorf(ERR_FMT_INVALID_ATTACH_TYPE, att.ContentType)
		}
		attHeader.Set(HEADER_CONTENT_TYPE, contentType)
		attHeader.Set(HEADER_CONTENT_TRANSFER_ENCODING, MIME_ENCODING_BASE64)
		disposition := mime.FormatMediaType("attachment", map[string]string{"filename": att.ContentName})
		if disposition == "" {
			return nil, fmt.Errorf(ERR_FMT_CONTENT_DISP, att.ContentName)
		}
		attHeader.Set(HEADER_CONTENT_DISPOSITION, disposition)

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
			if _, werr := attPart.Write([]byte(encoded[i:end] + CRLF)); werr != nil {
				return nil, fmt.Errorf(ERR_FMT_WRITE_ATTACHMENT, att.ContentName, werr)
			}
		}
	}

	if err = mixedWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ValidateMIMEType returns an error if contentType is not a valid MIME type.
// A valid MIME type must have the form "type/subtype" (e.g. "application/pdf").
func ValidateMIMEType(contentType string) error {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return fmt.Errorf(ERR_FMT_INVALID_MIME_TYPE, contentType, err)
	}
	// mediaType must contain exactly one "/" separating type and subtype.
	if !strings.Contains(mediaType, "/") {
		return fmt.Errorf(ERR_FMT_MISSING_SUBTYPE, contentType)
	}
	return nil
}
