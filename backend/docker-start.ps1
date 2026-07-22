param(
  [switch]$Rebuild
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ScriptDir

$DockerExe = (Get-Command docker -ErrorAction Stop).Source
$EnvFile = Join-Path $ScriptDir ".env.docker"
$ExampleFile = Join-Path $ScriptDir ".env.docker.example"

& $DockerExe info | Out-Null
if ($LASTEXITCODE -ne 0) {
  throw "Docker daemon is not running. Please start Docker Desktop first, then run .\\docker-start.ps1 again."
}

if (-not (Test-Path $EnvFile)) {
  Copy-Item $ExampleFile $EnvFile
  Write-Host "Created .env.docker from template. You can edit it later if needed."
}

New-Item -ItemType Directory -Force -Path (Join-Path $ScriptDir "logs") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $ScriptDir "uploads") | Out-Null

$composeArgs = @(
  "compose",
  "--env-file", ".env.docker",
  "up",
  "-d"
)
if ($Rebuild) {
  $composeArgs += "--build"
}
$composeArgs += @("mysql", "backend")

& $DockerExe @composeArgs
if ($LASTEXITCODE -ne 0) {
  throw "docker compose up failed"
}

$statusArgs = @("compose", "--env-file", ".env.docker", "ps")
& $DockerExe @statusArgs

$backendPort = 8080
Get-Content $EnvFile | ForEach-Object {
  if ($_ -match '^BACKEND_PORT=(.+)$') {
    $backendPort = $Matches[1]
  }
}

Write-Host ""
Write-Host "MySQL and backend are starting with Docker."
Write-Host "Backend URL: http://127.0.0.1:$backendPort"
Write-Host "Check logs: docker compose --env-file .env.docker logs -f backend"
