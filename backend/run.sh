#!/usr/bin/env bash
if [ -z "${BASH_VERSION:-}" ]; then
  if [ -x /usr/bin/bash ]; then
    exec /usr/bin/bash "$0" "$@"
  fi
  if [ -x /bin/bash ]; then
    exec /bin/bash "$0" "$@"
  fi
  if command -v bash >/dev/null 2>&1; then
    exec "$(command -v bash)" "$0" "$@"
  fi
  echo "bash is required to run this script." >&2
  exit 1
fi

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

MODE="docker"
REBUILD=false
WITH_PROXY=false
ENV_FILE=""
APP_DOMAIN=""
LETSENCRYPT_EMAIL=""
BACKEND_PORT=""
MYSQL_HOST="${MYSQL_HOST:-}"
MYSQL_PORT="${MYSQL_PORT:-}"
MYSQL_USER="${MYSQL_USER:-}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-}"
MYSQL_DATABASE="${MYSQL_DATABASE:-}"
MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-}"
APP_PORT="${PORT:-}"
ADMIN_USERNAME="${ADMIN_USERNAME:-}"
ADMIN_PASSWORD_VALUE=""
ADMIN_PASSWORD_SET=false
ADMIN_TITLE="${ADMIN_TITLE:-}"
AUTH_TOKEN_SECRET="${AUTH_TOKEN_SECRET:-}"
AUTH_TOKEN_TTL_HOURS="${AUTH_TOKEN_TTL_HOURS:-}"
WECHAT_APP_ID="${WECHAT_APP_ID:-}"
WECHAT_SECRET="${WECHAT_SECRET:-}"
SKIP_ADMIN_BUILD=false

if [[ "${ADMIN_PASSWORD+x}" == "x" ]]; then
  ADMIN_PASSWORD_VALUE="$ADMIN_PASSWORD"
  ADMIN_PASSWORD_SET=true
fi

usage() {
  cat <<'EOF'
Usage: ./run.sh [options]

Options:
  --mode docker|local
  --build
  --proxy
  --env-file FILE
  --app-domain VALUE
  --letsencrypt-email VALUE
  --backend-port VALUE
  --mysql-host VALUE
  --mysql-port VALUE
  --mysql-user VALUE
  --mysql-password VALUE
  --mysql-root-password VALUE
  --mysql-database VALUE
  --port VALUE
  --admin-username VALUE
  --admin-password VALUE
  --admin-title VALUE
  --auth-token-secret VALUE
  --auth-token-ttl-hours VALUE
  --wechat-app-id VALUE
  --wechat-secret VALUE
  --skip-admin-build
  --help
EOF
}

set_env_value() {
  local path="$1"
  local key="$2"
  local value="$3"
  local temp_file
  temp_file="$(mktemp)"

  if [[ -f "$path" ]] && grep -q "^${key}=" "$path"; then
    awk -v key="$key" -v value="$value" '
      BEGIN { updated = 0 }
      $0 ~ ("^" key "=") { print key "=" value; updated = 1; next }
      { print }
      END { if (updated == 0) print key "=" value }
    ' "$path" >"$temp_file"
  else
    if [[ -f "$path" ]]; then
      cat "$path" >"$temp_file"
    fi
    printf '%s=%s\n' "$key" "$value" >>"$temp_file"
  fi

  mv "$temp_file" "$path"
}

get_env_value() {
  local path="$1"
  local key="$2"
  if [[ ! -f "$path" ]]; then
    return 0
  fi
  grep "^${key}=" "$path" | head -n 1 | cut -d '=' -f 2-
}

