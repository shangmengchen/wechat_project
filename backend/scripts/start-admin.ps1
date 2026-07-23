param(
  [string]$MySqlHost = $env:MYSQL_HOST,
  [string]$MySqlPort = $env:MYSQL_PORT,
  [string]$MySqlUser = $env:MYSQL_USER,
  [string]$MySqlPassword = $env:MYSQL_PASSWORD,
  [string]$MySqlDatabase = $env:MYSQL_DATABASE,
  [string]$AppPort = $env:PORT,
  [string]$AdminUsername = $env:ADMIN_USERNAME,
  [string]$AdminPassword = $env:ADMIN_PASSWORD,
  [string]$AdminTitle = $env:ADMIN_TITLE
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RunScript = Join-Path (Split-Path -Parent $ScriptDir) "run.ps1"

& $RunScript `
  -Mode local `
  -MySqlHost $MySqlHost `
  -MySqlPort $MySqlPort `
  -MySqlUser $MySqlUser `
  -MySqlPassword $MySqlPassword `
  -MySqlDatabase $MySqlDatabase `
  -AppPort $AppPort `
  -AdminUsername $AdminUsername `
  -AdminPassword $AdminPassword `
  -AdminTitle $AdminTitle
