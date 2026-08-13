"""Anti-overfit scoring for backtest parameter and template selection.

This package ports the layered anti-overfit methodology that prevents
false-positive strategy promotion. It complements the existing walk-forward and
multiple-testing gates (orca/sizing) with the cross-sectional and plateau-focused
checks that those gates do not cover:

- ``ticker_split``    — deterministic SHA-256 train/validation ticker split
  (cross-sectional out-of-sample).
- ``param_score``     — per-parameter-set scoring: percentile-rank core,
  exponential drawdown penalty, neighbourhood stability (plateau preference),
  and a train/validation balance penalty.
- ``template_score``  — whole strategy-family ranking across multiple periods
  with a verification multiplier.

The scoring functions are pure: they consume plain dict rows and frozen settings
objects, so they are unit-testable without a database or API dependency.
"""

from orca.scoring.param_score import (
    ParamScoreSettings,
    score_backtest_parameters,
)
from orca.scoring.template_score import (
    TemplateScoreSettings,
    compute_template_scores,
)
from orca.scoring.ticker_split import (
    is_training_ticker,
    split_tickers,
)

__all__ = [
    "ParamScoreSettings",
    "TemplateScoreSettings",
    "compute_template_scores",
    "is_training_ticker",
    "score_backtest_parameters",
    "split_tickers",
]
