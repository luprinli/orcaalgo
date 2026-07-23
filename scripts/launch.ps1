param(
    [switch]$ForceCleanup = $true,
    [switch]$SkipReact = $false,
    [switch]$SkipMonitoring = $false,
    [switch]$Native = $false,
    [switch]$DryRun = $false,
    [int]$HealthTimeout = 90,
    [string]$LogFile = ""
)

$ErrorActionPreference = "Stop"
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent $scriptDir
Set-Location $projectRoot

$ts = { Get-Date -Format "HH:mm:ss" }
function Write-Log { param([string]$Level, [string]$Service, [string]$Message)
    $c = @{ STEP="Magenta"; OK="Green"; WARN="Yellow"; ERROR="Red"; INFO="Cyan" }
    $color = $c[$Level]; if (-not $color) { $color = "White" }
    Write-Host "$(& $ts) " -NoNewline -ForegroundColor DarkGray
    Write-Host ("{0,-5}" -f $Level) -NoNewline -ForegroundColor $color
    Write-Host (" [{0,-14}] " -f $Service) -NoNewline
    Write-Host $Message
    if ($LogFile) { "$(& $ts) $Level [$Service] $Message" | Out-File $LogFile -Append -Encoding utf8 }
}

function Test-PortInUse { param([int]$Port)
    $conns = netstat -ano 2>$null | Select-String -Pattern ":$Port\s" | Select-String -Pattern "LISTENING"
    return $conns -ne $null
}

function Test-HttpHealth { param([int]$Port, [string]$Path, [int]$ExpectedStatus = 200)
    try {
        $uri = "http://localhost:$Port$Path"
        $req = [System.Net.WebRequest]::Create($uri)
        $req.Timeout = 5000
        $resp = $req.GetResponse()
        $status = [int]$resp.StatusCode
        $resp.Close()
        return $status -eq $ExpectedStatus
    } catch { return $false }
}

function Test-TcpConnect { param([int]$Port)
    try {
        $client = New-Object System.Net.Sockets.TcpClient
        $result = $client.BeginConnect("127.0.0.1", $Port, $null, $null)
        $success = $result.AsyncWaitHandle.WaitOne(3000)
        $client.Close()
        return $success
    } catch { return $false }
}

function Wait-Health { param([string]$Name, [int]$Port, [string]$Path = "", [int]$MaxSec = 60, [int]$Interval = 2)
    $elapsed = 0
    while ($elapsed -lt $MaxSec) {
        if ($Path) { $healthy = Test-HttpHealth -Port $Port -Path $Path }
        else { $healthy = Test-TcpConnect -Port $Port }
        if ($healthy) {
            Write-Log "OK" $Name "Healthy on :$Port ($elapsed s)"
            return $true
        }
        Start-Sleep -Seconds $Interval
        $elapsed += $Interval
        if ($elapsed % 10 -eq 0) {
            Write-Log "INFO" $Name "Waiting... ($elapsed s / $MaxSec s)"
        }
    }
    Write-Log "ERROR" $Name "Failed to become healthy within $MaxSec seconds"
    return $false
}