build_admin_ui() {
  local admin_ui_dir="$SCRIPT_DIR/admin-ui"
  if [[ ! -d "$admin_ui_dir" ]]; then
    echo "admin-ui directory not found: $admin_ui_dir" >&2
    exit 1
  fi

  local npm_bin
  npm_bin="$(command -v npm)"
  pushd "$admin_ui_dir" >/dev/null
  if [[ ! -d node_modules ]]; then
    "$npm_bin" install
  fi
  "$npm_bin" run build
  popd >/dev/null
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    docker|local)
      MODE="$1"
      shift
      ;;
    build)
      REBUILD=true
      shift
      ;;
    --mode)
      MODE="$2"
      shift 2
      ;;
    --build|--rebuild)
      REBUILD=true
      shift
      ;;
    --proxy)
      WITH_PROXY=true
      shift
      ;;
    --env-file)
      ENV_FILE="$2"
      shift 2
      ;;
    --app-domain)
      APP_DOMAIN="$2"
      shift 2
      ;;
    --letsencrypt-email)
      LETSENCRYPT_EMAIL="$2"
      shift 2
      ;;
    --backend-port)
      BACKEND_PORT="$2"
      shift 2
      ;;
    --mysql-host)
      MYSQL_HOST="$2"
      shift 2
      ;;
    --mysql-port)
      MYSQL_PORT="$2"
      shift 2
      ;;
    --mysql-user)
      MYSQL_USER="$2"
      shift 2
      ;;
    --mysql-password)
      MYSQL_PASSWORD="$2"
      shift 2
      ;;
    --mysql-root-password)
      MYSQL_ROOT_PASSWORD="$2"
      shift 2
      ;;
    --mysql-database)
      MYSQL_DATABASE="$2"
      shift 2
      ;;
    --port)
      APP_PORT="$2"
      shift 2
      ;;
    --admin-username)
      ADMIN_USERNAME="$2"
      shift 2
      ;;
    --admin-password)
      ADMIN_PASSWORD_VALUE="$2"
      ADMIN_PASSWORD_SET=true
      shift 2
      ;;
    --admin-title)
      ADMIN_TITLE="$2"
      shift 2
      ;;
    --auth-token-secret)
      AUTH_TOKEN_SECRET="$2"
      shift 2
      ;;
    --auth-token-ttl-hours)
      AUTH_TOKEN_TTL_HOURS="$2"
      shift 2
      ;;
    --wechat-app-id)
      WECHAT_APP_ID="$2"
      shift 2
      ;;
    --wechat-secret)
      WECHAT_SECRET="$2"
      shift 2
      ;;
    --skip-admin-build)
      SKIP_ADMIN_BUILD=true
      shift
      ;;
    --help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

