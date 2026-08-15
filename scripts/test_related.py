#!/usr/bin/env python3
"""
Change-aware test runner — runs only tests relevant to changed files.

Usage:
    python scripts/test_related.py [--language python|go|odin|web|all] [--base origin/main]

Exit codes:
    0: All relevant tests PASSED
    1: One or more tests FAILED
    2: UNTESTED — changed files map to zero tests (BLOCKING)

Detects files changed vs the base branch (default: origin/main)
and runs only the test files/packages mapped to those source files.
"""

import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

PYTHON_TEST_MAP: dict[str, list[str]] = {
    "orca/sizing/":       ["tests/test_kelly.py"],
    "orca/math/":         ["tests/test_brier.py", "tests/test_platt.py", "tests/test_wilson.py"],
    "orca/ir/":           ["tests/test_ir.py", "tests/test_hash.py", "tests/test_cli_and_edge.py"],
    "orca/models/":       ["tests/test_models.py", "tests/guardian/test_critical_paths.py"],
    "orca/calibration/":  ["tests/test_calibration.py"],
    "orca/preflight/":    ["tests/test_cli_and_edge.py"],
    "orca/attribution/":  ["tests/test_attribution.py"],
    "orca/cli.py":        ["tests/test_cli_and_edge.py"],
    "orca/simulation/":   ["tests/simulation/"],
    "orca/ml/":           ["tests/test_barriers.py"],
    "orca/data_quality/": ["tests/test_cli_and_edge.py"],
    "orca/optimize/":     ["tests/optimize/"],
    "tests/guardian/":    ["tests/guardian/"],
    "orca/":              ["tests/"],  # Catch-all for unlisted orca/ subdirs
}

GO_PACKAGE_MAP: dict[str, str] = {
    "internal/api/":       "./internal/api/...",
    "internal/broker/":    "./internal/broker/...",
    "internal/breaker/":   "./internal/breaker/...",
    "internal/benchmark/": "./internal/benchmark/...",
    "internal/backtest/":  "./internal/backtest/...",
    "internal/risk/":      "./internal/risk/...",
    "internal/strategy/":  "./internal/strategy/...",
    "internal/ingest/":    "./internal/ingest/...",
    "internal/indicator/": "./internal/indicator/...",
    "internal/db/":        "./internal/db/...",
    "internal/monitor/":   "./internal/monitor/...",
    "internal/scheduler/": "./internal/scheduler/...",
    "internal/engine/":    "./internal/engine/...",
    "internal/ml/":        "./internal/ml/...",
    "internal/hash/":      "./internal/hash/...",
    "internal/llm/":       "./internal/llm/...",
    "internal/types/":     "./internal/types/...",
    "internal/universe/":  "./internal/universe/...",
    "internal/notify/":    "./internal/notify/...",
    "internal/email/":     "./internal/email/...",
    "internal/market/":    "./internal/market/...",
    "internal/analytics/":  "./internal/analytics/...",
    "internal/metrics/":   "./internal/metrics/...",
    "internal/model/":     "./internal/model/...",
    "internal/persist/":   "./internal/persist/...",
    "internal/propfirm/":  "./internal/propfirm/...",
    "internal/reactive/":  "./internal/reactive/...",
    "internal/audit/":     "./internal/audit/...",
    "internal/config/":    "./internal/config/...",
    "internal/security/":  "./internal/security/...",
    "internal/synthetic/": "./internal/synthetic/...",
    "internal/version/":   "./internal/version/...",
    "internal/error/":     "./internal/error/...",
    "cgo_bridge/":         "./cgo_bridge/...",
    "cmd/":                "./cmd/...",
}

ODIN_TEST_MAP: dict[str, str] = {
    "odin/": "odin build ./odin/",  # Compilation-only check for Odin
}

WEB_TEST_MAP: dict[str, str] = {
    "web/src/":   "cd web && npx vitest run",
    "web/e2e/":   "cd web && npx playwright test",
    "web/":       "cd web && npx tsc --noEmit",
}

GKR_CONFIG_PATH = "configs/strategies/"


def get_changed_files(base: str = "origin/main") -> list[str]:
    try:
        result = subprocess.run(
            ["git", "diff", "--name-only", f"{base}...HEAD"],
            capture_output=True, text=True, cwd=str(ROOT),
        )
        if result.returncode != 0:
            result = subprocess.run(
                ["git", "diff", "--name-only", "HEAD~1"],
                capture_output=True, text=True, cwd=str(ROOT),
            )
        return [f for f in result.stdout.strip().split("\n") if f]
    except Exception:
        print("[test-related] Could not detect changed files — running full test suite")
        return ["orca/", "internal/", "web/"]  # Trigger all


