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

## MySQL unhealthy — how to diagnose

On the server:

```bash
cd /opt/flexfence/backend

# 1. Check container status
docker compose -f docker-compose.prod.yml ps

# 2. Read MySQL logs (most useful)
docker logs flexfence-backend-mysql --tail 100

# 3. Inspect healthcheck failures
docker inspect flexfence-backend-mysql --format='{{json .State.Health}}' | jq

# 4. Confirm .env has non-empty passwords
grep -E '^(MYSQL_ROOT_PASSWORD|DB_PASSWORD|DB_USER|DB_NAME)=' .env
```

**Common causes**

| Symptom in logs | Fix |
|-----------------|-----|
| `MYSQL_ROOT_PASSWORD` / `MYSQL_PASSWORD` not set | Set both in `.env` |
| `Access denied` on healthcheck | Wrong password vs existing volume — see below |
| Slow first start | Wait 60s+ or re-run `up -d` |
| Volume initialized with old credentials | Reset volume (destroys DB data): `docker compose -f docker-compose.prod.yml down` then `docker volume rm backend_mysql_data` (volume name may be `flexfence-backend_mysql_data` — check `docker volume ls`) |

**Reset MySQL volume** (only if you can lose existing DB data):

```bash
cd /opt/flexfence/backend
docker compose -f docker-compose.prod.yml down
docker volume ls | grep mysql
docker volume rm <volume_name>
docker compose -f docker-compose.prod.yml up -d --build
```

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
