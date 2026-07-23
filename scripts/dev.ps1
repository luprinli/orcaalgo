$ErrorActionPreference = "Continue"

Write-Host "=== Orca Core Development Mode ==="
Write-Host "Starting Go server with hot-reload (air)..."
Write-Host "Starting Odin file watcher..."

$projectRoot = Split-Path $PSScriptRoot -Parent
Push-Location $projectRoot

$goJob = Start-Job -Name "go-air" -ScriptBlock {
    Set-Location $using:projectRoot
    go run github.com/cosmtrek/air@latest
}

$odinJob = Start-Job -Name "odin-watch" -ScriptBlock {
    $watcher = New-Object System.IO.FileSystemWatcher
    $watcher.Path = "$using:projectRoot\odin"
    $watcher.Filter = "*.odin"
    $watcher.IncludeSubdirectories = $true
    $watcher.EnableRaisingEvents = $true

    $lastBuild = [DateTime]::MinValue
    $action = {
        $now = [DateTime]::Now
        if (($now - $Event.MessageData).TotalSeconds -gt 3) {
            Write-Host "Odin source changed, rebuilding..."
            & "$using:projectRoot\scripts\build_odin.ps1"
            $Event.MessageData = $now
        }
    }
    Register-ObjectEvent $watcher "Changed" -Action $action -MessageData $lastBuild | Out-Null

    while ($true) { Start-Sleep -Seconds 1 }
}

Write-Host "Dev servers running. Press Ctrl+C to stop."
try {
    Wait-Job -Name "go-air", "odin-watch"
} finally {
    Stop-Job -Name "go-air", "odin-watch" -ErrorAction SilentlyContinue
    Remove-Job -Name "go-air", "odin-watch" -ErrorAction SilentlyContinue
    Pop-Location
}
