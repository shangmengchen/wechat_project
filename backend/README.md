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

The server listens on `http://127.0.0.1:8080` by default. Tables are created automatically on startup, and demo data is seeded when the database is empty.

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
- `PATCH /api/v1/goals/{id}/status`
- `DELETE /api/v1/goals/{id}`
