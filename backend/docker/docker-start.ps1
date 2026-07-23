param(
  [switch]$Rebuild,
  [switch]$WithProxy,
  [string]$EnvFile,
  [string]$AppDomain,
  [string]$LetsEncryptEmail,
  [string]$BackendPort,
  [string]$MySqlUser = $env:MYSQL_USER,
  [string]$MySqlPassword = $env:MYSQL_PASSWORD,
  [string]$MySqlRootPassword = $env:MYSQL_ROOT_PASSWORD,
  [string]$MySqlDatabase = $env:MYSQL_DATABASE,
  [string]$AdminUsername = $env:ADMIN_USERNAME,
  [string]$AdminPassword = $env:ADMIN_PASSWORD,
  [string]$AdminTitle = $env:ADMIN_TITLE
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RunScript = Join-Path (Split-Path -Parent $ScriptDir) "run.ps1"

& $RunScript `
  -Mode docker `
  -Rebuild:$Rebuild `
  -WithProxy:$WithProxy `
  -EnvFile $EnvFile `
  -AppDomain $AppDomain `
  -LetsEncryptEmail $LetsEncryptEmail `
  -BackendPort $BackendPort `
  -MySqlUser $MySqlUser `
  -MySqlPassword $MySqlPassword `
  -MySqlRootPassword $MySqlRootPassword `
  -MySqlDatabase $MySqlDatabase `
  -AdminUsername $AdminUsername `
  -AdminPassword $AdminPassword `
  -AdminTitle $AdminTitle
