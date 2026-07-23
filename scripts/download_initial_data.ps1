# Bootstrap: download initial universe data via Tiingo API
param(
    [string]$StartDate = "2024-01-01",
    [string]$EndDate = "",
    [string]$ApiKey = $env:TIINGO_API_KEY
)

$ErrorActionPreference = "Stop"
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent $scriptDir

if (-not $EndDate) {
    $EndDate = (Get-Date).ToString("yyyy-MM-dd")
}

if (-not $ApiKey) {
    Write-Host "TIINGO_API_KEY not set. Set it via environment variable or pass -ApiKey." -ForegroundColor Red
    Write-Host "Usage: .\scripts\download_initial_data.ps1 -ApiKey YOUR_KEY" -ForegroundColor Yellow
    exit 1
}

$env:TIINGO_API_KEY = $ApiKey

Write-Host "Downloading universe data from $StartDate to $EndDate..." -ForegroundColor Cyan

$symbols = @(
    "EURUSD", "GBPUSD", "USDJPY", "USDCHF", "AUDUSD", "USDCAD", "NZDUSD",
    "US30", "SPX500", "NAS100", "UK100", "GER40", "JPN225",
    "XAUUSD", "XAGUSD", "USOIL", "UKOIL",
    "BTCUSD", "ETHUSD"
)

$total = $symbols.Count
$done = 0
$stored = 0

foreach ($sym in $symbols) {
    $done++
    Write-Host "[$done/$total] $sym..." -NoNewline

    try {
        $result = & go run cmd/orca-fetch/main.go --symbols=$sym --start=$StartDate --end=$EndDate --source=tiingo 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host " OK" -ForegroundColor Green
            if ($result -match "(\d+) stored") { $stored += [int]$Matches[1] }
        } else {
            Write-Host " FAIL" -ForegroundColor Red
            Write-Host $result
        }
    } catch {
        Write-Host " ERROR: $_" -ForegroundColor Red
    }
}

Write-Host "`nDone. Total stored candles: $stored" -ForegroundColor Cyan
