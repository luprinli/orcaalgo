"""Regime-aware synthetic data generator.

Generates 1-minute OHLCV candles and tick data conditioned on a
market regime sequence. Supports batch generation with halt/resume
progress tracking.

Usage:
    from orca.simulation.regime_generator import generate_regime_aware

    gen_id, candles_df = generate_regime_aware(
        symbol="SPY", start="2020-01-01", end="2024-12-31",
        model="heston", seed=42,
        progress_callback=print,
    )
"""

from __future__ import annotations

import hashlib
import json
from collections.abc import Callable
from datetime import datetime, timedelta
from pathlib import Path
from typing import Any

import numpy as np
import pandas as pd

from orca.simulation.calibrate_regime import load_regime_params
from orca.simulation.regime import (
    DEFAULT_TRANSITION_MATRIX,
    REGIME_CALM,
    RegimeBatchState,
    RegimeSequenceGenerator,
    regime_params_for_state,
)

TRADING_MINUTES_PER_DAY = 390
TRADING_DAYS_PER_YEAR = 252


def generate_regime_aware(
    symbol: str = "SPY",
    start_date: str = "2020-01-01",
    end_date: str = "2024-12-31",
    model: str = "heston",
    transition_matrix: np.ndarray | None = None,
    regime_params_override: dict[int, dict[str, Any]] | None = None,
    config_dir: str | Path = "configs/simulation",
    output_dir: str | Path | None = None,
    seed: int | None = None,
    progress_callback: Callable[[dict[str, Any]], None] | None = None,
    halt_dir: str | Path | None = None,
    use_factor_model: bool = False,
) -> tuple[str, pd.DataFrame, np.ndarray, RegimeBatchState]:
    """Generate regime-aware synthetic 1-minute OHLCV data.

    When use_factor_model is True, delegates to the multi-factor factor generator
    instead of the Heston model, producing data with explicit trend, mean-reversion,
    and volatility factors per regime.

    Returns:
        (generation_id, candles_df, regime_labels, batch_state)
    """
    if use_factor_model:
        from orca.simulation.factor_generator import generate_1m_candles_from_factors

        candles_df = generate_1m_candles_from_factors(
            symbol=symbol,
            start_date=start_date,
            end_date=end_date,
            transition_matrix=transition_matrix,
            seed=seed,
        )
        regime_labels = candles_df["regime_label"].values
        start = datetime.fromisoformat(start_date)
        end = datetime.fromisoformat(end_date)
        n_trading_days = int(np.busday_count(start.date(), end.date()))
        gid = hashlib.sha256(
            json.dumps({"symbol": symbol, "model": "factor", "seed": seed}, sort_keys=True).encode()
        ).hexdigest()[:12]
        state = RegimeBatchState(gid, n_trading_days)
        state.start(None)
        return gid, candles_df, regime_labels, state

    rng = np.random.default_rng(seed)

    start = datetime.fromisoformat(start_date)
    end = datetime.fromisoformat(end_date)
    n_trading_days = int(np.busday_count(start.date(), end.date()))
    if n_trading_days < 1:
        n_trading_days = 252

    gen = RegimeSequenceGenerator(
        transition_matrix=transition_matrix or DEFAULT_TRANSITION_MATRIX,
        seed=seed,
    )
    regime_labels, _ = gen.generate_sequence(n_trading_days, start_date=start)

    generation_id = _compute_generation_id(symbol, start_date, end_date, model, seed, regime_labels)

    if regime_params_override is None:
        regime_params_override = load_regime_params(symbol, Path(config_dir))

    state = RegimeBatchState(generation_id, n_trading_days)
    halt_path = Path(halt_dir) if halt_dir else None
    state.start(halt_path)

    times = []
    opens = []
    highs = []
    lows = []
    closes = []
    volumes = []
    regimes = []

    price = 100.0
    rng = np.random.default_rng(seed)

    business_days = pd.bdate_range(start=start, end=end)
    day_idx = 0

    for bday in business_days:
        if state.check_halt():
            break
        if day_idx >= n_trading_days:
            break

        regime = int(regime_labels[day_idx])
        params = regime_params_for_state(regime, regime_params_override)

        daily_returns = _generate_daily_minute_bars(
            price=price,
            params=params,
            n_minutes=TRADING_MINUTES_PER_DAY,
            rng=rng,
            model=model,
        )

        minute_prices = price * np.exp(np.cumsum(daily_returns))
        minute_prices = np.insert(minute_prices, 0, price)

        for m in range(TRADING_MINUTES_PER_DAY):
            bar_open = minute_prices[m]
            bar_close = minute_prices[m + 1]
            bar_high = max(bar_open, bar_close) * (1 + abs(rng.normal(0, params.sigma * 0.2)))
            bar_low = min(bar_open, bar_close) * (1 - abs(rng.normal(0, params.sigma * 0.2)))
            bar_vol = max(1000, int(rng.exponential(5000) * params.volume_mult))

            minute_dt = bday + timedelta(hours=9, minutes=30 + m)

            times.append(minute_dt)
            opens.append(bar_open)
            highs.append(bar_high)
            lows.append(bar_low)
            closes.append(bar_close)
            volumes.append(bar_vol)
            regimes.append(regime)

        price = minute_prices[-1]
        state.advance(days=1, regime=regime)

        if progress_callback and day_idx % 20 == 0:
            progress_callback(state.progress_dict())

        day_idx += 1

    df = pd.DataFrame(
        {
            "timestamp": times,
            "open": opens,
            "high": highs,
            "low": lows,
            "close": closes,
            "volume": volumes,
            "regime_label": regimes if len(regimes) == len(times) else None,
            "data_source": "synthetic",
            "generation_id": generation_id,
            "symbol": symbol,
            "timeframe": "1m",
        }
    )

    if output_dir:
        out_path = Path(output_dir)
        out_path.mkdir(parents=True, exist_ok=True)
        df.to_parquet(out_path / f"{symbol}_{generation_id}_1m.parquet")
        np.save(out_path / f"{symbol}_{generation_id}_labels.npy", regime_labels)
        meta = {
            "generation_id": generation_id,
            "symbol": symbol,
            "start": start_date,
            "end": end_date,
            "model": model,
            "n_candles": len(df),
            "n_trading_days": n_trading_days,
            "seed": seed,
        }
        with open(out_path / f"{generation_id}_meta.json", "w") as f:
            json.dump(meta, f, indent=2, default=str)

    if progress_callback:
        progress_callback(state.progress_dict())

    return generation_id, df, regime_labels, state


