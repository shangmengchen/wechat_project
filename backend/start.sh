#!/usr/bin/env bash
set -euo pipefail

# Local development entrypoint.
# This script builds the Go binary, prepares DB-related env vars, then runs the app.
# Production deployment should use docker compose instead of this script.

# Always run relative to the backend directory so paths like ./bin remain stable.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Default local MySQL and app settings.
# These can be overridden by exporting env vars before running the script.
MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-password}"
MYSQL_DATABASE="${MYSQL_DATABASE:-couple_mini}"
APP_PORT="${PORT:-8080}"

GO_BIN="$(command -v go)"

# Export runtime env vars for the Go process.
# The app reads these values on startup to build DB connections and server config.
export PORT="$APP_PORT"
export MYSQL_HOST
export MYSQL_PORT
export MYSQL_USER
export MYSQL_PASSWORD
export MYSQL_DATABASE
export MYSQL_CREATE_DATABASE=true
export MYSQL_AUTO_MIGRATE=true
# Local runs default to seeding demo data unless the caller explicitly disables it.
export MYSQL_AUTO_SEED="${MYSQL_AUTO_SEED:-true}"

# If the password is intentionally empty, provide a DSN without the password field.
# Otherwise let the app build its normal DSN from the discrete env vars above.
if [[ -z "$MYSQL_PASSWORD" ]]; then
  export MYSQL_DSN="$MYSQL_USER@tcp($MYSQL_HOST:$MYSQL_PORT)/$MYSQL_DATABASE?charset=utf8mb4&parseTime=true&loc=Local"
else
  unset MYSQL_DSN
fi

# Build the backend into ./bin so repeated local runs do not clutter the repo root.
mkdir -p bin
"$GO_BIN" build -trimpath -o bin/backend .

# Start the compiled backend in the foreground.
echo "Backend ready. The app will auto-create the database and tables, then listen on http://127.0.0.1:$APP_PORT"
exec ./bin/backend
