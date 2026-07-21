# Couple Mini Backend

Gin + MySQL REST API for the couple mini program.

## MySQL

Create the database first:

```sql
CREATE DATABASE couple_mini DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

Then run:

```bash
set MYSQL_DSN=root:password@tcp(127.0.0.1:3306)/couple_mini?charset=utf8mb4&parseTime=true&loc=Local
go run ./cmd/server
```

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
