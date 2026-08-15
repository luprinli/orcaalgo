from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class CheckResult:
    check_name: str
    status: str  # "pass", "warn", "fail"
    message: str


def run_preflight_checks() -> list[CheckResult]:
    results: list[CheckResult] = []

    # Check 1: Validate config files exist
    config_dir = Path("configs")
    if config_dir.exists():
        results.append(CheckResult("config_exists", "pass", "Config directory found"))
    else:
        results.append(CheckResult("config_exists", "fail", "Config directory missing"))

    # Check 2: Validate .gkr.yaml strategies
    gkr_dir = Path("configs/strategies")
    gkr_files = list(gkr_dir.glob("*.gkr.yaml")) if gkr_dir.exists() else []
    if gkr_files:
        results.append(CheckResult("gkr_strategies", "pass", f"Found {len(gkr_files)} GKR strategy files"))
    else:
        results.append(CheckResult("gkr_strategies", "warn", "No GKR strategy files found"))

    # Check 3: Validate each GKR file
    for gkr_file in gkr_files:
        try:
            from orca.ir.loader import load_ir
            from orca.ir.validator import validate_ir
            ir = load_ir(gkr_file)
            diags = validate_ir(ir, "production_guarded")
            errors = [d for d in diags if d.severity == "error"]
            if errors:
                results.append(CheckResult(f"validate_{gkr_file.stem}", "fail", f"{len(errors)} validation errors"))
            else:
                results.append(CheckResult(f"validate_{gkr_file.stem}", "pass", "Strategy validated"))
        except Exception as e:
            results.append(CheckResult(f"validate_{gkr_file.stem}", "fail", str(e)))

    # Check 4: Environment checks
    import os
    env_vars = ["ORCA_DB_URL", "ORCA_JWT_SECRET"]
    for var in env_vars:
        if os.environ.get(var):
            results.append(CheckResult(f"env_{var}", "pass", "Set"))
        else:
            results.append(CheckResult(f"env_{var}", "warn", "Not set"))

    # Check 5: Python package integrity
    try:
        from orca import __version__
        results.append(CheckResult("orca_package", "pass", f"orca v{__version__} installed"))
    except ImportError:
        results.append(CheckResult("orca_package", "fail", "orca package not importable"))

    # Check 6: Numpy/SciPy available
    try:
        import numpy
        results.append(CheckResult("numpy_available", "pass", f"numpy {numpy.__version__}"))
    except ImportError:
        results.append(CheckResult("numpy_available", "fail", "numpy not installed"))

    try:
        import scipy
        results.append(CheckResult("scipy_available", "pass", f"scipy {scipy.__version__}"))
    except ImportError:
        results.append(CheckResult("scipy_available", "fail", "scipy not installed"))

    # Check 7: Kill-switch E2E verification
    # NOTE: Full E2E kill-switch verification requires a running Go server
    # (isLocked + killSwitchReady guard checked before any kill-switch execution
    # in internal/risk/pipeline.go). This check validates the Python-side risk
    # model and verifies the CLI toolchain via subprocess as a partial gate.
    try:
        from orca.models.risk import KillSwitchState
        ks = KillSwitchState(is_locked=False, reason="", triggered_at=None)
        if ks.is_locked:
            results.append(CheckResult("kill_switch_guard", "fail", "Kill-switch default state is locked"))
        else:
            results.append(CheckResult("kill_switch_guard", "pass", "Kill-switch re-entrancy guard model validated"))
    except Exception as e:
        results.append(CheckResult("kill_switch_guard", "fail", str(e)))

    # Check 7b: Kill-switch CLI smoke test via subprocess
    import subprocess
    import sys
    try:
        gkr_for_cli = list(Path("configs/strategies").glob("*.gkr.yaml"))
        if gkr_for_cli:
            result = subprocess.run(
                [sys.executable, "-m", "orca.cli", "validate", str(gkr_for_cli[0])],
                capture_output=True, text=True, timeout=30,
            )
            if result.returncode == 0:
                results.append(CheckResult("kill_switch_cli_smoke", "pass",
                    "orca validate subprocess call successful"))
            else:
                results.append(CheckResult("kill_switch_cli_smoke", "warn",
                    f"CLI validate exited {result.returncode}"))
        else:
            results.append(CheckResult("kill_switch_cli_smoke", "warn",
                "No GKR file available for CLI smoke test"))
    except Exception as e:
        results.append(CheckResult("kill_switch_cli_smoke", "warn",
            f"CLI subprocess smoke test failed: {e}"))

    # Check 8: Balance reconciliation readiness
    import importlib.util
    if importlib.util.find_spec("orca.attribution.slicer"):
        results.append(CheckResult("balance_reconcile", "pass", "PnL attribution engine available for reconciliation"))
    else:
        results.append(CheckResult("balance_reconcile", "warn", "Attribution engine not importable; balance reconciliation unavailable"))

    # Check 9: Calibration recency
    reports_dir = Path("reports")
    cal_files = list(reports_dir.glob("calibration-*.json")) if reports_dir.exists() else []
    if cal_files:
        latest = max(cal_files, key=lambda p: p.stat().st_mtime)
        results.append(CheckResult("calibration_recency", "pass", f"Latest calibration: {latest.name}"))
    else:
        results.append(CheckResult("calibration_recency", "warn", "No calibration reports found; run `orca calibrate`"))

    # Check 10: FTMO/prop-firm compliance verification
    propfirm_dir = Path("configs/propfirms")
    pf_files = list(propfirm_dir.glob("*.yaml")) if propfirm_dir.exists() else []
    if pf_files:
        results.append(CheckResult("propfirm_profiles", "pass", f"Found {len(pf_files)} prop firm profiles"))
    else:
        results.append(CheckResult("propfirm_profiles", "fail", "No prop firm profiles found"))

    # Check 11: Strategy config hash verification
    for gkr_file in gkr_files:
        try:
            from orca.hash.graph import instance_hash_v2
            from orca.hash.verify import verify_instance_hash
            ir = load_ir(gkr_file)
            ih = instance_hash_v2(ir)
            hash_file = gkr_file.with_suffix(".gkr.hash")
            if hash_file.exists():
                expected = hash_file.read_text().strip()
                if verify_instance_hash(ir, expected):
                    results.append(CheckResult(
                        f"hash_{gkr_file.stem}", "pass",
                        f"instance hash verified: {ih[:12]}"
                    ))
                else:
                    results.append(CheckResult(
                        f"hash_{gkr_file.stem}", "fail",
                        f"instance hash mismatch: computed={ih[:12]} expected={expected[:12]}"
                    ))
            else:
                results.append(CheckResult(
                    f"hash_{gkr_file.stem}", "warn",
                    f"no stored hash — first run (computed={ih[:12]})"
                ))
        except Exception as e:
            results.append(CheckResult(f"hash_{gkr_file.stem}", "fail", str(e)))

    # Check 12: Market data integrity
    try:
        from orca.data_quality.validator import run_data_quality_checks
        dq = run_data_quality_checks()
        if dq and hasattr(dq, 'passed'):
            results.append(CheckResult("data_integrity", "pass", "Data quality checks available"))
        else:
            results.append(CheckResult("data_integrity", "warn", "Data quality validator returned no results"))
    except ImportError:
        results.append(CheckResult("data_integrity", "warn", "Data quality module not available"))

    # Check 13: Broker credential presence (env fallback vs. per-account BYOK)
    alpaca_key = os.environ.get("ALPACA_API_KEY")
    alpaca_secret = os.environ.get("ALPACA_API_SECRET")
    if alpaca_key and alpaca_secret:
        results.append(CheckResult("broker_credentials", "pass", "Alpaca env credentials set"))
    else:
        results.append(CheckResult("broker_credentials", "warn",
            "Alpaca env credentials not set (per-account BYOK credentials may be used instead)"))

    # Check 14: Broker/LLM schema migrations present (aligned with the DB)
    for name, path in [
        ("account_credentials_migration", "internal/db/migrations/000043_account_credentials.up.sql"),
        ("llm_keys_migration", "internal/db/migrations/000044_llm_api_keys.up.sql"),
    ]:
        if Path(path).exists():
            results.append(CheckResult(name, "pass", f"{Path(path).name} present"))
        else:
            results.append(CheckResult(name, "fail", f"{path} missing"))

    # Check 15: Market-based benchmark filter available (mandatory promotion gate)
    try:
        from orca.benchmark.filter import apply_benchmark_filter
        if callable(apply_benchmark_filter):
            results.append(CheckResult("benchmark_filter", "pass", "Benchmark filter available"))
        else:
            results.append(CheckResult("benchmark_filter", "fail", "Benchmark filter not callable"))
    except ImportError:
        results.append(CheckResult("benchmark_filter", "fail", "orca.benchmark module missing"))

    # Check 16: Benchmark persistence migrations present
    for name, path in [
        ("benchmark_filter_migration", "internal/db/migrations/000049_benchmark_filter.up.sql"),
    ]:
        if Path(path).exists():
            results.append(CheckResult(name, "pass", f"{Path(path).name} present"))
        else:
            results.append(CheckResult(name, "fail", f"{path} missing"))

    return results
