# Couple Mini Program

This workspace contains:

- `miniprogram/`: native WeChat Mini Program UI built from the provided screenshots.
- `backend/`: Gin + MySQL REST backend with seeded demo data, CRUD management, and task state transitions.

Open `miniprogram` in WeChat Developer Tools. Run the backend with:

```bash
cd backend
set MYSQL_DSN=root:password@tcp(127.0.0.1:3306)/couple_mini?charset=utf8mb4&parseTime=true&loc=Local
go run ./cmd/server
```
