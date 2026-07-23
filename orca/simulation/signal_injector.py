"""Signal injection for synthetic data.

Provides injectors for trend, mean-reversion, and breakout signals
that can be applied to existing Heston-generated price paths.
"""

from __future__ import annotations

import numpy as np


class BaseInjector:
    """Base class for signal injectors."""

    def __init__(self, strength: float = 0.3) -> None:
        self.strength = strength

    def inject(self, prices: np.ndarray) -> np.ndarray:
        raise NotImplementedError


class TrendInjector(BaseInjector):
    """Inject AR(1) serial correlation with positive phi."""

    def __init__(self, strength: float = 0.3, lag: int = 20, phi: float = 0.6) -> None:
        super().__init__(strength)
        self.lag = lag
        self.phi = phi

    def inject(self, prices: np.ndarray) -> np.ndarray:
        n = len(prices)
        trend = np.zeros(n)
        rng = np.random.default_rng()
        for i in range(1, n):
            trend[i] = self.phi * trend[i - 1] + rng.normal(0, 0.01)
        log_prices = np.log(prices)
        trend_std = np.std(trend)
        if trend_std > 0:
            trend_scaled = self.strength * trend * float(np.std(log_prices)) / trend_std
        else:
            trend_scaled = 0
        new_log = log_prices + trend_scaled
        return np.exp(new_log)


class MeanReversionInjector(BaseInjector):
    """Inject Ornstein-Uhlenbeck mean-reversion."""

    def __init__(self, strength: float = 0.3, theta: float = 0.1) -> None:
        super().__init__(strength)
        self.theta = theta

    def inject(self, prices: np.ndarray) -> np.ndarray:
        n = len(prices)
        mr = np.zeros(n)
        rng = np.random.default_rng()
        for i in range(1, n):
            mr[i] = mr[i - 1] - self.theta * mr[i - 1] + rng.normal(0, 0.01)
        log_prices = np.log(prices)
        mr_std = np.std(mr)
        if mr_std > 0:
            mr_scaled = self.strength * mr * float(np.std(log_prices)) / mr_std
        else:
            mr_scaled = 0
        new_log = log_prices + mr_scaled
        return np.exp(new_log)


class BreakoutInjector(BaseInjector):
    """Inject directional drift after range expansion."""

    def __init__(self, strength: float = 0.3, lookback: int = 20, drift_bars: int = 5) -> None:
        super().__init__(strength)
        self.lookback = lookback
        self.drift_bars = drift_bars

    def inject(self, prices: np.ndarray) -> np.ndarray:
        n = len(prices)
        breakout = np.zeros(n)
        for i in range(self.lookback, n):
            high = float(np.max(prices[i - self.lookback : i]))
            low = float(np.min(prices[i - self.lookback : i]))
            if prices[i] > high * 1.01:
                for j in range(i, min(i + self.drift_bars, n)):
                    breakout[j] += (j - i + 1) * 0.001 * self.strength
            elif prices[i] < low * 0.99:
                for j in range(i, min(i + self.drift_bars, n)):
                    breakout[j] -= (j - i + 1) * 0.001 * self.strength
        log_prices = np.log(prices)
        breakout_std = float(np.std(breakout))
        if breakout_std > 0:
            breakout_scaled = breakout * float(np.std(log_prices)) / breakout_std
        else:
            breakout_scaled = breakout
        new_log = log_prices + breakout_scaled
        return np.exp(new_log)
