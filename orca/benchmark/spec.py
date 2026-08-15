"""Benchmark specification models (frozen, validated — HP #7).

Defines the declarative benchmark choice that is folded into a strategy's GKR
config hash (HP #3), so the benchmark is fixed *before* results are seen and
cannot be tuned post-hoc.
"""

from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, ConfigDict, Field, model_validator

BenchmarkKind = Literal[
    "equity_index", "growth_index", "sector_etf", "buy_hold", "risk_free", "custom"
]

DEFAULT_TICKER: dict[str, str] = {
    "equity_index": "SPY",
    "growth_index": "QQQ",
}


class BenchmarkThresholds(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    information_ratio: float = Field(default=0.4)
    alpha: float = Field(default=0.0)
    # Threshold for the *deflated* active Sharpe (DSR of the active-return
    # series). 0 means "must beat the benchmark after selection-bias deflation".
    active_sharpe: float = Field(default=0.0)


class BenchmarkSpec(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    kind: BenchmarkKind = "equity_index"
    ticker: str | None = None
    tickers: list[str] = Field(default_factory=list)
    weights: list[float] = Field(default_factory=list)
    rebalance: str = "1d"
    risk_free_hurdle: float = Field(default=0.0, ge=0.0)
    thresholds: BenchmarkThresholds = Field(default_factory=BenchmarkThresholds)

    @model_validator(mode="after")
    def _validate(self) -> BenchmarkSpec:
        if self.kind == "custom":
            if not self.tickers:
                raise ValueError("custom benchmark requires 'tickers'")
            if self.weights:
                if len(self.weights) != len(self.tickers):
                    raise ValueError("'weights' length must match 'tickers' length")
                if abs(sum(self.weights) - 1.0) > 1e-6:
                    raise ValueError("'weights' must sum to 1.0")
        elif self.kind == "sector_etf":
            if not self.ticker:
                raise ValueError("sector_etf benchmark requires 'ticker'")
        elif self.kind in DEFAULT_TICKER and not self.ticker:
            # Frozen models can't self-mutate; normalize via object.__setattr__.
            object.__setattr__(self, "ticker", DEFAULT_TICKER[self.kind])
        return self

    def resolved_tickers(self) -> list[str]:
        """Tickers that make up this benchmark (single-element for single-ticker kinds)."""
        if self.kind == "custom":
            return list(self.tickers)
        if self.ticker:
            return [self.ticker]
        return []

    def resolved_weights(self) -> list[float]:
        """Weights aligned with ``resolved_tickers`` (equal-weight when unset)."""
        tickers = self.resolved_tickers()
        if not tickers:
            return []
        if self.kind == "custom" and self.weights:
            return list(self.weights)
        w = 1.0 / len(tickers)
        return [w] * len(tickers)


__all__ = ["DEFAULT_TICKER", "BenchmarkKind", "BenchmarkSpec", "BenchmarkThresholds"]
