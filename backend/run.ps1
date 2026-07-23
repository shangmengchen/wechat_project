param(
  [ValidateSet("docker", "local")]
  [string]$Mode = "docker",
  [switch]$Rebuild,
  [switch]$WithProxy,
  [string]$EnvFile,
  [string]$AppDomain,
  [string]$LetsEncryptEmail,
  [string]$BackendPort,
  [string]$MySqlHost = $env:MYSQL_HOST,
  [string]$MySqlPort = $env:MYSQL_PORT,
  [string]$MySqlUser = $env:MYSQL_USER,
  [string]$MySqlPassword = $env:MYSQL_PASSWORD,
  [string]$MySqlDatabase = $env:MYSQL_DATABASE,
  [string]$MySqlRootPassword = $env:MYSQL_ROOT_PASSWORD,
  [string]$AppPort = $env:PORT,
  [string]$AdminUsername = $env:ADMIN_USERNAME,
  [string]$AdminPassword = $env:ADMIN_PASSWORD,
  [string]$AdminTitle = $env:ADMIN_TITLE,
  [switch]$SkipAdminBuild
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ScriptDir

function Set-EnvValue {
  param(
    [string]$Path,
    [string]$Key,
    [string]$Value
  )

  $lines = [System.Collections.Generic.List[string]]::new()
  if (Test-Path $Path) {
    foreach ($line in Get-Content $Path) {
      $lines.Add($line)
    }
  }

  $prefix = "$Key="
  $updated = $false
  for ($i = 0; $i -lt $lines.Count; $i++) {
    if ($lines[$i].StartsWith($prefix)) {
      $lines[$i] = "$Key=$Value"
      $updated = $true
      break
    }
  }
  if (-not $updated) {
    $lines.Add("$Key=$Value")
  }
  Set-Content -Path $Path -Value $lines
}

function Get-EnvValue {
  param(
    [string]$Path,
    [string]$Key
  )

  if (-not (Test-Path $Path)) {
    return $null
  }

  foreach ($line in Get-Content $Path) {
    if ($line -match "^$([regex]::Escape($Key))=(.*)$") {
      return $Matches[1]
    }
  }
  return $null
}

function Build-AdminUi {
  param([string]$RootDir)

  $AdminUiDir = Join-Path $RootDir "admin-ui"
  if (-not (Test-Path $AdminUiDir)) {
    throw "admin-ui directory not found: $AdminUiDir"
  }

  $NpmExe = (Get-Command npm -ErrorAction Stop).Source
  Push-Location $AdminUiDir
  try {
    if (-not (Test-Path (Join-Path $AdminUiDir "node_modules"))) {
      & $NpmExe install
      if ($LASTEXITCODE -ne 0) {
        throw "npm install failed"
      }
    }

    & $NpmExe run build
    if ($LASTEXITCODE -ne 0) {
      throw "npm run build failed"
    }
  } finally {
    Pop-Location
  }
}

function Start-Local {
	if (-not $MySqlHost) { $MySqlHost = "127.0.0.1" }
	if (-not $MySqlPort) { $MySqlPort = "3306" }
	if (-not $MySqlUser) { $MySqlUser = "root" }
	if ($null -eq $MySqlPassword) { $MySqlPassword = "password" }
	if (-not $MySqlDatabase) { $MySqlDatabase = "couple_mini" }
	if (-not $AppPort) { $AppPort = "8080" }

	New-Item -ItemType Directory -Force -Path (Join-Path $ScriptDir "logs") | Out-Null
	New-Item -ItemType Directory -Force -Path (Join-Path $ScriptDir "logs\admin") | Out-Null
	New-Item -ItemType Directory -Force -Path (Join-Path $ScriptDir "logs\backend") | Out-Null

	if (-not $SkipAdminBuild) {
		Build-AdminUi -RootDir $ScriptDir
	}

  $GoExe = (Get-Command go -ErrorAction Stop).Source

  $env:PORT = $AppPort
  $env:MYSQL_HOST = $MySqlHost
  $env:MYSQL_PORT = $MySqlPort
  $env:MYSQL_USER = $MySqlUser
  $env:MYSQL_PASSWORD = $MySqlPassword
  $env:MYSQL_DATABASE = $MySqlDatabase
  $env:MYSQL_CREATE_DATABASE = "true"
  $env:MYSQL_AUTO_MIGRATE = "true"
  $env:ADMIN_ENABLED = "true"

  if (-not $env:MYSQL_AUTO_SEED) {
    $env:MYSQL_AUTO_SEED = "true"
  }

  if (-not [string]::IsNullOrWhiteSpace($AdminUsername)) {
    $env:ADMIN_USERNAME = $AdminUsername
  }
  if ($null -ne $AdminPassword) {
    $env:ADMIN_PASSWORD = $AdminPassword
  }
  if (-not [string]::IsNullOrWhiteSpace($AdminTitle)) {
    $env:ADMIN_TITLE = $AdminTitle
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
  if (-not $SkipAdminBuild) {
    Write-Host "Admin UI built. Open http://127.0.0.1:$AppPort/admin after startup."
  }
  & $Binary
}

function Start-Docker {
  $DockerExe = (Get-Command docker -ErrorAction Stop).Source
  & $DockerExe info | Out-Null
	if ($LASTEXITCODE -ne 0) {
		throw "Docker daemon is not running. Please start Docker first, then run .\\run.ps1 again."
	}

	$selectedEnvFile = $EnvFile
	if (-not $selectedEnvFile) {
		if ($WithProxy) {
			$selectedEnvFile = "scripts/.env"
		} else {
			$selectedEnvFile = "scripts/.env.docker"
		}
	}
	if (-not [System.IO.Path]::IsPathRooted($selectedEnvFile)) {
		$selectedEnvFile = Join-Path $ScriptDir $selectedEnvFile
	}
	$templateName = if ($WithProxy) { ".env.example" } else { ".env.docker.example" }
	$templateFile = Join-Path $ScriptDir "scripts\$templateName"
	$composeFile = Join-Path $ScriptDir "docker\docker-compose.yml"

	New-Item -ItemType Directory -Force -Path (Split-Path -Parent $selectedEnvFile) | Out-Null

	if (-not (Test-Path $selectedEnvFile)) {
		Copy-Item $templateFile $selectedEnvFile
		Write-Host "Created $(Split-Path -Leaf $selectedEnvFile) from template."
	}

  if (-not [string]::IsNullOrWhiteSpace($AppDomain)) {
    Set-EnvValue -Path $selectedEnvFile -Key "APP_DOMAIN" -Value $AppDomain
  }
  if (-not [string]::IsNullOrWhiteSpace($LetsEncryptEmail)) {
    Set-EnvValue -Path $selectedEnvFile -Key "LETSENCRYPT_EMAIL" -Value $LetsEncryptEmail
  }
  if (-not [string]::IsNullOrWhiteSpace($BackendPort)) {
    Set-EnvValue -Path $selectedEnvFile -Key "BACKEND_PORT" -Value $BackendPort
  }
  if (-not [string]::IsNullOrWhiteSpace($MySqlUser)) {
    Set-EnvValue -Path $selectedEnvFile -Key "MYSQL_USER" -Value $MySqlUser
  }
  if ($null -ne $MySqlPassword -and $MySqlPassword -ne "") {
    Set-EnvValue -Path $selectedEnvFile -Key "MYSQL_PASSWORD" -Value $MySqlPassword
  }
  if ($null -ne $MySqlRootPassword -and $MySqlRootPassword -ne "") {
    Set-EnvValue -Path $selectedEnvFile -Key "MYSQL_ROOT_PASSWORD" -Value $MySqlRootPassword
  }
  if (-not [string]::IsNullOrWhiteSpace($MySqlDatabase)) {
    Set-EnvValue -Path $selectedEnvFile -Key "MYSQL_DATABASE" -Value $MySqlDatabase
  }
  if (-not [string]::IsNullOrWhiteSpace($AdminUsername)) {
    Set-EnvValue -Path $selectedEnvFile -Key "ADMIN_USERNAME" -Value $AdminUsername
  }
  if ($null -ne $AdminPassword -and $AdminPassword -ne "") {
    Set-EnvValue -Path $selectedEnvFile -Key "ADMIN_PASSWORD" -Value $AdminPassword
  }
	if (-not [string]::IsNullOrWhiteSpace($AdminTitle)) {
		Set-EnvValue -Path $selectedEnvFile -Key "ADMIN_TITLE" -Value $AdminTitle
	}

	New-Item -ItemType Directory -Force -Path (Join-Path $ScriptDir "logs") | Out-Null
	New-Item -ItemType Directory -Force -Path (Join-Path $ScriptDir "logs\admin") | Out-Null
	New-Item -ItemType Directory -Force -Path (Join-Path $ScriptDir "logs\backend") | Out-Null
	New-Item -ItemType Directory -Force -Path (Join-Path $ScriptDir "logs\backend\caddy") | Out-Null
	New-Item -ItemType Directory -Force -Path (Join-Path $ScriptDir "uploads") | Out-Null

	$composeArgs = @("compose", "-f", $composeFile, "--env-file", $selectedEnvFile, "up", "-d")
	if ($Rebuild) {
		$composeArgs += "--build"
	}

  $services = @("mysql", "backend")
  if ($WithProxy) {
    $services += "caddy"
  }
	$composeArgs += $services

	& $DockerExe @composeArgs
	if ($LASTEXITCODE -ne 0) {
		throw "docker compose up failed"
	}

	& $DockerExe compose -f $composeFile --env-file $selectedEnvFile ps

	$resolvedBackendPort = Get-EnvValue -Path $selectedEnvFile -Key "BACKEND_PORT"
	if (-not $resolvedBackendPort) {
		$resolvedBackendPort = "8080"
  }

  $resolvedDomain = Get-EnvValue -Path $selectedEnvFile -Key "APP_DOMAIN"

  Write-Host ""
  Write-Host "Environment is ready and services are starting."
	if ($WithProxy) {
		if ($resolvedDomain) {
			Write-Host "Public URL: https://$resolvedDomain"
			Write-Host "Admin URL:  https://$resolvedDomain/admin"
		} else {
			Write-Host "Caddy is enabled, but APP_DOMAIN is empty. Set it in $selectedEnvFile before public deployment."
		}
	} else {
		Write-Host "Backend URL: http://127.0.0.1:$resolvedBackendPort"
		Write-Host "Admin URL:   http://127.0.0.1:$resolvedBackendPort/admin"
	}
	Write-Host "Check logs: docker compose -f $composeFile --env-file $selectedEnvFile logs -f backend"
}

switch ($Mode) {
  "local" { Start-Local }
  "docker" { Start-Docker }
  default { throw "unsupported mode: $Mode" }
}
