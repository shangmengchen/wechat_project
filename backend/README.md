# Couple Mini Backend

Gin + GORM + MySQL REST API for the couple mini program.

The backend now follows a `ppm_web_go`-style layered layout:

- `main.go` / `cmd/server/main.go`
- `configs/`
- `router/`
- `api/`
- `internal/service/`
- `internal/repo/`
- `internal/model/`
- `internal/pkg/gormcli/`

## MySQL

One command for init + build + run:

```bash
# Linux
./start.sh

# Windows PowerShell
.\start.ps1
```

Override `MYSQL_HOST`, `MYSQL_PORT`, `MYSQL_USER`, `MYSQL_PASSWORD`, `MYSQL_DATABASE`, or `PORT` if your local MySQL differs from the defaults.

The server listens on `http://127.0.0.1:8080` by default. On startup the app now does these automatically:

- Creates the database when `MYSQL_CREATE_DATABASE=true`
- Runs GORM auto-migration when `MYSQL_AUTO_MIGRATE=true`
- Seeds demo data only when `MYSQL_AUTO_SEED=true` and the database is empty

Recommended local defaults:

```bash
MYSQL_CREATE_DATABASE=true
MYSQL_AUTO_MIGRATE=true
MYSQL_AUTO_SEED=true
```

Recommended server defaults:

```bash
MYSQL_CREATE_DATABASE=true
MYSQL_AUTO_MIGRATE=true
MYSQL_AUTO_SEED=false
```

## Logs

The backend now writes two log streams by default:

- App log: `logs/app.log`
- Access log: `logs/access.log`

Each request gets an `X-Request-ID` response header, and both HTTP failures and panic recovery logs include that ID for troubleshooting. GORM logs SQL errors and slow queries with `LOG_SQL_LEVEL` and `LOG_SQL_SLOW_MS`.

Useful log env vars:

```bash
LOG_LEVEL=info
LOG_FORMAT=json
LOG_DIR=logs
LOG_APP_FILE=app.log
LOG_ACCESS_FILE=access.log
LOG_TO_STDOUT=true
LOG_SQL_LEVEL=warn
LOG_SQL_SLOW_MS=500
```

## Admin Console

The backend now includes an admin console:

- Page: `GET /admin`
- API: `GET /admin/api/dashboard`
- Error logs: `GET /admin/api/errors`

The admin page is now maintained as a Vue + Vite app:

- Source: `backend/admin-ui`
- Build output: `backend/web/admin`

Build commands:

```bash
cd backend/admin-ui
npm install
npm run build
```

For local UI development:

```bash
cd backend/admin-ui
npm run dev
```

Default credentials come from env:

```bash
ADMIN_USERNAME=admin
ADMIN_PASSWORD=change_admin_password
```

Please change the admin password before deployment.

## Starter Deployment

The starter cloud deployment files are included:

- `Dockerfile`
- `docker-compose.yml`
- `.env.example`
- `.env.docker.example`
- `Caddyfile`
- `DEPLOY.md`

Basic flow:

1. Copy `.env.example` to `.env` and fill in your real domain and passwords.
2. Run `docker compose up -d --build`.
3. Add `https://your-domain` to the WeChat legal request domain list.
4. Check logs with `docker compose logs -f backend` and `docker compose logs -f caddy`.

## One-Click Docker Start

For local or first-server startup, you can now start only `mysql + backend` with one command.

Windows PowerShell:

```powershell
.\docker-start.ps1 -Rebuild
```

Linux:

```bash
./docker-start.sh --build
```

Before running, make sure Docker Desktop or the Docker daemon is already running.

Behavior:

- Auto-creates `.env.docker` from `.env.docker.example` on first run
- Starts MySQL container
- Builds and starts backend container
- Backend connects to MySQL automatically
- Backend auto-creates database and tables

By default the backend is exposed at `http://127.0.0.1:8080`.

The admin page will be available at `http://127.0.0.1:8080/admin`.

## Main Endpoints

- `POST /api/v1/auth/login`
- `POST /api/v1/pair/code`
- `POST /api/v1/pair/confirm`
- `PATCH /api/v1/couple/love-date`
- `PATCH /api/v1/users/{id}/profile`
- `GET /api/v1/dashboard`
- `POST /api/v1/uploads/images` (`multipart/form-data`, field name `file`; returns `{ url }`)
- `GET /api/v1/moments`
- `POST /api/v1/moments`
- `DELETE /api/v1/moments/{id}`
- `PATCH /api/v1/moments/{id}/liked`
- `GET /api/v1/tasks`
- `POST /api/v1/tasks`
- `DELETE /api/v1/tasks/{id}`
- `POST /api/v1/tasks/{id}/complete`
- `POST /api/v1/tasks/{id}/approve`
- `POST /api/v1/tasks/{id}/reject`
- `GET /api/v1/scheduled-tasks`
- `POST /api/v1/scheduled-tasks`
- `DELETE /api/v1/scheduled-tasks/{id}`
- `POST /api/v1/scheduled-tasks/{id}/confirm`
- `GET /api/v1/dishes`
- `POST /api/v1/dishes`
- `DELETE /api/v1/dishes/{id}`
- `PATCH /api/v1/dishes/{id}/enabled`
- `GET /api/v1/orders`
- `POST /api/v1/orders`
- `GET /api/v1/goals`
- `POST /api/v1/goals`
- `PATCH /api/v1/goals/{id}/value`
- `PATCH /api/v1/goals/{id}/status`
- `DELETE /api/v1/goals/{id}`
