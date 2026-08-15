"""Walk-forward validation and Island Volume Selection (IVS)."""

from itertools import product
from typing import Any

import numpy as np

from orca.optimize.sweeper import _evaluate_params


def walk_forward_validate(
    csv_path: str,
    strategy_id: str,
    param_grid: dict[str, list],
    window_size: int = 252,
    step_size: int = 63,
    oos_size: int = 63,
) -> dict[str, Any]:
    """Rolling walk-forward optimization.

    For each window:
      - Optimize params on IS (window_size days)
      - Test best params on OOS (oos_size days)
      - Collect OOS metrics
    """
    import pandas as pd

    df = pd.read_csv(csv_path, parse_dates=["Date"])
    df = df.set_index("Date").sort_index()

    n = len(df)
    if n < window_size + oos_size:
        return {"error": f"Not enough data: {n} rows, need {window_size + oos_size}"}

    windows = []
    start = 0
    window_num = 0
    while start + window_size + oos_size <= n:
        is_df = df.iloc[start : start + window_size]
        oos_df = df.iloc[start + window_size : start + window_size + oos_size]

        best_is = _find_best(is_df, param_grid)
        oos_metrics = _evaluate_params(oos_df, best_is["params"])

        windows.append(
            {
                "window": window_num,
                "train_start": str(is_df.index[0])[:10],
                "train_end": str(is_df.index[-1])[:10],
                "test_start": str(oos_df.index[0])[:10],
                "test_end": str(oos_df.index[-1])[:10],
                "is_sharpe": best_is.get("sharpe_ratio", 0),
                "oos_sharpe": oos_metrics.get("sharpe_ratio", 0),
                "oos_win_rate": oos_metrics.get("win_rate", 0),
                "oos_return": oos_metrics.get("total_return", 0),
                "oos_trades": oos_metrics.get("num_trades", 0),
                "best_params": best_is["params"],
            }
        )
        start += step_size
        window_num += 1

    if not windows:
        return {"error": "No valid windows"}

    oos_sharpes = [w["oos_sharpe"] for w in windows]
    is_sharpes = [w["is_sharpe"] for w in windows]
    avg_oos = np.mean(oos_sharpes) if oos_sharpes else 0
    avg_is = np.mean(is_sharpes) if is_sharpes else 0
    degradation = (avg_is - avg_oos) / max(avg_is, 1e-10) * 100 if avg_is > 0 else 0
    passed = sum(1 for s in oos_sharpes if s > 1.0)

    return {
        "avg_oos_sharpe": round(float(avg_oos), 4),
        "avg_is_sharpe": round(float(avg_is), 4),
        "degradation_pct": round(float(degradation), 2),
        "passed_windows": passed,
        "total_windows": len(windows),
        "windows": windows,
    }


def _find_best(df, param_grid: dict) -> dict:
    """Find best parameters from grid search on IS data."""
    keys = list(param_grid.keys())
    values = list(param_grid.values())
    best = {"sharpe_ratio": -999, "params": {}}
    for combo in product(*values):
        params = dict(zip(keys, combo, strict=False))
        metrics = _evaluate_params(df, params)
        if metrics.get("sharpe_ratio", -999) > best["sharpe_ratio"]:
            best = {"params": params, **metrics}
    return best
