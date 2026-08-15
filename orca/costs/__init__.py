"""Transaction-cost calibration primitives (R2).

Spread and market-impact estimators computed from OHLCV candles. These feed the
``SlippageModel`` coefficients (``SpreadBps`` / ``VolumeImpactFactor``) in
``internal/backtest/slippage.go`` so that backtest costs are calibrated from data
rather than hand-set constants (HP #9).
"""

from __future__ import annotations

from orca.costs.impact import fit_sqrt_impact, kyle_lambda
from orca.costs.spread import corwin_schultz, roll_spread

__all__ = ["corwin_schultz", "fit_sqrt_impact", "kyle_lambda", "roll_spread"]
