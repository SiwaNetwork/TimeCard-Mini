# Дизассемблирование бинарника timebeat через WSL (objdump/nm).
# Запуск: из корня репозитория:
#   .\code_analysis\run_disassembly_wsl.ps1
# или с указанием бинарника и каталога:
#   .\code_analysis\run_disassembly_wsl.ps1 -Binary "timebeat-extracted\usr\share\timebeat\bin\timebeat" -Output "code_analysis\disassembly"
# Опционально: -Patterns "clocksync","phc" — только эти подстроки в именах символов.

param(
    [string]$Binary = "timebeat-extracted\usr\share\timebeat\bin\timebeat",
    [string]$Output = "code_analysis\disassembly",
    [string[]]$Patterns = @(
        "clocksync/servo",
        "clocksync/hostclocks",
        "clocksync/phc",
        "clocksync/clients",
        "clocksync/sources",
        "adjusttime",
        "filters",
        "offsets",
        "algos",
        "statistics",
        "Controller",
        "TimeSource",
        "Offsets",
        "EMA",
        "MovingMedian",
        "NoneGaussianFilter",
        "BestFitFiltered",
        "Pi.",
        "LinReg",
        "AlgoPID",
        "NTP",
        "nmea",
        "ntp",
        "sources/store",
        "runGNSSRunloop",
        "ProcessEvent",
        "SubmitOffset",
        "receiveMessage",
        "UpdateTimeSource",
        "addSource",
        "CreateSource",
        "interactive",
        "daemon",
        "logging",
        "GetSecondarySourcesOffset",
        "GetClockWithURI",
        "ubx/conf",
        "ShouldLog",
        "uriRegister",
        "parseSourceForCLI",
        "GetSourcesForCLI"
    )
)

$ErrorActionPreference = "Stop"
$repoRoot = if ($PSScriptRoot) { Split-Path $PSScriptRoot -Parent } else { Get-Location }
$repoRoot = (Resolve-Path $repoRoot).Path
$binPath = Join-Path $repoRoot $Binary
$outPath = Join-Path $repoRoot $Output

if (-not (Test-Path $binPath -PathType Leaf)) {
    Write-Error "Binary not found: $binPath"
    exit 1
}

# Путь к бинарнику и репо в формате WSL (/mnt/c/...)
$driveLetter = $repoRoot.Substring(0, 1).ToLower()
$wslRepo = "/mnt/$driveLetter" + ($repoRoot.Substring(2) -replace '\\', '/')
$wslBin  = "/mnt/$driveLetter" + ($binPath.Substring(2) -replace '\\', '/')
$wslOut  = "/mnt/$driveLetter" + ($outPath.Substring(2) -replace '\\', '/')

$argsList = @(
    "cd", "`"$wslRepo`"", "&&",
    "python3", "code_analysis/extract_disassembly_for_functions.py", "`"$wslBin`"",
    "-o", "`"$wslOut`""
)
foreach ($p in $Patterns) {
    $argsList += "-f", "`"$p`""
}
$cmd = $argsList -join " "
Write-Host "Running in WSL: $cmd"
& wsl -e bash -c $cmd
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Write-Host "Disassembly saved to: $outPath"
