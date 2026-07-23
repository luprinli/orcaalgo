"""Parameter sweep using VectorBT's built-in optimization.

Falls back to orca.optimize.sweeper when vectorbt is not installed.
Output format is identical regardless of backend used.
"""

import os
import sys
from pathlib import Path
from typing import Any

import numpy as np
import pandas as pd

from orca.vectorbt.data import load_candles
from orca.vectorbt.strategies import HAS_VBT, STRATEGY_MAP

if HAS_VBT:
    import vectorbt as vbt


def sweep_strategy(
    symbol: str,
    start: str,
    end: str,
    strategy_name: str,
    param_grid: dict[str, list[float]],
    metric: str = "sharpe",
    timeframe: str = "1h",
    fallback: str = "auto",
) -> dict[str, Any]:
    """Run a parameter sweep for a given strategy.

    Backend selection:
      - "vectorbt": Use vbt.GridSearch (requires vectorbt installed)
      - "native": Use orca.optimize.sweeper.sweep_strategy (always available)
      - "auto": Try vectorbt, fall back to native

    Returns the SAME result schema regardless of backend:
      {
        "strategy_id": "...",
        "method": "grid",
        "n_trials": ...,
        "best_params": {...},
        "best_metrics": {
            "sharpe_ratio": ..., "max_drawdown": ..., "total_return": ...,
            "win_rate": ..., "num_trades": ...,
        },
        "all_results": [...],
        "backend_used": "vectorbt" | "native"
      }
    """
    if strategy_name not in STRATEGY_MAP:
        available = ", ".join(sorted(STRATEGY_MAP.keys()))
        raise ValueError(f"Unknown strategy '{strategy_name}'. Available: {available}")

    df = load_candles(symbol, start, end, timeframe)

    if fallback == "auto":
        backend = "vectorbt" if HAS_VBT else "native"
    else:
        backend = fallback

    if backend == "vectorbt" and HAS_VBT:
        return _sweep_vectorbt(df, strategy_name, param_grid, metric)
    else:
        return _sweep_native(df, strategy_name, param_grid)


def _sweep_vectorbt(
    df: pd.DataFrame,
    strategy_name: str,
    param_grid: dict[str, list[float]],
    metric: str,
) -> dict[str, Any]:
    """Use VectorBT's GridSearch for parameter optimization."""
    close = df["close"]
    high = df.get("high", close)
    low = df.get("low", close)
    volume = df.get("volume", pd.Series(0, index=df.index))

    strategy_fn = STRATEGY_MAP[strategy_name]

    try:
        results = vbt.GridSearch(
            strategy_fn,
            params=param_grid,
            param_combinations="product",
            close=close,
            high=high,
            low=low,
            volume=volume,
        )
        best_idx = results[metric].idxmax()
        best_params = dict(zip(param_grid.keys(), best_idx, strict=False))
        best_metrics = _extract_vbt_metrics(results, best_idx)

        all_results = []
        shape = results[metric].shape
        for i in range(int(np.prod(shape))):
            idx_tuple = np.unravel_index(i, shape)
            combo = dict(zip(param_grid.keys(), idx_tuple, strict=False))
            all_results.append({
                "params": combo,
                "sharpe_ratio": round(float(results[metric].values.flat[i]), 4),
            })

        return {
            "strategy_id": strategy_name,
            "method": "grid",
            "n_trials": len(all_results),
            "best_params": best_params,
            "best_metrics": best_metrics,
            "all_results": all_results,
            "backend_used": "vectorbt",
        }
    except Exception as exc:
        print(
            f"VectorBT GridSearch failed ({exc}). Falling back to native sweeper.",
            file=sys.stderr,
        )
        return _sweep_native(df, strategy_name, param_grid)


def _sweep_native(
    df: pd.DataFrame,
    strategy_name: str,
    param_grid: dict[str, list[float]],
) -> dict[str, Any]:
    """Fall back to existing orca.optimize.sweeper.

    Converts DataFrame to temporary CSV because sweeper expects a file path.
    """
    from orca.optimize.sweeper import sweep_strategy as native_sweep

    tmp_dir = Path(os.getenv("ORCA_TMP_DIR", "data/tmp"))
    tmp_dir.mkdir(parents=True, exist_ok=True)
    tmp_path = tmp_dir / f"_vbt_fallback_{strategy_name}.csv"

    df.to_csv(tmp_path)

    result = native_sweep(strategy_name, str(tmp_path), param_grid, method="grid")
    result["backend_used"] = "native"

    tmp_path.unlink(missing_ok=True)
    return result


def _extract_vbt_metrics(results, best_idx) -> dict[str, Any]:
    """Extract metrics in the standard format matching sweeper.py output."""
    return {
        "sharpe_ratio": round(float(results.get("sharpe", pd.Series([0])).iloc[0]), 4),
        "max_drawdown": round(
            float(results.get("max_drawdown", pd.Series([0])).iloc[0]), 2
        ),
        "total_return": round(
            float(results.get("total_return", pd.Series([0])).iloc[0]), 2
        ),
        "win_rate": round(
            float(results.get("win_rate", pd.Series([0])).iloc[0]), 1
        ),
        "num_trades": int(results.get("trades", pd.Series([0])).iloc[0]),
    }
