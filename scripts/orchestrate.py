#!/usr/bin/env python3
"""
OrcaAlgo — Multi-Service Startup Orchestrator

Manages the initialization sequence for the OrcaAlgo application stack:
  - PostgreSQL → Redis → Go API (8080) → React Dev (5173)
  - Optional: Prometheus (9091), Grafana (3000)

Features:
  - Identifies and terminates processes occupying required ports
  - Launches dependency services in correct order
  - Health-checks each service before proceeding to the next
  - Supports Docker-based, Native, and Local-database modes
  - Auto-detects local PostgreSQL installations on Windows/Mac/Linux
  - Graceful shutdown on SIGINT/SIGTERM (Ctrl+C)
  - Structured logging with timestamps and service tags

Usage:
  python scripts/orchestrate.py                    # Docker mode, exits after services healthy
  python scripts/orchestrate.py --monitor          # Keep running + health-check (Ctrl+C to stop)
  python scripts/orchestrate.py --production       # Production: pre-built binary, preflight + migration checks
  python scripts/orchestrate.py --production --seed # Production + seed database
  python scripts/orchestrate.py --local            # Use local PostgreSQL (auto-detect), skip Docker
  python scripts/orchestrate.py --no-react         # Skip frontend dev server
  python scripts/orchestrate.py --dry-run          # Show what would happen, don't execute
"""

from __future__ import annotations

import argparse
import http.client
import os
import platform
import shutil
import signal
import socket
import subprocess
import sys
import time
import venv
from dataclasses import dataclass, field
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

PROJECT_ROOT = Path(__file__).resolve().parent.parent
VENV_DIR = PROJECT_ROOT / ".venv"
PYTHON_MIN_VERSION = (3, 11)
WINDOWS = platform.system() == "Windows"

COLORS = {
    "reset":   "\033[0m",
    "cyan":    "\033[96m",
    "green":   "\033[92m",
    "yellow":  "\033[93m",
    "red":     "\033[91m",
    "magenta": "\033[95m",
    "gray":    "\033[90m",
    "bold":    "\033[1m",
}
if platform.system() == "Windows":
    import ctypes
    kernel32 = ctypes.windll.kernel32
    kernel32.SetConsoleMode(kernel32.GetStdHandle(-11), 7)
    for key in list(COLORS.keys()):
        COLORS[key] = ""


def get_python_exe(min_version: tuple[int, int] = PYTHON_MIN_VERSION) -> str | None:
    """Find a Python interpreter meeting the minimum version requirement.

    Checks, in order:
      1. VIRTUAL_ENV / CONDA_PREFIX environment variable
      2. python3 / python on PATH
      3. Windows: py launcher with version flag

    Returns path to the Python executable, or None if no suitable interpreter found.
    """
    venv_python = os.environ.get("VIRTUAL_ENV")
    if venv_python:
        candidate = Path(venv_python) / ("Scripts" if WINDOWS else "bin") / "python.exe" if WINDOWS else Path(venv_python) / "bin" / "python3"
        if candidate.exists():
            return str(candidate)

    conda_prefix = os.environ.get("CONDA_PREFIX")
    if conda_prefix:
        candidate = Path(conda_prefix) / "python.exe" if WINDOWS else Path(conda_prefix) / "bin" / "python3"
        if candidate.exists():
            return str(candidate)

    for name in ("python3", "python"):
        found = shutil.which(name)
        if found:
            try:
                result = subprocess.run(
                    [found, "--version"], capture_output=True, text=True, timeout=5
                )
                version_str = result.stdout.strip() or result.stderr.strip()
                if version_str.startswith("Python "):
                    parts = version_str.split()[1].split(".")
                    ver = (int(parts[0]), int(parts[1]))
                    if ver >= min_version:
                        return found
            except (subprocess.CalledProcessError, ValueError, IndexError):
                continue

    if WINDOWS:
        try:
            for tag in ("3.11", "3.12", "3.13", "3.14"):
                result = subprocess.run(
                    ["py", f"-{tag}", "--version"], capture_output=True, text=True, timeout=5
                )
                if result.returncode == 0 and "Python" in (result.stdout or result.stderr):
                    return f"py -{tag}"
        except (subprocess.CalledProcessError, FileNotFoundError):
            pass

    return None


def find_venv_python(venv_path: Path) -> Path:
    """Return the path to the Python executable inside a virtual environment."""
    if WINDOWS:
        return venv_path / "Scripts" / "python.exe"
    return venv_path / "bin" / "python3"


def is_venv_valid(venv_path: Path) -> bool:
    """Check if an existing venv is functional."""
    python_exe = find_venv_python(venv_path)
    if not python_exe.exists():
        return False
    try:
        result = subprocess.run(
            [str(python_exe), "--version"], capture_output=True, text=True, timeout=10
        )
        return result.returncode == 0 and "Python" in (result.stdout or result.stderr)
    except Exception:
        return False


def create_venv(venv_path: Path, python_exe: str | None = None,
                clear: bool = False, with_pip: bool = True) -> bool:
    """Create a Python virtual environment at venv_path.

    Args:
        venv_path: Target directory for the virtual environment.
        python_exe: Python interpreter to use. Auto-detected if None.
        clear: Remove existing venv before creating.
        with_pip: Ensure pip is installed in the venv.

    Returns True on success.
    """
    if clear and venv_path.exists():
        log("INFO", "venv", f"Removing existing venv at {venv_path}")
        shutil.rmtree(venv_path, ignore_errors=True)

    if venv_path.exists():
        if is_venv_valid(venv_path):
            log("OK", "venv", f"Existing venv at {venv_path} is valid — reusing")
            return True
        log("WARN", "venv", f"Existing venv at {venv_path} is broken — recreating")
        shutil.rmtree(venv_path, ignore_errors=True)

    resolved_python = python_exe or get_python_exe()
    if not resolved_python:
        log("ERROR", "venv", f"No Python {PYTHON_MIN_VERSION[0]}.{PYTHON_MIN_VERSION[1]}+ interpreter found")
        log("ERROR", "venv", "Install Python >= 3.11 from https://python.org or use --python-path")
        return False

    log("STEP", "venv", f"Creating virtual environment at {venv_path} (Python: {resolved_python})")

    class _VenvBuilder(venv.EnvBuilder):
        def __init__(self) -> None:
            super().__init__(with_pip=with_pip, clear=False, prompt="orca")

        def post_setup(self, context: Any) -> None:
            pass

    try:
        builder = _VenvBuilder()
        builder.create(str(venv_path))
        log("OK", "venv", f"Virtual environment created at {venv_path}")
        return True
    except Exception as e:
        log("ERROR", "venv", f"Failed to create venv: {e}")
        return False


