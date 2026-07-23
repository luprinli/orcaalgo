$ErrorActionPreference = "Continue"

Write-Host "=== Orca Core Build Pipeline ==="

Write-Host "`n[1/3] Building Odin Engine..."
& "$PSScriptRoot\build_odin.ps1"
if ($LASTEXITCODE -ne 0) {
    Write-Host "WARNING: Odin build skipped or failed (engine not required for Go-only development)"
}

Write-Host "`n[2/3] Building Go binaries..."
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    go build -o bin/orca-server.exe ./cmd/orca-server
    if ($LASTEXITCODE -ne 0) { Write-Error "Go server build failed"; exit 1 }

    go build -o bin/orca-cli.exe ./cmd/orca-cli
    if ($LASTEXITCODE -ne 0) { Write-Error "Go CLI build failed"; exit 1 }
} finally {
    Pop-Location
}

Write-Host "`n[3/3] Running tests..."
go test ./internal/... -count=1 -timeout 30s
if ($LASTEXITCODE -ne 0) { Write-Host "WARNING: Some tests failed" }

Write-Host "`nBuild complete. Binaries: bin/orca-server.exe, bin/orca-cli.exe"