case "$MODE" in
  local)
    MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
    MYSQL_PORT="${MYSQL_PORT:-3306}"
    MYSQL_USER="${MYSQL_USER:-root}"
    MYSQL_PASSWORD="${MYSQL_PASSWORD:-password}"
    MYSQL_DATABASE="${MYSQL_DATABASE:-couple_mini}"
    APP_PORT="${APP_PORT:-8080}"

    mkdir -p logs/admin logs/backend

    if [[ "$SKIP_ADMIN_BUILD" != "true" ]]; then
      build_admin_ui
    fi

    export PORT="$APP_PORT"
    export MYSQL_HOST
    export MYSQL_PORT
    export MYSQL_USER
    export MYSQL_PASSWORD
    export MYSQL_DATABASE
    export MYSQL_CREATE_DATABASE=true
    export MYSQL_AUTO_MIGRATE=true
    export MYSQL_AUTO_SEED="${MYSQL_AUTO_SEED:-true}"
    export ADMIN_ENABLED=true

    if [[ -n "$ADMIN_USERNAME" ]]; then
      export ADMIN_USERNAME
    fi
    if [[ "$ADMIN_PASSWORD_SET" == "true" ]]; then
      export ADMIN_PASSWORD="$ADMIN_PASSWORD_VALUE"
    fi
    if [[ -n "$ADMIN_TITLE" ]]; then
      export ADMIN_TITLE
    fi
    if [[ -n "$AUTH_TOKEN_SECRET" ]]; then
      export AUTH_TOKEN_SECRET
    fi
    if [[ -n "$AUTH_TOKEN_TTL_HOURS" ]]; then
      export AUTH_TOKEN_TTL_HOURS
    fi
    if [[ -n "$WECHAT_APP_ID" ]]; then
      export WECHAT_APP_ID
    fi
    if [[ -n "$WECHAT_SECRET" ]]; then
      export WECHAT_SECRET
    fi

    if [[ -z "$MYSQL_PASSWORD" ]]; then
      export MYSQL_DSN="$MYSQL_USER@tcp($MYSQL_HOST:$MYSQL_PORT)/$MYSQL_DATABASE?charset=utf8mb4&parseTime=true&loc=Local"
    else
      unset MYSQL_DSN
    fi

    mkdir -p bin
    "$(command -v go)" build -trimpath -o bin/backend .

    echo "Backend ready. The app will auto-create the database and tables, then listen on http://127.0.0.1:${APP_PORT}"
    if [[ "$SKIP_ADMIN_BUILD" != "true" ]]; then
      echo "Admin UI built. Open http://127.0.0.1:${APP_PORT}/admin after startup."
    fi
    exec ./bin/backend
    ;;
  docker)
    if ! docker info >/dev/null 2>&1; then
      echo "Docker daemon is not running. Please start Docker Desktop or docker service first, then run ./run.sh again." >&2
      exit 1
    fi

    if [[ -z "$ENV_FILE" ]]; then
      if [[ "$WITH_PROXY" == "true" ]]; then
        ENV_FILE="scripts/.env"
      else
        ENV_FILE="scripts/.env.docker"
      fi
    fi

    if [[ "$ENV_FILE" = /* || "$ENV_FILE" =~ ^[A-Za-z]:[\\/] ]]; then
      local_env_file="$ENV_FILE"
    else
      local_env_file="$SCRIPT_DIR/$ENV_FILE"
    fi
    template_name=".env.docker.example"
    if [[ "$WITH_PROXY" == "true" ]]; then
      template_name=".env.example"
    fi
    template_file="$SCRIPT_DIR/scripts/$template_name"
    compose_file="$SCRIPT_DIR/docker/docker-compose.yml"

    mkdir -p "$(dirname "$local_env_file")"

    if [[ ! -f "$local_env_file" ]]; then
      cp "$template_file" "$local_env_file"
      echo "Created $(basename "$local_env_file") from template."
    fi

    [[ -n "$APP_DOMAIN" ]] && set_env_value "$local_env_file" "APP_DOMAIN" "$APP_DOMAIN"
    [[ -n "$LETSENCRYPT_EMAIL" ]] && set_env_value "$local_env_file" "LETSENCRYPT_EMAIL" "$LETSENCRYPT_EMAIL"
    [[ -n "$BACKEND_PORT" ]] && set_env_value "$local_env_file" "BACKEND_PORT" "$BACKEND_PORT"
    [[ -n "$MYSQL_USER" ]] && set_env_value "$local_env_file" "MYSQL_USER" "$MYSQL_USER"
    [[ -n "$MYSQL_PASSWORD" ]] && set_env_value "$local_env_file" "MYSQL_PASSWORD" "$MYSQL_PASSWORD"
    [[ -n "$MYSQL_ROOT_PASSWORD" ]] && set_env_value "$local_env_file" "MYSQL_ROOT_PASSWORD" "$MYSQL_ROOT_PASSWORD"
    [[ -n "$MYSQL_DATABASE" ]] && set_env_value "$local_env_file" "MYSQL_DATABASE" "$MYSQL_DATABASE"
    [[ -n "$ADMIN_USERNAME" ]] && set_env_value "$local_env_file" "ADMIN_USERNAME" "$ADMIN_USERNAME"
    [[ "$ADMIN_PASSWORD_SET" == "true" ]] && set_env_value "$local_env_file" "ADMIN_PASSWORD" "$ADMIN_PASSWORD_VALUE"
    [[ -n "$ADMIN_TITLE" ]] && set_env_value "$local_env_file" "ADMIN_TITLE" "$ADMIN_TITLE"
    [[ -n "$AUTH_TOKEN_SECRET" ]] && set_env_value "$local_env_file" "AUTH_TOKEN_SECRET" "$AUTH_TOKEN_SECRET"
    [[ -n "$AUTH_TOKEN_TTL_HOURS" ]] && set_env_value "$local_env_file" "AUTH_TOKEN_TTL_HOURS" "$AUTH_TOKEN_TTL_HOURS"
    [[ -n "$WECHAT_APP_ID" ]] && set_env_value "$local_env_file" "WECHAT_APP_ID" "$WECHAT_APP_ID"
    [[ -n "$WECHAT_SECRET" ]] && set_env_value "$local_env_file" "WECHAT_SECRET" "$WECHAT_SECRET"

    mkdir -p logs/admin logs/backend/caddy uploads

    compose_args=(compose -f "$compose_file" --env-file "$local_env_file" up -d)
    if [[ "$REBUILD" == "true" ]]; then
      compose_args+=(--build)
    fi

    services=(mysql backend)
    if [[ "$WITH_PROXY" == "true" ]]; then
      services+=(caddy)
    fi

    docker "${compose_args[@]}" "${services[@]}"
    docker compose -f "$compose_file" --env-file "$local_env_file" ps

    resolved_backend_port="$(get_env_value "$local_env_file" "BACKEND_PORT")"
    resolved_backend_port="${resolved_backend_port:-8080}"
    resolved_domain="$(get_env_value "$local_env_file" "APP_DOMAIN")"

    echo
    echo "Environment is ready and services are starting."
    if [[ "$WITH_PROXY" == "true" ]]; then
      if [[ -n "$resolved_domain" ]]; then
        echo "Public URL: https://${resolved_domain}"
        echo "Admin URL:  https://${resolved_domain}/admin"
      else
        echo "Caddy is enabled, but APP_DOMAIN is empty. Set it in $local_env_file before public deployment."
      fi
    else
      echo "Backend URL: http://127.0.0.1:${resolved_backend_port}"
      echo "Admin URL:   http://127.0.0.1:${resolved_backend_port}/admin"
    fi
    echo "Check logs: docker compose -f $compose_file --env-file $local_env_file logs -f backend"
    ;;
  *)
    echo "Unsupported mode: $MODE" >&2
    exit 1
    ;;
esac