def _generate_daily_minute_bars(
    price: float,
    params,
    n_minutes: int,
    rng: np.random.Generator,
    model: str = "heston",
) -> np.ndarray:
    """Generate intraday minute returns conditioned on regime parameters."""
    dt = 1.0 / (TRADING_DAYS_PER_YEAR * n_minutes)
    mu = params.mu * dt
    sigma = params.sigma * np.sqrt(dt)

    base_returns = rng.normal(mu, sigma, n_minutes)

    if params.trend_bias != 0:
        trend = np.linspace(0, params.trend_bias * params.sigma, n_minutes)
        base_returns += trend * dt

    if model == "heston" and params.sigma > 0.01:
        v = np.ones(n_minutes) * params.sigma**2
        for t in range(1, n_minutes):
            v[t] = max(
                0.0001,
                v[t - 1] + 0.05 * (params.sigma**2 - v[t - 1]) + params.sigma * 0.1 * rng.normal(),
            )
        vol_factor = np.sqrt(v) / params.sigma
        base_returns *= vol_factor

    if params.jump_intensity > 0 and rng.random() < params.jump_intensity:
        jump_magnitude = rng.normal(params.jump_mean, params.jump_std)
        base_returns[rng.integers(0, n_minutes)] += jump_magnitude

    return base_returns


def _compute_generation_id(
    symbol: str,
    start: str,
    end: str,
    model: str,
    seed: int | None,
    regime_labels: np.ndarray,
) -> str:
    """Deterministic generation ID from config hash."""
    config = {
        "symbol": symbol,
        "start": start,
        "end": end,
        "model": model,
        "seed": seed,
        "regime_hash": hashlib.sha256(regime_labels.tobytes()).hexdigest()[:16],
    }
    payload = json.dumps(config, sort_keys=True, default=str)
    return hashlib.sha256(payload.encode()).hexdigest()[:12]


def generate_regime_ticks(
    candles_df: pd.DataFrame,
    generation_id: str,
    ticks_per_minute: int = 60,
    output_dir: str | Path | None = None,
    seed: int | None = None,
) -> pd.DataFrame:
    """Disaggregate regime-aware 1-minute candles into tick data.

    Adjusts bid-ask spread and volume based on regime.
    """
    rng = np.random.default_rng(seed)
    ticks: list[dict] = []

    for _, row in candles_df.iterrows():
        n = ticks_per_minute
        t = np.linspace(0, 1, n + 1)
        dt = t[1]
        dW = rng.normal(0, np.sqrt(dt), n)
        bridge = np.zeros(n + 1)
        bridge[1:] = np.cumsum(dW)
        bridge -= t * bridge[-1]

        price_range = row["high"] - row["low"]
        vol = bridge * price_range
        base = row["close"] - vol[-1] if not np.isnan(row["close"]) else row["open"]
        path = base + vol

        regime = int(row.get("regime_label", REGIME_CALM))
        from orca.simulation.regime import DEFAULT_REGIME_PARAMS

        regime_defaults = DEFAULT_REGIME_PARAMS.get(regime, DEFAULT_REGIME_PARAMS[REGIME_CALM])
        spread_factor = regime_defaults.get("spread_mult", 1.0)
        half_spread = 0.00005 * spread_factor

        volume_per_tick = max(1, int(row["volume"] / n))

        for i in range(n):
            price = max(0.01, path[i])
            tick_time = row["timestamp"] + timedelta(seconds=i * (60 / n))
            ticks.append(
                {
                    "timestamp_ms": int(tick_time.timestamp() * 1000),
                    "price": round(price, 4),
                    "bid": round(price * (1 - half_spread), 4),
                    "ask": round(price * (1 + half_spread), 4),
                    "volume": volume_per_tick,
                    "symbol": row.get("symbol", ""),
                    "generation_id": generation_id,
                    "regime_label": regime,
                }
            )

    df = pd.DataFrame(ticks)
    if output_dir:
        out = Path(output_dir)
        out.mkdir(parents=True, exist_ok=True)
        df.to_parquet(out / f"ticks_{generation_id}.parquet")
    return df