def install_python_deps(venv_path: Path, extras: list[str] | None = None,
                         upgrade_pip: bool = True) -> bool:
    """Install Python project dependencies into the virtual environment.

    Args:
        venv_path: Path to the virtual environment.
        extras: List of optional dependency groups (e.g., ['dev', 'ml']).
        upgrade_pip: Upgrade pip before installing dependencies.

    Returns True on success.
    """
    python_exe = find_venv_python(venv_path)
    if not python_exe.exists():
        log("ERROR", "venv", f"Python not found in venv at {python_exe}")
        return False

    def _pip_install(args: list[str]) -> bool:
        cmd = [str(python_exe), "-m", "pip", "install"] + args
        log("INFO", "venv", f"Running: {' '.join(cmd)}")
        timeout = 600 if any("ml" in str(a) or "lightgbm" in str(a) or "xgboost" in str(a) for a in args) else 180
        result = subprocess.run(
            cmd, cwd=str(PROJECT_ROOT), capture_output=True, text=True, timeout=timeout
        )
        if result.returncode != 0:
            stderr_tail = result.stderr.strip()[-300:] if result.stderr else ""
            log("ERROR", "venv", f"pip install failed: {stderr_tail}")
            return False
        return True

    if upgrade_pip:
        if not _pip_install(["--upgrade", "pip", "--quiet"]):
            return False

    install_target = str(PROJECT_ROOT)
    extra_flag = f"[{','.join(extras)}]" if extras else ""

    if extra_flag:
        if not _pip_install(["-e", f"{install_target}{extra_flag}", "--quiet"]):
            return False
    else:
        if not _pip_install(["-e", install_target, "--quiet"]):
            return False

    log("OK", "venv", "Dependencies installed successfully" + (f" (extras: {extras})" if extras else ""))

    pyproject = PROJECT_ROOT / "pyproject.toml"
    if pyproject.exists():
        _verify_deps(python_exe)

    return True


def _verify_deps(python_exe: Path) -> None:
    """Quick-verify core dependencies are importable."""
    for mod in ("pydantic", "yaml", "numpy", "typer"):
        result = subprocess.run(
            [str(python_exe), "-c", f"import {mod}"],
            capture_output=True, text=True, timeout=10
        )
        if result.returncode != 0:
            log("WARN", "venv", f"Module '{mod}' not importable — may need reinstall")


def setup_python_environment(venv_path: Path, force_recreate: bool = False,
                               install_dev: bool = True, install_ml: bool = False,
                               python_exe: str | None = None,
                               skip_install: bool = False,
                               dry_run: bool = False) -> Path | None:
    """Set up the Python virtual environment and install dependencies.

    This is the main entry point for venv management. Called early in the
    orchestrator startup sequence to ensure Python services have a working
    environment before any subprocess calls.

    Returns the path to the venv's Python executable, or None on failure.
    """
    if dry_run:
        log("OK", "venv", f"DRY-RUN: Would create venv at {venv_path}")
        log("OK", "venv", f"DRY-RUN: Would install deps from {PROJECT_ROOT}")
        return find_venv_python(venv_path)

    if not create_venv(venv_path, python_exe=python_exe, clear=force_recreate):
        return None

    if skip_install:
        log("INFO", "venv", "Skipping dependency installation (--skip-pip-install)")
        return find_venv_python(venv_path)

    extras: list[str] = []
    if install_dev:
        extras.append("dev")
    if install_ml:
        extras.append("ml")

    if not install_python_deps(venv_path, extras=extras if extras else None):
        log("WARN", "venv", "Dependency installation had issues — some Python commands may fail")
        return find_venv_python(venv_path)

    return find_venv_python(venv_path)


@dataclass
class Service:
    name: str
    port: int
    health_url: str = ""
    health_method: str = "GET"
    health_expected_status: int = 200
    health_max_retries: int = 30
    health_retry_delay: float = 2.0
    depends_on: list[str] = field(default_factory=list)
    startup_cmd: list[str] | None = None
    startup_cwd: str | None = None
    env: dict[str, str] = field(default_factory=dict)
    docker_service: str | None = None
    optional: bool = False
    process: subprocess.Popen | None = field(default=None, init=False)

    @property
    def label(self) -> str:
        return f"[{self.name}:{self.port}]"


def _build_services(local_mode: bool = False,
                    db_host: str = "localhost", db_port: int = 5432,
                    db_user: str = "postgres", db_password: str = "",
                    db_name: str = "orca_core", db_sslmode: str = "disable",
                    skip_metrics: bool = False,
                    production: bool = False) -> list[Service]:
    """Build service list, adapting for local DB mode when requested."""
    go_cmd = ["go", "run", "./cmd/orca-server"]
    go_name = "Go API"
    if production:
        bin_path = str(PROJECT_ROOT / "bin" / "orca-server.exe")
        if Path(bin_path).exists():
            go_cmd = [bin_path]
            go_name = "Orca Server (prod)"
        else:
            log("WARN", "Orchestrator", f"Binary not found at {bin_path}, falling back to go run")
            go_cmd = ["go", "run", "./cmd/orca-server"]

    if local_mode:
        postgres_svc = Service(
            name="PostgreSQL",
            port=db_port,
            health_url="",
            env={
                "ORCA_DB_HOST": db_host,
                "ORCA_DB_PORT": str(db_port),
                "ORCA_DB_USER": db_user,
                "ORCA_DB_PASSWORD": db_password,
                "ORCA_DB_NAME": db_name,
                "ORCA_DB_SSLMODE": db_sslmode,
            },
        )
    else:
        postgres_svc = Service(
            name="PostgreSQL",
            port=5433,
            docker_service="postgres",
            health_url="",
        )

    services: list[Service] = [postgres_svc]

    go_env: dict[str, str] = {}
    if local_mode:
        go_env = {
            "ORCA_DB_HOST": db_host,
            "ORCA_DB_PORT": str(db_port),
            "ORCA_DB_USER": db_user,
            "ORCA_DB_PASSWORD": db_password,
            "ORCA_DB_NAME": db_name,
            "ORCA_DB_SSLMODE": db_sslmode,
            "PAPER_TRADING": "true",
        }

    services.append(Service(
        name=go_name,
        port=8080,
        health_url="/api/v1/system/health",
        health_max_retries=30,
        health_retry_delay=3.0,
        depends_on=["PostgreSQL"],
        startup_cmd=go_cmd,
        startup_cwd=str(PROJECT_ROOT),
        env=go_env,
    ))

    if not skip_metrics:
        services.append(Service(
            name="Metrics",
            port=9090,
            health_url="/healthz",
            health_expected_status=200,
            depends_on=["Go API"],
        ))

    services.append(Service(
        name="React Dev",
        port=5173,
        health_url="/",
        health_max_retries=20,
        health_retry_delay=2.0,
        depends_on=["Go API"],
        startup_cmd=["npx", "vite", "--port", "5173", "--host", "0.0.0.0"],
        startup_cwd=str(PROJECT_ROOT / "web"),
        optional=True,
    ))

    services.append(Service(
        name="Prometheus",
        port=9091,
        docker_service="prometheus",
        optional=True,
    ))
    services.append(Service(
        name="Grafana",
        port=3000,
        docker_service="grafana",
        optional=True,
    ))

    return services


SERVICES: list[Service] = _build_services(local_mode=False)

_log_file: Path | None = None


