"""Multi-factor regime generation with explicit trend, mean-reversion,
and volatility factors. Replaces pure Heston with a two-layer factor model."""

from __future__ import annotations

from dataclasses import dataclass

import numpy as np
import pandas as pd


@dataclass
class FactorConfig:
    """Regime-specific factor parameters."""
    trend_phi: float
    mr_theta: float
    vol_sigma: float
    drift: float


REGIME_FACTORS: dict[int, FactorConfig] = {
    0: FactorConfig(trend_phi=0.05, mr_theta=0.15, vol_sigma=0.008, drift=0.0001),
    1: FactorConfig(trend_phi=0.85, mr_theta=0.02, vol_sigma=0.014, drift=0.0003),
    2: FactorConfig(trend_phi=0.30, mr_theta=0.40, vol_sigma=0.028, drift=0.0000),
    3: FactorConfig(trend_phi=-0.60, mr_theta=0.70, vol_sigma=0.065, drift=-0.0005),
}


class FactorGenerator:
    """Generate price paths from a regime-dependent multi-factor model."""

    def __init__(
        self,
        regime_sequence: np.ndarray,
        config: dict[int, FactorConfig] | None = None,
    ) -> None:
        self.regime_sequence = regime_sequence
        self.config = config or REGIME_FACTORS
        self.n = len(regime_sequence)

    def generate_log_returns(self) -> np.ndarray:
        """Generate log-returns from trend, MR, and residual factors."""
        trend = np.zeros(self.n)
        mr = np.zeros(self.n)
        log_returns = np.zeros(self.n)
        rng = np.random.default_rng()

        for t in range(self.n):
            regime = int(self.regime_sequence[t])
            cfg = self.config[regime]

            if t == 0:
                trend[t] = rng.normal(0, 0.01)
            else:
                trend[t] = cfg.trend_phi * trend[t - 1] + rng.normal(0, 0.01)

            if t == 0:
                mr[t] = rng.normal(0, 0.01)
            else:
                mr[t] = mr[t - 1] - cfg.mr_theta * mr[t - 1] + rng.normal(0, 0.01)

            residual = rng.normal(0, cfg.vol_sigma)
            beta_trend = 0.5
            beta_mr = 0.3
            log_returns[t] = (
                cfg.drift + beta_trend * trend[t] + beta_mr * mr[t] + residual
            )

        return log_returns

    def generate_prices(self, start_price: float = 100.0) -> np.ndarray:
        """Generate price path from log-returns. Returns array where prices[0] == start_price."""
        log_returns = self.generate_log_returns()
        log_prices = np.insert(np.cumsum(log_returns), 0, 0) + np.log(start_price)
        return np.exp(log_prices)[:self.n]


def generate_1m_candles_from_factors(
    symbol: str,
    start_date: str,
    end_date: str,
    transition_matrix: np.ndarray,
    seed: int | None = None,
) -> pd.DataFrame:
    """Generate 1-minute OHLCV candles using the multi-factor regime model."""
    from orca.simulation.regime import (
        DEFAULT_TRANSITION_MATRIX,
        RegimeSequenceGenerator,
    )
    from orca.simulation.regime_generator import TRADING_MINUTES_PER_DAY

    rng = np.random.default_rng(seed)
    start = pd.Timestamp(start_date)
    end = pd.Timestamp(end_date)

    n_trading_days = int(np.busday_count(start.date(), end.date()))
    if n_trading_days < 1:
        n_trading_days = 252

    gen = RegimeSequenceGenerator(
        transition_matrix=transition_matrix or DEFAULT_TRANSITION_MATRIX,
        seed=seed,
    )
    regime_labels, _ = gen.generate_sequence(n_trading_days, start_date=start_date)

    factor_gen = FactorGenerator(regime_labels)
    daily_drift = factor_gen.generate_log_returns()

    times: list[pd.Timestamp] = []
    opens: list[float] = []
    highs: list[float] = []
    lows: list[float] = []
    closes: list[float] = []
    volumes: list[int] = []
    regimes: list[int] = []

    price = 100.0
    business_days = pd.bdate_range(start=start, end=end)

    for day_idx, bday in enumerate(business_days):
        if day_idx >= n_trading_days:
            break

        regime = int(regime_labels[day_idx])
        day_return = daily_drift[day_idx]
        cfg = REGIME_FACTORS[regime]

        intra_vol = cfg.vol_sigma * np.sqrt(1.0 / TRADING_MINUTES_PER_DAY)
        minute_returns = rng.normal(
            day_return / TRADING_MINUTES_PER_DAY, intra_vol, TRADING_MINUTES_PER_DAY
        )
        minute_prices = price * np.exp(np.cumsum(minute_returns))
        minute_prices = np.insert(minute_prices, 0, price)

        for m in range(TRADING_MINUTES_PER_DAY):
            bar_open = minute_prices[m]
            bar_close = minute_prices[m + 1]
            vol_factor = max(0.5, cfg.vol_sigma * 10)
            bar_high = max(bar_open, bar_close) * (1 + abs(rng.normal(0, vol_factor * 0.002)))
            bar_low = min(bar_open, bar_close) * (1 - abs(rng.normal(0, vol_factor * 0.002)))
            bar_vol = max(1000, int(rng.exponential(5000)))

            minute_dt = bday + pd.Timedelta(hours=9, minutes=30 + m)
            times.append(minute_dt)
            opens.append(bar_open)
            highs.append(bar_high)
            lows.append(bar_low)
            closes.append(bar_close)
            volumes.append(bar_vol)
            regimes.append(regime)

        price = minute_prices[-1]

    df = pd.DataFrame({
        "timestamp": times,
        "open": opens,
        "high": highs,
        "low": lows,
        "close": closes,
        "volume": volumes,
        "regime_label": regimes,
        "data_source": "synthetic",
        "symbol": symbol,
        "timeframe": "1m",
    })
    return df
