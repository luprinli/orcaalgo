"""Synthetic 1-minute OHLCV data generator.

Generates realistic 1-minute candles from calibrated statistical models
(GBM, OU, Heston) with intraday volume profiles and residual KDE refinement.
"""

from __future__ import annotations

import hashlib
import json
from datetime import UTC, datetime, timedelta
from pathlib import Path
from typing import Any

import numpy as np
import pandas as pd

from orca.simulation.calibrate import calibrate_symbol

TRADING_MINUTES_PER_DAY = 390


def _generate_volume_profile(n_minutes: int, profile: str = "u_shaped") -> np.ndarray:
    """Generate intraday volume weights.

    u_shaped: higher volume at market open and close.
    sine: highest volume mid-session.
    flat: uniform volume.
    """
    t = np.linspace(0.0, 1.0, n_minutes)
    if profile == "u_shaped":
        weights = np.cos(np.pi * t) ** 2 + 0.3
    elif profile == "sine":
        weights = np.sin(np.pi * t) ** 2 + 0.1
    elif profile == "flat":
        weights = np.ones(n_minutes)
    else:
        weights = np.ones(n_minutes)
    weights = weights / weights.sum()
    weights = np.maximum(weights, 0.0001)
    weights = weights / weights.sum()
    return weights


def _apply_kde_refinement(returns: np.ndarray, kde_samples: list[float]) -> np.ndarray:
    """Refine normally-distributed returns using KDE residuals."""
    if not kde_samples or len(kde_samples) < 5:
        return returns
    kde = np.array(kde_samples, dtype=np.float64)
    kde = kde / kde.sum()
    xs = np.linspace(-4.0, 4.0, len(kde))
    cdf = np.cumsum(kde)
    cdf = cdf / cdf[-1]

    uniform = np.random.uniform(0, 1, len(returns))
    indices = np.searchsorted(cdf, uniform)
    indices = np.clip(indices, 0, len(kde) - 1)
    noise = xs[indices]
    return returns + noise * np.std(returns) * 0.1


def _generate_daily_returns_gbm(
    n_minutes: int, mu: float, sigma: float, rng: np.random.Generator
) -> np.ndarray:
    """Generate intraday returns using GBM.

    Parameters are daily-scale: the sum of n_minutes returns
    has expectation mu and standard deviation sigma.
    """
    dt = 1.0 / n_minutes
    drift = (mu - 0.5 * sigma**2) * dt
    diffusion = sigma * np.sqrt(dt)
    shocks = rng.normal(0, 1, n_minutes)
    returns = drift + diffusion * shocks
    return returns


def _generate_daily_returns_ou(
    n_minutes: int, theta: float, mu: float, sigma: float, rng: np.random.Generator
) -> np.ndarray:
    """Generate intraday returns using Ornstein-Uhlenbeck process.

    Parameters are daily-scale: sum of n_minutes returns has mean ~0 and std ~sigma.
    """
    dt = 1.0 / n_minutes
    if theta <= 0:
        return _generate_daily_returns_gbm(n_minutes, mu, sigma, rng)
    prices = np.zeros(n_minutes + 1)
    prices[0] = mu
    for i in range(n_minutes):
        shock = rng.normal(0, 1)
        prices[i + 1] = prices[i] + theta * (mu - prices[i]) * dt + sigma * np.sqrt(dt) * shock
    returns = np.diff(prices)
    return returns


def _generate_daily_returns_heston(
    n_minutes: int,
    kappa: float,
    theta: float,
    sigma_v: float,
    rho: float,
    dt_daily: float,
    rng: np.random.Generator,
) -> np.ndarray:
    """Generate intraday returns using Heston stochastic volatility model."""
    dt = dt_daily / n_minutes
    returns = np.zeros(n_minutes)
    v = theta

    for i in range(n_minutes):
        z1 = rng.normal(0, 1)
        z2 = rho * z1 + np.sqrt(1 - rho**2) * rng.normal(0, 1)

        v = max(v + kappa * (theta - v) * dt + sigma_v * np.sqrt(max(v, 0) * dt) * z2, 1e-10)
        returns[i] = np.sqrt(v * dt) * z1

    return returns


