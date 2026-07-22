#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

ENV_FILE="$SCRIPT_DIR/.env.docker"
EXAMPLE_FILE="$SCRIPT_DIR/.env.docker.example"

if ! docker info >/dev/null 2>&1; then
  echo "Docker daemon is not running. Please start Docker Desktop or docker service first, then run ./docker-start.sh again." >&2
  exit 1
fi

if [[ ! -f "$ENV_FILE" ]]; then
  cp "$EXAMPLE_FILE" "$ENV_FILE"
  echo "Created .env.docker from template. You can edit it later if needed."
fi

mkdir -p logs uploads

COMPOSE_ARGS=(compose --env-file .env.docker up -d)
if [[ "${1:-}" == "--build" ]]; then
  COMPOSE_ARGS+=(--build)
fi
COMPOSE_ARGS+=(mysql backend)

docker "${COMPOSE_ARGS[@]}"
docker compose --env-file .env.docker ps

BACKEND_PORT="$(grep '^BACKEND_PORT=' "$ENV_FILE" | head -n 1 | cut -d '=' -f 2)"
BACKEND_PORT="${BACKEND_PORT:-8080}"

echo
echo "MySQL and backend are starting with Docker."
echo "Backend URL: http://127.0.0.1:${BACKEND_PORT}"
echo "Check logs: docker compose --env-file .env.docker logs -f backend"
