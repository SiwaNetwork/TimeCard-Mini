# Добавляет Go в PATH для текущей сессии (если установлен в стандартное место).
# Использование: . .\scripts\ensure-go.ps1   или   & .\scripts\ensure-go.ps1
$goRoot = "C:\Program Files\Go"
$goBin = "$goRoot\bin"
if (Test-Path $goBin) {
    $env:Path = "$goBin;$env:Path"
    Write-Host "Go добавлен в PATH: $goBin"
    & go version
} else {
    Write-Host "Go не найден в $goRoot. Установите: winget install GoLang.Go"
    exit 1
}
