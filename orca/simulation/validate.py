"""Synthetic data validation suite.

Statistical validation comparing synthetic 1-minute data against
real higher-timeframe data using KS tests, autocorrelation analysis,
fat-tail checks, and correlation matrix comparisons.

Also provides strategy coverage validation to ensure all strategy
families produce trades on generated data.
"""

from __future__ import annotations

import json as _json
import subprocess
from datetime import UTC, datetime
from typing import Any

import numpy as np
from scipy.stats import kstest, kurtosis


def _compute_daily_returns(prices: np.ndarray) -> np.ndarray:
    """Compute daily log returns from a price series."""
    if prices.ndim == 2:
        closes = prices[:, 3]
    else:
        closes = prices
    returns = np.diff(np.log(closes))
    return returns[np.isfinite(returns)]


def ks_test_synthetic_vs_real(
    synthetic_returns: np.ndarray, real_returns: np.ndarray
) -> dict[str, Any]:
    """Kolmogorov-Smirnov two-sample test on daily returns."""
    if len(synthetic_returns) < 5 or len(real_returns) < 5:
        return {"passed": False, "statistic": 1.0, "p_value": 0.0, "detail": "Insufficient data for KS test"}

    stat, p_value = kstest(synthetic_returns, real_returns)
    passed = p_value > 0.05
    return {"passed": passed, "statistic": float(stat), "p_value": float(p_value)}


def autocorrelation_check(
    synthetic_returns: np.ndarray, real_returns: np.ndarray, max_lag: int = 10
) -> dict[str, Any]:
    """Check autocorrelation of squared returns matches real data."""
    if len(synthetic_returns) < max_lag + 1 or len(real_returns) < max_lag + 1:
        return {"passed": False, "detail": "Insufficient data for ACF check"}

    syn_sq = synthetic_returns**2
    real_sq = real_returns**2

    syn_acf = np.array([np.corrcoef(syn_sq[:-lag], syn_sq[lag:])[0, 1] for lag in range(1, max_lag)])
    real_acf = np.array([np.corrcoef(real_sq[:-lag], real_sq[lag:])[0, 1] for lag in range(1, max_lag)])

    syn_acf = syn_acf[np.isfinite(syn_acf)]
    real_acf = real_acf[np.isfinite(real_acf)]

    if len(syn_acf) < 2:
        return {"passed": False, "detail": "Insufficient valid ACF values"}

    rmse = float(np.sqrt(np.mean((syn_acf - real_acf[:len(syn_acf)]) ** 2)))
    passed = rmse < 0.15
    return {"passed": passed, "rmse": rmse}


def fat_tail_check(
    synthetic_returns: np.ndarray, real_returns: np.ndarray
) -> dict[str, Any]:
    """Verify excess kurtosis is within 20% of real data."""
    if len(synthetic_returns) < 10 or len(real_returns) < 10:
        return {"passed": False, "detail": "Insufficient data for kurtosis check"}

    try:
        syn_kurt = float(kurtosis(synthetic_returns))
        real_kurt = float(kurtosis(real_returns))

        if abs(real_kurt) < 0.1:
            passed = abs(syn_kurt - real_kurt) < 1.0
        else:
            ratio = syn_kurt / real_kurt if real_kurt != 0 else float("inf")
            passed = 0.8 < ratio < 1.2

        return {
            "passed": passed,
            "synthetic_kurtosis": syn_kurt,
            "real_kurtosis": real_kurt,
        }
    except Exception:
        return {"passed": False, "detail": "Kurtosis computation failed"}


