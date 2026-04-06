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
	"log/slog"
	"net/http"
)

// HealthHandler holds the dependencies for the health-check HTTP handler.
type HealthHandler struct {
	client Mailer
}

// NewHealthHandler creates a HealthHandler backed by the given Mailer.
func NewHealthHandler(client Mailer) *HealthHandler {
	return &HealthHandler{client: client}
}

// HealthCheck handles GET /health-check.
// It pings the SMTP server to confirm the client is operational.
func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ResponseMessage{Message: "method not allowed"})
		return
	}

	if err := h.client.Ping(r.Context()); err != nil {
		slog.Error("health-check: SMTP ping failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, HealthResponse{Status: "unhealthy"})
		return
	}

	writeJSON(w, http.StatusOK, HealthResponse{Status: "healthy"})
}
