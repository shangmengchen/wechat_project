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

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ScriptDir

if (-not $MySqlHost) { $MySqlHost = "127.0.0.1" }
if (-not $MySqlPort) { $MySqlPort = "3306" }
if (-not $MySqlUser) { $MySqlUser = "root" }
if ($null -eq $MySqlPassword) { $MySqlPassword = "password" }
if (-not $MySqlDatabase) { $MySqlDatabase = "couple_mini" }
if (-not $AppPort) { $AppPort = "8080" }

$GoExe = (Get-Command go -ErrorAction Stop).Source

$env:PORT = $AppPort
$env:MYSQL_HOST = $MySqlHost
$env:MYSQL_PORT = $MySqlPort
$env:MYSQL_USER = $MySqlUser
$env:MYSQL_PASSWORD = $MySqlPassword
$env:MYSQL_DATABASE = $MySqlDatabase
$env:MYSQL_CREATE_DATABASE = "true"
$env:MYSQL_AUTO_MIGRATE = "true"

if (-not $env:MYSQL_AUTO_SEED) {
  $env:MYSQL_AUTO_SEED = "true"
}

if ([string]::IsNullOrEmpty($MySqlPassword)) {
  $env:MYSQL_DSN = "${MySqlUser}@tcp($($MySqlHost):$MySqlPort)/${MySqlDatabase}?charset=utf8mb4&parseTime=true&loc=Local"
} else {
  Remove-Item Env:MYSQL_DSN -ErrorAction SilentlyContinue
}

$BinDir = Join-Path $ScriptDir "bin"
New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
$Binary = Join-Path $BinDir "backend.exe"

& $GoExe build -trimpath -o $Binary .
if ($LASTEXITCODE -ne 0) {
  throw "go build failed"
}

Write-Host "Backend ready. The app will auto-create the database and tables, then listen on http://127.0.0.1:$AppPort"
& $Binary