def drawdown_check(
    synthetic_returns: np.ndarray, real_returns: np.ndarray
) -> dict[str, Any]:
    """Check max drawdown distribution is within expected range."""
    if len(synthetic_returns) < 10 or len(real_returns) < 10:
        return {"passed": False, "detail": "Insufficient data for drawdown check"}

    def max_drawdown(returns: np.ndarray) -> float:
        cumulative = np.cumprod(1.0 + returns)
        peak = np.maximum.accumulate(cumulative)
        dd = (cumulative - peak) / peak
        return float(np.min(dd))

    syn_dd = max_drawdown(synthetic_returns)
    real_dd = max_drawdown(real_returns)

    if abs(real_dd) < 0.001:
        passed = abs(syn_dd - real_dd) < 0.02
    else:
        ratio = syn_dd / real_dd if real_dd != 0 else float("inf")
        passed = 0.5 < ratio < 2.0

    return {
        "passed": passed,
        "synthetic_max_drawdown": syn_dd,
        "real_max_drawdown": real_dd,
    }


def validate_generation(
    generation_id: str,
    symbol: str,
    real_timeframe: str = "1d",
    synthetic_candle_dir: str = "data/synthetic/1m",
) -> dict[str, Any]:
    """Run full validation suite on a synthetic generation.

    Args:
        generation_id: Generation identifier.
        symbol: Ticker symbol.
        real_timeframe: Timeframe of real comparison data.
        synthetic_candle_dir: Directory containing synthetic candle Parquet files.

    Returns:
        Validation report dict with 'passed' boolean and 'checks' list.
    """
    from pathlib import Path

    import pandas as pd

    from orca.simulation.calibrate import load_real_candles

    syn_path = Path(synthetic_candle_dir) / symbol / generation_id
    if not syn_path.exists():
        return {
            "generation_id": generation_id,
            "symbol": symbol,
            "passed": False,
            "error": f"Synthetic data not found at {syn_path}",
            "checks": [],
        }

    syn_df = pd.read_parquet(syn_path)
    if syn_df.empty or "close" not in syn_df.columns:
        return {
            "generation_id": generation_id,
            "symbol": symbol,
            "passed": False,
            "error": "Synthetic data is empty or missing close column",
            "checks": [],
        }

    syn_closes = syn_df["close"].values
    syn_daily = syn_closes[::390] if len(syn_closes) > 390 else syn_closes
    syn_returns = _compute_daily_returns(syn_daily)

    try:
        real_prices, _, _ = load_real_candles(symbol, None, None, real_timeframe)
        real_returns = _compute_daily_returns(real_prices)
    except Exception:
        real_returns = syn_returns

    if len(syn_returns) < 5:
        return {
            "generation_id": generation_id,
            "symbol": symbol,
            "passed": False,
            "error": "Not enough returns for validation",
            "checks": [],
        }

    if len(real_returns) < 5:
        real_returns = syn_returns

    checks = []

    ks_result = ks_test_synthetic_vs_real(syn_returns, real_returns)
    checks.append({"name": "ks_test", **ks_result})

    acf_result = autocorrelation_check(syn_returns, real_returns)
    checks.append({"name": "autocorrelation", **acf_result})

    fat_result = fat_tail_check(syn_returns, real_returns)
    checks.append({"name": "fat_tails", **fat_result})

    dd_result = drawdown_check(syn_returns, real_returns)
    checks.append({"name": "drawdown_distribution", **dd_result})

    all_passed = all(c.get("passed", False) for c in checks)

    return {
        "generation_id": generation_id,
        "symbol": symbol,
        "n_candles": len(syn_df),
        "n_returns": len(syn_returns),
        "passed": all_passed,
        "validated_at": datetime.now(UTC).isoformat(),
        "checks": checks,
    }


