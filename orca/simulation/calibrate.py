"""Parameter calibration from real OHLCV data for synthetic data generation.

Extracts statistical parameters (GBM, Ornstein-Uhlenbeck, Heston SV) from
real TimescaleDB candles to drive realistic synthetic 1-minute generation.
"""

from __future__ import annotations

import json
import os
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

import numpy as np
from scipy.optimize import minimize
from scipy.stats import gaussian_kde


def _get_db_url() -> str:
    return os.environ.get("ORCA_DB_URL", "postgresql://orca:orca@localhost:5432/orca_core")


def load_real_candles(
    symbol: str,
    start: datetime | None = None,
    end: datetime | None = None,
    timeframe: str = "1d",
) -> tuple[np.ndarray, np.ndarray, np.ndarray]:
    """Load real OHLCV candles from TimescaleDB.

    Returns:
        (prices, volumes, timestamps) as numpy arrays.
        prices is shape (N, 4) for O/H/L/C.
    """
    try:
        import psycopg2
    except ImportError as e:
        raise ImportError(
            "psycopg2 is required for database access. "
            "Install with: pip install psycopg2-binary"
        ) from e

    conn = psycopg2.connect(_get_db_url())
    cur = conn.cursor()

    query = """
        SELECT c.time, c.open_raw, c.high_raw, c.low_raw, c.close_raw, c.volume
        FROM candles c
        JOIN symbols s ON c.symbol_id = s.id
        WHERE s.ticker = %s AND c.timeframe = %s AND c.source != 'synthetic'
    """
    params = [symbol, timeframe]

    if start is not None:
        query += " AND c.time >= %s"
        params.append(start)
    if end is not None:
        query += " AND c.time <= %s"
        params.append(end)

    query += " ORDER BY c.time ASC"

    cur.execute(query, params)
    rows = cur.fetchall()
    cur.close()
    conn.close()

    if not rows:
        raise ValueError(f"No real candles found for {symbol} (timeframe={timeframe})")

    times = np.array([r[0].timestamp() for r in rows])
    prices = np.array(
        [[float(r[1]) / 100_000, float(r[2]) / 100_000, float(r[3]) / 100_000, float(r[4]) / 100_000]
         for r in rows],
        dtype=np.float64,
    )
    volumes = np.array([float(r[5]) for r in rows], dtype=np.float64)

    return prices, volumes, times


def compute_gbm_params(prices: np.ndarray) -> dict[str, float]:
    """Compute Geometric Brownian Motion parameters from close prices.

    Returns mu (drift per period) and sigma (volatility per period).
    """
    closes = prices[:, 3] if prices.ndim == 2 else prices
    log_returns = np.diff(np.log(closes))
    log_returns = log_returns[np.isfinite(log_returns)]

    mu = float(np.mean(log_returns))
    sigma = float(np.std(log_returns, ddof=1))
    return {"mu": mu, "sigma": sigma}


def compute_ou_params(prices: np.ndarray) -> dict[str, float]:
    """Compute Ornstein-Uhlenbeck parameters from log prices.

    dX = theta * (mu - X) * dt + sigma * dW

    Returns theta (mean-reversion speed), mu (long-term mean), sigma (volatility).
    """
    closes = prices[:, 3] if prices.ndim == 2 else prices
    log_prices = np.log(closes)

    x_t = log_prices[:-1]
    x_t1 = log_prices[1:]

    mask = np.isfinite(x_t) & np.isfinite(x_t1)
    x_t = x_t[mask]
    x_t1 = x_t1[mask]

    if len(x_t) < 2:
        return {"theta": 0.0, "mu": 0.0, "sigma": 0.0}

    a = np.column_stack([np.ones_like(x_t), x_t])
    coeffs, residuals, _, _ = np.linalg.lstsq(a, x_t1, rcond=None)

    if len(residuals) == 0:
        sigma = 0.0
    else:
        sigma = float(np.sqrt(residuals[0] / (len(x_t) - 2)))

    theta = float(1.0 - coeffs[1]) if coeffs[1] < 1.0 else 0.0
    mu = float(coeffs[0] / theta) if theta > 1e-10 else float(np.mean(log_prices))

    return {"theta": max(theta, 0.0), "mu": mu, "sigma": sigma}