def log(level: str, service: str, message: str) -> None:
    ts = datetime.now(UTC).strftime("%H:%M:%S")
    color = {"INFO": COLORS["cyan"], "OK": COLORS["green"], "WARN": COLORS["yellow"],
             "ERROR": COLORS["red"], "STEP": COLORS["magenta"]}.get(level, "")
    line = f"{COLORS['gray']}{ts}{COLORS['reset']} {color}{level:5}{COLORS['reset']} [{service:14}] {message}"
    print(line, flush=True)
    if _log_file:
        with open(_log_file, "a") as f:
            f.write(f"{ts} {level:5} [{service:14}] {message}\n")


def is_port_in_use(port: int) -> list[tuple[int, str]]:
    """Return list of unique (pid, process_name) tuples occupying the port."""
    results: set[tuple[int, str]] = set()
    system = platform.system()

    if system == "Windows":
        try:
            output = subprocess.check_output(
                ["netstat", "-ano"], text=True, timeout=10
            )
            for line in output.splitlines():
                if f":{port}" in line and "LISTENING" in line:
                    parts = line.strip().split()
                    pid_str = parts[-1]
                    if pid_str.isdigit() and int(pid_str) != 0:
                        pid = int(pid_str)
                        try:
                            proc_output = subprocess.check_output(
                                ["tasklist", "/FI", f"PID eq {pid}", "/FO", "CSV", "/NH"],
                                text=True, timeout=5
                            )
                            name = proc_output.strip().split('","')[0].strip('"')
                        except Exception:
                            name = "unknown"
                        results.add((pid, name))
        except (subprocess.CalledProcessError, FileNotFoundError):
            pass
    else:
        try:
            output = subprocess.check_output(
                ["lsof", "-i", f":{port}", "-t", "-sTCP:LISTEN"],
                text=True, timeout=10
            )
            for pid_str in output.strip().splitlines():
                if pid_str.isdigit():
                    pid = int(pid_str)
                    try:
                        name = subprocess.check_output(
                            ["ps", "-p", str(pid), "-o", "comm="], text=True, timeout=5
                        ).strip()
                    except Exception:
                        name = "unknown"
                    results.add((pid, name))
        except (subprocess.CalledProcessError, FileNotFoundError):
            pass

    return sorted(results)


def stop_windows_service(service_names: list[str]) -> None:
    """Stop Windows services by name (e.g., 'postgresql-x64-16')."""
    for name in service_names:
        try:
            result = subprocess.run(
                ["sc", "stop", name], capture_output=True, text=True, timeout=15
            )
            if result.returncode == 0 or "STOP_PENDING" in result.stdout or "not started" in result.stdout.lower():
                log("OK", "Service", f"Stopped '{name}'")
            else:
                log("WARN", "Service", f"Could not stop '{name}': {result.stderr.strip() or result.stdout.strip()}")
        except Exception as e:
            log("WARN", "Service", f"Failed to stop '{name}': {e}")


def find_postgresql_service() -> list[str]:
    """Find PostgreSQL Windows services by scanning service names."""
    try:
        result = subprocess.run(
            ["sc", "query", "state=", "all"], capture_output=True, text=True, timeout=10
        )
        services = []
        for line in result.stdout.splitlines():
            if "postgres" in line.lower():
                parts = line.strip().split(":")
                if len(parts) >= 2:
                    services.append(parts[1].strip())
        return services
    except Exception:
        return []


def is_windows_service(pid: int) -> bool:
    """Check if a process is running as a Windows service (should not be killed)."""
    if platform.system() != "Windows":
        return False
    try:
        result = subprocess.run(
            ["tasklist", "/FI", f"PID eq {pid}", "/FO", "CSV", "/NH", "/SVC"],
            capture_output=True, text=True, timeout=5
        )
        return "N/A" not in result.stdout and len(result.stdout.strip().split('","')) > 3
    except Exception:
        return False


def analyze_port_conflict(port: int, local_mode: bool = False) -> dict[str, Any]:
    """Analyze what is occupying a port and return structured info.
    Returns dict with: occupied (bool), processes (list of {pid, name, is_service}), guidance (str)."""
    procs = is_port_in_use(port)
    if not procs:
        return {"occupied": False, "processes": [], "guidance": ""}

    details = []
    for pid, name in procs:
        is_svc = is_windows_service(pid)
        details.append({"pid": pid, "name": name, "is_service": is_svc})

    names = [d["name"] for d in details]
    is_docker = any("docker" in n.lower() or "com.docker" in n.lower() for n in names)
    is_postgres = any("postgres" in n.lower() for n in names)
    is_orca = any("orca" in n.lower() for n in names)
    is_node = any("node" in n.lower() for n in names)
    has_service = any(d["is_service"] for d in details)

    if local_mode and is_postgres:
        return {"occupied": True, "processes": details, "guidance": "", "local_ok": True,
                "message": "Local PostgreSQL already running — using existing service"}
    if local_mode and is_orca:
        return {"occupied": True, "processes": details, "guidance": "", "local_ok": True,
                "message": "Orca server already running — using existing process"}
    if local_mode and is_node and port == 5173:
        return {"occupied": True, "processes": details, "guidance": "", "local_ok": True,
                "message": "React dev server already running — using existing process"}

    if is_orca:
        guidance = "Orca server already running. Use --force to stop and restart it."
    elif is_docker:
        guidance = "Port used by Docker. Run 'docker compose stop postgres redis' to free it."
    elif has_service:
        guidance = "Port used by a Windows service. Stop it with 'net stop <service_name>' or run as Administrator with --force."
    elif is_postgres:
        guidance = "Port used by PostgreSQL. Stop the service or run with --force to auto-stop it."
    else:
        guidance = "Port occupied by an unknown process. Use --force to kill it, or stop it manually."

    return {"occupied": True, "processes": details, "guidance": guidance, "local_ok": False}


def stop_project_docker_services(docker_svcs: list[str]) -> bool:
    """Gracefully stop Docker Compose services that belong to this project.
    Returns True if all specified services are stopped."""
    if not docker_available() or not docker_svcs:
        return True

    log("INFO", "Docker", f"Gracefully stopping existing project containers: {', '.join(docker_svcs)}")
    subprocess.run(
        ["docker", "compose", "stop"] + docker_svcs,
        cwd=str(PROJECT_ROOT), capture_output=True, text=True, timeout=30
    )
    time.sleep(2)

    all_stopped = True
    for svc in docker_svcs:
        check = subprocess.run(
            ["docker", "compose", "ps", "-q", svc],
            cwd=str(PROJECT_ROOT), capture_output=True, text=True, timeout=10
        )
        if check.stdout.strip():
            log("WARN", "Docker", f"Container '{svc}' may still be running")
            all_stopped = False
        else:
            log("OK", "Docker", f"Container '{svc}' stopped")

    return all_stopped


