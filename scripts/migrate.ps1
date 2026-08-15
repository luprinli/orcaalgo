param(
    [string]$DatabaseUrl = $env:DATABASE_URL
)

# Migrations are applied by the Go-managed runner (internal/db.RunMigrations),
# the single source of truth also used at server startup. The connection is
# configured via the standard ORCA_DB_* environment variables (not the
# DATABASE_URL positional above, kept only for backward compatibility).

$projectRoot = Split-Path $PSScriptRoot -Parent

Write-Host "Running migrations (Go-managed runner)..."
Write-Host "Project: $projectRoot"

# Optional: surface the legacy DATABASE_URL as a hint if the modern vars are unset.
if (-not $env:ORCA_DB_HOST -and $DatabaseUrl) {
    Write-Warning "DATABASE_URL is deprecated; set ORCA_DB_HOST/PORT/USER/PASSWORD/NAME instead."
}

go run ./cmd/migrate
if ($LASTEXITCODE -eq 0) {
    Write-Host "Migrations completed successfully."
} else {
    Write-Error "Migration failed."
    exit $LASTEXITCODE
}
