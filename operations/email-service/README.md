# Go Email Service

A robust, simple, and dependency-free (other than the standard library) HTTP microservice built in Go for dispatching emails through an SMTP server.

## Features

- **Standardized Architecture**: Adheres strictly to the standard Go project layout with `cmd` and `internal` separation.
- **SMTP Gateway**: Secure email sending utilizing native STARTTLS upgrading.
- **RESTful Endpoints**: Delivers `/send-email` for dispatch and `/health-check` for verifying SMTP uptime.
- **Highly Testable**: Full abstraction through Mailer interfaces enabling fast, reliable mocking and testing.

## Endpoints

### `GET /health-check`

Checks if the service is running and if the upstream SMTP server is reachable via a `Ping()`.

**Response**
- `200 OK`
  ```json
  {
    "status": "healthy"
  }
  ```
- `503 Service Unavailable`
  ```json
  {
    "status": "unhealthy"
  }
  ```

### `POST /send-email`

Sends an email with an HTML template and optional attachments.

**Request Body (JSON)**:
```json
{
  "to":          ["addr@example.com"],          // required, non-empty
  "cc":          ["cc@example.com"],            // optional
  "bcc":         ["bcc@example.com"],           // optional
  "replyTo":     ["reply@example.com"],         // optional
  "from":        "sender@example.com",          // required, non-empty
  "subject":     "Hello",                       // required
  "template":    "<base64-encoded HTML>",       // required
  "attachments": [{                             // optional
    "contentName": "file.pdf",
    "contentType": "application/pdf",
    "attachment":  "<base64-encoded bytes>"
  }]
}
```

**Response**:
- `200 OK`
  ```json
  {
    "message": "Email sent successfully"
  }
  ```
- `413 Payload Too Large`
  ```json
  {
    "message": "request body too large"
  }
  ```

## Configuration

This service configures itself primarily via environment variables. It prioritizes OS environment variables, enabling seamless deployment in containerized environments (Kubernetes, Docker, Choreo), and will fall back to loading from a `.env` file at the root of the project if one is present.

| Variable | Description | Default |
| -------- | ----------- | ------- |
| `SMTP_HOSTNAME` | The SMTP server host (e.g. email-smtp.us-east-1.amazonaws.com) | **Required** |
| `SMTP_USERNAME` | The SMTP username or access key | **Required** |
| `SMTP_PASSWORD` | The SMTP password or secret key | **Required** |
| `SMTP_PORT` | The SMTP connection port | `587` |
| `PORT` | The HTTP listening port for the service | `9090` |
| `HTTP_READ_HEADER_TIMEOUT` | Timeout for reading HTTP request headers | `5s` |
| `HTTP_READ_TIMEOUT` | Timeout for reading the entire HTTP request | `10s` |
| `HTTP_WRITE_TIMEOUT` | Timeout for writing the HTTP response | `10s` |
| `HTTP_IDLE_TIMEOUT` | Timeout for keep-alive HTTP connections | `120s` |
| `MAX_REQUEST_BODY_SIZE` | Maximum allowed request body size in bytes | `10485760` (10MB) |

An example `.env.example` file is provided in the repository. Provide your own `.env` file with these values before running.

## Running Locally

To build and run the service locally:

```bash
# copy the env example
cp .env.example .env

# populate your credentials in .env

# start the service
go run cmd/email-service/main.go
```

## Testing

To run the unit tests (which employ a mocked Mailer without triggering network dial timeouts):

```bash
go test -v ./...
```
