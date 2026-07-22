# Starter Deployment Guide

This is the recommended first production setup for the current project:

1. One Linux cloud VM
2. `docker compose` to manage services
3. Caddy for HTTPS and reverse proxy
4. MySQL in the same stack for the first stage
5. Backend logs and uploads mounted to host directories

## Files

- `.env.example`: environment template
- `docker-compose.yml`: service orchestration
- `Dockerfile`: backend image build
- `Caddyfile`: HTTPS reverse proxy config

## First Deploy

1. Install Docker Engine and Docker Compose Plugin on the server.
2. Upload the `backend` directory to the server.
3. Copy `.env.example` to `.env`.
4. Replace these values in `.env`:
   - `APP_DOMAIN`
   - `LETSENCRYPT_EMAIL`
   - `MYSQL_ROOT_PASSWORD`
   - `MYSQL_PASSWORD`
5. Start services:

```bash
docker compose up -d --build

If you only want database + backend first, without HTTPS reverse proxy yet:

```bash
docker compose --env-file .env.docker up -d --build mysql backend
```

Or use the one-click scripts:

```bash
./docker-start.sh --build
```
```

## Runtime Behavior

- The backend auto-creates the database when `MYSQL_CREATE_DATABASE=true`.
- The backend auto-creates or updates tables when `MYSQL_AUTO_MIGRATE=true`.
- The backend should use `MYSQL_AUTO_SEED=false` in production to avoid demo data.
- Caddy terminates HTTPS and forwards traffic to the backend container.

## Logs And Troubleshooting

Check service status:

```bash
docker compose ps
```

Check backend logs:

```bash
docker compose logs -f backend
```

Check reverse proxy logs:

```bash
docker compose logs -f caddy
```

Check MySQL logs:

```bash
docker compose logs -f mysql
```

The backend also writes:

- `./logs/app.log`
- `./logs/access.log`
- `./logs/caddy/access.log`

Each backend response carries an `X-Request-ID` header. Use that ID to correlate user-reported failures with app logs.

## WeChat Side

After the backend domain is online:

1. Add the HTTPS domain to the mini program server domain whitelist.
2. Make sure uploads also go through the same allowed HTTPS domain.
3. Complete ICP and public network compliance requirements for your server region if needed.
