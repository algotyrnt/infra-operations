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
	env := loadDotEnv(".env")

	hostname := mustEnv(env, "SMTP_HOSTNAME")
	username := mustEnv(env, "SMTP_USERNAME")
	password := mustEnv(env, "SMTP_PASSWORD")
	smtpPort := envOrDefault(env, "SMTP_PORT", "587")
	httpPort := envOrDefault(env, "PORT", "9090")

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

// mustEnv returns the value of the named config key or exits with
// a clear error message if it is not set.
func mustEnv(env map[string]string, key string) string {
	v, ok := env[key]
	if !ok || v == "" {
		slog.Error("required config not set in .env", "key", key)
		os.Exit(1)
	}
	return v
}

// envOrDefault returns the value of the named config key, or
// defaultValue if it is not set.
func envOrDefault(env map[string]string, key, defaultValue string) string {
	if v, ok := env[key]; ok && v != "" {
		return v
	}
	return defaultValue
}

// loadDotEnv reads KEY=VALUE pairs from the named file.
func loadDotEnv(filename string) map[string]string {
	env := make(map[string]string)
	f, err := os.Open(filename)
	if err != nil {
		return env // file absent — return empty map
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

		env[key] = value
	}
	return env
}
