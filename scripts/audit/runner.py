"""Audit runner: orchestrates checks, aggregates results, emits report.

Short-lived by design — it connects to an already-running server, runs a set of
bounded checks, writes a JSON report, prints a console summary, and exits. It
never starts servers or browsers and never leaves a background process behind.
"""

from __future__ import annotations

import json
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Callable

from . import checks, config
from .checks import CheckResult, Context
from .client import APIClient, PrerequisiteError


def _now() -> str:
    return datetime.now(timezone.utc).strftime("%H:%M:%S")


def _log(level: str, msg: str) -> None:
    print(f"{_now()} {level:5} {msg}", flush=True)


def run_check(fn: Callable[[Context], CheckResult], ctx: Context) -> CheckResult:
    """Execute one check with timing and total error isolation."""
    start = time.time()
    try:
        res = fn(ctx)
    except PrerequisiteError:
        raise
    except Exception as e:  # noqa: BLE001
        res = CheckResult(
            name=getattr(fn, "__name__", "unknown").replace("check_", ""),
            status="ERROR", severity="high", guards="-",
            detail=f"unhandled exception: {e!r}",
        )
    res.duration_s = round(time.time() - start, 2)
    return res


def build_context(client: APIClient, timeframe: str) -> Context:
    symbols = client.synthetic_symbols(limit=3)
    if not symbols:
        symbols = list(config.FALLBACK_SYMBOLS)
    return Context(client=client, symbols=symbols, timeframe=timeframe)


def run_suite(*, quick: bool, timeframe: str,
              fail_fast: bool, resume: bool = False,
              checkpoint: str = "reports/.matrix_audit_checkpoint.json",
              ) -> tuple[list[CheckResult], Context]:
    client = APIClient()
    client.login()  # raises PrerequisiteError on failure
    ctx = build_context(client, timeframe)
    _log("INFO", f"Using synthetic symbols: {ctx.symbols} timeframe={timeframe}")

    # Resume: load the set of checks already recorded PASS so we can skip them.
    done: set[str] = set()
    cp_path = Path(checkpoint)
    if resume and cp_path.exists():
        try:
            done = set(json.loads(cp_path.read_text()).get("passed", []))
            _log("INFO", f"resume: skipping {len(done)} previously-passed checks")
        except Exception:  # noqa: BLE001
            done = set()

    selected = checks.QUICK_CHECKS if quick else checks.ALL_CHECKS
    results: list[CheckResult] = []
    for fn in selected:
        cname = getattr(fn, "__name__", "check").replace("check_", "")
        if resume and cname in done:
            results.append(CheckResult(cname, "SKIP", "medium", "-", "skipped (resume: already passed)"))
            _log("SKIP", f"[SKIP] {cname} (resume)")
            continue
        _log("STEP", f"running {cname} ...")
        res = run_check(fn, ctx)
        results.append(res)
        icon = {"PASS": "OK ", "FAIL": "FAIL", "ERROR": "ERR ", "SKIP": "SKIP"}.get(res.status, "?")
        _log(icon, f"[{res.status}] {res.name} ({res.duration_s}s) — {res.detail}")
        if fail_fast and res.status in ("FAIL", "ERROR"):
            _log("INFO", "fail-fast enabled; stopping after first failure")
            break

    # Persist the checkpoint of passed checks for a future --resume.
    try:
        passed = sorted({r.name for r in results if r.status == "PASS"} | done)
        cp_path.parent.mkdir(parents=True, exist_ok=True)
        cp_path.write_text(json.dumps({"passed": passed}, indent=2))
    except Exception:  # noqa: BLE001
        pass

    return results, ctx


def summarize(results: list[CheckResult]) -> dict:
    counts = {"PASS": 0, "FAIL": 0, "ERROR": 0, "SKIP": 0}
    for r in results:
        counts[r.status] = counts.get(r.status, 0) + 1
    critical_failed = any(
        r.status in ("FAIL", "ERROR") and r.severity == "critical" for r in results
    )
    verdict = "PASS"
    if counts["FAIL"] or counts["ERROR"]:
        verdict = "FAIL"
    return {"counts": counts, "critical_failed": critical_failed, "verdict": verdict}


def write_report(results: list[CheckResult], ctx: Context, summary: dict,
                 report_path: Path) -> None:
    report_path.parent.mkdir(parents=True, exist_ok=True)
    payload = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "api_base": config.API_BASE,
        "symbols": ctx.symbols,
        "verdict": summary["verdict"],
        "counts": summary["counts"],
        "critical_failed": summary["critical_failed"],
        "checks": [
            {
                "name": r.name, "status": r.status, "severity": r.severity,
                "guards": r.guards, "detail": r.detail, "duration_s": r.duration_s,
            }
            for r in results
        ],
    }
    report_path.write_text(json.dumps(payload, indent=2))


def print_summary(results: list[CheckResult], summary: dict, report_path: Path) -> None:
    print()
    print("=" * 74)
    print("  ORCAALGO MATRIX REGRESSION AUDIT — SUMMARY")
    print("=" * 74)
    for r in results:
        print(f"  [{r.status:4}] {r.name:30} {r.guards:8} {r.duration_s:>5.1f}s  {r.detail}")
    c = summary["counts"]
    print("-" * 74)
    print(f"  PASS={c['PASS']} FAIL={c['FAIL']} ERROR={c['ERROR']} SKIP={c['SKIP']}")
    print(f"  Report: {report_path}")
    print("=" * 74)
    print(f"  VERDICT: {summary['verdict']}")
    print("=" * 74)
    print()
