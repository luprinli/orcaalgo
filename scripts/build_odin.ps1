# Build Odin engine as shared library (DLL/.so)
# Requires Odin compiler: https://odin-lang.org/docs/install/

param(
    [string]$OutDir = "bin",
    [string]$BuildMode = "dll",
    [switch]$Release = $true,
    [switch]$Debug = $false
)

$ErrorActionPreference = "Stop"

$odinCmd = Get-Command odin -ErrorAction SilentlyContinue
if (-not $odinCmd) {
    $odinPaths = @(
        "C:\odin\odin.exe",
        "$env:USERPROFILE\odin\odin.exe",
        "/usr/local/bin/odin"
    )
    foreach ($p in $odinPaths) {
        if (Test-Path $p) {
            $odinCmd = $p
            break
        }
    }
}

if (-not $odinCmd) {
    Write-Error "Odin compiler not found. Install from https://odin-lang.org/docs/install/"
    exit 1
}

if (-not (Test-Path $OutDir)) {
    New-Item -ItemType Directory -Path $OutDir -Force | Out-Null
}

$odinFlags = @(
    "build", "odin/",
    "-out:$OutDir/orca_engine.$ext",
    "-build-mode:$BuildMode",
    "-o:speed",
    "-no-bounds-check",
    "-no-type-assert"
)

if ($Release) {
    $odinFlags += "-o:speed"
    $odinFlags += "-subsystem:windows"
}
if ($Debug) {
    $odinFlags += "-o:none"
    $odinFlags += "-debug"
}

$ext = if ($IsWindows -or $env:OS -eq "Windows_NT") { "dll" } else { "so" }

Write-Host "Building Odin engine: $odinCmd $odinFlags"
& $odinCmd $odinFlags

if ($LASTEXITCODE -ne 0) {
    Write-Error "Odin build failed with exit code $LASTEXITCODE"
    exit $LASTEXITCODE
}

Write-Host "Odin engine built successfully: $OutDir/orca_engine.$ext"
