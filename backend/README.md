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

## Unified Start

Use one script as the main entry point:

```powershell
.\run.ps1 -Rebuild
```

```bash
./run.sh --build
```

By default this will:

- create `scripts/.env.docker` from the template if needed
- create `logs/admin`, `logs/backend` and `uploads`
- start `mysql + backend` with Docker
- make the admin page available at `/admin`

For cloud deployment with HTTPS proxy:

```powershell
.\run.ps1 -Rebuild -WithProxy -AppDomain api.example.com -LetsEncryptEmail ops@example.com -AdminUsername admin -AdminPassword change_me
```

```bash
./run.sh --build --proxy --app-domain api.example.com --letsencrypt-email ops@example.com --admin-username admin --admin-password change_me
```

## Compatibility Scripts

The older scripts are now thin wrappers around `run.ps1` / `run.sh`:

```text
scripts/start.ps1
scripts/start-admin.ps1
scripts/start.sh
scripts/start-admin.sh
docker/docker-start.ps1
docker/docker-start.sh
```

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

- App log: `logs/backend/app.log`
- Access log: `logs/backend/access.log`
- Error-only log: `logs/backend/error.log`

Each request gets an `X-Request-ID` response header, and both HTTP failures and panic recovery logs include that ID for troubleshooting. GORM logs SQL errors and slow queries with `LOG_SQL_LEVEL` and `LOG_SQL_SLOW_MS`.

Useful log env vars:

```bash
LOG_LEVEL=info
LOG_FORMAT=json
LOG_DIR=logs/backend
LOG_APP_FILE=app.log
LOG_ACCESS_FILE=access.log
LOG_ERROR_FILE=error.log
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

You can also keep the defaults in `configs/config.yml` and change them there:

```yaml
admin:
  enabled: true
  username: "admin"
  password: "admin123456"
  title: "Couple Mini Admin"
```

Priority is:

1. Script parameters such as `.\run.ps1 -AdminUsername ops -AdminPassword your_password`
2. Environment variables such as `ADMIN_USERNAME` and `ADMIN_PASSWORD`
3. `configs/config.yml`

Please change the admin password before deployment.

Examples:

```powershell
.\run.ps1 -Rebuild -BackendPort 18080 -AdminUsername admin -AdminPassword change_me
```

```bash
./run.sh --build --backend-port 18080 --admin-username admin --admin-password change_me
```

## Starter Deployment

The starter cloud deployment files are included:

- `docker/Dockerfile`
- `docker/docker-compose.yml`
- `docker/Caddyfile`
- `scripts/.env.example`
- `scripts/.env.docker.example`
- `DEPLOY.md`

Basic flow:

1. Copy `scripts/.env.example` to `scripts/.env` and fill in your real domain and passwords.
2. Run `./run.sh --build --proxy` or `.\run.ps1 -Rebuild -WithProxy`.
3. Add `https://your-domain` to the WeChat legal request domain list.
4. Check logs with `docker compose logs -f backend` and `docker compose logs -f caddy`.

Cloud deployment now includes the admin console in the Docker image build, and the backend container reads:

- `ADMIN_ENABLED`
- `ADMIN_USERNAME`
- `ADMIN_PASSWORD`
- `ADMIN_TITLE`
- `ADMIN_SAMPLE_INTERVAL_SEC`
- `ADMIN_HISTORY_LIMIT`

After deployment, the admin page is available at `https://your-domain/admin`.

## One-Click Behavior

Before running, make sure Docker Desktop or the Docker daemon is already running.

The unified script will auto-create the matching env file, patch values from the command line when you pass them, and then start the full stack you asked for.

Directory layout after the cleanup:

```text
backend/
  docker/
  scripts/
  logs/
    admin/
    backend/
```

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
