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
	// Default context timeouts applied when the caller supplies no deadline.
	DEFAULT_PING_TIMEOUT = 10 * time.Second
	DEFAULT_SEND_TIMEOUT = 30 * time.Second

	// SMTP ports.
	PORT_SMTPS    = "465"
	PORT_STARTTLS = "587"

	// Connection and handshake — shared by Ping and SendEmail.
	ERR_FMT_DIAL       = "dial SMTP server: %w"
	ERR_FMT_TLS_DIAL   = "TLS dial SMTP server: %w"
	ERR_FMT_NEW_CLIENT = "create SMTP client: %w"
	ERR_FMT_STARTTLS   = "STARTTLS: %w"
	ERR_FMT_AUTH       = "SMTP auth: %w"

	// SendEmail envelope and data phases.
	ERR_FMT_BUILD_MIME = "build MIME message: %w"
	ERR_FMT_MAIL_FROM  = "MAIL FROM: %w"
	ERR_FMT_RCPT_TO    = "RCPT TO <%s>: %w"
	ERR_FMT_DATA       = "DATA command: %w"
	ERR_FMT_WRITE_BODY = "write message body: %w"

	// MIME message construction.
	ERR_FMT_INVALID_MIME_TYPE = "invalid MIME type %q: %w"
	ERR_FMT_MISSING_SUBTYPE   = "invalid MIME type %q: missing subtype"
	ERR_FMT_CONTENT_DISP      = "could not format Content-Disposition for attachment %q"
	ERR_FMT_WRITE_ATTACHMENT  = "write attachment data for %q: %w"

	// MIME content values written by buildMIMEMessage.
	MIME_VERSION              = "1.0"
	MIME_CHARSET_UTF8         = "UTF-8"
	MIME_TYPE_MULTIPART_MIXED = "multipart/mixed"
	MIME_TYPE_TEXT_HTML       = "text/html"
	MIME_ENCODING_QP          = "quoted-printable"
	MIME_ENCODING_BASE64      = "base64"

	// MIME headers.
	HEADER_MIME_VERSION              = "MIME-Version"
	HEADER_DATE                      = "Date"
	HEADER_FROM                      = "From"
	HEADER_TO                        = "To"
	HEADER_CC                        = "Cc"
	HEADER_REPLY_TO                  = "Reply-To"
	HEADER_SUBJECT                   = "Subject"
	HEADER_CONTENT_TYPE              = "Content-Type"
	HEADER_CONTENT_TRANSFER_ENCODING = "Content-Transfer-Encoding"
	HEADER_CONTENT_DISPOSITION       = "Content-Disposition"

	// Delimiters.
	CRLF = "\r\n"
)
