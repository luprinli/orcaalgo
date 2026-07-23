# Pre-commit scope detection — runs only relevant linters for changed files
# Usage: powershell -File scripts/pre-commit-scope.ps1

$ErrorActionPreference = "Stop"
$changed = git diff --cached --name-only --diff-filter=ACM
$exitCode = 0

$pythonChanged = $changed | Where-Object { $_ -match "^orca/|^tests/" }
$goChanged = $changed | Where-Object { $_ -match "\.go$" }
$webChanged = $changed | Where-Object { $_ -match "^web/src/.*\.tsx?$" }
$odinChanged = $changed | Where-Object { $_ -match "^odin/" }
$gkrChanged = $changed | Where-Object { $_ -match "^configs/strategies/.*\.gkr\.yaml$" }

if ($pythonChanged) {
    Write-Host "[pre-commit] Python files changed — running ruff + mypy"
    ruff check orca/ tests/
    if ($LASTEXITCODE -ne 0) { $exitCode = 1 }
    mypy orca/
    if ($LASTEXITCODE -ne 0) { $exitCode = 1 }
}

if ($goChanged) {
    Write-Host "[pre-commit] Go files changed — running go vet + gofmt"
    go vet ./...
    if ($LASTEXITCODE -ne 0) { $exitCode = 1 }
    $goFiles = ($changed | Where-Object { $_ -match "\.go$" }) -join " "
    if ($goFiles) {
        $fmtResult = gofmt -l $goFiles
        if ($fmtResult) {
            Write-Host "gofmt would format these files:"
            Write-Host $fmtResult
            $exitCode = 1
        }
    }
}

if ($webChanged) {
    Write-Host "[pre-commit] React files changed — running tsc + eslint"
    Push-Location web
    try {
        npx tsc --noEmit
        if ($LASTEXITCODE -ne 0) { $exitCode = 1 }
        npx eslint src/ --max-warnings 0
        if ($LASTEXITCODE -ne 0) { $exitCode = 1 }
    } finally {
        Pop-Location
    }
}

if ($odinChanged) {
    Write-Host "[pre-commit] Odin files changed — attempting compilation check"
    powershell -File scripts/build_odin.ps1
    if ($LASTEXITCODE -ne 0) { $exitCode = 1 }
}

if ($gkrChanged) {
    Write-Host "[pre-commit] GKR strategy files changed — running orca validate"
    $gkrFiles = ($changed | Where-Object { $_ -match "^configs/strategies/.*\.gkr\.yaml$" }) -join " "
    orca validate $gkrFiles
    if ($LASTEXITCODE -ne 0) { $exitCode = 1 }
}

exit $exitCode
