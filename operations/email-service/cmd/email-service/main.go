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
	"strconv"
	"strings"
	"time"

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

	readHeaderTimeout := envOrDefaultDuration(env, "HTTP_READ_HEADER_TIMEOUT", 5*time.Second)
	readTimeout := envOrDefaultDuration(env, "HTTP_READ_TIMEOUT", 10*time.Second)
	writeTimeout := envOrDefaultDuration(env, "HTTP_WRITE_TIMEOUT", 10*time.Second)
	idleTimeout := envOrDefaultDuration(env, "HTTP_IDLE_TIMEOUT", 120*time.Second)

	maxRequestBodySize := envOrDefaultInt64(env, "MAX_REQUEST_BODY_SIZE", 10*1024*1024)

	client := smtpclient.New(smtpclient.Config{
		Hostname: hostname,
		Username: username,
		Password: password,
		Port:     smtpPort,
	})

	emailHandler := handler.NewEmailHandler(client, maxRequestBodySize)
	healthHandler := handler.NewHealthHandler(client)

	mux := http.NewServeMux()
	mux.HandleFunc("/send-email", emailHandler.SendEmail)
	mux.HandleFunc("/health-check", healthHandler.HealthCheck)

	addr := ":" + httpPort
	slog.Info("email-service starting", "addr", addr, "smtp_host", hostname, "smtp_port", smtpPort, "max_req_size", maxRequestBodySize)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
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

// envOrDefaultDuration returns the parsed duration of the named config key, or
// defaultValue if it is not set or fails to parse.
func envOrDefaultDuration(env map[string]string, key string, defaultValue time.Duration) time.Duration {
	v, ok := env[key]
	if !ok || v == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		slog.Warn("invalid duration for config, using default", "key", key, "value", v, "default", defaultValue, "error", err)
		return defaultValue
	}
	return d
}

// envOrDefaultInt64 parses the named config key as an int64, or
// defaultValue if it is not set or fails to parse.
func envOrDefaultInt64(env map[string]string, key string, defaultValue int64) int64 {
	v, ok := env[key]
	if !ok || v == "" {
		return defaultValue
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		slog.Warn("invalid int64 for config, using default", "key", key, "value", v, "default", defaultValue, "error", err)
		return defaultValue
	}
	return i
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