def resolve_port_conflicts(ports: list[int], force: bool, dry_run: bool,
                            docker_svcs: list[str], optional_ports: set[int] | None = None,
                            local_mode: bool = False) -> bool:
    """Resolve port conflicts with a conflict-first, graceful approach.

    Strategy (in order of escalation):
      1. If ports are free → proceed
      2. In --local mode, if ports are occupied by expected local services → proceed
      3. If Docker containers own the ports → 'docker compose stop' gracefully
      4. If ports still occupied → report what's there with specific guidance
      5. If --force and Windows PostgreSQL service → 'sc stop' then kill stale orphans
      6. If --force and non-service processes → kill them

    Optional ports that remain occupied after cleanup are silently skipped.
    Returns True if all required ports are free after resolution."""
    if optional_ports is None:
        optional_ports = set()

    if dry_run:
        for port in ports:
            conflict = analyze_port_conflict(port, local_mode=local_mode)
            if conflict["occupied"]:
                if conflict.get("local_ok"):
                    log("OK", f"Port {port}", conflict.get("message", "Local service detected"))
                else:
                    log("WARN", f"Port {port}", f"DRY-RUN: {conflict['guidance']}")
            else:
                log("OK", f"Port {port}", "Free (dry-run)")
        return True

    local_ok_ports: set[int] = set()
    for port in ports:
        conflict = analyze_port_conflict(port, local_mode=local_mode)
        if not conflict["occupied"]:
            log("OK", f"Port {port}", "Free")
            continue
        if conflict.get("local_ok"):
            log("OK", f"Port {port}", conflict.get("message", "Local service detected"))
            local_ok_ports.add(port)
            continue
        names = ", ".join(f"PID {d['pid']} ({d['name']})" for d in conflict["processes"])
        log("INFO", f"Port {port}", f"Occupied by: {names}")

    ports_to_clean = [p for p in ports if p not in optional_ports and p not in local_ok_ports]
    if not ports_to_clean:
        return True

    stop_project_docker_services(docker_svcs)

    for port in ports_to_clean:
        conflict = analyze_port_conflict(port, local_mode=local_mode)
        if not conflict["occupied"]:
            log("OK", f"Port {port}", "Free (after Docker stop)")
            continue

        if not force:
            log("ERROR", f"Port {port}", conflict["guidance"])
            continue

        log("INFO", f"Port {port}", "Scanning for related Windows services...")
        pg_services = find_postgresql_service()
        if pg_services:
            log("INFO", f"Port {port}", f"Stopping PostgreSQL services: {pg_services}")
            stop_windows_service(pg_services)
            time.sleep(3)

        remaining = is_port_in_use(port)
        for pid, name in remaining:
            details = analyze_port_conflict(port, local_mode=local_mode)
            is_svc = any(d["is_service"] for d in details.get("processes", []))
            if is_svc:
                log("WARN", f"Port {port}",
                    f"PID {pid} ({name}) is a Windows service — cannot kill. Stop it manually.")
                continue
            if any(x in name.lower() for x in ("docker", "com.docker", "vpnkit")):
                log("WARN", f"Port {port}", f"PID {pid} ({name}) is Docker daemon — cannot kill")
                continue
            if any(x in name.lower() for x in ("orca-server", "orca")):
                log("INFO", f"Port {port}", f"Stopping stale orca process PID {pid} ({name})")
            try:
                if platform.system() == "Windows":
                    subprocess.run(["taskkill", "/F", "/T", "/PID", str(pid)],
                                  capture_output=True, timeout=10)
                else:
                    os.kill(pid, signal.SIGKILL)
                log("OK", f"Port {port}", f"Killed stale PID {pid} ({name})")
            except Exception as e:
                log("WARN", f"Port {port}", f"Could not kill PID {pid} ({name}): {e}")

    time.sleep(2)
    for attempt in range(3):
        blocked = [port for port in ports_to_clean if port not in optional_ports and is_port_in_use(port)]
        if not blocked:
            return True
        if attempt < 2:
            remaining_names: dict[int, list[str]] = {}
            for port in blocked:
                remaining_names[port] = [f"PID {p} ({n})" for p, n in is_port_in_use(port)]
            log("INFO", "Port Cleanup", f"Retry {attempt + 2}/3 — waiting for: {dict(remaining_names)}")
            time.sleep(3)

    final_blocked = [port for port in ports_to_clean if port not in optional_ports and is_port_in_use(port)]
    if final_blocked:
        for port in final_blocked:
            details = analyze_port_conflict(port, local_mode=local_mode)
            names = ", ".join(f"PID {d['pid']} ({d['name']})" for d in details["processes"])
            log("ERROR", f"Port {port}", f"Still occupied by: {names}")
            log("ERROR", f"Port {port}", details["guidance"])
        return False
    return True


def docker_available() -> bool:
    try:
        result = subprocess.run(["docker", "ps"], capture_output=True, timeout=10)
        return result.returncode == 0
    except (FileNotFoundError, subprocess.TimeoutExpired):
        return False


def start_docker_services(services: list[str]) -> bool:
    if not docker_available():
        log("ERROR", "Docker", "Docker is not available or not running")
        return False
    for svc in services:
        log("STEP", svc, "Starting via Docker Compose...")
    result = subprocess.run(
        ["docker", "compose", "up", "-d"] + services,
        cwd=str(PROJECT_ROOT), capture_output=True, text=True, timeout=120
    )
    if result.returncode != 0:
        log("ERROR", "Docker", result.stderr.strip() or "Failed to start services")
        return False
    log("OK", "Docker", f"Started: {', '.join(services)}")
    return True


def check_health(host: str, port: int, path: str, method: str = "GET",
                 expected: int = 200) -> bool:
    try:
        conn = http.client.HTTPConnection(host, port, timeout=5)
        conn.request(method, path)
        resp = conn.getresponse()
        conn.close()
        return resp.status == expected
    except Exception:
        return False


def db_health(port: int, _kind: str = "postgres") -> bool:
    """Check if a database service is accepting connections via TCP socket."""
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(3)
        result = sock.connect_ex(("127.0.0.1", port))
        sock.close()
        return result == 0
    except Exception:
        return False


def wait_for_service(svc: Service, dry_run: bool) -> bool:
    if dry_run:
        log("OK", svc.name, f"DRY-RUN: Would wait for health on :{svc.port}")
        return True

    for attempt in range(svc.health_max_retries):
        if svc.health_url:
            healthy = check_health("127.0.0.1", svc.port, svc.health_url,
                                    svc.health_method, svc.health_expected_status)
        else:
            healthy = db_health(svc.port, "postgres" if "postgres" in svc.name.lower() else "redis")

        if healthy:
            log("OK", svc.name, f"Healthy on :{svc.port} (attempt {attempt + 1})")
            return True

        elapsed = (attempt + 1) * svc.health_retry_delay
        log("INFO", svc.name,
            f"Waiting for health... ({elapsed:.0f}s / {svc.health_max_retries * svc.health_retry_delay:.0f}s)")
        time.sleep(svc.health_retry_delay)

    log("ERROR", svc.name, f"Failed to become healthy within {svc.health_max_retries * svc.health_retry_delay:.0f}s")
    return False


