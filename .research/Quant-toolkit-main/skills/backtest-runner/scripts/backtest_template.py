#!/usr/bin/env python3
"""Walk-forward backtest harness for a prediction-market strategy.

This is a TEMPLATE — adapt the four customization points below to your
specific market structure, model, and sizing logic. The harness handles
the walk-forward bookkeeping, fill simulation, and trade logging so you
can focus on the strategy.

Customization points:
    1. load_universe(date)         — return candidate markets active on date
    2. forecast(market, date)      — return your model's probability for market on date
    3. should_trade(market, p)     — return (do_trade, side) given price + probability
    4. settle(market, trade_date)  — return realized outcome (0 or 1)

Usage:
    python backtest_template.py --start 2026-01-01 --end 2026-04-30 --bankroll 1000
"""
from __future__ import annotations

import argparse
import csv
import math
from dataclasses import dataclass, asdict
from datetime import date, timedelta


# ---------- customization points ----------------------------------------------

@dataclass
class Market:
    ticker: str
    bid: float          # YES bid in [0, 1]
    ask: float          # YES ask in [0, 1]
    # Add any other state your strategy needs at decision time.


def load_universe(d: date) -> list[Market]:
    """Return candidate markets active on date d. Must NOT use data from d's settlement."""
    raise NotImplementedError("Implement load_universe for your data source.")


def forecast(market: Market, d: date) -> float:
    """Return your model's win probability for `market` as of decision time on d."""
    raise NotImplementedError("Implement forecast using only pre-d information.")


def should_trade(market: Market, p: float) -> tuple[bool, str]:
    """Decide whether to trade. Return (trade?, side) where side is 'yes' or 'no'."""
    raise NotImplementedError("Implement should_trade with your filters.")


def settle(market: Market, trade_date: date) -> int:
    """Return realized outcome (0 = NO, 1 = YES). May use data from settlement time."""
    raise NotImplementedError("Implement settle using actual outcomes.")


# ---------- harness (no customization required below) -------------------------

@dataclass
class TradeRow:
    date: str
    ticker: str
    side: str
    forecast_p: float
    fill_price: float
    contracts: int
    cost: float
    outcome: int
    pnl: float
    bankroll_after: float


def maker_fill(side: str, market: Market, fill_prob_at_one_tick: float = 0.7) -> tuple[float | None, float]:
    """Simulate maker fill one tick inside the touch.

    Returns (fill_price_per_contract, fill_probability).
    Price is in dollars-per-contract for the side being bought.
    """
    tick = 0.01
    if side == "yes":
        target = round(market.ask - tick, 2)
        if target <= market.bid:
            return None, 0.0  # would cross — treat as no maker available
        return target, fill_prob_at_one_tick
    else:  # no
        target = round((1 - market.bid) - tick, 2)
        no_ask = 1 - market.bid
        no_bid = 1 - market.ask
        if target <= no_bid:
            return None, 0.0
        return target, fill_prob_at_one_tick


def kelly_contracts(p: float, fill_price: float, bankroll: float, multiplier: float = 0.25, cap_pct: float = 0.02) -> int:
    if fill_price <= 0 or fill_price >= 1:
        return 0
    edge = (p - fill_price) / (1 - fill_price)
    if edge <= 0:
        return 0
    edge_after_mult = edge * multiplier
    edge_after_cap = min(edge_after_mult, cap_pct)
    dollars = edge_after_cap * bankroll
    return math.floor(dollars / fill_price)


def run(start: date, end: date, bankroll: float, *, multiplier: float = 0.25, fill_prob: float = 0.7) -> list[TradeRow]:
    rows: list[TradeRow] = []
    cur = start
    day = timedelta(days=1)
    while cur <= end:
        universe = load_universe(cur)
        for m in universe:
            p = forecast(m, cur)
            ok, side = should_trade(m, p)
            if not ok:
                continue
            fill_price, fill_p = maker_fill(side, m, fill_prob_at_one_tick=fill_prob)
            if fill_price is None:
                continue
            # Simulate fill probability with deterministic-bookkeeping approach:
            # multiply position size by fill_p instead of randomizing. This produces
            # a smoothed expected-value backtest. Use random simulation for
            # variance estimates instead.
            n = kelly_contracts(p, fill_price, bankroll, multiplier=multiplier)
            n_eff = int(round(n * fill_p))
            if n_eff <= 0:
                continue
            cost = n_eff * fill_price
            if cost > bankroll:
                continue
            outcome = settle(m, cur)
            payoff = n_eff * (1 if (outcome == 1 and side == "yes") or (outcome == 0 and side == "no") else 0)
            pnl = payoff - cost
            bankroll += pnl
            rows.append(TradeRow(
                date=cur.isoformat(),
                ticker=m.ticker,
                side=side,
                forecast_p=p,
                fill_price=fill_price,
                contracts=n_eff,
                cost=cost,
                outcome=outcome,
                pnl=pnl,
                bankroll_after=round(bankroll, 2),
            ))
        cur += day
    return rows


def summarize(rows: list[TradeRow], starting_bankroll: float) -> None:
    if not rows:
        print("no trades")
        return
    final = rows[-1].bankroll_after
    pnl = final - starting_bankroll
    wins = sum(1 for r in rows if r.pnl > 0)
    capital_deployed = sum(r.cost for r in rows)
    print(f"  trades             {len(rows)}")
    print(f"  starting bankroll  ${starting_bankroll:.2f}")
    print(f"  final bankroll     ${final:.2f}")
    print(f"  net P&L            ${pnl:+.2f}")
    print(f"  capital deployed   ${capital_deployed:.2f}")
    print(f"  ROI on deployed    {(pnl / capital_deployed * 100):+.2f}%" if capital_deployed > 0 else "  ROI                n/a")
    print(f"  hit rate           {wins}/{len(rows)} = {wins/len(rows)*100:.1f}%")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--start", required=True, help="YYYY-MM-DD")
    parser.add_argument("--end", required=True, help="YYYY-MM-DD")
    parser.add_argument("--bankroll", type=float, default=1000.0)
    parser.add_argument("--multiplier", type=float, default=0.25)
    parser.add_argument("--fill-prob", type=float, default=0.7)
    parser.add_argument("--out", default="backtest_trades.csv")
    args = parser.parse_args()

    start = date.fromisoformat(args.start)
    end = date.fromisoformat(args.end)
    rows = run(start, end, args.bankroll, multiplier=args.multiplier, fill_prob=args.fill_prob)

    with open(args.out, "w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=list(TradeRow.__dataclass_fields__.keys()))
        writer.writeheader()
        for r in rows:
            writer.writerow(asdict(r))

    summarize(rows, args.bankroll)
    print(f"\n  trades written to {args.out}")


if __name__ == "__main__":
    main()