def _compute_generation_id(config: dict[str, Any]) -> str:
    """Compute a deterministic SHA-256 generation ID from config."""
    canonical = json.dumps(config, sort_keys=True, default=str)
    return hashlib.sha256(canonical.encode()).hexdigest()[:16]


def _get_us_trading_days(start: datetime, end: datetime) -> list[datetime]:
    """Return list of US market trading days (Monday-Friday, excluding NYSE holidays).

    Uses the real NYSE holiday calendar from orca.data.nyse_calendar.
    """
    try:
        from orca.data.nyse_calendar import get_trading_days
        return [
            datetime(d.year, d.month, d.day, tzinfo=UTC)
            for d in get_trading_days(start.date(), end.date())
        ]
    except ImportError:
        pass

    trading_days = []
    current = start.replace(hour=0, minute=0, second=0, microsecond=0)
    if current.tzinfo is None:
        current = current.replace(tzinfo=UTC)
    end_tz = end
    if end_tz.tzinfo is None:
        end_tz = end_tz.replace(tzinfo=UTC)

    while current <= end_tz:
        if current.weekday() < 5:
            trading_days.append(current)
        current += timedelta(days=1)
    return trading_days


def _market_open(day: datetime) -> datetime:
    """Return 9:30 AM ET for a given trading day."""
    return day.replace(hour=13, minute=30, second=0, microsecond=0)


def generate_1m_candles(
    symbol: str,
    start: datetime,
    end: datetime,
    model: str = "heston",
    calibration: dict[str, Any] | None = None,
    seed: int | None = None,
    volume_profile: str = "u_shaped",
    timeframe: str = "1d",
    daily_volume_avg: float | None = None,
) -> pd.DataFrame:
    """Generate synthetic 1-minute OHLCV candles.

    Args:
        symbol: Ticker symbol.
        start: Start date.
        end: End date.
        model: Model to use ('gbm', 'ou', 'heston').
        calibration: Pre-computed calibration dict. If None, calibrates from DB.
        seed: Random seed for reproducibility.
        volume_profile: Intraday volume shape ('u_shaped', 'sine', 'flat').
        timeframe: Timeframe of calibration data ('1d', '1h', '5m').
        daily_volume_avg: Average daily volume override.

    Returns:
        DataFrame with columns: time, open, high, low, close, volume, symbol.
    """
    rng = np.random.default_rng(seed)

    if calibration is None:
        calibration = calibrate_symbol(symbol, timeframe, start, end)

    if model == "gbm":
        params = calibration["gbm"]
        mu, sigma = params["mu"], params["sigma"]
    elif model == "ou":
        params = calibration["ou"]
        mu, sigma = params["mu"], params["sigma"]
        theta = params.get("theta", 0.05)
    else:
        params = calibration.get("heston", {"kappa": 2.0, "theta": 0.04, "sigma_v": 0.3, "rho": -0.7})
        kappa = params.get("kappa", 2.0)
        theta = params.get("theta", 0.04)
        sigma_v = params.get("sigma_v", 0.3)
        rho = params.get("rho", -0.7)

    kde_residuals = calibration.get("kde_residuals", [])

    start_price = 100.0
    if "last_close" in calibration:
        start_price = float(calibration["last_close"])
    elif "gbm" in calibration:
        try:
            prices, _, _ = __import__("orca.simulation.calibrate", fromlist=["load_real_candles"]).load_real_candles(
                symbol, None, None, timeframe
            )
            if len(prices) > 0:
                start_price = float(prices[-1, 3])
        except Exception:
            pass

    vol_weights = _generate_volume_profile(TRADING_MINUTES_PER_DAY, volume_profile)

    if daily_volume_avg is None:
        daily_volume_avg = 1_000_000.0

    trading_days = _get_us_trading_days(start, end)

    all_rows: list[dict[str, Any]] = []

    current_price = start_price

    for day in trading_days:
        market_open = _market_open(day)

        if model == "gbm":
            returns = _generate_daily_returns_gbm(TRADING_MINUTES_PER_DAY, mu, sigma, rng)
        elif model == "ou":
            returns = _generate_daily_returns_ou(TRADING_MINUTES_PER_DAY, theta, mu, sigma, rng)
        else:
            returns = _generate_daily_returns_heston(
                TRADING_MINUTES_PER_DAY, kappa, theta, sigma_v, rho, 1.0, rng
            )

        if kde_residuals:
            returns = _apply_kde_refinement(returns, kde_residuals)

        intraday_prices = current_price * np.exp(np.cumsum(returns))
        intraday_prices = np.clip(intraday_prices, current_price * 0.5, current_price * 2.0)

        daily_vol = daily_volume_avg * (0.5 + rng.random() * 1.0)
        volumes = (vol_weights * daily_vol).astype(np.int64)

        open_price = current_price
        for minute in range(TRADING_MINUTES_PER_DAY):
            t_now = market_open + timedelta(minutes=minute)

            if minute == 0:
                segment_prices = [current_price, intraday_prices[0]]
            else:
                segment_prices = [intraday_prices[minute - 1], intraday_prices[minute]]

            high = max(np.max(segment_prices) * (1.0 + rng.uniform(0, 0.0005)), open_price)
            low = min(np.min(segment_prices) * (1.0 - rng.uniform(0, 0.0005)), open_price)
            close = intraday_prices[minute]

            all_rows.append({
                "time": t_now,
                "open": open_price,
                "high": high,
                "low": low,
                "close": close,
                "volume": int(volumes[minute]),
                "symbol": symbol,
            })
            open_price = close

        current_price = intraday_prices[-1]
        if current_price <= 0 or not np.isfinite(current_price):
            current_price = start_price

    df = pd.DataFrame(all_rows)
    if df.empty:
        return df

    df["time"] = pd.to_datetime(df["time"], utc=True)
    df = df.sort_values("time").reset_index(drop=True)
    return df


