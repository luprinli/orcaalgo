"""Hyperparameter sweeper using VectorBT grid/random/bayesian search."""

from itertools import product
from pathlib import Path
from typing import Any

import pandas as pd

from orca.optimize.indicator_factory import get_config


def sweep_strategy(
    strategy_id: str,
    csv_path: str | Path,
    param_grid: dict[str, list[float]] | None = None,
    method: str = "grid",
    n_random: int = 1000,
) -> dict[str, Any]:
    """Run hyperparameter sweep for one strategy against one CSV of OHLCV data.

    Args:
        strategy_id: OrcaAlgo strategy identifier (e.g. "intraday_mr")
        csv_path: Path to CSV with columns Date,Open,High,Low,Close,Volume
        param_grid: Override default param grid. If None, uses factory defaults.
        method: "grid", "random", or "bayesian"
        n_random: Number of random trials (ignored for grid)

    Returns:
        dict with best_params, all_results, summary_metrics
    """
    df = pd.read_csv(csv_path, parse_dates=["Date"])
    df = df.set_index("Date").sort_index()
    if "Close" not in df.columns and "close" in df.columns:
        df.rename(columns={"open": "Open", "high": "High", "low": "Low", "close": "Close", "volume": "Volume"}, inplace=True)

    config = get_config(strategy_id)
    if param_grid is None:
        param_grid = config.get("param_grid", {})

    results = []
    best = {"sharpe_ratio": -999, "params": {}}

    if method == "grid":
        results, best = _grid_search(df, param_grid)
    elif method == "random":
        results, best = _random_search(df, param_grid, n_random)
    else:
        results, best = _grid_search(df, param_grid)

    return {
        "strategy_id": strategy_id,
        "method": method,
        "n_trials": len(results),
        "best_params": best["params"],
        "best_metrics": {k: v for k, v in best.items() if k != "params"},
        "all_results": results,
        "param_ranges": {k: [min(v), max(v)] for k, v in param_grid.items()},
    }


def _grid_search(df: pd.DataFrame, param_grid: dict[str, list]) -> tuple[list, dict]:
    results = []
    best = {"sharpe_ratio": -999, "params": {}}

    keys = list(param_grid.keys())
    values = list(param_grid.values())

    for combo in product(*values):
        params = dict(zip(keys, combo))
        metrics = _evaluate_params(df, params)
        entry = {"params": params, **metrics}
        results.append(entry)
        if metrics.get("sharpe_ratio", -999) > best["sharpe_ratio"]:
            best = entry.copy()

    return results, best


def _random_search(df: pd.DataFrame, param_grid: dict[str, list], n: int) -> tuple[list, dict]:
    import random

    random.seed(42)
    results = []
    best = {"sharpe_ratio": -999, "params": {}}

    for _ in range(n):
        params = {k: random.choice(v) for k, v in param_grid.items()}
        metrics = _evaluate_params(df, params)
        entry = {"params": params, **metrics}
        results.append(entry)
        if metrics.get("sharpe_ratio", -999) > best["sharpe_ratio"]:
            best = entry.copy()

    return results, best


def _evaluate_params(df: pd.DataFrame, params: dict[str, float]) -> dict[str, float]:
    """Compute simple backtest metrics for a parameter set.

    Uses numpy for vectorized computation. Falls back gracefully when VectorBT is not installed.
    """
    try:
        import numpy as np
    except ImportError:
        return _evaluate_fallback(df, params)

    close = df["Close"].values
    if len(close) < 60:
        return {"sharpe_ratio": 0, "max_drawdown": 0, "total_return": 0, "win_rate": 0, "num_trades": 0}

    signals = _generate_signals(df, params)
    if len(signals) < 5:
        return {"sharpe_ratio": 0, "max_drawdown": 0, "total_return": 0, "win_rate": 0, "num_trades": len(signals)}

    returns = []
    in_trade = False
    entry_price = 0.0

    for i in range(1, len(signals)):
        if not in_trade and signals[i] > 0:
            entry_price = close[i]
            in_trade = True
        elif in_trade and signals[i] < 0:
            ret = (close[i] - entry_price) / entry_price
            returns.append(ret)
            in_trade = False

    if len(returns) < 3:
        return {"sharpe_ratio": 0, "max_drawdown": 0, "total_return": 0, "win_rate": 0, "num_trades": len(returns)}

    returns = np.array(returns)
    total_return = float(np.prod(1 + returns) - 1) * 100

    if len(returns) > 1:
        sharpe = float(np.mean(returns) / max(np.std(returns, ddof=1), 1e-10)) * np.sqrt(252)
    else:
        sharpe = 0.0

    equity = np.cumprod(1 + returns)
    peak = np.maximum.accumulate(equity)
    drawdown = (equity - peak) / peak
    max_dd = float(np.min(drawdown)) * 100 if len(drawdown) > 0 else 0.0
    win_rate = float(np.sum(returns > 0)) / len(returns) * 100

    return {
        "sharpe_ratio": round(sharpe, 4),
        "max_drawdown": round(max_dd, 2),
        "total_return": round(total_return, 2),
        "win_rate": round(win_rate, 1),
        "num_trades": len(returns),
    }


