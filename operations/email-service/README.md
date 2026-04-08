# Go Email Service

A robust, simple, and dependency-free (other than the standard library) HTTP microservice built in Go for dispatching emails through an SMTP server.

## Features

- **Zero external dependencies**: Built entirely on the Go standard library.
- **Standard Go layout**: Strict `cmd/` and `internal/` separation for maintainability.
- **Secure by default**: STARTTLS with TLS 1.2 minimum; context-aware timeouts on all SMTP operations.
- **Testable design**: Full Mailer interface abstraction allowing fast, mock-based unit tests with no network calls.
- **RESTful API**: `/send-email` for dispatch and `/health-check` for deep SMTP connectivity verification.
- **OpenAPI Specification**: Defined in [`openapi.yaml`](./openapi.yaml) for easy integration and documentation.

## Endpoints

### `GET /health-check`

Performs a real SMTP dial + STARTTLS + AUTH + QUIT to confirm the upstream server is reachable and credentials are valid.

**Responses**

| Status                    | Body                      |
| ------------------------- | ------------------------- |
| `200 OK`                  | `{"status": "healthy"}`   |
| `503 Service Unavailable` | `{"status": "unhealthy"}` |

### `POST /send-email`

Sends an email with an HTML body and optional file attachments.

**Request Body (JSON)**:

```json
{
  "to":          ["addr@example.com"],          // required, non-empty
  "cc":          ["cc@example.com"],            // optional
  "bcc":         ["bcc@example.com"],           // optional
  "replyTo":     ["reply@example.com"],         // optional
  "from":        "sender@example.com",          // required, non-empty
  "subject":     "Hello",                       // required
  "template":    "<base64-encoded HTML>",       // required — see note below
  "attachments": [{                             // optional
    "contentName": "file.pdf",
    "contentType": "application/pdf",
    "attachment":  "<base64-encoded bytes>"
  }]
}
```

> The HTML body is transmitted as a base64 string to avoid ambiguity when the template contains characters that clash with JSON encoding (e.g., unescaped `<`, `>`, or embedded quotes in inline scripts/styles). Callers encode the raw HTML once; this service decodes it and encodes it again as `quoted-printable` inside the MIME message, which is the standard encoding for HTML email bodies.

**Responses**

| Status                      | Body                                     |
| --------------------------- | ---------------------------------------- |
| `200 OK`                    | `{"message": "Email sent successfully"}` |
| `400 Bad Request`           | `{"message": "<validation error>"}`      |
| `413 Payload Too Large`     | `{"message": "request body too large"}`  |
| `500 Internal Server Error` | `{"message": "failed to send email"}`    |

## Configuration

The service prioritises OS environment variables, enabling seamless deployment in containerised environments (Kubernetes, Docker, Choreo). If a variable is not found in the environment it falls back to a `.env` file at the project root.

| Variable                   | Description                                    | Default            |
| -------------------------- | ---------------------------------------------- | ------------------ |
| `SMTP_HOSTNAME`            | SMTP server hostname (e.g. `smtp.example.com`) | **Required**       |
| `SMTP_USERNAME`            | SMTP username or access key                    | **Required**       |
| `SMTP_PASSWORD`            | SMTP password or secret key                    | **Required**       |
| `SMTP_PORT`                | SMTP port                                      | `587`              |
| `PORT`                     | HTTP listening port                            | `9090`             |
| `HTTP_READ_HEADER_TIMEOUT` | Timeout to read request headers                | `5s`               |
| `HTTP_READ_TIMEOUT`        | Timeout to read the full request body          | `10s`              |
| `HTTP_WRITE_TIMEOUT`       | Timeout to write the full response             | `10s`              |
| `HTTP_IDLE_TIMEOUT`        | Keep-alive idle connection timeout             | `120s`             |
| `MAX_REQUEST_BODY_SIZE`    | Maximum request body size in bytes             | `10485760` (10 MB) |
| `SHUTDOWN_TIMEOUT`         | Graceful shutdown timeout                      | `30s`              |

All timeout values accept standard Go duration strings (e.g. `5s`, `1m30s`).

An `.env.example` file is provided. Copy it to `.env` and fill in your SMTP credentials before running locally.

## Running Locally

```bash
# Copy and populate the environment file
cp .env.example .env

# Start the service
go run ./cmd/email-service/
```

## Testing

### Unit Tests

The unit tests use a mock `Mailer` — no network connections are made:

```bash
go test -race -count=1 ./...
```

## License

This project is licensed under the [Apache License 2.0](../../LICENSE). The full license text is in the repository root.
