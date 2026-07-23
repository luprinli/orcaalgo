"""CLI entry point for the OrcaAlgo matrix regression audit.

This is a short-lived command. It connects to an already-running server, runs a
suite of fast regression checks (each guarding a specific fix from the
2026-07-08 session), writes a JSON report, prints a summary, and exits with a
non-zero code if anything failed. It never starts servers or Playwright.

Usage:
    # Ensure the server is running first (separate process), e.g.:
    #   python scripts/orchestrate.py
    python scripts/run_matrix_audit.py                 # full suite
    python scripts/run_matrix_audit.py --quick         # fast checks only (~15s)
    python scripts/run_matrix_audit.py --timeframe 15m
    python scripts/run_matrix_audit.py --fail-fast
    python scripts/run_matrix_audit.py --report reports/my_audit.json

Exit codes:
    0  all checks passed (SKIP allowed)
    1  one or more checks failed or errored
    2  prerequisite failure (server unreachable / auth failed)
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

# Make the sibling `audit` package importable regardless of CWD.
sys.path.insert(0, str(Path(__file__).resolve().parent))

from audit.client import PrerequisiteError  # noqa: E402
from audit.runner import (  # noqa: E402
    print_summary,
    run_suite,
    summarize,
    write_report,
)


def build_cli() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description="OrcaAlgo matrix regression audit (fast, API-only)")
    p.add_argument("--quick", action="store_true",
                   help="Run only the fast single-backtest checks (skip matrix + optimizer)")
    p.add_argument("--timeframe", default="1h", help="Timeframe for backtests (default: 1h)")
    p.add_argument("--fail-fast", action="store_true",
                   help="Stop at the first failing check")
    p.add_argument("--report", default="reports/matrix_audit_report.json",
                   help="Path to write the JSON report")
    p.add_argument("--resume", action="store_true",
                   help="Skip checks already recorded as PASS in the checkpoint and only re-run the rest")
    p.add_argument("--checkpoint", default="reports/.matrix_audit_checkpoint.json",
                   help="Checkpoint file of completed checks (for --resume)")
    return p


def main() -> int:
    # UTF-8-safe console on Windows cp1252 terminals.
    try:
        sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    except Exception:  # noqa: BLE001
        pass

    args = build_cli().parse_args()

    print("=" * 74)
    print("  ORCAALGO MATRIX REGRESSION AUDIT")
    print(f"  mode={'quick' if args.quick else 'full'} timeframe={args.timeframe}")
    print("=" * 74)

    try:
        results, ctx = run_suite(
            quick=args.quick, timeframe=args.timeframe, fail_fast=args.fail_fast,
            resume=args.resume, checkpoint=args.checkpoint,
        )
    except PrerequisiteError as e:
        print(f"\nPREREQUISITE FAILURE: {e}\n")
        return 2

    summary = summarize(results)
    report_path = Path(args.report)
    write_report(results, ctx, summary, report_path)
    print_summary(results, summary, report_path)

    return 0 if summary["verdict"] == "PASS" else 1


if __name__ == "__main__":
    sys.exit(main())
