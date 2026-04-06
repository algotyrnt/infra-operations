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
package main

import (
	"bufio"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/wso2-open-operations/infra-operations/operations/email-service/internal/handler"
	smtpclient "github.com/wso2-open-operations/infra-operations/operations/email-service/internal/smtp"
)

func main() {
	loadDotEnv(".env")

	hostname := mustEnv("SMTP_HOSTNAME")
	username := mustEnv("SMTP_USERNAME")
	password := mustEnv("SMTP_PASSWORD")
	smtpPort := envOrDefault("SMTP_PORT", "587")
	httpPort := envOrDefault("PORT", "9090")

	client := smtpclient.New(smtpclient.Config{
		Hostname: hostname,
		Username: username,
		Password: password,
		Port:     smtpPort,
	})

	emailHandler := handler.NewEmailHandler(client)
	healthHandler := handler.NewHealthHandler(client)

	mux := http.NewServeMux()
	mux.HandleFunc("/send-email", emailHandler.SendEmail)
	mux.HandleFunc("/health-check", healthHandler.HealthCheck)

	addr := ":" + httpPort
	slog.Info("email-service starting", "addr", addr, "smtp_host", hostname, "smtp_port", smtpPort)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		slog.Error("server exited", "error", err)
		os.Exit(1)
	}
}

// mustEnv returns the value of the named environment variable or exits with
// a clear error message if it is not set.
func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required environment variable not set", "key", key)
		os.Exit(1)
	}
	return v
}

// envOrDefault returns the value of the named environment variable, or
// defaultValue if it is not set.
func envOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// loadDotEnv reads KEY=VALUE pairs from the named file to the environment.
// It skips existing keys so that real environment variables win.
func loadDotEnv(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		return // file absent — not an error
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip blank lines and comments.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Split on the first '=' only so values may contain '='.
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Only set if not already present in the environment.
		if os.Getenv(key) == "" {
			os.Setenv(key, value) //nolint:errcheck
		}
	}
}
