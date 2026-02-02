# Кросс-сборка tc-sync для Linux (целевая ОС).
# Запуск из корня репозитория или из tc-sync:
#   .\scripts\build-linux.ps1
#   .\scripts\build-linux.ps1 -Arch arm64
param(
    [ValidateSet("amd64", "arm64", "arm")]
    [string]$Arch = "amd64"
)

$ErrorActionPreference = "Stop"
# Скрипт лежит в tc-sync/scripts/
$tcSync = Split-Path $PSScriptRoot -Parent
if (-not (Test-Path "$tcSync\go.mod")) {
    Write-Error "go.mod не найден в $tcSync"
}

Push-Location $tcSync
try {
    $env:GOOS = "linux"
    $env:GOARCH = $Arch
    $out = "tc-sync-linux-$Arch"
    Write-Host "Building $out (GOOS=linux GOARCH=$Arch)..."
    go build -o $out ./cmd/tc-sync
    Write-Host "OK: $tcSync\$out"
} finally {
    Pop-Location
}
