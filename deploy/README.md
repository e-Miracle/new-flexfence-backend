# flexfence-backend deployment

API at **https://api.flexfence.app**. The container runs **nginx** (port 80) proxying to the Go API on `127.0.0.1:8080`.

## Container port on host

| Service | Host bind | Subdomain |
|---------|-----------|-----------|
| flexfence-web | `127.0.0.1:3000` | flexfence.app |
| flexfence-dashboard | `127.0.0.1:3001` | app.flexfence.app |
| flexfence-backend | `127.0.0.1:3002` | api.flexfence.app |

## Server setup

```bash
sudo mkdir -p /opt/flexfence/backend
sudo cp docker-compose.prod.yml /opt/flexfence/backend/
sudo cp .env.production.example /opt/flexfence/backend/.env
# Edit .env with DB credentials, JWT_SECRET, SMTP, etc.
```

MySQL can run on the host or in a separate container; set `DB_HOST` accordingly in `.env`.

Host nginx TLS routing: see `host-nginx.example.conf` in the flexfence-web repo (covers all three subdomains).

## GitHub secrets

| Secret | Description |
|--------|-------------|
| `DEPLOY_HOST` | Server IP or hostname |
| `DEPLOY_USER` | SSH user |
| `DEPLOY_SSH_KEY` | SSH private key |
| `DEPLOY_APP_DIR` | Optional; default `/opt/flexfence/backend` |

Ensure production `.env` on the server sets:

- `CORS_ALLOWED_ORIGINS=https://app.flexfence.app,https://flexfence.app`
- `DASHBOARD_URL=https://app.flexfence.app`
- `JOIN_PUBLIC_BASE_URL=https://app.flexfence.app`
