# OrcaAlgo Database Setup Script
# Run in PowerShell as Administrator

param(
    [string]$PgPassword = $env:PG_SUPERUSER_PASSWORD
)

$pgBin = "C:\Program Files\PostgreSQL\18\bin"
$psql = "$pgBin\psql.exe"

if (-not $PgPassword) {
    $PgPassword = Read-Host "Enter PostgreSQL superuser (postgres) password" -AsSecureString
    $BSTR = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($PgPassword)
    $PgPassword = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto($BSTR)
}

$env:PGPASSWORD = $PgPassword

Write-Host "Creating orca database user..."
& $psql -h localhost -p 5432 -U postgres -d postgres -w -c @"
DO
`$`$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'orca') THEN
      CREATE ROLE orca WITH LOGIN PASSWORD 'change_me' CREATEDB;
   END IF;
END
`$`$;
"@

if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to create orca user. Check postgres password."
    exit 1
}

Write-Host "Creating orca_core database..."
& $psql -h localhost -p 5432 -U postgres -d postgres -w -c "CREATE DATABASE orca_core OWNER orca" 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "Database may already exist (continuing)..."
}

Write-Host "Granting privileges..."
& $psql -h localhost -p 5432 -U postgres -d orca_core -w -c "GRANT ALL ON SCHEMA public TO orca" 2>&1
& $psql -h localhost -p 5432 -U postgres -d orca_core -w -c "ALTER DATABASE orca_core OWNER TO orca" 2>&1

Write-Host "Setup complete. You can now start orca-server."