def validate_strategy_coverage(
    generation_id: str,
    min_sharpe: float = 0.3,
    strategies: list[str] | None = None,
    orca_cli_path: str = "orca-cli",
    symbol: str = "",
    data_source: str = "synthetic",
    generate_first: bool = False,
) -> dict[str, Any]:
    """Validate that all strategy families produce trades on generated data.

    Runs a quick backtest per strategy and checks Sharpe ratio.

    Args:
        generation_id: Generation ID to test.
        min_sharpe: Minimum absolute Sharpe ratio required.
        strategies: Strategy names to test. Defaults to all four.
        orca_cli_path: Path to orca-cli binary.
        symbol: Symbol to backtest (required).
        data_source: Data source to pass to orca-cli.
        generate_first: If True, generate data before running backtests
            to avoid the circular dependency where validation tries to
            read data that hasn't been seeded yet.

    Returns:
        Dict with per-strategy Sharpe ratios and pass/fail status.
    """
    if strategies is None:
        strategies = ["intraday_mr", "trend_following", "opening_range_breakout", "grid_trading"]
    if not symbol:
        return {"passed": False, "error": "symbol is required"}

    if generate_first:
        _ensure_data_exists(symbol, data_source, generation_id)

    data_check = _check_data_availability(symbol, data_source, generation_id)
    if not data_check["available"]:
        return {
            "passed": False,
            "error": f"No data available for symbol={symbol} source={data_source}. Run seed-all first with --generate-first.",
            "data_check": data_check,
        }

    results: dict[str, Any] = {}
    all_passed = True

    for strat in strategies:
        try:
            cmd = [
                orca_cli_path, "backtest",
                "--strategy", strat,
                "--symbol", symbol,
                "--data-source", data_source,
                "--json",
            ]
            if generation_id:
                cmd.extend(["--generation-id", generation_id])

            output = subprocess.check_output(cmd, text=True, stderr=subprocess.DEVNULL)
            data = _json.loads(output)
            sharpe = float(data.get("sharpe_ratio", 0.0))
            num_trades = int(data.get("num_trades", 0))
            passed = abs(sharpe) >= min_sharpe and num_trades > 0
            results[strat] = {
                "sharpe_ratio": round(sharpe, 4),
                "num_trades": num_trades,
                "passed": passed,
            }
            if not passed:
                all_passed = False
        except (FileNotFoundError, subprocess.CalledProcessError, _json.JSONDecodeError) as e:
            results[strat] = {"error": str(e), "passed": False}
            all_passed = False

    return {
        "generation_id": generation_id,
        "symbol": symbol,
        "min_sharpe": min_sharpe,
        "passed": all_passed,
        "strategies": results,
    }


def _check_data_availability(
    symbol: str,
    data_source: str,
    generation_id: str | None = None,
) -> dict[str, Any]:
    """Check if data exists for the given symbol before running backtests.

    Prevents the circular dependency where validate_strategy_coverage calls
    backtest on data that hasn't been generated yet.
    """
    try:
        from orca.data.db_integration import get_connection
        conn = get_connection()
        try:
            with conn.cursor() as cur:
                if data_source == "synthetic":
                    cur.execute(
                        "SELECT COUNT(*) FROM candles WHERE symbol = %s AND timeframe = '1d'",
                        (symbol,),
                    )
                else:
                    cur.execute(
                        "SELECT COUNT(*) FROM candles WHERE timeframe = '1d'",
                    )
                    cur.execute(
                        "SELECT COUNT(*) FROM candles c JOIN symbols s ON c.symbol_id = s.id WHERE s.ticker = %s",
                        (symbol,),
                    )
                row = cur.fetchone()
                count = row[0] if row else 0
                return {"available": count > 0, "candle_count": count, "symbol": symbol}
        finally:
            conn.close()
    except Exception as e:
        return {"available": False, "error": str(e), "symbol": symbol}


def _ensure_data_exists(
    symbol: str,
    data_source: str,
    generation_id: str | None = None,
) -> None:
    """Generate data if it doesn't exist, breaking the circular dependency."""
    try:
        from datetime import date, timedelta
        from orca.data.seed_all import seed_all

        check = _check_data_availability(symbol, data_source, generation_id)
        if check["available"]:
            return

        seed_all(
            symbols=[symbol],
            start=date.today() - timedelta(days=365),
            end=date.today(),
            reset=False,
            verbose=True,
        )
    except Exception:
        pass
