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

const (
	// Generic HTTP errors.
	ERR_REQUEST_BODY_TOO_LARGE = "request body too large"
	ERR_INVALID_REQUEST_BODY   = "invalid request body"

	// HTTP headers and values.
	HEADER_CONTENT_TYPE = "Content-Type"
	CONTENT_TYPE_JSON   = "application/json"

	// Email validation errors.
	ERR_RECIPIENTS_REQUIRED  = "at least one recipient is required"
	ERR_FROM_REQUIRED        = "'from' address is required"
	ERR_INVALID_FROM         = "invalid 'from' address"
	ERR_INVALID_TO           = "invalid 'to' address"
	ERR_INVALID_CC           = "invalid 'cc' address"
	ERR_INVALID_BCC          = "invalid 'bcc' address"
	ERR_INVALID_REPLY_TO     = "invalid 'replyTo' address"
	ERR_SUBJECT_REQUIRED     = "'subject' is required"
	ERR_INVALID_CONTENT_TYPE = "unsupported attachment content type"

	// Email send outcomes.
	ERR_EMAIL_SEND_ERR     = "failed to send email"
	MSG_EMAIL_SENT_SUCCESS = "email sent successfully"

	// Health-check status values.
	STATUS_HEALTHY   HealthStatus = "healthy"
	STATUS_UNHEALTHY HealthStatus = "unhealthy"
)