def compute_heston_params(prices: np.ndarray) -> dict[str, float]:
    """Estimate Heston SV parameters from price series.

    Uses method of moments for initial estimates, then MLE refinement.
    All parameters are on daily scale.

    Returns kappa, theta (long-run daily variance), sigma_v (vol-of-vol), rho (correlation).
    """
    closes = prices[:, 3] if prices.ndim == 2 else prices
    log_returns = np.diff(np.log(closes))
    log_returns = log_returns[np.isfinite(log_returns)]

    if len(log_returns) < 10:
        return {"kappa": 2.0, "theta": 0.04, "sigma_v": 0.3, "rho": -0.7}

    daily_var = float(np.var(log_returns, ddof=1))
    kappa = 2.0
    theta = max(daily_var, 1e-8)

    squared_returns = log_returns**2
    var_of_variance = float(np.var(squared_returns, ddof=1))
    sigma_v = max(np.sqrt(var_of_variance) / max(np.sqrt(theta), 1e-8), 0.01)
    sigma_v = min(sigma_v, 0.5)

    if len(log_returns) > 1:
        rho = float(np.clip(np.corrcoef(log_returns[:-1], np.abs(log_returns[1:]))[0, 1], -0.99, 0.99))
    else:
        rho = -0.7

    try:
        initial = [kappa, theta, sigma_v, rho]
        rng = np.random.default_rng(42)

        def neg_log_likelihood(params: np.ndarray) -> float:
            k, t, s, r = params
            if k <= 0 or t <= 0 or s <= 0 or abs(r) >= 1:
                return 1e10
            v = np.zeros(len(log_returns) + 1, dtype=np.float64)
            v[0] = t
            nll = 0.0
            for i in range(len(log_returns)):
                v[i + 1] = max(v[i] + k * (t - v[i]) + s * np.sqrt(max(v[i], 0)) * rng.normal(), 1e-10)
                nll += 0.5 * (np.log(2 * np.pi * v[i]) + (log_returns[i] ** 2) / max(v[i], 1e-10))
            return float(nll)

        result = minimize(
            neg_log_likelihood,
            initial,
            bounds=[(0.1, 50.0), (1e-8, 0.1), (0.01, 0.5), (-0.99, 0.99)],
            method="L-BFGS-B",
            options={"maxiter": 200},
        )
        if result.success:
            kappa = float(result.x[0])
            theta = float(result.x[1])
            sigma_v = float(result.x[2])
            rho = float(result.x[3])
    except Exception:
        pass

    return {"kappa": kappa, "theta": theta, "sigma_v": sigma_v, "rho": rho}


def compute_residual_distribution(
    prices: np.ndarray, model: str = "gbm"
) -> list[float]:
    """Fit KDE on model residuals and return sample points."""
    closes = prices[:, 3] if prices.ndim == 2 else prices
    log_returns = np.diff(np.log(closes))
    log_returns = log_returns[np.isfinite(log_returns)]

    if model == "gbm":
        params = compute_gbm_params(prices)
        mu = params["mu"]
        sigma = params["sigma"]
        residuals = (log_returns - mu) / max(sigma, 1e-10)
    else:
        residuals = (log_returns - np.mean(log_returns)) / max(np.std(log_returns, ddof=1), 1e-10)

    residuals = residuals[np.isfinite(residuals)]
    if len(residuals) < 10:
        return [0.0]

    residuals = np.clip(residuals, -5.0, 5.0)
    try:
        kde = gaussian_kde(residuals)
        xs = np.linspace(-4.0, 4.0, 1000)
        pdf_values = kde(xs)
        pdf_values = pdf_values / pdf_values.sum()
        return pdf_values.tolist()
    except Exception:
        return residuals[:1000].tolist()


def calibrate_all(
    symbols: list[str],
    timeframe: str = "1d",
    start: datetime | None = None,
    end: datetime | None = None,
    output_dir: str = "configs/simulation",
) -> dict[str, str]:
    """Calibrate all models for a list of symbols and save calibation JSON files.

    Returns dict of symbol -> output file path.
    """
    root = Path(output_dir)
    root.mkdir(parents=True, exist_ok=True)

    results: dict[str, str] = {}

    for symbol in symbols:
        try:
            prices, volumes, times = load_real_candles(symbol, start, end, timeframe)

            calibration: dict[str, Any] = {
                "symbol": symbol,
                "timeframe": timeframe,
                "n_candles": len(prices),
                "calibrated_at": datetime.now(UTC).isoformat(),
            }
            calibration["gbm"] = compute_gbm_params(prices)
            calibration["ou"] = compute_ou_params(prices)
            calibration["heston"] = compute_heston_params(prices)
            calibration["kde_residuals"] = compute_residual_distribution(prices, "gbm")

            filepath = root / f"calibration_{symbol}.json"
            with open(filepath, "w") as f:
                json.dump(calibration, f, default=str, indent=2)

            results[symbol] = str(filepath)
        except Exception as e:
            results[symbol] = f"ERROR: {e}"

    return results


def calibrate_symbol(
    symbol: str,
    timeframe: str = "1d",
    start: datetime | None = None,
    end: datetime | None = None,
) -> dict[str, Any]:
    """Calibrate a single symbol and return the calibration dict.

    Does not write to disk.
    """
    prices, _volumes, _times = load_real_candles(symbol, start, end, timeframe)

    calibration: dict[str, Any] = {
        "symbol": symbol,
        "timeframe": timeframe,
        "n_candles": len(prices),
        "calibrated_at": datetime.now(UTC).isoformat(),
    }
    calibration["gbm"] = compute_gbm_params(prices)
    calibration["ou"] = compute_ou_params(prices)
    calibration["heston"] = compute_heston_params(prices)
    calibration["kde_residuals"] = compute_residual_distribution(prices, "gbm")

    return calibration


