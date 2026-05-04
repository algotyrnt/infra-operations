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
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"sync"
	"time"
)

// headerSanitizer is built once at package init to avoid rebuilding its
// internal trie on every sanitizeHeader call.
var headerSanitizer = strings.NewReplacer("\r", "", "\n", "")

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
	cfg         Config
	healthMu    sync.Mutex
	lastHealthy time.Time
}

// lineWrapper wraps an io.Writer and inserts CRLF after every lineLen bytes
// written.
type lineWrapper struct {
	w       io.Writer
	lineLen int
	col     int
}

// New creates a new Client using the provided Config.
func New(cfg Config) *Client {
	if cfg.Port == "" {
		cfg.Port = PortSTARTTLS
	}
	return &Client{cfg: cfg}
}

// clone returns a deep copy of the Message.
func (m *Message) clone() Message {
	c := *m
	c.To = append([]string(nil), m.To...)
	c.CC = append([]string(nil), m.CC...)
	c.BCC = append([]string(nil), m.BCC...)
	c.ReplyTo = append([]string(nil), m.ReplyTo...)
	c.Attachments = make([]Attachment, len(m.Attachments))
	for i, a := range m.Attachments {
		ac := a
		ac.Data = append([]byte(nil), a.Data...)
		c.Attachments[i] = ac
	}
	return c
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

	if c.cfg.Port == PortSMTPS {
		// Immediate TLS dial for port 465.
		var d tls.Dialer
		d.Config = tlsCfg
		conn, err = d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, nil, fmt.Errorf(errFmtTLSDial, err)
		}
	} else {
		// Regular TCP dial followed by STARTTLS (standard for 587).
		var d net.Dialer
		conn, err = d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, nil, fmt.Errorf(errFmtDial, err)
		}
	}

	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			conn.Close()
			return nil, nil, fmt.Errorf("set connection deadline: %w", err)
		}
	}
	stop := context.AfterFunc(ctx, func() {
		// Ignore error: we only care about interrupting blocked I/O on context
		// cancellation. If SetDeadline fails, the connection is likely dead.
		conn.SetDeadline(time.Now())
	})

	sc, err := smtp.NewClient(conn, c.cfg.Hostname)
	if err != nil {
		stop()
		conn.Close()
		return nil, nil, fmt.Errorf(errFmtNewClient, err)
	}

	cleanup := func() {
		if stop != nil {
			stop()
		}
		_ = sc.Quit()
	}

	// Upgrade to TLS via STARTTLS if not already on an encrypted connection.
	if c.cfg.Port != PortSMTPS {
		if err = sc.StartTLS(tlsCfg); err != nil {
			cleanup()
			return nil, nil, fmt.Errorf(errFmtSTARTTLS, err)
		}
	}

	auth := smtp.PlainAuth("", c.cfg.Username, c.cfg.Password, c.cfg.Hostname)
	if err = sc.Auth(auth); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf(errFmtAuth, err)
	}

	return sc, cleanup, nil
}

// Ping verifies that the SMTP server is reachable and credentials are accepted.
// Results are cached for defaultHealthTTL to avoid hammering the server on
// every liveness probe. It performs a full handshake on cache miss.
func (c *Client) Ping(ctx context.Context) error {
	c.healthMu.Lock()
	defer c.healthMu.Unlock()

	if time.Since(c.lastHealthy) < defaultHealthTTL {
		return nil
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultPingTimeout)
		defer cancel()
	}

	_, cleanup, err := c.dialAndAuth(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	c.lastHealthy = time.Now()
	return nil
}

