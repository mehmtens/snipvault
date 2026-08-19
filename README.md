# SnipVault

SnipVault is a secure Pastebin-style application for storing and sharing code or text through short, durable links. It combines a Go HTTP API, PostgreSQL persistence, an embedded responsive frontend, owner-level authorization, expiration policies, and production-oriented operations.

**Live application:** [snipvault-mehmetenesaldag-8600s-projects.vercel.app](https://snipvault-mehmetenesaldag-8600s-projects.vercel.app)

## Highlights

- Public, unlisted, and private pastes
- Secure random slugs and raw-text links
- Syntax highlighting for ten languages
- Registration and login with bcrypt password hashing
- HttpOnly JWT session cookies, SameSite protection, and CSRF validation
- Owner-only listing, editing, and deletion
- Paste expiration and background database cleanup
- IP rate limiting and stricter authentication limits
- CSP and defensive HTTP headers
- Structured JSON request logs and request IDs
- Graceful shutdown and server timeouts
- Multi-stage Docker image and PostgreSQL Compose stack
- GitHub Actions quality and image-build checks

## Architecture

```text
Browser / API client
        │
        ▼
Go HTTP server
  ├── Security headers, request IDs, logging, rate limiting
  ├── Authentication and CSRF validation
  ├── Paste and user handlers
  └── Embedded HTML, CSS, and JavaScript
        │
        ▼
PostgreSQL
  ├── users
  └── pastes
        ▲
        │
Expiration cleanup worker
```

Database migrations are embedded into the binary and run idempotently at startup.

Production runs as a Go Vercel Function in Frankfurt with a pooled Neon PostgreSQL 18 connection. Database migrations use a PostgreSQL advisory lock so concurrent serverless cold starts remain safe. The long-running cleanup worker is used by the Docker service; serverless reads exclude expired pastes directly in SQL.

## Quick start with Docker

Requirements: Docker Desktop with Docker Compose.

```powershell
Copy-Item .env.docker.example .env.docker
notepad .env.docker
docker compose --env-file .env.docker up --build -d
```

Open:

- Application: `http://localhost:8090`
- API documentation: `http://localhost:8090/docs`
- OpenAPI specification: `http://localhost:8090/openapi.yaml`
- Health endpoint: `http://localhost:8090/health`

Inspect or stop the stack:

```powershell
docker compose --env-file .env.docker ps
docker compose --env-file .env.docker down
```

Normal `down` preserves PostgreSQL data.

## Local development

Requirements: Go 1.26 and PostgreSQL 18.

```powershell
Copy-Item .env.example .env
notepad .env
go run ./cmd/api
```

| Variable | Purpose | Example |
| --- | --- | --- |
| `DATABASE_URL` | PostgreSQL connection URL | `postgres://postgres:password@localhost:5432/snipvault?sslmode=disable` |
| `JWT_SECRET` | JWT HMAC key; minimum 32 characters | Random 256-bit value |
| `PORT` | HTTP listen port | `8090` |
| `CLEANUP_INTERVAL` | Expired-paste cleanup frequency | `5m` |

Never commit `.env` or `.env.docker`. Both are ignored by Git and excluded from Docker build context.

## API overview

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/health` | Service health |
| `POST` | `/register` | Create account and session |
| `POST` | `/login` | Start session |
| `POST` | `/logout` | End session |
| `POST` | `/pastes` | Create paste |
| `GET` | `/pastes/{slug}` | Read visible paste |
| `GET` | `/raw/{slug}` | Read raw content |
| `PUT` | `/pastes/{slug}` | Update owned paste |
| `DELETE` | `/pastes/{slug}` | Delete owned paste |
| `GET` | `/me/pastes` | List owned active pastes |

Browser authentication uses an HttpOnly session cookie and `X-CSRF-Token` for state-changing requests. Programmatic clients can use `Authorization: Bearer <token>`.

## Security model

- Passwords are hashed with bcrypt and never returned by the API.
- Session JWTs expire after 24 hours and are stored in HttpOnly cookies.
- Private paste lookups return `404` to avoid revealing existence.
- Cookie-based state changes require constant-time CSRF validation.
- API bodies are limited to 1 MB and reject unknown JSON fields.
- Rate limiting is applied per client IP; authentication uses a stricter limit.
- Content is HTML-escaped before syntax highlighting.
- CSP, frame protection, MIME protection, permissions policy, and referrer policy are enabled.

For public deployment, terminate TLS at a trusted reverse proxy and configure trusted proxy handling before using forwarded client-IP headers.

## Tests and quality

```powershell
gofmt -w .
go vet ./...
go test ./...
& go 'test' '-covermode=atomic' '-coverprofile=coverage.out' './...'
& go 'tool' 'cover' '-func=coverage.out'
```

GitHub Actions checks formatting, static analysis, race-enabled tests, coverage, and the production Docker build. Dependabot monitors Go modules, Docker images, and Actions dependencies.

## Project layout

```text
cmd/api/                 Application entrypoint
internal/auth/           Password and JWT service
internal/cleanup/        Expiration worker
internal/database/       PostgreSQL stores and migrations
internal/httpapi/        HTTP handlers and embedded frontend
internal/middleware/     Rate limiting, logging, security headers
internal/paste/          Paste domain and store contract
internal/user/           User domain and store contract
.github/workflows/       CI pipeline
compose.yaml             API and PostgreSQL stack
Dockerfile               Production image
```

## License

No license has been selected yet. Add one before accepting external contributions.
