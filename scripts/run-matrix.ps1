#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Runs the full OrcaAlgo backtest matrix (17 strategies × 35 symbols × 6 timeframes = 3570 combos)
    and generates a summary report.

.DESCRIPTION
    This script builds and executes the matrix backtest runner against the local database.
    It outputs:
      - data/.backtest_results/matrix_results (N).csv  — full backtest results
      - data/.backtest_results/matrix_summary (N).txt   — human-readable summary

.PARAMETER OutputDir
    Directory for output files. Default: data/.backtest_results/

.PARAMETER Strategies
    Comma-separated list of strategy IDs. Default: all 17 canonical strategies.

.PARAMETER Symbols
    Comma-separated list of symbols. Default: all 35 matrix symbols.

.PARAMETER StartDate
    Backtest start date (YYYY-MM-DD). Default: 1 year ago.

.PARAMETER EndDate
    Backtest end date (YYYY-MM-DD). Default: today.

.EXAMPLE
    ./scripts/run-matrix.ps1
    ./scripts/run-matrix.ps1 -Strategies "grid_trading,trend_following" -Symbols "ES,NQ" -StartDate "2025-01-01"
#>

param(
    [string]$OutputDir = "data/.backtest_results",
    [string]$Strategies = "",
    [string]$Symbols = "",
    [string]$StartDate = "",
    [string]$EndDate = "",
    [int]$DBPort = 5433
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

Write-Host "=== OrcaAlgo Matrix Backtest ==="
Write-Host "Root: $root"
Write-Host ""

$version = (Get-ChildItem "$OutputDir/matrix_results (*).csv" -ErrorAction SilentlyContinue | ForEach-Object {
    if ($_.Name -match 'matrix_results \((\d+)\)') { [int]$matches[1] }
} | Measure-Object -Maximum).Maximum
$version = if ($version) { $version + 1 } else { 1 }
$csvFile = "$OutputDir/matrix_results ($version).csv"
$sumFile = "$OutputDir/matrix_summary ($version).txt"

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

Write-Host "Building matrix runner..."
go build -o bin/matrix-runner.exe ./cmd/matrix-runner 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Error "Build failed."
    exit 1
}
Write-Host "Built OK."
Write-Host ""

Write-Host "Starting backtest matrix v$version..."
Write-Host "Output CSV: $csvFile"
Write-Host ""

$env:MATRIX_OUTPUT = (Get-Item $csvFile -ErrorAction SilentlyContinue).FullName
if (-not $env:MATRIX_OUTPUT) { $env:MATRIX_OUTPUT = (Join-Path (Get-Location) $csvFile) }
$env:MATRIX_STRATEGIES = $Strategies
$env:MATRIX_SYMBOLS = $Symbols
$env:MATRIX_START = $StartDate
$env:MATRIX_END = $EndDate
$env:MATRIX_DB_PORT = "$DBPort"

$stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
$output = & "$root/bin/matrix-runner.exe" 2>&1
$stopwatch.Stop()

$exitCode = $LASTEXITCODE
$output | Out-File "$OutputDir/matrix_log ($version).txt"

$output | Select-Object -Last 5 | ForEach-Object { Write-Host $_ }
Write-Host ""
Write-Host "Done in $([math]::Round($stopwatch.Elapsed.TotalMinutes, 1)) min. Exit code: $exitCode"

if ($exitCode -ne 0) {
    Write-Error "Matrix runner failed. Check $OutputDir/matrix_log ($version).txt"
    exit 1
}

if (-not (Test-Path $csvFile)) {
    Write-Error "CSV not found: $csvFile"
    exit 1
}

Write-Host "Generating summary..."
$csv = Import-Csv $csvFile

$total = $csv.Count
$trading = ($csv | Where-Object { [int]$_.Trades -gt 0 }).Count
$zeroTrade = $total - $trading
$posSharp = ($csv | Where-Object { [int]$_.Trades -gt 0 -and [double]$_.Sharpe -gt 0 }).Count
$sharpGt1 = ($csv | Where-Object { [int]$_.Trades -gt 0 -and [double]$_.Sharpe -gt 1.0 }).Count
$absurdWin = ($csv | Where-Object { [double]$_.AvgWin -gt 1000 }).Count
$maxDD100 = ($csv | Where-Object { [double]$_.'MaxDD%' -ge 99 }).Count
$kellyViolations = ($csv | Where-Object { $_.Params -match '"kelly_fraction":\s*0\.[3-9]|1\.0' }).Count

$summary = @"

=== OrcaAlgo Matrix Backtest v$version ===
Date:       $(Get-Date -Format 'yyyy-MM-dd HH:mm')
Runtime:    $([math]::Round($stopwatch.Elapsed.TotalMinutes, 1)) min
Total rows: $total ($([math]::Round($trading/$total*100, 1))% trading)

DATA QUALITY:
  Absurd AvgWin (>$1K):    $absurdWin rows
  Kelly fraction > 0.25:   $kellyViolations rows
  100% MaxDD:               $maxDD100 rows

PERFORMANCE:
  Positive Sharpe:          $posSharp rows ($([math]::Round($posSharp/$total*100, 1))%)
  Sharpe > 1.0:             $sharpGt1 rows ($([math]::Round($sharpGt1/$total*100, 1))%)

STRATEGY SUMMARY:
$(($csv | Group-Object Strategy | ForEach-Object {
    $s = $_.Name; $g = $_.Group
    $tr = ($g | Where-Object { [int]$_.Trades -gt 0 }).Count
    $zp = $g.Count - $tr
    $ps = ($g | Where-Object { [int]$_.Trades -gt 0 -and [double]$_.Sharpe -gt 0 }).Count
    $avgSh = if ($tr -gt 0) { [math]::Round(($g | Where-Object { [int]$_.Trades -gt 0 } | ForEach-Object { [double]$_.Sharpe } | Measure-Object -Average).Average, 3) } else { 0 }
    "  $($s.PadRight(24)) $($tr.ToString().PadLeft(3))/$($g.Count) trading  zero=$zp  posSh=$ps  avgSharpe=$avgSh"
}) -join "`n")

TOP 10 SHARPE:
$(($csv | Where-Object { [int]$_.Trades -gt 0 } | Sort-Object {[double]$_.Sharpe} -Descending | Select-Object -First 10 | ForEach-Object {
    "  $($_.Strategy.PadRight(24)) $($_.Symbol.PadRight(8)) $($_.Tf.PadRight(4)) trades=$($_.Trades.ToString().PadLeft(5)) sharpe=$([math]::Round([double]$_.Sharpe, 4).ToString().PadRight(10)) return=$([math]::Round([double]$_.'Return%', 1))%"
}) -join "`n")

"@

$summary | Out-File $sumFile -Encoding utf8
Write-Host $summary
Write-Host ""
Write-Host "Summary written to: $sumFile"
Write-Host "CSV written to: $csvFile"
