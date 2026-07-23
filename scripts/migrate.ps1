param(
    [string]$DatabaseUrl = $env:DATABASE_URL
)

if (-not $DatabaseUrl) {
    $DatabaseUrl = "postgres://orca:orca@localhost:5432/orca_core?sslmode=disable"
}

$projectRoot = Split-Path $PSScriptRoot -Parent
$migrationsPath = "$projectRoot\internal\db\migrations"

Write-Host "Running migrations..."
Write-Host "Database: $DatabaseUrl"
Write-Host "Migrations: $migrationsPath"

migrate -path "$migrationsPath" -database "$DatabaseUrl" up

if ($LASTEXITCODE -eq 0) {
    Write-Host "Migrations completed successfully."
} else {
    Write-Error "Migration failed."
    exit $LASTEXITCODE
}
