"""Export VectorBT sweep results as Go-compatible pipeline JSON.

Converts VBT's parameter sweep output (top-K candidates with params + metrics)
to the format expected by POST /api/v1/backtests/pipeline. This is Stage 1→2
of the 5-stage optimization pipeline:

  VBT coarse sweep → Go fine-grid purged CV → Bayesian opt → Walk-forward

Usage:
    python -m orca.vectorbt.sweep_exporter \
        --symbol SPY --start 2022-01-01 --end 2024-12-31 \
        --strategy intraday_mr --top-k 20 --output sweep.json
"""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

import pandas as pd

from orca.vectorbt.optimize import sweep_strategy
from orca.vectorbt.strategies import STRATEGY_MAP


def export_sweep_to_pipeline(
    symbol: str,
    start: str,
    end: str,
    strategy_name: str,
    param_grid: dict[str, list[float]] | None = None,
    top_k: int = 20,
    timeframe: str = "1h",
    metric: str = "sharpe",
) -> dict[str, Any]:
    """Export VBT sweep results as a Go pipeline-ready JSON payload.

    Returns a dict with:
        pipeline: {strategy_id, symbols, timeframes, start_date, end_date, combos}
        candidates: [{rank, params, metrics}]
    """
    if strategy_name not in STRATEGY_MAP:
        raise ValueError(
            f"Unknown strategy '{strategy_name}'. Available: {list(STRATEGY_MAP)}"
        )

    if param_grid is None:
        strategy_fn = STRATEGY_MAP[strategy_name]
        param_grid = strategy_fn["default_grid"]

    result = sweep_strategy(
        symbol=symbol,
        start=start,
        end=end,
        strategy_name=strategy_name,
        param_grid=param_grid,
        metric=metric,
        timeframe=timeframe,
    )

    df: pd.DataFrame = result["results"]
    if df.empty:
        return {"pipeline": None, "candidates": [], "error": "No candidates found"}

    top = df.nlargest(top_k, metric)

    candidates = []
    for rank, (idx, row) in enumerate(top.iterrows(), 1):
        params = {}
        for col in df.columns:
            if col not in (metric, "start", "end", "duration", "exposure"):
                if not col.startswith("_"):
                    params[col] = float(row[col]) if pd.notna(row[col]) else 0.0

        candidates.append({
            "rank": rank,
            "params": params,
            "metrics": {metric: float(row[metric])},
        })

    combos = []
    for c in candidates:
        combos.append({
            "strategy": strategy_name,
            "symbol": [symbol],
            "timeframe": [timeframe],
            "params": c["params"],
        })

    return {
        "pipeline": {
            "strategy_id": strategy_name,
            "symbols": [symbol],
            "timeframes": [timeframe],
            "start_date": start,
            "end_date": end,
            "combos": combos,
        },
        "candidates": candidates,
        "top_k": top_k,
        "total_evaluated": len(df),
    }


def export_to_file(
    symbol: str,
    start: str,
    end: str,
    strategy_name: str,
    output_path: str | Path,
    param_grid: dict[str, list[float]] | None = None,
    top_k: int = 20,
) -> Path:
    """Export sweep results to a JSON file."""
    result = export_sweep_to_pipeline(
        symbol=symbol, start=start, end=end,
        strategy_name=strategy_name, param_grid=param_grid, top_k=top_k,
    )
    output_path = Path(output_path)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    with open(output_path, "w") as f:
        json.dump(result, f, indent=2, default=str)
    return output_path


if __name__ == "__main__":
    import argparse
    p = argparse.ArgumentParser(description="Export VBT sweep to Go pipeline JSON")
    p.add_argument("--symbol", default="SPY")
    p.add_argument("--start", default="2022-01-01")
    p.add_argument("--end", default="2024-12-31")
    p.add_argument("--strategy", default="intraday_mr")
    p.add_argument("--top-k", type=int, default=20)
    p.add_argument("--output", default="sweep_export.json")
    args = p.parse_args()
    path = export_to_file(args.symbol, args.start, args.end, args.strategy, args.output, top_k=args.top_k)
    print(f"Exported to {path}")
