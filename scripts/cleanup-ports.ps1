param(
    [int[]]$Ports = @(8080, 9090, 9091, 5173, 3000, 5433, 6379),
    [switch]$DryRun = $false,
    [switch]$Force = $false
)

$ErrorActionPreference = "Stop"
$preferredPorts = @{
    "8080" = "orca-server API"
    "9090" = "Prometheus metrics"
    "9091" = "orca-server WebSocket/metrics"
    "5173" = "React dev server"
    "3000" = "Grafana dashboard"
    "5433" = "Orca PostgreSQL (Docker)"
    "6379" = "Redis"
}

$killed = @()

foreach ($port in $Ports) {
    $portStr = "$port"
    $name = $preferredPorts[$portStr]
    if (-not $name) { $name = "Port $port" }

    $connections = netstat -ano | Select-String -Pattern ":$port\s" | Select-String -Pattern "LISTENING"

    if (-not $connections) {
        Write-Host "[OK] $name :$port — free" -ForegroundColor Green
        continue
    }

    $pids = @()
    foreach ($line in $connections) {
        $parts = (-split ($line -replace '\s+', ' ').Trim())
        $pidStr = $parts[-1]
        if ($pidStr -match '^\d+$' -and [int]$pidStr -ne 0) {
            $pids += [int]$pidStr
        }
    }
    $pids = $pids | Select-Object -Unique

    foreach ($pid in $pids) {
        try {
            $process = Get-Process -Id $pid -ErrorAction Stop
            $procName = $process.ProcessName

            if ($procName -eq "com.docker.backend" -or $procName -eq "com.docker.vpnkit" -or $procName -eq "vpnkit") {
                Write-Host "[SKIP] $name :$port — PID $pid is Docker daemon ($procName), will not kill" -ForegroundColor Yellow
                continue
            }

            if ($Force) {
                Write-Host "[KILL] $name :$port — PID $pid ($procName)" -ForegroundColor Yellow
                if (-not $DryRun) {
                    Stop-Process -Id $pid -Force -ErrorAction Stop
                    Write-Host "[DONE] Killed PID $pid ($procName)" -ForegroundColor Green
                    $killed += @{ Port = $port; PID = $pid; Name = $procName }
                }
            }
            else {
                Write-Host "[BLOCK] $name :$port — PID $pid ($procName) — use -Force to kill" -ForegroundColor Red
            }
        }
        catch {
            Write-Host "[WARN] $name :$port — PID $pid no longer exists, skipping" -ForegroundColor DarkYellow
        }
    }
}

if ($killed.Count -gt 0) {
    Write-Host ""
    Write-Host "Killed $($killed.Count) process(es):" -ForegroundColor Cyan
    foreach ($k in $killed) {
        Write-Host "  Port $($k.Port): PID $($k.PID) ($($k.Name))" -ForegroundColor Cyan
    }

    Start-Sleep -Seconds 2
    Write-Host ""
    Write-Host "Verifying ports after cleanup..." -ForegroundColor Cyan

    $stillBlocked = $false
    foreach ($k in $killed) {
        $port = $k.Port
        $check = netstat -ano | Select-String -Pattern ":$port\s" | Select-String -Pattern "LISTENING"
        if ($check) {
            Write-Host "[FAIL] Port $port still occupied" -ForegroundColor Red
            $stillBlocked = $true
        }
        else {
            Write-Host "[OK] Port $port now free" -ForegroundColor Green
        }
    }

    if ($stillBlocked) {
        Write-Host ""
        Write-Host "Some ports could not be freed. Run with elevated permissions or check Docker." -ForegroundColor Red
        exit 1
    }
}
else {
    Write-Host ""
    Write-Host "All ports are free." -ForegroundColor Green
}

if ($DryRun) {
    Write-Host "Dry run — no processes were killed." -ForegroundColor Cyan
}
