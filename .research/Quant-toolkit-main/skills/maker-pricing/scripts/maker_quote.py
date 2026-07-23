#!/usr/bin/env python3
"""Compute a maker quote one tick inside the touch.

Returns the limit price, the breakeven probability including fees, and
the recommended re-quote thresholds. Use this when placing a limit order
on a prediction market with a taker fee structure.

Usage:
    python maker_quote.py --side yes --bid 0.42 --ask 0.48
    python maker_quote.py --side no  --bid 0.42 --ask 0.48 --tick 0.01 --fee 0.0
"""
from __future__ import annotations

import argparse
import math
from dataclasses import dataclass


@dataclass
class MakerQuote:
    limit_price: float | None
    breakeven_prob: float | None
    fee_saved_vs_taker: float | None
    requote_lower: float | None
    requote_upper: float | None
    reason: str


def compute_maker_quote(
    side: str,
    bid: float,
    ask: float,
    tick: float = 0.01,
    taker_fee: float = 0.0,
    maker_fee: float = 0.0,
    adverse_selection_haircut: float = 0.02,
) -> MakerQuote:
    """Compute a maker limit one tick inside the touch.

    side: 'yes' or 'no'
    bid, ask: best YES bid/ask in [0, 1]
    tick: minimum price increment (typically 0.01)
    taker_fee, maker_fee: in dollars per contract
    adverse_selection_haircut: expected adverse selection cost per contract

    Returns a MakerQuote. If no maker quote is sensible (the spread is one
    tick wide or already inverted), returns None for limit_price with a reason.
    """
    if not 0 <= bid <= ask <= 1:
        return MakerQuote(None, None, None, None, None, f"invalid book: bid={bid} ask={ask}")

    spread = ask - bid
    if spread < 2 * tick - 1e-9:
        return MakerQuote(None, None, None, None, None,
                          f"spread {spread:.3f} too tight for maker quote (need ≥ {2*tick:.2f})")

    if side.lower() == "yes":
        limit = round(ask - tick, 4)
        # Breakeven probability accounts for: paying `limit` per contract, payoff 1 if YES wins.
        # p_break = limit + (fee + adverse_selection) / (1 - limit)
        cost = limit + maker_fee + adverse_selection_haircut
        breakeven = cost  # the break-even probability equals the cost in the unit-payoff binary case
        # Re-quote thresholds: if ask moves to ≤ limit, we may be filled; if it moves up, re-quote up.
        requote_upper = limit + tick
        requote_lower = bid
    elif side.lower() == "no":
        # Buying NO at 1 - limit_yes; equivalently quote selling YES at bid + tick
        sell_limit_yes = round(bid + tick, 4)
        limit = round(1 - sell_limit_yes, 4)  # NO contract price
        cost = limit + maker_fee + adverse_selection_haircut
        breakeven = cost
        requote_upper = round(1 - bid, 4)
        requote_lower = round(1 - (sell_limit_yes + tick), 4)
    else:
        return MakerQuote(None, None, None, None, None, f"invalid side: {side}")

    if not 0.01 <= limit <= 0.99:
        return MakerQuote(None, None, None, None, None,
                          f"limit {limit} outside [0.01, 0.99]")

    fee_saved = taker_fee - maker_fee
    return MakerQuote(
        limit_price=limit,
        breakeven_prob=round(breakeven, 4),
        fee_saved_vs_taker=round(fee_saved, 4),
        requote_lower=requote_lower,
        requote_upper=requote_upper,
        reason="ok",
    )


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--side", choices=["yes", "no"], required=True)
    parser.add_argument("--bid", type=float, required=True, help="Best YES bid (0-1)")
    parser.add_argument("--ask", type=float, required=True, help="Best YES ask (0-1)")
    parser.add_argument("--tick", type=float, default=0.01)
    parser.add_argument("--taker-fee", type=float, default=0.0)
    parser.add_argument("--maker-fee", type=float, default=0.0)
    parser.add_argument("--adverse-selection", type=float, default=0.02,
                        help="Adverse-selection haircut per contract (default 0.02)")
    args = parser.parse_args()

    q = compute_maker_quote(
        side=args.side,
        bid=args.bid,
        ask=args.ask,
        tick=args.tick,
        taker_fee=args.taker_fee,
        maker_fee=args.maker_fee,
        adverse_selection_haircut=args.adverse_selection,
    )

    if q.limit_price is None:
        print(f"  no maker quote: {q.reason}")
        return

    print(f"  side                {args.side}")
    print(f"  book                bid={args.bid} ask={args.ask}")
    print(f"  maker limit         {q.limit_price}")
    print(f"  breakeven prob      {q.breakeven_prob}  (need true p above this)")
    print(f"  fee saved/contract  ${q.fee_saved_vs_taker:.4f}")
    print(f"  re-quote if touch moves outside [{q.requote_lower}, {q.requote_upper}]")


if __name__ == "__main__":
    main()
