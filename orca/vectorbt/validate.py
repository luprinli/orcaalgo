"""Validate VectorBT results against Orca's Go backtest engine.

Performs a cross-engine comparison and reports metric deltas.
Handles Go CLI version differences gracefully — detects supported flags,
falls back to text parsing for legacy CLI versions, and never crashes on
missing orca-cli.
"""

import json
import subprocess
from typing import Any

import pandas as pd

from orca.vectorbt.data import load_candles

DEFAULT_TOLERANCE: dict[str, float] = {
    "sharpe": 0.30,
    "max_dd": 5.0,
    "win_rate": 10.0,
}


def compare_results(
    symbol: str,
    strategy: str,
    params: dict[str, float],
    start: str,
    end: str,
    timeframe: str = "1h",
    tolerance: dict[str, float] | None = None,
) -> dict[str, Any]:
    """Run both VectorBT/native and Orca Go backtest, compare metrics.

    Args:
        tolerance: Max acceptable metric differences.
                   Default: {"sharpe": 0.30, "max_dd": 5.0, "win_rate": 10.0}

    Returns:
        {
          "vectorbt_metrics": {...},
          "orca_go_metrics": {...},
          "diffs": {"sharpe": ..., "max_dd": ..., "win_rate": ...},
          "passed": bool,
          "validation_failures": [...],
          "go_cli_version": "...",
        }
    """
    if tolerance is None:
        tolerance = DEFAULT_TOLERANCE

    df = _load_data(symbol, start, end, timeframe)
    vbt_metrics = _compute_metrics_native(df, strategy, params)
    go_metrics = _run_orca_go_backtest(symbol, strategy, params, start, end, timeframe)

    diffs: dict[str, float] = {}
    failures: list[str] = []
    metric_keys = ["sharpe", "max_dd", "win_rate"]

    for key in metric_keys:
        vbt_val = float(vbt_metrics.get(key, 0))
        go_val = float(go_metrics.get(key, 0))
        diff = round(abs(vbt_val - go_val), 4)
        diffs[key] = diff
        if diff > tolerance[key]:
            failures.append(
                f"{key}: vbt={vbt_val:.4f} go={go_val:.4f} diff={diff:.4f} > tol={tolerance[key]}"
            )

    go_version = go_metrics.get("_cli_version", "unknown")

    return {
        "vectorbt_metrics": vbt_metrics,
        "orca_go_metrics": {k: v for k, v in go_metrics.items() if k != "_cli_version"},
        "diffs": diffs,
        "passed": len(failures) == 0,
        "validation_failures": failures,
        "go_cli_version": go_version,
    }


def _run_orca_go_backtest(
    symbol: str,
    strategy: str,
    params: dict[str, float],
    start: str,
    end: str,
    timeframe: str,
) -> dict[str, Any]:
    """Run orca-cli backtest. Detects available CLI flags and adapts.

    Backward compat: tries --output json first; if unsupported, parses stdout.
    """
    cmd = [
        "orca-cli",
        "backtest",
        "--symbol",
        symbol,
        "--strategy",
        strategy,
        "--start",
        start,
        "--end",
        end,
        "--timeframe",
        timeframe,
        "--params",
        ",".join(f"{k}={v}" for k, v in params.items()),
    ]

    version = _detect_cli_version()
    if version >= (0, 3):
        cmd.extend(["--output", "json"])

    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
    except FileNotFoundError:
        return {
            "_cli_version": "unknown",
            "error": "orca-cli not found in PATH",
        }
    except subprocess.TimeoutExpired:
        return {
            "_cli_version": ".".join(map(str, version)),
            "error": "orca-cli backtest timed out (120s)",
        }

    if result.returncode != 0:
        return {
            "_cli_version": ".".join(map(str, version)),
            "error": result.stderr.strip() or f"exit code {result.returncode}",
        }

    if "--output" in cmd and "json" in cmd:
        try:
            return {"_cli_version": ".".join(map(str, version)), **json.loads(result.stdout)}
        except json.JSONDecodeError:
            pass

    return _parse_legacy_output(result.stdout, version)


def _detect_cli_version() -> tuple[int, ...]:
    """Detect orca-cli version. Returns (0, 0, 0) if undetectable."""
    try:
        r = subprocess.run(
            ["orca-cli", "--version"],
            capture_output=True,
            text=True,
            timeout=5,
        )
        for token in r.stdout.split():
            cleaned = token.lstrip("v")
            if cleaned and (cleaned[0].isdigit() or cleaned[:1].replace(".", "").isdigit()):
                return tuple(int(x) for x in cleaned.split("."))
    except Exception:
        pass
    return (0, 0, 0)


def _parse_legacy_output(stdout: str, version: tuple[int, ...]) -> dict[str, Any]:
    """Parse text output from older orca-cli versions."""
    metrics: dict[str, Any] = {"_cli_version": ".".join(map(str, version))}
    for line in stdout.splitlines():
        line_lower = line.strip().lower()
        if "sharpe" in line_lower:
            try:
                metrics["sharpe"] = float(line.split(":")[-1].strip())
            except (ValueError, IndexError):
                pass
        if "max drawdown" in line_lower or "max_dd" in line_lower:
            try:
                metrics["max_dd"] = float(line.split(":")[-1].strip().replace("%", ""))
            except (ValueError, IndexError):
                pass
        if "win rate" in line_lower or "win_rate" in line_lower:
            try:
                metrics["win_rate"] = float(line.split(":")[-1].strip().replace("%", ""))
            except (ValueError, IndexError):
                pass
    return metrics


def _load_data(symbol: str, start: str, end: str, timeframe: str) -> pd.DataFrame:
    return load_candles(symbol, start, end, timeframe)


def _compute_metrics_native(
    df: pd.DataFrame,
    strategy: str,
    params: dict[str, float],
) -> dict[str, Any]:
    """Compute metrics using existing sweeper._evaluate_params."""
    from orca.optimize.sweeper import _evaluate_params

    metrics = _evaluate_params(df, params)
    return {
        "sharpe": metrics.get("sharpe_ratio", 0),
        "max_dd": metrics.get("max_drawdown", 0),
        "win_rate": metrics.get("win_rate", 0),
    }