def start_native_service(svc: Service, env_overrides: dict[str, str], dry_run: bool) -> bool:
    if dry_run:
        log("OK", svc.name, f"DRY-RUN: Would start: {' '.join(svc.startup_cmd or [])}")
        return True

    if not svc.startup_cmd:
        return True

    if svc.health_url:
        if check_health("127.0.0.1", svc.port, svc.health_url, svc.health_method, svc.health_expected_status):
            log("OK", svc.name, f"Already healthy on :{svc.port} — skipping start")
            return True

    env = os.environ.copy()
    env.update(env_overrides)
    env.update(svc.env)

    try:
        proc = subprocess.Popen(
            svc.startup_cmd,
            cwd=svc.startup_cwd or str(PROJECT_ROOT),
            env=env,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            shell=(platform.system() == "Windows"),
        )
        svc.process = proc
        log("OK", svc.name, f"Started (PID {proc.pid}): {' '.join(svc.startup_cmd)}")
        return True
    except FileNotFoundError as e:
        log("ERROR", svc.name, f"Command not found: {e}")
        return False
    except Exception as e:
        log("ERROR", svc.name, f"Failed to start: {e}")
        return False


def resolve_service_order(services: list[Service], enabled_names: set[str],
                           skip_names: set[str]) -> list[Service]:
    active = {s.name: s for s in services if s.name in enabled_names and s.name not in skip_names}
    resolved: list[Service] = []
    visited: set[str] = set()

    def visit(svc: Service) -> None:
        if svc.name in visited:
            return
        visited.add(svc.name)
        for dep_name in svc.depends_on:
            dep = active.get(dep_name)
            if dep is not None:
                visit(dep)
        resolved.append(svc)

    for s in sorted(active.values(), key=lambda s: (s.optional, s.port)):
        visit(s)

    return resolved


def shutdown(services: list[Service], docker_services: list[str]) -> None:
    print()
    log("STEP", "Shutdown", "Stopping all services...")
    for svc in reversed(services):
        if svc.process and svc.process.poll() is None:
            log("INFO", svc.name, f"Stopping PID {svc.process.pid}...")
            svc.process.terminate()
            try:
                svc.process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                svc.process.kill()
            log("OK", svc.name, "Stopped")

    if docker_services and docker_available():
        log("INFO", "Docker", f"Stopping: {', '.join(docker_services)}")
        subprocess.run(
            ["docker", "compose", "stop"] + docker_services,
            cwd=str(PROJECT_ROOT), capture_output=True, timeout=30
        )
    log("OK", "Shutdown", "All services stopped.")


def load_dotenv(dotenv_path: Path) -> dict[str, str]:
    """Parse a .env file into a dict of key=value pairs. Does NOT override os.environ."""
    result: dict[str, str] = {}
    if not dotenv_path.exists():
        return result
    with open(dotenv_path) as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            if "=" in line:
                key, _, value = line.partition("=")
                key = key.strip()
                value = value.strip().strip('"').strip("'")
                result[key] = value
    return result


def build_cli() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="OrcaAlgo — Multi-Service Startup Orchestrator",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--local", action="store_true",
                        help="Use local PostgreSQL (auto-detect), skip Docker")
    parser.add_argument("--docker", action="store_true",
                        help="Run ALL services via Docker Compose (legacy full-Docker mode)")
    parser.add_argument("--production", action="store_true",
                        help="Production mode: use pre-built binary, skip React dev, run preflight + migration checks")
    parser.add_argument("--seed", action="store_true",
                        help="Seed database with initial data before starting API")
    parser.add_argument("--reset-reseed", action="store_true",
                        help="Reset and regenerate all data: fetch real market data from Yahoo, "
                             "resample to all timeframes, infer regimes, and generate sentiment. "
                             "Requires yfinance and psycopg2-binary installed.")
    parser.add_argument("--reseed-symbols", type=str, nargs="*",
                        help="Symbols to reseed (default: SPY,QQQ,IWM,DIA,AAPL,MSFT,NVDA,GLD,BTC-USD,ETH-USD and major FX pairs)")
    parser.add_argument("--reseed-start", type=str, default="2026-06-12",
                        help="Start date for reseed (YYYY-MM-DD, default: 60d ago for Yahoo 5m)")
    parser.add_argument("--reseed-end", type=str, default="2026-08-12",
                        help="End date for reseed (YYYY-MM-DD, default: 2026-08-12)")
    parser.add_argument("--no-react", action="store_true",
                        help="Skip React frontend dev server")
    parser.add_argument("--no-monitoring", action="store_true",
                        help="Skip Prometheus and Grafana")
    parser.add_argument("--force", action="store_true",
                        help="Force-kill stale processes on ports (use only for orphaned processes)")
    parser.add_argument("--skip-port-cleanup", action="store_true",
                        help="Skip port cleanup (use if services are already running)")
    parser.add_argument("--ports", type=int, nargs="*",
                        help="Override default ports to manage")
    parser.add_argument("--db-host", type=str, default="localhost",
                        help="PostgreSQL host (default: localhost)")
    parser.add_argument("--db-port", type=int, default=5432,
                        help="PostgreSQL port (default: 5432)")
    parser.add_argument("--db-user", type=str, default="postgres",
                        help="PostgreSQL user (default: postgres)")
    parser.add_argument("--db-password", type=str, default="",
                        help="PostgreSQL password (default: empty)")
    parser.add_argument("--db-name", type=str, default="orca_core",
                        help="PostgreSQL database name (default: orca_core)")
    parser.add_argument("--db-sslmode", type=str, default="disable",
                        help="PostgreSQL SSL mode (default: disable)")
    parser.add_argument("--health-timeout", type=int, default=90,
                        help="Max seconds to wait for health checks (default: 90)")
    parser.add_argument("--log-file", type=str,
                        help="Write structured log to file")
    parser.add_argument("--dry-run", action="store_true",
                        help="Show what would happen without executing")
    parser.add_argument("--monitor", action="store_true",
                        help="Keep running and health-check services (Ctrl+C to stop)")

    venv_group = parser.add_argument_group("Python Virtual Environment")
    venv_group.add_argument("--setup-venv", action="store_true",
                            help="Create and configure Python virtual environment before starting services")
    venv_group.add_argument("--venv-path", type=str, default=str(VENV_DIR),
                            help=f"Path to Python virtual environment (default: {VENV_DIR})")
    venv_group.add_argument("--python-path", type=str,
                            help="Path to Python interpreter for venv creation (auto-detected if not set)")
    venv_group.add_argument("--force-recreate-venv", action="store_true",
                            help="Delete and recreate the virtual environment even if it exists")
    venv_group.add_argument("--install-ml", action="store_true",
                            help="Install ML dependencies (xgboost, lightgbm, scikit-learn, etc.)")
    venv_group.add_argument("--skip-pip-install", action="store_true",
                            help="Skip pip install — assume dependencies are already installed in venv")
    venv_group.add_argument("--skip-venv", action="store_true",
                            help="Skip virtual environment setup entirely (use system Python)")
    return parser


