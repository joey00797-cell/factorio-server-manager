param(
    [ValidateSet("start", "stop", "logs", "reset", "build")]
    [string]$Action = "start"
)

$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$Compose = Join-Path $Root "docker\docker-compose.windows-test.yaml"
$BuildDir = Join-Path $Root "build"
$Zip = Join-Path $Root "docker\factorio-server-manager-linux.zip"

function Build-Image {
    Push-Location $Root
    try {
        if (Test-Path $BuildDir) {
            Remove-Item -Recurse -Force $BuildDir
        }
        docker build -f docker/Dockerfile-build --target output -o build .
        $artifact = Get-ChildItem $BuildDir -Filter "factorio-server-manager-linux*.zip" | Select-Object -First 1
        if (-not $artifact) {
            throw "Linux build artifact was not produced in $BuildDir"
        }
        Copy-Item -Force $artifact.FullName $Zip
        docker build -f docker/Dockerfile-local -t factorio-server-manager:windows-test docker
        Remove-Item -Force $Zip
    } finally {
        Pop-Location
    }
}

switch ($Action) {
    "build" {
        Build-Image
    }
    "start" {
        Build-Image
        docker compose -f $Compose up -d
        Write-Host "FSM is starting at http://localhost:8080"
        Write-Host "Follow logs with: .\scripts\docker-windows-test.ps1 logs"
    }
    "stop" {
        docker compose -f $Compose down
    }
    "logs" {
        docker compose -f $Compose logs -f factorio-server-manager
    }
    "reset" {
        docker compose -f $Compose down -v
        Write-Host "Removed the Windows test containers and named volumes."
    }
}
