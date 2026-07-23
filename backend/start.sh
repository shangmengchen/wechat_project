#!/usr/bin/env bash
set -euo pipefail

# 本地开发启动入口。
# 这个脚本会先编译 Go 二进制，再准备数据库相关环境变量，最后启动后端服务。
# 生产环境部署应使用 docker compose，而不是这个脚本。

# 始终切换到 backend 目录下执行，避免 ./bin 这类相对路径失效。
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 本地默认的 MySQL 和服务端口配置。
# 如果你在执行前已经导出了同名环境变量，这里会优先使用你的值。
MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-password}"
MYSQL_DATABASE="${MYSQL_DATABASE:-couple_mini}"
APP_PORT="${PORT:-8080}"

GO_BIN="$(command -v go)"

# 导出运行时环境变量给 Go 进程使用。
# 后端启动时会读取这些变量，构造数据库连接和服务配置。
export PORT="$APP_PORT"
export MYSQL_HOST
export MYSQL_PORT
export MYSQL_USER
export MYSQL_PASSWORD
export MYSQL_DATABASE
export MYSQL_CREATE_DATABASE=true
export MYSQL_AUTO_MIGRATE=true
# 本地运行默认允许写入演示数据，除非你手动把 MYSQL_AUTO_SEED 设为 false。
export MYSQL_AUTO_SEED="${MYSQL_AUTO_SEED:-true}"

# 如果密码被明确设为空，就手动拼一个不带密码字段的 DSN。
# 否则让应用自己根据上面的分散环境变量组装正常的 DSN。
if [[ -z "$MYSQL_PASSWORD" ]]; then
  export MYSQL_DSN="$MYSQL_USER@tcp($MYSQL_HOST:$MYSQL_PORT)/$MYSQL_DATABASE?charset=utf8mb4&parseTime=true&loc=Local"
else
  unset MYSQL_DSN
fi

# 把编译产物放到 ./bin 目录，避免反复本地运行时污染仓库根目录。
mkdir -p bin
"$GO_BIN" build -trimpath -o bin/backend .

# 前台启动编译好的后端服务。
echo "Backend ready. The app will auto-create the database and tables, then listen on http://127.0.0.1:$APP_PORT"
exec ./bin/backend