// SendEmail snapshots msg immediately on entry and sends it.
// Concurrent calls are safe; each send uses its own connection.
func (c *Client) SendEmail(ctx context.Context, msg *Message) error {
	if msg == nil {
		return fmt.Errorf("SendEmail: message must not be nil")
	}

	snap := msg.clone()

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultSendTimeout)
		defer cancel()
	}

	raw, err := buildMIMEMessage(&snap)
	if err != nil {
		return fmt.Errorf(errFmtBuildMIME, err)
	}

	sc, cleanup, err := c.dialAndAuth(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	// Parse From address for envelope.
	envelopeFrom := snap.From
	if parsed, err := mail.ParseAddress(snap.From); err == nil {
		envelopeFrom = parsed.Address
	}

	// Set envelope sender.
	if err = sc.Mail(envelopeFrom); err != nil {
		return fmt.Errorf(errFmtMailFrom, err)
	}

	// Add unique recipients (To + CC + BCC) to the envelope.
	seen := make(map[string]struct{})
	allRecipients := make([]string, 0, len(snap.To)+len(snap.CC)+len(snap.BCC))
	allRecipients = append(allRecipients, snap.To...)
	allRecipients = append(allRecipients, snap.CC...)
	allRecipients = append(allRecipients, snap.BCC...)

	for _, rcpt := range allRecipients {
		envelopeRcpt := rcpt
		if parsed, err := mail.ParseAddress(rcpt); err == nil {
			envelopeRcpt = parsed.Address
		}

		normalized := strings.ToLower(envelopeRcpt)
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}

		if err = sc.Rcpt(envelopeRcpt); err != nil {
			return fmt.Errorf(errFmtRcptTo, err)
		}
	}

	// Stream the message body.
	wc, err := sc.Data()
	if err != nil {
		return fmt.Errorf(errFmtData, err)
	}
	if _, err = wc.Write(raw); err != nil {
		wc.Close()
		return fmt.Errorf(errFmtWriteBody, err)
	}
	return wc.Close()
}

// generateMessageID returns a globally unique Message-ID per RFC 5322 §3.6.4.
// Format: <timestamp.random@hostname>
func generateMessageID(hostname string) (string, error) {
	max := new(big.Int).Lsh(big.NewInt(1), 64) // 2^64
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("generate Message-ID random component: %w", err)
	}
	ts := time.Now().UnixNano()
	return fmt.Sprintf("<%d.%s@%s>", ts, n.Text(16), hostname), nil
}

// sanitizeHeader strips CR and LF characters from a header value to prevent
// header injection attacks.
func sanitizeHeader(value string) string {
	return headerSanitizer.Replace(value)
}

