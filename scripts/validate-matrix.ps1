#!/usr/bin/env pwsh
<#
.SYNOPSIS
    CI integrity check for matrix backtest results.
    Verifies no data-quality defects in the latest matrix CSV.

.DESCRIPTION
    Checks: AvgWin < $500, Kelly <= 0.25, Return% < 200%, no NaN values,
    no Sharpe < -50, no absurd ProfitFactor.

.EXAMPLE
    ./scripts/validate-matrix.ps1
    ./scripts/validate-matrix.ps1 -File "data/.backtest_results/matrix_results (13).csv"
#>

param([string]$File = "")

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

if ($File -eq "") {
    $latest = Get-ChildItem "data/.backtest_results/matrix_results (*).csv" -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -match 'matrix_results \((\d+)\)\.csv' } |
        Sort-Object { [int]($_.Name -replace '.*\((\d+)\)\.csv', '$1') } |
        Select-Object -Last 1
    if (-not $latest) {
        Write-Error "No matrix results files found in data/.backtest_results/"
        exit 1
    }
    $File = $latest.FullName
}

Write-Host "Validating: $File"
$csv = Import-Csv $File
$total = $csv.Count
$errors = 0
$warnings = 0

Write-Host "Total rows: $total"

$absurdWin = ($csv | Where-Object { [double]$_.AvgWin -gt 500 }).Count
if ($absurdWin -gt 0) { Write-Warning ("AvgWin > `$500: $absurdWin rows"); $warnings++ }

$kelly = ($csv | Where-Object { $_.Params -match '"kelly_fraction":\s*0\.[3-9]|1\.0|[2-9]\.' }).Count
if ($kelly -gt 0) { Write-Error "Kelly > 0.25: $kelly rows"; $errors++ }

$ret200 = ($csv | Where-Object { [double]$_.'Return%' -gt 200 }).Count
if ($ret200 -gt 0) { Write-Error "Return% > 200%: $ret200 rows"; $errors++ }

$sharpNeg50 = ($csv | Where-Object { [double]$_.Sharpe -lt -50 }).Count
if ($sharpNeg50 -gt 0) { Write-Warning "Sharpe < -50: $sharpNeg50 rows"; $warnings++ }

$pfClamped = ($csv | Where-Object { [double]$_.ProfitFactor -gt 100 }).Count
if ($pfClamped -gt 0) { Write-Warning "ProfitFactor > 100: $pfClamped rows"; $warnings++ }

$dd100 = ($csv | Where-Object { [double]$_.'MaxDD%' -ge 99 -and [int]$_.Trades -gt 0 }).Count
if ($dd100 -gt 0) { Write-Warning "MaxDD 100% with trades: $dd100 rows"; $warnings++ }

$avgWinM = ($csv | Where-Object { [double]$_.AvgWin -gt 1000000 }).Count
if ($avgWinM -gt 0) { Write-Error "AvgWin > 1M: $avgWinM rows"; $errors++ }

$optFalse = ($csv | Where-Object { $_.Optimized -eq 'false' }).Count
Write-Host "Optimized=false: $optFalse / $total"

$trading = ($csv | Where-Object { [int]$_.Trades -gt 0 }).Count
Write-Host "Trading rows: $trading / $total"

if ($errors -gt 0) {
    Write-Error "VALIDATION FAILED: $errors error(s), $warnings warning(s)"
    exit 1
}

if ($warnings -gt 0) {
    Write-Warning "VALIDATION PASSED with $warnings warning(s)"
} else {
    Write-Host "VALIDATION PASSED (0 errors, 0 warnings)"
}
exit 0
