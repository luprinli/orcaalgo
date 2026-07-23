"""Monte Carlo simulation for prop-firm pass probability estimation."""

from typing import Any

import numpy as np


def monte_carlo_pass_probability(
    trades: list[float],
    initial_capital: float = 100_000,
    daily_loss_pct: float = 5.0,
    max_drawdown_pct: float = 10.0,
    profit_target_pct: float = 10.0,
    n_simulations: int = 10_000,
) -> dict[str, Any]:
    """Shuffle trade sequence N times, estimate pass probability against prop-firm rules.

    Args:
        trades: List of per-trade returns (e.g., [0.01, -0.005, 0.02, ...])
        initial_capital: Starting account balance
        daily_loss_pct: Max daily loss as % of starting balance
        max_drawdown_pct: Max drawdown as % of peak balance
        profit_target_pct: Required return to pass the challenge
        n_simulations: Number of trade-sequence shuffles
    """
    if len(trades) < 10:
        return {"error": f"Too few trades: {len(trades)}, need at least 10"}

    trades = np.array(trades, dtype=np.float64)
    np.random.seed(42)

    passes = 0
    worst_drawdowns = []
    final_returns = []

    for _ in range(n_simulations):
        shuffled = np.random.permutation(trades)
        equity = initial_capital
        peak = initial_capital
        daily_pnl = 0.0
        violated = False

        for ret in shuffled:
            equity *= (1 + ret)
            daily_pnl += equity * ret

            if abs(daily_pnl) > initial_capital * daily_loss_pct / 100:
                violated = True
                break

            if equity > peak:
                peak = equity

            dd = (equity - peak) / peak * 100
            if abs(dd) > max_drawdown_pct:
                violated = True
                break

        if not violated:
            final_return = (equity - initial_capital) / initial_capital * 100
            if final_return >= profit_target_pct:
                passes += 1

        final_returns.append((equity - initial_capital) / initial_capital * 100)
        daily_peak = initial_capital
        for ret in shuffled:
            daily_peak = max(daily_peak, initial_capital * np.prod(1 + np.array([ret])))
        worst_drawdowns.append(min(0.0, (equity - daily_peak) / daily_peak * 100))

    final_returns = np.array(final_returns)
    worst_drawdowns = np.array(worst_drawdowns)

    return {
        "pass_probability": round(passes / n_simulations * 100, 1),
        "n_simulations": n_simulations,
        "n_trades": len(trades),
        "expected_return": round(float(np.mean(final_returns)), 2),
        "return_95ci": [round(float(np.percentile(final_returns, 2.5)), 2), round(float(np.percentile(final_returns, 97.5)), 2)],
        "worst_drawdown_mean": round(float(np.mean(worst_drawdowns)), 2),
        "worst_drawdown_95ci": [round(float(np.percentile(worst_drawdowns, 2.5)), 2), round(float(np.percentile(worst_drawdowns, 97.5)), 2)],
    }