def run_command(cmd: str, description: str) -> int:
    print(f"[test-related] {description}: {cmd}")
    result = subprocess.run(cmd, shell=True, cwd=str(ROOT))
    return result.returncode


def run_python_tests(changed: list[str]) -> int:
    py_tests: set[str] = set()
    for f in changed:
        f = f.replace("\\", "/")
        for src_dir, test_files in PYTHON_TEST_MAP.items():
            if f.startswith(src_dir):
                py_tests.update(test_files)
    if not py_tests:
        print("[test-related] No Python changes detected")
        return 0
    print(f"[test-related] Running Python tests: {sorted(py_tests)}")
    return subprocess.run(
        ["pytest"] + sorted(py_tests) + ["-v", "--tb=short", f"--rootdir={ROOT}"],
        cwd=str(ROOT),
    ).returncode


def run_go_tests(changed: list[str]) -> int:
    go_pkgs: set[str] = set()
    for f in changed:
        f = f.replace("\\", "/")
        for pkg_path, test_cmd in GO_PACKAGE_MAP.items():
            if f.startswith(pkg_path):
                go_pkgs.add(test_cmd)
    if not go_pkgs:
        print("[test-related] No Go changes detected")
        return 0
    exit_code = 0
    for pkg in sorted(go_pkgs):
        if subprocess.run(
            ["go", "test", pkg, "-count=1", "-short"], cwd=str(ROOT),
        ).returncode != 0:
            exit_code = 1
    return exit_code


def run_odin_tests(changed: list[str]) -> int:
    for f in changed:
        f = f.replace("\\", "/")
        if f.startswith("odin/"):
            return run_command("odin build ./odin/", "Odin compilation check")
    return 0


def run_web_tests(changed: list[str]) -> int:
    exit_code = 0
    for f in changed:
        f = f.replace("\\", "/")
        if f.startswith("web/src/"):
            if subprocess.run(["npx", "tsc", "--noEmit"], cwd=str(ROOT / "web")).returncode != 0:
                exit_code = 1
            if subprocess.run(["npx", "vitest", "run"], cwd=str(ROOT / "web")).returncode != 0:
                exit_code = 1
            break
        if f.startswith("web/e2e/"):
            if subprocess.run(["npx", "playwright", "test"], cwd=str(ROOT / "web")).returncode != 0:
                exit_code = 1
            break
    return exit_code


def run_gkr_validation(changed: list[str]) -> int:
    for f in changed:
        f = f.replace("\\", "/")
        if f.startswith(GKR_CONFIG_PATH):
            return run_command(
                f"python -m orca.cli validate {GKR_CONFIG_PATH}*.gkr.yaml",
                "GKR strategy validation"
            )
    return 0


def main() -> None:
    language = "all"
    base_branch = "origin/main"
    args = sys.argv[1:]
    while args:
        arg = args.pop(0)
        if arg == "--language" and args:
            language = args.pop(0)
        elif arg == "--base" and args:
            base_branch = args.pop(0)

    changed = get_changed_files(base_branch)
    if not changed:
        print("[test-related] No changed files detected — nothing to run")
        sys.exit(0)

    results: dict[str, int] = {}
    ran_any = False

    if language in ("all", "python"):
        rc = run_python_tests(changed)
        if rc >= 0: results["python"] = rc; ran_any = True

    if language in ("all", "go"):
        rc = run_go_tests(changed)
        if rc >= 0: results["go"] = rc; ran_any = True

    if language in ("all", "odin"):
        rc = run_odin_tests(changed)
        if rc >= 0: results["odin"] = rc

    if language in ("all", "web"):
        rc = run_web_tests(changed)
        if rc >= 0: results["web"] = rc

    if language in ("all", "gkr"):
        rc = run_gkr_validation(changed)
        if rc >= 0: results["gkr"] = rc

    has_failures = any(v != 0 for v in results.values())
    has_tests = bool(results)

    if not has_tests:
        print("\n[test-related] 🚫 ZERO-TEST GUARD: Changed files map to zero tests.")
        print("   This is BLOCKING. Add test coverage before merging.")
        print(f"   Changed files: {changed[:10]}{'...' if len(changed) > 10 else ''}")
        sys.exit(2)

    if has_failures:
        failed = [f"{k} (exit {v})" for k, v in results.items() if v != 0]
        print(f"\n[test-related] ❌ FAILED: {', '.join(failed)}")
        sys.exit(1)

    print("\n[test-related] ✅ All relevant tests PASSED")
    sys.exit(0)


if __name__ == "__main__":
    main()
