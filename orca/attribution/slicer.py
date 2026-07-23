from __future__ import annotations

from dataclasses import dataclass, field
from datetime import UTC

from orca.math.wilson import wilson_ci


@dataclass(frozen=True)
class SliceStats:
    n: int
    wins: int
    hit_rate: float
    hit_rate_ci_low: float
    hit_rate_ci_high: float
    total_pnl: float
    total_cost: float
    roi: float

    @property
    def sufficient_data(self) -> bool:
        return self.n >= 30

    @property
    def avg_win(self) -> float:
        return self.total_pnl / self.n if self.n > 0 else 0.0


@dataclass(frozen=True)
class AttributionReport:
    overall: SliceStats
    by_side: dict[str, SliceStats] = field(default_factory=dict)
    by_price_bucket: dict[str, SliceStats] = field(default_factory=dict)
    by_edge_bucket: dict[str, SliceStats] = field(default_factory=dict)
    generated_at: str = ""


def _compute_slice(trades_subset: list[dict]) -> SliceStats:
    n = len(trades_subset)
    if n == 0:
        ci_low, ci_high = 0.0, 1.0
    else:
        wins = sum(1 for t in trades_subset if t.get("pnl", 0) > 0)
        p = wins / n
        ci_low, ci_high = wilson_ci(wins, n)

    wins = sum(1 for t in trades_subset if t.get("pnl", 0) > 0)
    total_pnl = sum(float(t.get("pnl", 0)) for t in trades_subset)
    total_cost = sum(float(t.get("cost", 0)) for t in trades_subset)
    roi = (total_pnl / total_cost) if total_cost > 0 else 0.0

    return SliceStats(
        n=n,
        wins=wins,
        hit_rate=(wins / n) if n > 0 else 0.0,
        hit_rate_ci_low=ci_low,
        hit_rate_ci_high=ci_high,
        total_pnl=total_pnl,
        total_cost=total_cost,
        roi=roi,
    )


def attribute_pnl(trades: list[dict]) -> AttributionReport:
    """Multi-dimensional PnL attribution."""
    from datetime import datetime

    overall = _compute_slice(trades)

    by_side: dict[str, list[dict]] = {}
    by_price: dict[str, list[dict]] = {}
    by_edge: dict[str, list[dict]] = {}

    for t in trades:
        side = t.get("side", "unknown")
        by_side.setdefault(side, []).append(t)

        price = float(t.get("entry_price", 0))
        if price < 30:
            price_bucket = "0-30"
        elif price < 50:
            price_bucket = "30-50"
        elif price < 70:
            price_bucket = "50-70"
        else:
            price_bucket = "70+"
        by_price.setdefault(price_bucket, []).append(t)

        conf = float(t.get("confidence", 0))
        edge_pct = abs(conf - 0.5) * 100
        if edge_pct < 5:
            edge_bucket = "0-5%"
        elif edge_pct < 10:
            edge_bucket = "5-10%"
        elif edge_pct < 20:
            edge_bucket = "10-20%"
        else:
            edge_bucket = "20%+"
        by_edge.setdefault(edge_bucket, []).append(t)

    return AttributionReport(
        overall=overall,
        by_side={k: _compute_slice(v) for k, v in sorted(by_side.items())},
        by_price_bucket={k: _compute_slice(v) for k, v in sorted(by_price.items())},
        by_edge_bucket={k: _compute_slice(v) for k, v in sorted(by_edge.items())},
        generated_at=datetime.now(UTC).isoformat(),
    )
