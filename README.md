# flexfence-backend

Go backend for Flexfence MVP attendance flow.

## MVP Responsibilities

- Auth session/token validation
- Event and geofence management
- Join by invite link or QR
- Mark attendee present
- Movement and entry/exit ingestion

## Run

```bash
go mod tidy
go run ./cmd/api
```

`config.Load()` reads values from `.env` automatically (via `godotenv`), with process environment variables still able to override them.

### CORS (dashboard)

Set `CORS_ALLOWED_ORIGINS` (comma-separated), e.g. `http://localhost:5173`. Defaults include Vite dev URLs.

For local dashboard dev, you can also leave `VITE_API_BASE_URL` empty and use the Vite proxy (no cross-origin requests).

### SMTP (dashboard OTP email)

Password login sends a one-time code by email. Configure SMTP in `.env` (see `.env.example`):

| Variable | Description |
|----------|-------------|
| `SMTP_HOST` | SMTP server hostname |
| `SMTP_PORT` | Usually `587` (STARTTLS) |
| `SMTP_USER` / `SMTP_PASSWORD` | Auth credentials |
| `SMTP_FROM_EMAIL` | Sender address |
| `SMTP_FROM_NAME` | Sender display name (optional) |
| `OTP_LENGTH` | Code length (default `4`) |
| `OTP_EXPIRE_MINUTES` | Validity window (default `10`) |
| `DASHBOARD_URL` | Linked in the email body |

If `SMTP_HOST` and `SMTP_FROM_EMAIL` are unset, OTP content is **logged to the API process stdout** (local dev only).

## MySQL + ORM

- ORM: GORM
- Driver: MySQL
- Storage implementation: `internal/store/mysql`
- Set env values from `.env.example`
- If `DB_AUTO_MIGRATE=true`, tables are auto-created on startup

## Docker

Run backend + MySQL from this repo:

```bash
docker compose up --build
```

Or run backend image manually:

```bash
docker run --rm -p 8080:8080 \
  -e PORT=8080 \
  -e APP_ENV=development \
  -e DB_HOST=host.docker.internal \
  -e DB_PORT=3306 \
  -e DB_USER=root \
  -e DB_PASSWORD=changeme \
  -e DB_NAME=flexfence \
  -e DB_AUTO_MIGRATE=true \
  flexfence-backend
```

## API Contracts

- Request/response DTOs live in `internal/http/dto.go`
- Centralized API error model lives in `internal/http/errors.go`
- OpenAPI spec lives in `openapi/openapi.yaml`
- CI workflow lives in `.github/workflows/ci.yml`
- Drift check script lives in `scripts/check_openapi_drift.py`

### PR contract guard

On every PR, CI will:
- validate `openapi/openapi.yaml` structure
- run `scripts/check_openapi_drift.py` to fail if handlers and spec diverge

Run locally:

```bash
python3 scripts/check_openapi_drift.py
```

### Generate typed clients

Examples using [OpenAPI Generator](https://openapi-generator.tech/):

```bash
# Dashboard (TypeScript)
openapi-generator-cli generate \
  -i openapi/openapi.yaml \
  -g typescript-fetch \
  -o ../flexfence-dashboard/src/api-client

# iOS (Swift)
openapi-generator-cli generate \
  -i openapi/openapi.yaml \
  -g swift5 \
  -o ../flexfence-ios/GeneratedAPI

# Android (Kotlin)
openapi-generator-cli generate \
  -i openapi/openapi.yaml \
  -g kotlin \
  -o ../flexfence-android/generated-api
```

## Implemented Endpoints (MVP v0)

- `GET /health`
- `POST /v1/events`
- `GET /v1/events`
- `GET /v1/events/{eventId}`
- `POST /v1/events/{eventId}/fences`
- `POST /v1/events/{eventId}/join-by-qr`
- `POST /v1/events/{eventId}/attendance/mark-present`

## Seed local data

```bash
go run ./cmd/seed
```

Creates:
- Organization `acme-events`
- Owner `owner@acme.test` / `changeme`
- Sample event + fence

## Quick Test

Business login (dashboard) — sends OTP email, returns `challenge_id`:

```bash
curl -X POST http://localhost:8080/v1/auth/business/login \
  -H "Content-Type: application/json" \
  -d '{"email":"owner@acme.test","password":"changeme"}'
```

Verify OTP (issues JWT):

```bash
curl -X POST http://localhost:8080/v1/auth/business/otp/verify \
  -H "Content-Type: application/json" \
  -d '{"challenge_id":"<challenge_id>","code":"1234"}'
```

Create event (use `access_token` from OTP verify):

```bash
curl -X POST http://localhost:8080/v1/events \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <access_token>" \
  -d '{"title":"Tech Conference","description":"Main hall event","start_at":"2026-06-01T09:00:00Z","end_at":"2026-06-01T17:00:00Z"}'
```

Mobile Google login (dev mode without `GOOGLE_CLIENT_ID`):

```bash
curl -X POST http://localhost:8080/v1/auth/user/oauth/google \
  -H "Content-Type: application/json" \
  -d '{"google_sub":"google-dev-1","email":"attendee@example.com","first_name":"Sam","last_name":"Lee"}'
```

Create fence:

```bash
curl -X POST http://localhost:8080/v1/events/evt_1/fences \
  -H "Content-Type: application/json" \
  -d '{"name":"Main Hall","shape_type":"circle","center_lat":6.5244,"center_lng":3.3792,"radius_m":120}'
```

Join by QR:

```bash
curl -X POST http://localhost:8080/v1/events/evt_1/join-by-qr \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <user_access_token>" \
  -d '{"qr_token":"qr_evt_1"}'
```

Mark present:

```bash
curl -X POST http://localhost:8080/v1/events/evt_1/attendance/mark-present \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <user_access_token>" \
  -d '{"source":"qr_scan","lat":6.5244,"lng":3.3792,"accuracy_m":14.2}'
```

## SaaS identity model

See `docs/SAAS_IDENTITY_MODEL.md` for tenant (`organizations`), dashboard users (`business_users`), and mobile attendees (`users`).

## Auth guards

See `docs/AUTH_GUARDS.md` for middleware/guard behavior, role matrix, and per-route protection.

## Structure

- `cmd/api`: application entrypoint
- `docs/SAAS_IDENTITY_MODEL.md`: multi-tenant user design
- `internal/config`: environment configuration
- `internal/db`: MySQL connection and auto-migrations
- `internal/http`: router and handlers
- `internal/domain`: domain models
- `internal/store`: storage interface + shared errors
- `internal/store/mysql`: GORM MySQL persistence
- `internal/store/memory`: in-memory persistence (optional dev fallback)
