#!/usr/bin/env python3
"""Market scanner template.

Applies a stacked filter pipeline to a candidate universe and outputs
ranked actionable opportunities. Adapt the `forecast`, `model_spread`,
and data-loading functions to your strategy.

Filter stack (in order; first failure drops the market):
    1. liquidity / freshness
    2. extreme-price guard
    3. model disagreement
    4. asymmetric edge threshold (YES / NO)
    5. side-cost floor
    6. maker-pricing feasibility

Usage:
    python scanner_template.py  # uses the example synthetic universe
"""
from __future__ import annotations

import argparse
from dataclasses import dataclass, field
from datetime import datetime, timedelta
from typing import Callable


@dataclass
class Market:
    ticker: str
    bid: float                 # YES bid
    ask: float                 # YES ask
    last_trade_ts: datetime    # last trade timestamp
    metadata: dict = field(default_factory=dict)  # anything else you want (category, cohort, etc.)


@dataclass
class Candidate:
    market: Market
    side: str
    fill_price: float
    model_p: float
    edge: float
    expected_roi: float
    reason: str = "ok"


@dataclass
class FilterConfig:
    min_freshness_minutes: int = 30
    yes_price_lo: float = 0.05
    yes_price_hi: float = 0.95
    max_model_spread: float = 0.15
    min_edge_yes: float = 0.08
    min_edge_no: float = 0.06
    side_cost_floor: float = 0.30
    tick: float = 0.01
    fee_pct: float = 0.0
    adverse_selection_pct: float = 0.02


def scan(
    universe: list[Market],
    forecast: Callable[[Market], float],
    model_spread: Callable[[Market], float],
    config: FilterConfig,
    now: datetime,
) -> tuple[list[Candidate], dict[str, int]]:
    """Run the filter stack. Returns (ranked candidates, drop reasons)."""
    drops: dict[str, int] = {}
    surviving: list[Candidate] = []

    def drop(reason: str) -> None:
        drops[reason] = drops.get(reason, 0) + 1

    for m in universe:
        # 1. liquidity / freshness
        if now - m.last_trade_ts > timedelta(minutes=config.min_freshness_minutes):
            drop("stale")
            continue

        # 2. extreme-price guard
        if m.ask < config.yes_price_lo or m.bid > config.yes_price_hi:
            drop("extreme-price")
            continue
        if not (config.yes_price_lo <= (m.bid + m.ask) / 2 <= config.yes_price_hi):
            drop("extreme-price")
            continue

        # 3. model disagreement
        spread = model_spread(m)
        if spread > config.max_model_spread:
            drop("model-disagreement")
            continue

        p = forecast(m)

        # Decide side based on which has positive edge
        yes_maker = round(m.ask - config.tick, 4)
        no_maker = round((1 - m.bid) - config.tick, 4)

        edge_yes = p - yes_maker if 0 < yes_maker < 1 else -1
        edge_no = (1 - p) - no_maker if 0 < no_maker < 1 else -1

        # 4. asymmetric edge threshold
        if edge_yes >= config.min_edge_yes and edge_yes >= edge_no:
            side, fill_price, edge = "yes", yes_maker, edge_yes
        elif edge_no >= config.min_edge_no:
            side, fill_price, edge = "no", no_maker, edge_no
        else:
            drop("edge-below-threshold")
            continue

        # 5. side-cost floor
        if fill_price < config.side_cost_floor:
            drop("side-cost-floor")
            continue

        # 6. maker-pricing feasibility
        if (m.ask - m.bid) < 2 * config.tick - 1e-9:
            drop("spread-too-tight")
            continue
        if not 0.01 <= fill_price <= 0.99:
            drop("limit-out-of-band")
            continue

        # Compute expected ROI net of fees
        raw_roi = edge / fill_price
        net_roi = raw_roi - config.fee_pct - config.adverse_selection_pct
        if net_roi <= 0:
            drop("net-roi-negative")
            continue

        surviving.append(Candidate(
            market=m, side=side, fill_price=fill_price,
            model_p=p, edge=edge, expected_roi=net_roi,
        ))

    surviving.sort(key=lambda c: c.expected_roi, reverse=True)
    return surviving, drops


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--top", type=int, default=10, help="Show top N candidates")
    args = parser.parse_args()

    now = datetime.now()
    universe = [
        Market("DEMO-A", 0.40, 0.42, now - timedelta(minutes=5), {"cohort": "demo"}),
        Market("DEMO-B", 0.55, 0.58, now - timedelta(minutes=10), {"cohort": "demo"}),
        Market("DEMO-C", 0.02, 0.04, now - timedelta(minutes=2), {"cohort": "demo"}),
        Market("DEMO-D", 0.45, 0.47, now - timedelta(hours=3), {"cohort": "demo"}),
    ]
    def fake_forecast(m: Market) -> float:
        return {"DEMO-A": 0.55, "DEMO-B": 0.50, "DEMO-C": 0.20, "DEMO-D": 0.52}[m.ticker]
    def fake_spread(m: Market) -> float:
        return {"DEMO-A": 0.05, "DEMO-B": 0.08, "DEMO-C": 0.20, "DEMO-D": 0.04}[m.ticker]

    cands, drops = scan(universe, fake_forecast, fake_spread, FilterConfig(), now)

    print(f"\n  funnel:")
    for reason, n in sorted(drops.items(), key=lambda kv: -kv[1]):
        print(f"    {reason:<25} {n:>3}")
    print(f"    surviving:                {len(cands):>3}")

    print(f"\n  top {args.top} candidates:")
    print(f"  {'ticker':<10} {'side':<5} {'fill':>6} {'model_p':>8} {'edge':>8} {'roi':>8}")
    for c in cands[:args.top]:
        print(f"  {c.market.ticker:<10} {c.side:<5} {c.fill_price:>6.3f} {c.model_p:>8.3f} {c.edge:>+8.3f} {c.expected_roi*100:>+7.2f}%")


if __name__ == "__main__":
    main()
