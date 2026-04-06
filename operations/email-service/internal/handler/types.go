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

// EmailAttachment represents a file to attach to an email.
type EmailAttachment struct {
	ContentName string `json:"contentName"`
	ContentType string `json:"contentType"`
	Attachment []byte `json:"attachment"`
}

// EmailRequest is the JSON body accepted by POST /send-email.
type EmailRequest struct {
	To []string `json:"to"`
	CC []string `json:"cc,omitempty"`
	BCC []string `json:"bcc,omitempty"`
	ReplyTo []string `json:"replyTo,omitempty"`
	From string `json:"from"`
	Subject string `json:"subject"`
	Template string `json:"template"`
	Attachments []EmailAttachment `json:"attachments,omitempty"`
}

// ResponseMessage is the JSON body returned by all responses.
type ResponseMessage struct {
	Message string `json:"message"`
}

// HealthResponse is the JSON body returned by GET /health-check.
type HealthResponse struct {
	Status string `json:"status"`
}
