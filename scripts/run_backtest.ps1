param(
    [string]$StrategyId = "intraday_mr",
    [string]$Symbols = "SPY",
    [string]$StartDate = "2024-01-01",
    [string]$EndDate = "2025-12-31",
    [double]$Capital = 100000.0
)

$projectRoot = Split-Path $PSScriptRoot -Parent
Push-Location $projectRoot

$env:CONFIG_PATH = "$projectRoot\configs\config.dev.yaml"

Write-Host "Running backtest:"
Write-Host "  Strategy: $StrategyId"
Write-Host "  Symbols:  $Symbols"
Write-Host "  Range:    $StartDate to $EndDate"
Write-Host "  Capital:  $Capital"

go run ./cmd/orca-cli backtest `
    -strategy "$StrategyId" `
    -symbols "$Symbols" `
    -start "$StartDate" `
    -end "$EndDate" `
    -capital $Capital

Pop-Location
