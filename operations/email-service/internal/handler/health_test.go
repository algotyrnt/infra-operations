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
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHealthCheck validates the health-check handler's response to
// different SMTP server health states.
func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name             string
		mockErr          error
		wantStatus       int
		wantHealthStatus HealthStatus
	}{
		{
			name:             "Healthy status",
			mockErr:          nil,
			wantStatus:       http.StatusOK,
			wantHealthStatus: STATUS_HEALTHY,
		},
		{
			name:             "Unhealthy status",
			mockErr:          errors.New("ping failed"),
			wantStatus:       http.StatusServiceUnavailable,
			wantHealthStatus: STATUS_UNHEALTHY,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHealthHandler(&mockMailer{err: tt.mockErr})
			req := httptest.NewRequest(http.MethodGet, "/health-check", nil)
			rr := httptest.NewRecorder()
			h.HealthCheck(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", rr.Code, tt.wantStatus)
			}

			var resp HealthResponse
			rawBody := rr.Body.String()
			if err := json.Unmarshal([]byte(rawBody), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Status != tt.wantHealthStatus {
				t.Errorf("health status: got %v, want %v", resp.Status, tt.wantHealthStatus)
			}

			// Verify JSON serialization
			var wantStr string
			if tt.wantHealthStatus == STATUS_HEALTHY {
				wantStr = "healthy"
			} else {
				wantStr = "unhealthy"
			}

			if !strings.Contains(rawBody, wantStr) {
				t.Errorf("JSON output did not contain expected string %q: %s", wantStr, rawBody)
			}
		})
	}
}
