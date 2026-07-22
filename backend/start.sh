#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_PASSWORD="${MYSQL_PASSWORD-password}"
MYSQL_DATABASE="${MYSQL_DATABASE:-couple_mini}"
APP_PORT="${PORT:-8080}"

GO_BIN="$(command -v go)"
MYSQL_BIN="$(command -v mysql)"

MYSQL_ARGS=(
  --protocol=TCP
  --host="$MYSQL_HOST"
  --port="$MYSQL_PORT"
  --user="$MYSQL_USER"
)
if [[ -n "$MYSQL_PASSWORD" ]]; then
  MYSQL_ARGS+=(--password="$MYSQL_PASSWORD")
fi

"$MYSQL_BIN" "${MYSQL_ARGS[@]}" -e "CREATE DATABASE IF NOT EXISTS $MYSQL_DATABASE DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

export PORT="$APP_PORT"
if [[ -n "$MYSQL_PASSWORD" ]]; then
  export MYSQL_DSN="$MYSQL_USER:$MYSQL_PASSWORD@tcp($MYSQL_HOST:$MYSQL_PORT)/$MYSQL_DATABASE?charset=utf8mb4&parseTime=true&loc=Local"
else
  export MYSQL_DSN="$MYSQL_USER@tcp($MYSQL_HOST:$MYSQL_PORT)/$MYSQL_DATABASE?charset=utf8mb4&parseTime=true&loc=Local"
fi

mkdir -p bin
"$GO_BIN" build -trimpath -o bin/backend ./cmd/server

echo "Backend ready. Listening on http://127.0.0.1:$APP_PORT"
exec ./bin/backend