def generate_and_save(
    symbol: str,
    start: datetime,
    end: datetime,
    model: str = "heston",
    calibration: dict[str, Any] | None = None,
    seed: int | None = None,
    volume_profile: str = "u_shaped",
    timeframe: str = "1d",
    output_dir: str = "data/synthetic/1m",
    daily_volume_avg: float | None = None,
) -> str:
    """Generate 1-minute candles and save as Parquet partitioned by symbol/date.

    Returns the generation_id.
    """
    gen_config = {
        "symbol": symbol,
        "start": start.isoformat(),
        "end": end.isoformat(),
        "model": model,
        "seed": seed,
        "volume_profile": volume_profile,
        "timeframe": timeframe,
        "calibration": calibration,
    }
    generation_id = _compute_generation_id(gen_config)

    df = generate_1m_candles(
        symbol=symbol,
        start=start,
        end=end,
        model=model,
        calibration=calibration,
        seed=seed,
        volume_profile=volume_profile,
        timeframe=timeframe,
        daily_volume_avg=daily_volume_avg,
    )

    if df.empty:
        raise ValueError(f"No candles generated for {symbol} ({start} to {end})")

    out_path = Path(output_dir) / symbol / generation_id
    out_path.mkdir(parents=True, exist_ok=True)

    if "year" not in df.columns:
        df["year"] = df["time"].dt.year
    if "month" not in df.columns:
        df["month"] = df["time"].dt.month

    df.to_parquet(out_path, partition_cols=["year", "month"], index=False)

    meta_path = out_path / "_generation.json"
    gen_config["generation_id"] = generation_id
    gen_config["n_candles"] = len(df)
    gen_config["generated_at"] = datetime.now(UTC).isoformat()
    with open(meta_path, "w") as f:
        json.dump(gen_config, f, default=str, indent=2)

    return generation_id
