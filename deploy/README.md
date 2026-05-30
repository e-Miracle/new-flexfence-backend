# flexfence-backend deployment

API at **https://api.flexfence.app**. The container runs **nginx** (port 80) proxying to the Go API on `127.0.0.1:8080`.

## How deploy works

On merge to `main`, GitHub Actions:

1. Packs the repo source into `deploy.tar.gz`
2. Copies it to the server over SSH
3. Runs `docker compose -f docker-compose.prod.yml build` and `up -d` **on the server**

No container registry (GHCR) is used — images are built locally on the server.

## Server setup

```bash
sudo mkdir -p /opt/flexfence/backend
sudo chown -R $USER:$USER /opt/flexfence/backend
nano /opt/flexfence/backend/.env   # required — see .env.production.example
```

**`.env` is required** for the backend (DB credentials, JWT, SMTP, etc.). CI never copies or overwrites it.

Production compose **includes its own MySQL 8.4 container** (`flexfence-backend-mysql`) with a persistent `mysql_data` volume. The API connects via Docker network hostname `mysql` — you do not need a separate MySQL install on the host. MySQL is not exposed on the host port (internal to compose only).

## Manual deploy (on server)

```bash
cd /opt/flexfence/backend
docker compose -f docker-compose.prod.yml build
docker compose -f docker-compose.prod.yml up -d
```

## GitHub secrets

| Secret | Description |
|--------|-------------|
| `DEPLOY_HOST` | Server IP or hostname |
| `DEPLOY_USER` | SSH user |
| `DEPLOY_SSH_KEY` | SSH private key |
| `DEPLOY_APP_DIR` | Optional; default `/opt/flexfence/backend` |

## Host routing

| Service | Host bind | Subdomain |
|---------|-----------|-----------|
| flexfence-web | `127.0.0.1:3000` | flexfence.app |
| flexfence-dashboard | `127.0.0.1:3001` | app.flexfence.app |
| flexfence-backend | `127.0.0.1:3002` | api.flexfence.app |

Host nginx TLS: see `host-nginx.example.conf`.
