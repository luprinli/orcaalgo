"""Residual bootstrap from real market data.

Bootstraps empirical residuals from real data and recombines
them with synthetic factor paths to preserve empirical distributions.
"""

from __future__ import annotations

import numpy as np
import pandas as pd


class ResidualBootstrap:
    """Bootstrap residuals from real returns to generate synthetic paths."""

    def __init__(
        self,
        real_returns: np.ndarray,
        factor_returns: np.ndarray | None = None,
        block_size: int = 20,
    ) -> None:
        self.real_returns = real_returns
        self.factor_returns = factor_returns
        self.block_size = block_size
        self.n_real = len(real_returns)
        self.residuals = real_returns - float(np.mean(real_returns))

    def bootstrap(self, n_paths: int = 1, seed: int | None = None) -> np.ndarray:
        """Generate residual blocks and add to factors."""
        rng = np.random.default_rng(seed)
        resid_flat = self.residuals.flatten()
        n_blocks = max(1, self.n_real // self.block_size)
        generated: list[np.ndarray] = []

        for _ in range(n_paths):
            block_indices = rng.integers(0, n_blocks, size=n_blocks)
            residual_path = np.zeros(self.n_real)
            for i, bi in enumerate(block_indices):
                src_start = bi * self.block_size
                dst_start = i * self.block_size
                if dst_start + self.block_size <= self.n_real:
                    end = dst_start + self.block_size
                    residual_path[dst_start:end] = resid_flat[
                        src_start : src_start + self.block_size
                    ]

            drift = float(np.mean(self.real_returns))
            synthetic = drift + residual_path
            generated.append(synthetic)

        return np.array(generated)


def bootstrap_generate(
    symbol: str,
    start_date: str,
    end_date: str | None = None,
    lookback_years: int = 5,
) -> pd.DataFrame:
    """Generate synthetic prices via residual bootstrap from real data."""
    from orca.simulation.calibrate import load_real_candles

    real_prices, _, _ = load_real_candles(symbol, None, None, "1d")

    if len(real_prices) < 100:
        msg = f"Not enough real data for {symbol}: {len(real_prices)} prices"
        raise ValueError(msg)

    log_returns = np.diff(np.log(real_prices))
    log_returns = log_returns[np.isfinite(log_returns)]

    bootstrap = ResidualBootstrap(log_returns.reshape(-1, 1), None)
    bootstrapped_returns = bootstrap.bootstrap(n_paths=1)[0].flatten()

    start_price = float(real_prices[0])
    new_prices = start_price * np.exp(np.cumsum(bootstrapped_returns))

    n = len(new_prices)
    start = pd.Timestamp(start_date)
    end = pd.Timestamp(end_date) if end_date else start + pd.Timedelta(days=n - 1)
    timestamps = pd.date_range(start=start, end=end, periods=n)

    df = pd.DataFrame(
        {
            "timestamp": timestamps,
            "open": new_prices,
            "high": new_prices * 1.005,
            "low": new_prices * 0.995,
            "close": new_prices,
            "volume": np.random.poisson(1000, n),
            "regime_label": np.zeros(n, dtype=int),
            "data_source": "synthetic",
            "symbol": symbol,
            "timeframe": "1d",
        }
    )
    return df