def detect_local_postgresql() -> dict[str, Any]:
    """Auto-detect a local PostgreSQL installation and return connection parameters.

    Checks, in order:
      1. Environment variables (ORCA_DB_HOST, ORCA_DB_PORT, etc.)
      2. pgpass.conf / .pgpass file
      3. Well-known PostgreSQL service ports (5432, 5433)
      4. Windows service name detection

    Returns dict with keys: host, port, user, password, dbname, sslmode, source
    Returns empty dict if no PostgreSQL instance is detected.
    """
    result: dict[str, Any] = {}

    env_host = os.environ.get("ORCA_DB_HOST", "")
    env_port = os.environ.get("ORCA_DB_PORT", "")
    env_user = os.environ.get("ORCA_DB_USER", "")
    env_pass = os.environ.get("ORCA_DB_PASSWORD", "")
    env_db   = os.environ.get("ORCA_DB_NAME", "")
    env_ssl  = os.environ.get("ORCA_DB_SSLMODE", "disable")

    if env_host and env_port and env_user:
        result = {
            "host": env_host, "port": int(env_port), "user": env_user,
            "password": env_pass, "dbname": env_db or "orca_core",
            "sslmode": env_ssl, "source": "environment",
        }

    pgpass_entries: list[dict[str, str]] = []
    pgpass_paths = [
        Path(os.environ.get("APPDATA", "")) / "postgresql" / "pgpass.conf",
        Path.home() / ".pgpass",
    ]
    for pp in pgpass_paths:
        if pp.exists():
            try:
                with open(pp) as f:
                    for line in f:
                        line = line.strip()
                        if line and not line.startswith("#"):
                            parts = line.split(":")
                            if len(parts) >= 5:
                                pgpass_entries.append({
                                    "host": parts[0], "port": parts[1],
                                    "dbname": parts[2], "user": parts[3],
                                    "password": parts[4],
                                })
            except Exception:
                pass

    common_ports = [5432, 5433]
    for port in common_ports:
        if db_health(port):
            if not result:
                result = {"host": "localhost", "port": port, "sslmode": "disable", "source": f"port-scan:{port}"}
                for entry in pgpass_entries:
                    if entry["port"] == str(port) or entry["port"] == "*":
                        result["user"] = entry["user"]
                        result["password"] = entry["password"]
                        result["dbname"] = entry["dbname"] if entry["dbname"] != "*" else "orca_core"
                        result["source"] = f"pgpass+port:{port}"
                        break
            break

    if platform.system() == "Windows":
        try:
            output = subprocess.check_output(
                ["sc", "query", "state=", "all"], text=True, timeout=10
            )
            for line in output.splitlines():
                if "postgres" in line.lower():
                    parts = line.strip().split(":")
                    if len(parts) >= 2:
                        svc_name = parts[1].strip()
                        if not result or "port-scan" in result.get("source", ""):
                            result["windows_service"] = svc_name
            if not result and pgpass_entries:
                entry = pgpass_entries[0]
                result = {
                    "host": "localhost", "port": int(entry["port"]),
                    "user": entry["user"], "password": entry["password"],
                    "dbname": entry["dbname"] if entry["dbname"] != "*" else "orca_core",
                    "sslmode": "disable", "source": "pgpass",
                }
        except Exception:
            pass

    return result


def run_reseed(
    venv_python: str | None,
    symbols: list[str] | None,
    start: str,
    end: str,
    db_host: str,
    db_port: int,
    db_user: str,
    db_password: str,
    db_name: str,
    db_sslmode: str,
) -> bool:
    """Reset and regenerate all market data from Yahoo Finance.

    Pipeline: seed-all (reset) -> build-regime-logs -> build-candles.
    All commands run via subprocess in the project's venv or system Python.

    Returns True if all steps succeed.
    """
    python_cmd = venv_python or "python"
    from urllib.parse import quote
    db_url = f"postgresql://{db_user}:{quote(db_password, safe='')}@{db_host}:{db_port}/{db_name}?sslmode={db_sslmode}"

    env = os.environ.copy()
    env["ORCA_DB_URL"] = db_url

    sym_args: list[str] = []
    if symbols:
        flat: list[str] = []
        for s in symbols:
            flat.extend([x.strip() for x in s.replace(",", " ").split() if x.strip()])
        for s in flat:
            sym_args.extend(["--symbols", s])

    step = 0
    log("STEP", "Reseed", "=== Reset & Regenerate Data Pipeline ===")

    # Step 1: seed-all --reset
    step += 1
    log("STEP", f"Reseed/{step}", f"Fetching real market data from Yahoo Finance ({start} -> {end})...")
    log("INFO", f"Reseed/{step}", f"Symbols: {', '.join(symbols) if symbols else 'default (30 major instruments)'}")
    cmd = [python_cmd, "-c", "import orca.cli; orca.cli.app()", "seed-all",
           "--start", start, "--end", end, "--reset"] + sym_args
    result = subprocess.run(
        cmd, cwd=str(PROJECT_ROOT), env=env, capture_output=True, text=True, timeout=600
    )
    if result.returncode != 0:
        log("ERROR", f"Reseed/{step}", f"seed-all failed: {result.stderr.strip()[-500:]}")
        log("ERROR", f"Reseed/{step}", result.stdout.strip()[-500:] if result.stdout else "")
        return False
    log("OK", f"Reseed/{step}", "seed-all complete")
    log("INFO", f"Reseed/{step}", result.stdout.strip()[-200:] if result.stdout else "")

    # Step 2: build-regime-logs
    step += 1
    log("STEP", f"Reseed/{step}", "Inferring regimes from fresh candle data...")
    cmd = [python_cmd, "-c", "import orca.cli; orca.cli.app()", "build-regime-logs"] + sym_args
    result = subprocess.run(
        cmd, cwd=str(PROJECT_ROOT), env=env, capture_output=True, text=True, timeout=120
    )
    if result.returncode != 0:
        log("WARN", f"Reseed/{step}", f"regime inference had issues: {result.stderr.strip()[-200:]}")
    else:
        log("OK", f"Reseed/{step}", "Regime logs generated")

    # Step 3: build-candles (resample 5m -> higher timeframes)
    step += 1
    log("STEP", f"Reseed/{step}", "Resampling 5m candles to higher timeframes...")
    cmd = [python_cmd, "-c", "import orca.cli; orca.cli.app()", "build-candles",
           "--validate"] + sym_args
    result = subprocess.run(
        cmd, cwd=str(PROJECT_ROOT), env=env, capture_output=True, text=True, timeout=300
    )
    if result.returncode != 0:
        log("WARN", f"Reseed/{step}", f"candle resampling had issues: {result.stderr.strip()[-200:]}")
    else:
        log("OK", f"Reseed/{step}", "Candle resampling complete")
        log("INFO", f"Reseed/{step}", result.stdout.strip()[-300:] if result.stdout else "")

    log("OK", "Reseed", "Data regeneration pipeline complete")
    return True


