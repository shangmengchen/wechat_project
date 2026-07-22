param(
  [string]$MySqlHost = $env:MYSQL_HOST,
  [string]$MySqlPort = $env:MYSQL_PORT,
  [string]$MySqlUser = $env:MYSQL_USER,
  [string]$MySqlPassword = $env:MYSQL_PASSWORD,
  [string]$MySqlDatabase = $env:MYSQL_DATABASE,
  [string]$AppPort = $env:PORT
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Local development entrypoint.
# This script builds the Go binary, prepares DB-related env vars, then runs the app.
# Production deployment should use docker compose instead of this script.

# Always run relative to the backend directory so paths like .\bin remain stable.
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ScriptDir

# Default local MySQL and app settings.
# These can be overridden with parameters or environment variables.
if (-not $MySqlHost) { $MySqlHost = "127.0.0.1" }
if (-not $MySqlPort) { $MySqlPort = "3306" }
if (-not $MySqlUser) { $MySqlUser = "root" }
if ($null -eq $MySqlPassword) { $MySqlPassword = "password" }
if (-not $MySqlDatabase) { $MySqlDatabase = "couple_mini" }
if (-not $AppPort) { $AppPort = "8080" }

$GoExe = (Get-Command go -ErrorAction Stop).Source

# Export runtime env vars for the Go process.
# The app reads these values on startup to build DB connections and server config.
$env:PORT = $AppPort
$env:MYSQL_HOST = $MySqlHost
$env:MYSQL_PORT = $MySqlPort
$env:MYSQL_USER = $MySqlUser
$env:MYSQL_PASSWORD = $MySqlPassword
$env:MYSQL_DATABASE = $MySqlDatabase
$env:MYSQL_CREATE_DATABASE = "true"
$env:MYSQL_AUTO_MIGRATE = "true"
# Local runs default to seeding demo data unless the caller explicitly disables it.
if (-not $env:MYSQL_AUTO_SEED) {
  $env:MYSQL_AUTO_SEED = "true"
}

# If the password is intentionally empty, provide a DSN without the password field.
# Otherwise let the app build its normal DSN from the discrete env vars above.
if ([string]::IsNullOrEmpty($MySqlPassword)) {
  $env:MYSQL_DSN = "${MySqlUser}@tcp($($MySqlHost):$MySqlPort)/${MySqlDatabase}?charset=utf8mb4&parseTime=true&loc=Local"
} else {
  Remove-Item Env:MYSQL_DSN -ErrorAction SilentlyContinue
}

# Build the backend into .\bin so repeated local runs do not clutter the repo root.
$BinDir = Join-Path $ScriptDir "bin"
New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
$Binary = Join-Path $BinDir "backend.exe"

& $GoExe build -trimpath -o $Binary .
if ($LASTEXITCODE -ne 0) {
  throw "go build failed"
}

# Start the compiled backend in the foreground.
Write-Host "Backend ready. The app will auto-create the database and tables, then listen on http://127.0.0.1:$AppPort"
& $Binary