// Write implements io.Writer for lineWrapper by wrapping io.Writer.
// It inserts CRLF every lineLen bytes written.
func (lw *lineWrapper) Write(p []byte) (int, error) {
	total := 0
	for len(p) > 0 {
		avail := lw.lineLen - lw.col
		if avail <= 0 {
			if _, err := lw.w.Write([]byte(crlf)); err != nil {
				return total, err
			}
			lw.col = 0
			avail = lw.lineLen
		}
		n := avail
		if n > len(p) {
			n = len(p)
		}
		written, err := lw.w.Write(p[:n])
		total += written
		lw.col += written
		p = p[written:]
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// buildMIMEMessage constructs a MIME email message.
func buildMIMEMessage(msg *Message) ([]byte, error) {
	estimatedSize := 512 + len(msg.HTMLBody)
	for _, a := range msg.Attachments {
		estimatedSize += base64.StdEncoding.EncodedLen(len(a.Data))
	}
	var buf bytes.Buffer
	buf.Grow(estimatedSize)

	writeHeader := func(key, value string) {
		fmt.Fprintf(&buf, "%s: %s%s", key, value, crlf)
	}

	// Extract the bare hostname from the From address for the Message-ID, falling
	// back to "localhost" if the address cannot be parsed.
	msgIDHost := "localhost"
	if parsed, err := mail.ParseAddress(msg.From); err == nil {
		if parts := strings.SplitN(parsed.Address, "@", 2); len(parts) == 2 {
			msgIDHost = parts[1]
		}
	}
	msgID, err := generateMessageID(msgIDHost)
	if err != nil {
		return nil, fmt.Errorf("generate Message-ID: %w", err)
	}

	writeHeader(headerMIMEVersion, mimeVersion)
	writeHeader(headerMessageID, msgID)
	writeHeader(headerDate, time.Now().UTC().Format(time.RFC1123Z))
	writeHeader(headerFrom, sanitizeHeader(msg.From))
	writeHeader(headerTo, sanitizeHeader(strings.Join(msg.To, ", ")))

	if len(msg.CC) > 0 {
		writeHeader(headerCC, sanitizeHeader(strings.Join(msg.CC, ", ")))
	}
	if len(msg.ReplyTo) > 0 {
		writeHeader(headerReplyTo, sanitizeHeader(strings.Join(msg.ReplyTo, ", ")))
	}

	writeHeader(headerSubject, mime.QEncoding.Encode(mimeCharsetUTF8, sanitizeHeader(msg.Subject)))

	if len(msg.Attachments) == 0 {
		writeHeader(headerContentType, mimeTypeTextHTML+`; charset="`+mimeCharsetUTF8+`"`)
		writeHeader(headerContentTransferEncoding, mimeEncodingQP)
		buf.WriteString(crlf)
		qpWriter := quotedprintable.NewWriter(&buf)
		if _, err = qpWriter.Write([]byte(msg.HTMLBody)); err != nil {
			return nil, err
		}
		return buf.Bytes(), qpWriter.Close()
	}

	// One or more attachments — wrap everything in multipart/mixed.
	mixedWriter := multipart.NewWriter(&buf)
	writeHeader(headerContentType, fmt.Sprintf(`%s; boundary="%s"`, mimeTypeMultipartMixed, mixedWriter.Boundary()))

	// Blank line separates headers from body.
	buf.WriteString(crlf)

	htmlPartHeader := textproto.MIMEHeader{}
	htmlPartHeader.Set(headerContentType, mimeTypeTextHTML+`; charset="`+mimeCharsetUTF8+`"`)
	htmlPartHeader.Set(headerContentTransferEncoding, mimeEncodingQP)

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
		mediatype, params, err := mime.ParseMediaType(att.ContentType)
		if err != nil {
			return nil, fmt.Errorf(errFmtInvalidAttachType, att.ContentType)
		}
		contentType := mime.FormatMediaType(mediatype, params)
		if contentType == "" {
			return nil, fmt.Errorf(errFmtInvalidAttachType, att.ContentType)
		}
		attHeader.Set(headerContentType, contentType)
		attHeader.Set(headerContentTransferEncoding, mimeEncodingBase64)
		disposition := mime.FormatMediaType("attachment", map[string]string{"filename": att.ContentName})
		if disposition == "" {
			return nil, fmt.Errorf(errFmtContentDisp, att.ContentName)
		}
		attHeader.Set(headerContentDisposition, disposition)

		attPart, err := mixedWriter.CreatePart(attHeader)
		if err != nil {
			return nil, err
		}

		lw := &lineWrapper{w: attPart, lineLen: 76}
		enc := base64.NewEncoder(base64.StdEncoding, lw)
		if _, err = enc.Write(att.Data); err != nil {
			return nil, fmt.Errorf(errFmtWriteAttachment, att.ContentName, err)
		}
		if err = enc.Close(); err != nil {
			return nil, fmt.Errorf(errFmtWriteAttachment, att.ContentName, err)
		}
		// Terminate the final (potentially partial) line.
		if lw.col > 0 {
			if _, err = attPart.Write([]byte(crlf)); err != nil {
				return nil, fmt.Errorf(errFmtWriteAttachment, att.ContentName, err)
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
		return fmt.Errorf(errFmtInvalidMIMEType, contentType, err)
	}
	// mediaType must contain exactly one "/" separating type and subtype.
	if !strings.Contains(mediaType, "/") {
		return fmt.Errorf(errFmtMissingSubtype, contentType)
	}
	return nil
}
