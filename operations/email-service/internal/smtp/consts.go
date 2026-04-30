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

import "time"

const (
	// defaultPingTimeout and defaultSendTimeout are applied when the caller
	// supplies no deadline on the context.
	defaultPingTimeout = 10 * time.Second
	defaultSendTimeout = 30 * time.Second

	// defaultHealthTTL is how long a successful SMTP health-check result is
	// cached to avoid hammering the server on every liveness probe.
	defaultHealthTTL = 30 * time.Second

	// SMTP ports. Exported because callers (e.g. main) reference them.
	PortSMTPS    = "465"
	PortSTARTTLS = "587"

	// Connection and handshake errors — shared by Ping and SendEmail.
	errFmtDial      = "dial SMTP server: %w"
	errFmtTLSDial   = "TLS dial SMTP server: %w"
	errFmtNewClient = "create SMTP client: %w"
	errFmtSTARTTLS  = "STARTTLS: %w"
	errFmtAuth      = "SMTP auth: %w"

	// SendEmail envelope and data phases.
	errFmtBuildMIME = "build MIME message: %w"
	errFmtMailFrom  = "MAIL FROM: %w"
	errFmtRcptTo    = "RCPT TO: %w"
	errFmtData      = "DATA command: %w"
	errFmtWriteBody = "write message body: %w"

	// MIME message construction.
	errFmtInvalidMIMEType   = "invalid MIME type %q: %w"
	errFmtMissingSubtype    = "invalid MIME type %q: missing subtype"
	errFmtInvalidAttachType = "invalid attachment content type: %q"
	errFmtContentDisp       = "could not format Content-Disposition for attachment %q"
	errFmtWriteAttachment   = "write attachment data for %q: %w"

	// MIME content values written by buildMIMEMessage.
	mimeVersion            = "1.0"
	mimeCharsetUTF8        = "UTF-8"
	mimeTypeMultipartMixed = "multipart/mixed"
	mimeTypeTextHTML       = "text/html"
	mimeEncodingQP         = "quoted-printable"
	mimeEncodingBase64     = "base64"

	// MIME headers.
	headerMIMEVersion             = "MIME-Version"
	headerMessageID               = "Message-ID"
	headerDate                    = "Date"
	headerFrom                    = "From"
	headerTo                      = "To"
	headerCC                      = "Cc"
	headerReplyTo                 = "Reply-To"
	headerSubject                 = "Subject"
	headerContentType             = "Content-Type"
	headerContentTransferEncoding = "Content-Transfer-Encoding"
	headerContentDisposition      = "Content-Disposition"

	// crlf is the MIME line terminator per RFC 5322.
	crlf = "\r\n"
)
