# Authentication Guards

All protected routes use middleware from `internal/http/guards.go`.

## Guard types

| Guard | Use case | Checks |
|-------|----------|--------|
| `Public` | `/health`, login endpoints | No token |
| `BusinessRead` | Dashboard reads | Business JWT, active account, org/role match DB, viewers blocked on write methods |
| `BusinessWrite` | Strict write-only routes | Same as read + must be `owner` or `admin` |
| `User` | Mobile attendance | User JWT, active `users` row |
| `AllowMethods` | Per-route verb allowlist | HTTP method |
| `RequireEventTenant` | Event sub-resources | Event belongs to JWT `organization_id` |

## Role matrix (dashboard)

| Action | owner | admin | viewer |
|--------|-------|-------|--------|
| List/get events | yes | yes | yes |
| Create event | yes | yes | no |
| Create fence | yes | yes | no |

## Route map

| Route | Guards applied |
|-------|----------------|
| `GET /health` | Public |
| `POST /v1/auth/business/login` | Public |
| `POST /v1/auth/user/oauth/google` | Public |
| `GET/POST /v1/events` | AllowMethods + BusinessRead |
| `GET /v1/events/{id}` | AllowMethods(GET) + BusinessRead + RequireEventTenant |
| `POST /v1/events/{id}/fences` | AllowMethods(POST) + BusinessRead + RequireEventTenant |
| `POST .../join-by-qr` | AllowMethods(POST) + User |
| `POST .../mark-present` | AllowMethods(POST) + User |

## Error codes

- `401 unauthorized` — missing/invalid/expired token, account not found, claim mismatch
- `403 forbidden` — wrong token type, disabled account, insufficient role
- `404 event_not_found` — event outside tenant (no cross-org leakage)