def main() -> None:
    parser = build_cli()
    args = parser.parse_args()

    global _log_file
    if args.log_file:
        _log_file = Path(args.log_file)

    venv_python: str | None = None
    venv_path = Path(args.venv_path)
    if not args.skip_venv:
        venv_result = setup_python_environment(
            venv_path=venv_path,
            force_recreate=args.force_recreate_venv,
            install_dev=not args.production or args.setup_venv,
            install_ml=args.install_ml,
            python_exe=args.python_path,
            skip_install=args.skip_pip_install or (args.production and not args.setup_venv),
            dry_run=args.dry_run,
        )
        if venv_result:
            venv_python = str(venv_result)
    elif not args.dry_run:
        detected = get_python_exe()
        venv_python = detected
        if not detected:
            log("ERROR", "venv", f"--skip-venv specified but no Python {PYTHON_MIN_VERSION[0]}.{PYTHON_MIN_VERSION[1]}+ found on PATH")
            sys.exit(1)

    local_mode = args.local
    if local_mode:
        db_info = detect_local_postgresql()
        if not db_info:
            merged_env = load_dotenv(PROJECT_ROOT / ".env")
            db_info = {
                "host": merged_env.get("ORCA_DB_HOST", "localhost"),
                "port": int(merged_env.get("ORCA_DB_PORT", "5432")),
                "user": merged_env.get("ORCA_DB_USER", "postgres"),
                "password": merged_env.get("ORCA_DB_PASSWORD", ""),
                "dbname": merged_env.get("ORCA_DB_NAME", "orca_core"),
                "sslmode": merged_env.get("ORCA_DB_SSLMODE", "disable"),
                "source": ".env",
            }
        db_host = args.db_host if args.db_host != "localhost" else str(db_info.get("host", "localhost"))
        db_port = args.db_port if args.db_port != 5432 else int(db_info.get("port", 5432))
        db_user = args.db_user if args.db_user != "postgres" else str(db_info.get("user", "postgres"))
        db_password = args.db_password or str(db_info.get("password", ""))
        db_name = args.db_name if args.db_name != "orca_core" else str(db_info.get("dbname", "orca_core"))
        db_sslmode = args.db_sslmode if args.db_sslmode != "disable" else str(db_info.get("sslmode", "disable"))
    else:
        db_host = "localhost"
        db_port = 5433
        db_user = "orca"
        db_password = ""
        db_name = "orca_core"
        db_sslmode = "disable"

    active_services = _build_services(
        local_mode=local_mode,
        db_host=db_host, db_port=db_port,
        db_user=db_user, db_password=db_password,
        db_name=db_name, db_sslmode=db_sslmode,
        skip_metrics=args.no_monitoring,
        production=args.production,
    )
    if args.ports:
        active_services = [s for s in SERVICES if s.port in args.ports]

    wanted_ports = [s.port for s in active_services]

    print()
    print(f"{COLORS['cyan']}{COLORS['bold']}{'='*56}{COLORS['reset']}")
    print(f"{COLORS['cyan']}{COLORS['bold']}  OrcaAlgo — Startup Orchestrator{COLORS['reset']}")
    print(f"{COLORS['cyan']}{COLORS['bold']}{'='*56}{COLORS['reset']}")
    print()
    log("INFO", "Orchestrator", f"Mode: {'Production' if args.production else 'Local DB' if local_mode else 'Docker'}")
    log("INFO", "Orchestrator", f"Platform: {platform.system()} {platform.release()}")
    log("INFO", "Orchestrator", f"Project: {PROJECT_ROOT}")
    if args.production:
        log("INFO", "Orchestrator", "Binary: bin/orca-server.exe")
    if local_mode:
        log("INFO", "Orchestrator", f"Database: {db_user}@{db_host}:{db_port}/{db_name} (source: {db_info.get('source', 'cli')})")
    print()

    if args.reset_reseed:
        print()
        print(f"{COLORS['magenta']}{COLORS['bold']}{'='*56}{COLORS['reset']}")
        print(f"{COLORS['magenta']}{COLORS['bold']}  DATA REGENERATION — Reset & Reseed{COLORS['reset']}")
        print(f"{COLORS['magenta']}{COLORS['bold']}{'='*56}{COLORS['reset']}")
        print()
        if args.dry_run:
            log("OK", "Reseed", "DRY-RUN: Would reset and regenerate all market data")
        else:
            if not run_reseed(
                venv_python, args.reseed_symbols, args.reseed_start, args.reseed_end,
                db_host, db_port, db_user, db_password, db_name, db_sslmode,
            ):
                log("ERROR", "Orchestrator", "Data regeneration failed. Aborting.")
                sys.exit(1)
        print()


    skip_names: set[str] = set()
    if args.no_react or args.production:
        skip_names.add("React Dev")
    if args.no_monitoring:
        skip_names.update(["Prometheus", "Grafana", "Metrics"])

    docker_services: list[str] = []
    if not local_mode:
        for svc in active_services:
            if svc.docker_service and svc.name not in skip_names:
                docker_services.append(svc.docker_service)
    else:
        for svc in active_services:
            if svc.docker_service and svc.name in ("PostgreSQL", "Prometheus", "Grafana"):
                if svc.name not in skip_names and not svc.optional:
                    docker_services.append(svc.docker_service)

    step = 0
    total_steps = 4 + (1 if not args.skip_port_cleanup else 0)

    optional_ports = {s.port for s in active_services if s.optional}
    optional_ports.update({9091, 3000})

    if not args.skip_port_cleanup:
        step += 1
        print()
        print(f"{COLORS['bold']}[{step}/{total_steps}] Port Cleanup{COLORS['reset']}")
        print("-" * 40)
        if not resolve_port_conflicts(wanted_ports, args.force, args.dry_run, docker_services, optional_ports, local_mode=local_mode):
            log("ERROR", "Orchestrator",
                "Cannot proceed — required ports are occupied. Review guidance above or use --force for stale processes.")
            sys.exit(1)

    step += 1
    print()
    print(f"{COLORS['bold']}[{step}/{total_steps}] Dependency Services{COLORS['reset']}")
    print("-" * 40)
    if docker_services:
        if not start_docker_services(docker_services):
            log("ERROR", "Orchestrator", "Docker services failed to start. Aborting.")
            sys.exit(1)
    else:
        log("INFO", "Orchestrator", "No Docker services to start (all native)")

    enabled_names = {s.name for s in active_services if s.name not in skip_names}
    ordered = resolve_service_order(active_services, enabled_names, skip_names)
    native_services = [s for s in ordered if s.startup_cmd and not s.docker_service]

    step += 1
    print()
    print(f"{COLORS['bold']}[{step}/{total_steps}] Health Verification{COLORS['reset']}")
    print("-" * 40)
    for svc in ordered:
        if svc.name in skip_names:
            continue
        if svc.docker_service not in docker_services:
            continue
        if not wait_for_service(svc, args.dry_run):
            if not svc.optional:
                log("ERROR", "Orchestrator",
                    f"Required service {svc.name} is not healthy. Aborting.")
                shutdown(native_services, docker_services)
                sys.exit(1)
            log("WARN", svc.name, "Optional service not healthy — continuing")

    env_overrides: dict[str, str] = {}
    dotenv = load_dotenv(PROJECT_ROOT / ".env")
    env_overrides.update(dotenv)
    jwt = os.environ.get("ORCA_JWT_SECRET", env_overrides.get("ORCA_JWT_SECRET", ""))
    admin_pw = os.environ.get("ORCA_ADMIN_PASSWORD", env_overrides.get("ORCA_ADMIN_PASSWORD", ""))
    if not jwt:
        jwt = "orchestrator-auto-generated-jwt-secret"
    if not admin_pw:
        admin_pw = "orchestrator-auto-generated-admin-password"
    env_overrides["ORCA_JWT_SECRET"] = jwt
    env_overrides["ORCA_ADMIN_PASSWORD"] = admin_pw
    env_overrides["PAPER_TRADING"] = os.environ.get("PAPER_TRADING", "true")
    if jwt == "orchestrator-auto-generated-jwt-secret":
        log("WARN", "Orchestrator",
            "ORCA_JWT_SECRET not set — using auto-generated value for dev only")

    if args.production:
        log("STEP", "Production", "Running pre-flight checks...")
        python_cmd = venv_python or "python"
        result = subprocess.run(
            [python_cmd, "-m", "orca.cli", "preflight"],
            cwd=str(PROJECT_ROOT), capture_output=True, text=True, timeout=30
        )
        if result.returncode != 0:
            log("ERROR", "Production", "Preflight checks failed — aborting")
            log("ERROR", "Production", result.stdout.strip()[-500:] if result.stdout else result.stderr.strip())
            sys.exit(1)
        log("OK", "Production", "Preflight checks passed")

    if args.production or args.seed:
        log("STEP", "Database", "Verifying migrations...")
        result = subprocess.run(
            ["docker", "exec", "orca_algo-postgres-1", "psql", "-U", db_user, "-d", db_name,
             "-c", "SELECT COUNT(*) FROM information_schema.tables WHERE table_name='strategies'", "-t"],
            capture_output=True, text=True, timeout=15
        )
        count = result.stdout.strip()
        if not count or count == "0":
            log("WARN", "Database", "Migrations not applied. Running migrate script...")
            migrate_result = subprocess.run(
                ["powershell", "-File", str(PROJECT_ROOT / "scripts" / "migrate.ps1")],
                cwd=str(PROJECT_ROOT), capture_output=True, text=True, timeout=60
            )
            if migrate_result.returncode != 0:
                log("ERROR", "Database", f"Migration failed: {migrate_result.stderr.strip()}")
            else:
                log("OK", "Database", "Migrations applied")
        else:
            log("OK", "Database", f"Migrations verified ({count.strip()} tables)")

    if args.seed:
        log("STEP", "Database", "Seeding initial data...")
        seed_result = subprocess.run(
            ["docker", "exec", "orca_algo-postgres-1", "psql", "-U", db_user, "-d", db_name,
             "-c", "INSERT INTO providers (id, name, type, driver, is_active) VALUES ('paper','Paper Trading','broker','paper',true) ON CONFLICT DO NOTHING"],
            capture_output=True, text=True, timeout=15
        )
        log("OK", "Database", "Seed data applied" if seed_result.returncode == 0 else f"Seed warning: {seed_result.stderr.strip()}")

    if args.dry_run:
        log("INFO", "Orchestrator", "Dry-run complete — no services were started.")
        return

    def handle_shutdown(signum: int, frame: Any) -> None:
        shutdown(native_services, docker_services)
        sys.exit(0)

    signal.signal(signal.SIGINT, handle_shutdown)
    signal.signal(signal.SIGTERM, handle_shutdown)

    step += 1
    print()
    print(f"{COLORS['bold']}[{step}/{total_steps}] Application Services{COLORS['reset']}")
    print("-" * 40)

    # Ensure binary is built (avoids go run compile-time eating into health checks)
    bin_path = str(PROJECT_ROOT / "bin" / "orca-server.exe")
    if not Path(bin_path).exists() or not args.production:
        log("STEP", "Build", "Compiling orca-server binary...")
        build_result = subprocess.run(
            ["go", "build", "-o", bin_path, "./cmd/orca-server"],
            cwd=str(PROJECT_ROOT), capture_output=True, text=True, timeout=120
        )
        if build_result.returncode != 0:
            log("ERROR", "Build", f"Failed: {build_result.stderr.strip()[-300:]}")
            sys.exit(1)
        log("OK", "Build", "orca-server compiled")
    # Use binary instead of go run for reliable startup
    for svc in native_services:
        if svc.name in ("Go API", "Orca Server (prod)"):
            svc.startup_cmd = [bin_path]
    for svc in native_services:
        if svc.name in skip_names:
            continue
        if not start_native_service(svc, env_overrides, args.dry_run):
            if not svc.optional:
                log("ERROR", "Orchestrator",
                    f"Failed to start {svc.name}. Aborting.")
                shutdown(native_services, docker_services)
                sys.exit(1)
            log("WARN", svc.name, "Optional service failed to start — continuing")

    for svc in native_services:
        if svc.name in skip_names:
            continue
        if svc.health_url and not wait_for_service(svc, args.dry_run):
            if not svc.optional:
                log("ERROR", "Orchestrator",
                    f"{svc.name} did not become healthy. Aborting.")
                shutdown(native_services, docker_services)
                sys.exit(1)
            log("WARN", svc.name, "Optional service not healthy — continuing")

    print()
    print(f"{COLORS['green']}{COLORS['bold']}{'='*56}{COLORS['reset']}")
    print(f"{COLORS['green']}{COLORS['bold']}  All Services Ready{COLORS['reset']}")
    print(f"{COLORS['green']}{COLORS['bold']}{'='*56}{COLORS['reset']}")
    for svc in active_services:
        if svc.name not in skip_names:
            status = "healthy" if not svc.optional else "running"
            print(f"  {COLORS['cyan']}{svc.label:20}{COLORS['reset']} {status}")
    print()
    print(f"  API:        {COLORS['cyan']}http://localhost:8080{COLORS['reset']}")
    if not args.no_monitoring and "Metrics" not in skip_names:
        print(f"  Metrics:    {COLORS['cyan']}http://localhost:9091/metrics{COLORS['reset']}")
    if "React Dev" not in skip_names:
        print(f"  Dashboard:  {COLORS['cyan']}http://localhost:5173{COLORS['reset']}")
    if not args.no_monitoring:
        print(f"  Prometheus: {COLORS['cyan']}http://localhost:9091{COLORS['reset']}")
        print(f"  Grafana:    {COLORS['cyan']}http://localhost:3000{COLORS['reset']}")
    print()
    if args.monitor:
        print(f"  {COLORS['gray']}Monitoring mode — Press Ctrl+C to stop all services.{COLORS['reset']}")
    else:
        print(f"  {COLORS['gray']}All services running in background. Use 'docker compose stop' to shut down.{COLORS['reset']}")
    print()

    if not args.monitor:
        return

    all_pre_existing = all(svc.process is None for svc in native_services if svc.name not in skip_names)
    if all_pre_existing:
        log("INFO", "Orchestrator", "All services were already running — monitoring mode (Ctrl+C to stop)")
        try:
            while True:
                time.sleep(5)
                for svc in native_services:
                    if svc.name in skip_names:
                        continue
                    if svc.health_url and not check_health("127.0.0.1", svc.port, svc.health_url, svc.health_method, svc.health_expected_status):
                        log("WARN", svc.name, f"Health check failed on :{svc.port}")
        except KeyboardInterrupt:
            log("INFO", "Orchestrator", "Shutdown requested")
        return

    for svc in native_services:
        if svc.process:
            svc.process.wait()

    shutdown(native_services, docker_services)


if __name__ == "__main__":
    main()