def load_candles_from_csv(filepath: str | Path) -> tuple[np.ndarray, np.ndarray, np.ndarray]:
    """Load OHLCV candles from a Stooq-format CSV file.

    Format: TICKER,PER,DATE,TIME,OPEN,HIGH,LOW,CLOSE,VOL,OPENINT

    Returns:
        (prices, volumes, timestamps) as numpy arrays.
        prices is shape (N, 4) for O/H/L/C.
    """
    import csv

    path = Path(filepath)
    if not path.exists():
        raise FileNotFoundError(f"CSV file not found: {path}")

    rows = []
    with open(path) as f:
        reader = csv.DictReader(f)
        for row in reader:
            try:
                ts = datetime(
                    int(row["<DATE>"][:4]),
                    int(row["<DATE>"][4:6]),
                    int(row["<DATE>"][6:8]),
                    int(row["<TIME>"][:2]),
                    int(row["<TIME>"][2:4]),
                    int(row["<TIME>"][4:6]),
                )
                rows.append({
                    "time": ts,
                    "open": float(row["<OPEN>"]),
                    "high": float(row["<HIGH>"]),
                    "low": float(row["<LOW>"]),
                    "close": float(row["<CLOSE>"]),
                    "volume": float(row["<VOL>"]),
                })
            except (KeyError, ValueError):
                continue

    if not rows:
        raise ValueError(f"No valid rows in {path}")

    rows.sort(key=lambda r: r["time"])
    times = np.array([r["time"].timestamp() for r in rows])
    prices = np.array([[r["open"], r["high"], r["low"], r["close"]] for r in rows], dtype=np.float64)
    volumes = np.array([r["volume"] for r in rows], dtype=np.float64)

    return prices, volumes, times


def scale_params_to_daily(params: dict[str, float], source_timeframe_minutes: int) -> dict[str, float]:
    """Scale model parameters from sub-daily timeframe to daily.

    5-min -> daily: periods_per_day = 390 / 5 = 78
      daily_mu = 5min_mu * periods_per_day
      daily_sigma = 5min_sigma * sqrt(periods_per_day)
      daily_theta (OU) = 5min_theta (mean-reversion speed is time-scale invariant, stays same)
      daily_theta_variance (Heston) = 5min_theta * periods_per_day (variance scales with time)
    """
    periods_per_day = 390.0 / source_timeframe_minutes
    sqrt_n = np.sqrt(periods_per_day)

    scaled: dict[str, float] = {}
    for key, value in params.items():
        if key == "mu":
            scaled[key] = value * periods_per_day
        elif key == "sigma":
            scaled[key] = value * sqrt_n
        elif key == "theta" and not isinstance(value, dict):
            scaled[key] = value
        else:
            scaled[key] = value

    if "theta" in params and isinstance(params.get("theta"), (int, float)):
        scaled["mu"] = params.get("mu", 0.0) * periods_per_day
        scaled["sigma"] = params.get("sigma", 0.0) * sqrt_n

    return scaled


def calibrate_from_csv(
    symbol: str,
    csv_path: str | Path,
    source_timeframe_minutes: int = 5,
) -> dict[str, Any]:
    """Calibrate from a 5-min CSV file and return daily-scale parameters."""
    prices, _volumes, _times = load_candles_from_csv(csv_path)

    gbm = compute_gbm_params(prices)
    ou = compute_ou_params(prices)
    heston = compute_heston_params(prices)

    gbm_daily = scale_params_to_daily(gbm, source_timeframe_minutes)
    ou_daily = scale_params_to_daily(ou, source_timeframe_minutes)
    heston_daily = scale_params_to_daily(heston, source_timeframe_minutes)

    calibration: dict[str, Any] = {
        "symbol": symbol,
        "timeframe": f"daily (scaled from {source_timeframe_minutes}min)",
        "n_candles": len(prices),
        "calibrated_at": datetime.now(UTC).isoformat(),
        "gbm": gbm_daily,
        "ou": ou_daily,
        "heston": heston_daily,
        "kde_residuals": compute_residual_distribution(prices, "gbm"),
        "last_close": float(prices[-1, 3]),
    }

    return calibration


def calibrate_all_from_csv(
    csv_dir: str | Path,
    output_dir: str = "configs/simulation",
    source_timeframe_minutes: int = 5,
) -> dict[str, str]:
    """Calibrate all symbols from 5-min CSV files in a directory.

    Returns dict of symbol -> output file path.
    """
    root = Path(output_dir)
    root.mkdir(parents=True, exist_ok=True)

    csv_path = Path(csv_dir)
    results: dict[str, str] = {}

    for f in sorted(csv_path.glob("*.txt")):
        if f.stat().st_size == 0:
            continue
        symbol = f.stem
        try:
            cal = calibrate_from_csv(symbol, f, source_timeframe_minutes)
            filepath = root / f"calibration_{symbol}.json"
            with open(filepath, "w") as fh:
                json.dump(cal, fh, default=str, indent=2)
            results[symbol] = str(filepath)
        except Exception as e:
            results[symbol] = f"ERROR: {e}"

    return results
