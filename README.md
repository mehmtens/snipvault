# SnipVault

Share code and text with clean links, or keep private snippets in your personal vault.

[Open SnipVault](https://snipvault-mehmetenesaldag-8600s-projects.vercel.app) · [API docs](https://snipvault-mehmetenesaldag-8600s-projects.vercel.app/docs)

## What you can do

- Create public, unlisted, or private snippets
- Share a formatted page, raw text, or downloaded file
- Set snippets to expire automatically
- Register with email verification
- Search and filter your personal vault
- Mark important snippets as favorites
- Edit, delete, and manage your account securely

## Run locally — easiest way

You only need [Docker Desktop](https://www.docker.com/products/docker-desktop/) and a free [Brevo](https://www.brevo.com/) account for verification emails.

On Windows PowerShell:

```powershell
git clone https://github.com/mehmtens/snipvault.git
cd snipvault
.\setup.ps1
```

The setup asks for your Brevo API key and verified sender email. Database and JWT secrets are generated automatically.

Then open [http://localhost:8090](http://localhost:8090).

To stop SnipVault later:

```powershell
docker compose --env-file .env.docker down
```

Your database is preserved when the containers stop.

## Manual Docker setup

Use this path on macOS/Linux or when you prefer to configure everything yourself:

```bash
cp .env.docker.example .env.docker
# Fill in the values in .env.docker
docker compose --env-file .env.docker up --build -d
```

Required values:

- `POSTGRES_PASSWORD`: a strong local database password
- `JWT_SECRET`: at least 32 random characters
- `BREVO_API_KEY`: your Brevo transactional email API key
- `MAIL_FROM_EMAIL`: a verified Brevo sender address
- `MAIL_FROM_NAME`: sender name, normally `SnipVault`

Never commit `.env` or `.env.docker`. Both are ignored by Git.

## Development without Docker

Requirements: Go 1.26 and PostgreSQL 18.

```powershell
Copy-Item .env.example .env
notepad .env
go run ./cmd/api
```

The application runs at `http://localhost:8090`.

## Useful commands

```powershell
# Check containers
docker compose --env-file .env.docker ps

# Follow application logs
docker compose --env-file .env.docker logs -f api

# Run tests
go test ./...
```

## How it is built

- Go HTTP API and embedded HTML/CSS/JavaScript frontend
- PostgreSQL storage with automatic embedded migrations
- bcrypt passwords, HttpOnly JWT sessions, CSRF protection, and rate limiting
- Brevo transactional email for verification and password recovery
- Neon PostgreSQL and Vercel Functions in production
- GitHub Actions for tests, coverage, and Docker builds

Detailed endpoint documentation is available at `/docs` and `/openapi.yaml` after starting the application.

## Project structure

```text
cmd/api/              Application entrypoint
internal/auth/        Authentication and sessions
internal/database/    PostgreSQL stores and migrations
internal/httpapi/     API handlers and embedded frontend
internal/paste/       Snippet domain logic
internal/user/        User domain logic
serverless/           Vercel function adapter
```

## License

No license has been selected yet.