function Invoke-Shutdown { param($Procs, [string[]]$DockerSvcs)
    Write-Host ""
    Write-Log "STEP" "Shutdown" "Stopping all services..."
    if ($Procs) {
        foreach ($p in $Procs) {
            if ($p -and !$p.HasExited) {
                Write-Log "INFO" $p.ProcessName "Stopping PID $($p.Id)..."
                $p.Kill($true)
            }
        }
    }
    if ($DockerSvcs) {
        docker compose stop $DockerSvcs 2>$null
    }
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  OrcaAlgo — Full Launch Sequence" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Log "INFO" "Orchestrator" "Mode: $(if ($Native) {'Native'} else {'Docker'})"
Write-Log "INFO" "Orchestrator" "Platform: $($env:OS)"
Write-Log "INFO" "Orchestrator" "Project: $projectRoot"
Write-Host ""

# ── Step 0: Set required environment variables
if (-not $env:ORCA_JWT_SECRET) {
    $env:ORCA_JWT_SECRET = "launch-ps1-auto-generated-jwt-secret"
    Write-Log "WARN" "Env" "ORCA_JWT_SECRET not set — using auto-generated value for dev"
}
if (-not $env:ORCA_ADMIN_PASSWORD) {
    $env:ORCA_ADMIN_PASSWORD = "launch-ps1-auto-generated-admin-password"
    Write-Log "WARN" "Env" "ORCA_ADMIN_PASSWORD not set — using auto-generated value for dev"
}
if (-not $env:PAPER_TRADING) { $env:PAPER_TRADING = "true" }

# ── Step 1: Stop existing containers (Docker mode) or clean ports (Native mode)
Write-Log "STEP" "Pre-flight" "Preparing environment..."
if ($Native) {
    Write-Host ""
    Write-Host "[1/5] Cleaning up ports..." -ForegroundColor Yellow
    $ports = @(8080, 9090, 5173)
    if (-not $SkipMonitoring) { $ports += @(9091, 3000) }
    if ($ForceCleanup) {
        & "$scriptDir\cleanup-ports.ps1" -Force -Ports $ports
    } else {
        & "$scriptDir\cleanup-ports.ps1" -Ports $ports
    }
    if ($LASTEXITCODE -ne 0) {
        Write-Log "ERROR" "Pre-flight" "Port cleanup failed. Aborting."
        exit 1
    }
} else {
    Write-Host ""
    Write-Host "[1/4] Stopping existing containers..." -ForegroundColor Yellow
    docker compose down 2>$null
    Write-Log "OK" "Docker" "Existing containers stopped"
}

# ── Step 2: Start dependency services
Write-Host ""
Write-Host "[2/$(if ($Native) {5} else {4})] Starting dependency services..." -ForegroundColor Yellow

if ($Native) {
    $dockerSvcs = @("postgres", "redis")
    if (-not $SkipMonitoring) { $dockerSvcs += @("prometheus", "grafana") }
    Write-Log "STEP" "Docker" "Starting: $($dockerSvcs -join ', ')..."
    if (-not $DryRun) {
        docker compose up -d @dockerSvcs
        if ($LASTEXITCODE -ne 0) {
            Write-Log "ERROR" "Docker" "Failed to start dependency services. Aborting."
            exit 1
        }
    } else {
        Write-Log "INFO" "Docker" "DRY-RUN: Would start $($dockerSvcs -join ', ')"
    }
} else {
    Write-Host "      Building Docker image..." -ForegroundColor Yellow
    if (-not $DryRun) { docker build -t orca-core:latest . }
    Write-Host "      Starting all services via Docker Compose..." -ForegroundColor Yellow
    if (-not $DryRun) { docker compose up -d }
}

# ── Step 3: Health verification
Write-Host ""
Write-Host "[3/$(if ($Native) {5} else {4})] Health verification..." -ForegroundColor Yellow

if ($Native) {
    if (-not $DryRun) {
        if (-not (Wait-Health -Name "PostgreSQL" -Port 5433 -MaxSec 30)) { exit 1 }
        if (-not (Wait-Health -Name "Redis" -Port 6379 -MaxSec 15)) { exit 1 }
    } else {
        Write-Log "INFO" "Health" "DRY-RUN: Would wait for PostgreSQL and Redis"
    }
} else {
    Write-Host "      Waiting for orca-server to be healthy..."
    $maxWait = $HealthTimeout
    $waited = 0
    while ($waited -lt $maxWait) {
        Start-Sleep -Seconds 2
        $waited += 2
        $status = docker inspect --format='{{.State.Health.Status}}' orca-server 2>$null
        if ($status -eq "healthy") { break }
        if ($waited % 10 -eq 0) {
            Write-Log "INFO" "orca-server" "Waiting... ($waited s / $maxWait s)"
        }
    }
    if ($waited -ge $maxWait) {
        Write-Log "WARN" "orca-server" "Did not become healthy within ${maxWait}s"
    } else {
        Write-Log "OK" "orca-server" "Healthy ($waited s)"
    }
}

# ── Step 4: Start application (Native mode only — Docker already includes Go server)
$goProc = $null
if ($Native) {
    Write-Host ""
    Write-Host "[4/5] Starting Go API server..." -ForegroundColor Yellow
    if ($DryRun) {
        Write-Log "INFO" "Go API" "DRY-RUN: Would start: go run ./cmd/orca-server"
    } else {
        $psi = New-Object System.Diagnostics.ProcessStartInfo
        $psi.FileName = "go"
        $psi.Arguments = "run ./cmd/orca-server"
        $psi.WorkingDirectory = $projectRoot
        $psi.UseShellExecute = $false
        $psi.RedirectStandardOutput = $true
        $psi.RedirectStandardError = $true
        $psi.Environment["ORCA_JWT_SECRET"] = $env:ORCA_JWT_SECRET
        $psi.Environment["ORCA_ADMIN_PASSWORD"] = $env:ORCA_ADMIN_PASSWORD
        $psi.Environment["PAPER_TRADING"] = $env:PAPER_TRADING
        $goProc = [System.Diagnostics.Process]::Start($psi)
        Write-Log "OK" "Go API" "Started (PID $($goProc.Id)) — go run ./cmd/orca-server"
    }

    if (-not (Wait-Health -Name "Go API" -Port 8080 -Path "/api/v1/risk/status" -MaxSec 45 -Interval 3)) {
        if (-not $DryRun) {
            Write-Log "ERROR" "Go API" "Failed to start. Check logs above."
            Invoke-Shutdown -Procs @($goProc) -DockerSvcs @("postgres", "redis")
            exit 1
        }
    }
    if (-not (Wait-Health -Name "Metrics" -Port 9090 -Path "/healthz" -MaxSec 10)) {
        Write-Log "WARN" "Metrics" "Metrics server not responding on :9090"
    }
}

# ── Step 5: Start React dev server
$viteProc = $null
if (-not $SkipReact) {
    Write-Host ""
    Write-Host "[5/5] Starting React frontend..." -ForegroundColor Yellow

    Push-Location "$projectRoot\web"
    npm install --silent 2>$null
    if ($DryRun) {
        Write-Log "INFO" "React Dev" "DRY-RUN: Would start: npx vite --port 5173"
    } else {
        $vitePsi = New-Object System.Diagnostics.ProcessStartInfo
        $vitePsi.FileName = "npx"
        $vitePsi.Arguments = "vite --port 5173 --host 0.0.0.0"
        $vitePsi.WorkingDirectory = "$projectRoot\web"
        $vitePsi.UseShellExecute = $false
        $vitePsi.RedirectStandardOutput = $true
        $vitePsi.RedirectStandardError = $true
        $viteProc = [System.Diagnostics.Process]::Start($vitePsi)
        Write-Log "OK" "React Dev" "Started (PID $($viteProc.Id)) — npx vite --port 5173"
    }
    Pop-Location

    if (-not $DryRun) {
        if (-not (Wait-Health -Name "React Dev" -Port 5173 -Path "/" -MaxSec 30)) {
            Write-Log "WARN" "React Dev" "Not healthy — may still be compiling. Check http://localhost:5173"
        }
    }
}

# ── Registered shutdown handler
$shutdown = {
    Invoke-Shutdown -Procs @($goProc, $viteProc) -DockerSvcs @("postgres", "redis", "prometheus", "grafana")
    Write-Host "Shutdown complete." -ForegroundColor Green
}
if (-not $DryRun) {
    try { [Console]::TreatControlCAsInput = $false } catch {}
    $null = Register-EngineEvent -SourceIdentifier PowerShell.Exiting -Action $shutdown
}

# ── Final status
Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "  Launch Complete" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host "  API:        http://localhost:8080" -ForegroundColor White
Write-Host "  Health:     http://localhost:9090/healthz" -ForegroundColor White
Write-Host "  Dashboard:  http://localhost:5173" -ForegroundColor White
if (-not $SkipMonitoring) {
    Write-Host "  Prometheus: http://localhost:9091" -ForegroundColor White
    Write-Host "  Grafana:    http://localhost:3000" -ForegroundColor White
}
Write-Host "  PostgreSQL: localhost:5433" -ForegroundColor White
Write-Host "  Redis:      localhost:6379" -ForegroundColor White
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Press Ctrl+C to stop all services." -ForegroundColor DarkGray

if ($DryRun) {
    Write-Log "INFO" "Orchestrator" "Dry-run complete — no services were started."
    exit 0
}

try {
    while ($true) {
        Start-Sleep -Seconds 5
        if ($goProc -and $goProc.HasExited) {
            Write-Log "ERROR" "Go API" "Server exited unexpectedly (code $($goProc.ExitCode))"
            Invoke-Shutdown -Procs @($goProc, $viteProc) -DockerSvcs @("postgres", "redis")
            exit 1
        }
    }
} finally {
    Invoke-Shutdown -Procs @($goProc, $viteProc) -DockerSvcs @("postgres", "redis", "prometheus", "grafana")
}
