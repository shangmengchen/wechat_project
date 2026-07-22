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
$MysqlExe = (Get-Command mysql -ErrorAction Stop).Source

function Invoke-Mysql {
  param([string[]]$CommandArgs)
  & $MysqlExe @CommandArgs
  if ($LASTEXITCODE -ne 0) {
    throw "mysql command failed"
  }
}

$mysqlArgs = @(
  "--protocol=TCP",
  "--host=$MySqlHost",
  "--port=$MySqlPort",
  "--user=$MySqlUser"
)
if ($MySqlPassword) {
  $mysqlArgs += "--password=$MySqlPassword"
}
$mysqlArgs += "--execute=CREATE DATABASE IF NOT EXISTS $MySqlDatabase DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
Invoke-Mysql $mysqlArgs

$env:PORT = $AppPort
if ($MySqlPassword) {
  $encodedPassword = [uri]::EscapeDataString($MySqlPassword)
  $env:MYSQL_DSN = "${MySqlUser}:$encodedPassword@tcp($($MySqlHost):$MySqlPort)/${MySqlDatabase}?charset=utf8mb4&parseTime=true&loc=Local"
} else {
  $env:MYSQL_DSN = "${MySqlUser}@tcp($($MySqlHost):$MySqlPort)/${MySqlDatabase}?charset=utf8mb4&parseTime=true&loc=Local"
}

$BinDir = Join-Path $ScriptDir "bin"
New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
$Binary = Join-Path $BinDir "backend.exe"

& $GoExe build -trimpath -o $Binary ./cmd/server
if ($LASTEXITCODE -ne 0) {
  throw "go build failed"
}

Write-Host "Backend ready. Listening on http://127.0.0.1:$AppPort"
& $Binary
