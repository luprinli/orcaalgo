"""Block bootstrap Monte Carlo for strategy performance estimation.

Combines block resampling (preserving return autocorrelation) with Monte Carlo
simulation to produce robust performance confidence intervals.
"""

from __future__ import annotations

from typing import Any

import numpy as np


def block_bootstrap_monte_carlo(
    returns: np.ndarray,
    n_simulations: int = 2000,
    block_size: int = 20,
    initial_capital: float = 100000.0,
    seed: int | None = None,
) -> dict[str, Any]:
    """Run block-bootstrap Monte Carlo on a return series.

    Unlike IID bootstrap, block bootstrap preserves temporal dependencies
    (autocorrelation, volatility clustering) by resampling contiguous blocks
    of returns. This produces more realistic confidence intervals for
    auto-correlated return streams.

    Args:
        returns: Array of (log) returns. Shape: (n_periods,).
        n_simulations: Number of bootstrap simulations.
        block_size: Size of contiguous blocks to resample.
        initial_capital: Starting capital for equity simulation.
        seed: Random seed for reproducibility.

    Returns:
        Dict with {
            sharpe_ci: [lower, upper] 95% CI,
            max_drawdown_ci: [lower, upper] 95% CI,
            total_return_ci: [lower, upper] 95% CI,
            pass_probability: fraction of sims that end positive,
            simulated_sharpes: array of all simulated Sharpe ratios,
            simulated_drawdowns: array of all simulated max drawdowns,
        }
    """
    rng = np.random.default_rng(seed)
    n = len(returns)
    if n < block_size * 2:
        block_size = max(1, n // 3)

    n_blocks = n // block_size
    if n_blocks < 2:
        block_size = max(1, n // 2)
        n_blocks = n // block_size

    simulated_sharpes = np.zeros(n_simulations)
    simulated_drawdowns = np.zeros(n_simulations)
    simulated_returns = np.zeros(n_simulations)

    for sim in range(n_simulations):
        indices = rng.integers(0, n_blocks, size=n_blocks)
        sampled = np.concatenate([
            returns[i * block_size : min((i + 1) * block_size, n)]
            for i in indices
        ])
        if len(sampled) < 2:
            continue

        equity = initial_capital * np.exp(np.cumsum(sampled))

        sim_return = (equity[-1] / initial_capital - 1.0) * 100.0
        sim_sharpe = _compute_sharpe(sampled)
        sim_dd = _compute_max_drawdown(equity)

        simulated_returns[sim] = sim_return
        simulated_sharpes[sim] = sim_sharpe
        simulated_drawdowns[sim] = sim_dd

    alpha = 2.5
    return {
        "sharpe_ci": [
            float(np.percentile(simulated_sharpes, alpha)),
            float(np.percentile(simulated_sharpes, 100 - alpha)),
        ],
        "max_drawdown_ci": [
            float(np.percentile(simulated_drawdowns, alpha)),
            float(np.percentile(simulated_drawdowns, 100 - alpha)),
        ],
        "total_return_ci": [
            float(np.percentile(simulated_returns, alpha)),
            float(np.percentile(simulated_returns, 100 - alpha)),
        ],
        "pass_probability": float(np.mean(simulated_returns > 0)),
        "simulated_sharpes": simulated_sharpes.tolist(),
        "simulated_drawdowns": simulated_drawdowns.tolist(),
        "block_size": block_size,
        "n_simulations": n_simulations,
    }


def _compute_sharpe(returns: np.ndarray, periods_per_year: int = 252) -> float:
    """Compute annualized Sharpe ratio from return series."""
    if len(returns) < 2:
        return 0.0
    mean = np.mean(returns)
    std = np.std(returns, ddof=1)
    if std < 1e-12:
        return 0.0
    return float(mean / std * np.sqrt(periods_per_year))


def _compute_max_drawdown(equity: np.ndarray) -> float:
    """Compute maximum drawdown from equity curve."""
    peak = np.maximum.accumulate(equity)
    drawdown = (equity - peak) / peak
    return float(np.min(drawdown))