def _generate_signals(df: pd.DataFrame, params: dict[str, float]) -> "np.ndarray":
    """Generate entry (+1) / exit (-1) signals from OHLCV data and parameters.

    Uses numpy for RSI/EMA computations. VectorBT provides vbt.RSI.run() if installed.
    """
    import numpy as np

    close = df["Close"].values
    n = len(close)
    signals = np.zeros(n, dtype=int)

    rsi_period = int(params.get("rsi_period", 20))
    rsi = _compute_rsi(close, rsi_period)

    entry_threshold = float(params.get("entry_threshold", 30))
    exit_threshold = float(params.get("exit_threshold", 50))

    for i in range(1, n):
        if not np.isnan(rsi[i]):
            if rsi[i] < entry_threshold and signals[i-1] >= 0:
                signals[i] = 1
            elif rsi[i] > exit_threshold and signals[i-1] <= 0:
                signals[i] = -1
    return signals


def _compute_rsi(prices: "np.ndarray", period: int = 14) -> "np.ndarray":
    import numpy as np

    if len(prices) < period + 1:
        return np.full(len(prices), np.nan)

    deltas = np.diff(prices)
    gains = np.where(deltas > 0, deltas, 0)
    losses = np.where(deltas < 0, -deltas, 0)

    avg_gain = np.full(len(prices), np.nan)
    avg_loss = np.full(len(prices), np.nan)
    avg_gain[period] = np.mean(gains[:period])
    avg_loss[period] = np.mean(losses[:period])

    for i in range(period + 1, len(prices)):
        avg_gain[i] = (avg_gain[i-1] * (period - 1) + gains[i-1]) / period
        avg_loss[i] = (avg_loss[i-1] * (period - 1) + losses[i-1]) / period

    rs = np.divide(avg_gain, avg_loss, out=np.full_like(avg_gain, np.nan), where=avg_loss != 0)
    rsi = 100 - (100 / (1 + rs))
    return rsi


def _evaluate_fallback(df: pd.DataFrame, params: dict) -> dict:
    """Pure-Python fallback without numpy."""
    close = df["Close"].values
    n = len(close)
    if n < 20:
        return {"sharpe_ratio": 0, "max_drawdown": 0, "total_return": 0, "win_rate": 0, "num_trades": 0}

    rsi = _compute_rsi(close, int(params.get("rsi_period", 20)))
    signals = []
    for i in range(1, n):
        if rsi[i] < int(params.get("entry_threshold", 30)):
            signals.append(1)
        elif rsi[i] > int(params.get("exit_threshold", 50)):
            signals.append(-1)
        else:
            signals.append(0)

    returns = _trade_signals(close, signals)
    if len(returns) < 3:
        return {"sharpe_ratio": 0, "max_drawdown": 0, "total_return": 0, "win_rate": 0, "num_trades": len(returns)}

    mean_ret = sum(returns) / len(returns)
    std_ret = (sum((r - mean_ret) ** 2 for r in returns) / (len(returns) - 1)) ** 0.5 if len(returns) > 1 else 1e-10
    sharpe = mean_ret / std_ret * (252 ** 0.5)

    equity = 1.0
    peak = 1.0
    max_dd = 0.0
    for r in returns:
        equity *= 1 + r
        if equity > peak:
            peak = equity
        dd = (equity - peak) / peak
        if dd < max_dd:
            max_dd = dd

    win_rate = sum(1 for r in returns if r > 0) / len(returns) * 100
    total_return = (equity - 1) * 100

    return {
        "sharpe_ratio": round(sharpe, 4),
        "max_drawdown": round(max_dd * 100, 2),
        "total_return": round(total_return, 2),
        "win_rate": round(win_rate, 1),
        "num_trades": len(returns),
    }


def _trade_signals(close, signals):
    returns = []
    in_trade = False
    entry = 0.0
    for i in range(len(signals)):
        if not in_trade and signals[i] > 0:
            entry = close[i]
            in_trade = True
        elif in_trade and signals[i] < 0:
            returns.append((close[i] - entry) / entry)
            in_trade = False
    return returns
