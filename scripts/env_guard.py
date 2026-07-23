#!/usr/bin/env python3
"""OrcaAlgo Environment Guard — Prevents AI agents from executing destructive
operations against live brokerage accounts without explicit authorization.

Exit codes: 0 = safe to proceed, 1 = blocked (live environment), 2 = error
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
CONFIG_PATH = ROOT / "config" / "env_guard.json"

PROTECTED_OPERATIONS = [
    "kill_switch_e2e",
    "balance_reconciliation",
    "place_order",
    "cancel_all_orders",
    "preflight_strict",
    "deploy_gate",
    "orca_calibrate",
    "orca_simulate",
]


def load_config() -> dict:
    if not CONFIG_PATH.exists():
        return {"allowed_environments": {
            "paper": {"PAPER_TRADING": "true"},
            "staging": {"ORCA_ENV": "staging", "PAPER_TRADING": "true"},
        }}
    with open(CONFIG_PATH) as f:
        return json.load(f)


def check_environment(operation: str) -> tuple[bool, str]:
    """Return (safe, message)."""
    config = load_config()
    paper_trading = os.environ.get("PAPER_TRADING", "").lower() == "true"
    orca_env = os.environ.get("ORCA_ENV", "local")
    alpaca_live = os.environ.get("ALPACA_LIVE", "").lower() == "true"
    allow_live = os.environ.get("ALLOW_LIVE_GUARD", "") == "explicit"
    allow_live_env = argparse.Namespace()
    # Check for --allow-live flag
    if "--allow-live" in sys.argv:
        allow_live = True

    # Determine if this is a protected operation
    is_protected = any(op in operation for op in PROTECTED_OPERATIONS)

    if not is_protected:
        return True, f"Operation '{operation}' is not protected — proceeding"

    # Paper trading is always safe
    if paper_trading:
        return True, f"PAPER_TRADING=true — safe for '{operation}'"

    # Live trading detected — block unless explicitly authorized
    if alpaca_live or orca_env == "production":
        if allow_live:
            return True, f"LIVE environment explicitly authorized for '{operation}' (ALLOW_LIVE_GUARD=explicit)"
        return False, (
            f"BLOCKED: Operation '{operation}' cannot run against live environment.\n"
            f"  PAPER_TRADING: {paper_trading}\n"
            f"  ORCA_ENV: {orca_env}\n"
            f"  ALPACA_LIVE: {alpaca_live}\n"
            f"  To override: set ALLOW_LIVE_GUARD=explicit in environment and re-run with --allow-live"
        )

    return True, f"Environment check passed for '{operation}'"


def main():
    parser = argparse.ArgumentParser(description="OrcaAlgo Environment Guard")
    parser.add_argument("operation", nargs="?", default="default",
                        help="Operation name to check (e.g. kill_switch_e2e, preflight_strict)")
    parser.add_argument("--check", default=None, help="Alias for operation")
    parser.add_argument("--allow-live", action="store_true", help="Explicitly allow live environment operations")
    parser.add_argument("--json", action="store_true", help="Output JSON")
    args = parser.parse_args()

    op = args.check or args.operation
    safe, message = check_environment(op)

    if args.json:
        print(json.dumps({"operation": op, "safe": safe, "message": message}))
    else:
        print(f"🔒 Environment Guard: {op}")
        print(f"   {'✅ SAFE' if safe else '🚫 BLOCKED'}: {message}")

    sys.exit(0 if safe else 1)


if __name__ == "__main__":
    main()
