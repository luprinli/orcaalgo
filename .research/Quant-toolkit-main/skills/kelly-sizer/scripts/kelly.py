#!/usr/bin/env python3
"""Kelly sizer for prediction-market contracts.

Computes the Kelly-optimal stake for a binary contract, applies a
fractional-Kelly multiplier and edge-uncertainty discount, then enforces
per-trade and total-exposure caps.

Usage:
    python kelly.py --p 0.62 --price 0.48 --bankroll 1000 --multiplier 0.25
    python kelly.py --p 0.30 --price 0.52 --bankroll 1000 --side no
"""
from __future__ import annotations

import argparse
import math
from dataclasses import dataclass


@dataclass
class KellyResult:
    raw_fraction: float          # f* before any adjustment
    discounted_fraction: float   # after edge-uncertainty haircut
    fractional_kelly: float      # after multiplier
    final_fraction: float        # after caps
    dollar_stake: float
    contract_count: int
    binding_cap: str | None      # which constraint bound the size (or None)


def kelly_fraction(p: float, price: float, side: str = "yes") -> float:
    """Kelly fraction for a binary contract priced at `price` per unit payoff.

    side='yes' bets the event occurs (long).
    side='no'  bets the event does not occur (short the YES contract).
    """
    if not 0 < price < 1:
        raise ValueError(f"price must be in (0, 1), got {price}")
    if not 0 <= p <= 1:
        raise ValueError(f"p must be in [0, 1], got {p}")

    if side.lower() == "yes":
        # payout = 1/price - 1 per dollar; net odds b = (1-price)/price
        return (p - price) / (1 - price)
    elif side.lower() == "no":
        q = 1 - p
        no_price = 1 - price
        return (q - no_price) / (1 - no_price)
    else:
        raise ValueError(f"side must be 'yes' or 'no', got {side}")


def size_position(
    p: float,
    price: float,
    bankroll: float,
    side: str = "yes",
    multiplier: float = 0.25,
    edge_discount: float = 0.02,
    per_trade_cap_pct: float = 0.02,
    total_exposure_cap_pct: float = 0.30,
    current_exposure: float = 0.0,
) -> KellyResult:
    """Compute Kelly-sized stake with practical guardrails.

    multiplier: fractional-Kelly multiplier (¼ Kelly = 0.25, default).
    edge_discount: subtract from p before sizing as calibration buffer.
    per_trade_cap_pct: max fraction of bankroll per individual trade.
    total_exposure_cap_pct: max aggregate exposure as fraction of bankroll.
    current_exposure: dollars currently in other open positions.
    """
    raw = kelly_fraction(p, price, side)

    # Apply edge-uncertainty discount: shrink p toward the contract price.
    p_discounted = max(p - edge_discount, 0.0) if side.lower() == "yes" else min(p + edge_discount, 1.0)
    discounted = kelly_fraction(p_discounted, price, side)
    discounted = max(discounted, 0.0)

    # Apply fractional-Kelly multiplier.
    frac_kelly = discounted * multiplier

    # Apply caps.
    final = frac_kelly
    binding = None

    if final > per_trade_cap_pct:
        final = per_trade_cap_pct
        binding = "per-trade cap"

    headroom_pct = max(total_exposure_cap_pct - (current_exposure / bankroll), 0.0)
    if final > headroom_pct:
        final = headroom_pct
        binding = "total-exposure cap"

    dollar_stake = final * bankroll
    contract_cost = price if side.lower() == "yes" else (1 - price)
    contract_count = math.floor(dollar_stake / contract_cost) if contract_cost > 0 else 0

    return KellyResult(
        raw_fraction=raw,
        discounted_fraction=discounted,
        fractional_kelly=frac_kelly,
        final_fraction=final,
        dollar_stake=round(contract_count * contract_cost, 2),
        contract_count=contract_count,
        binding_cap=binding,
    )


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--p", type=float, required=True, help="Your estimated win probability for the bet (0-1)")
    parser.add_argument("--price", type=float, required=True, help="YES contract price (0-1)")
    parser.add_argument("--bankroll", type=float, required=True, help="Total bankroll in dollars")
    parser.add_argument("--side", choices=["yes", "no"], default="yes")
    parser.add_argument("--multiplier", type=float, default=0.25, help="Fractional-Kelly multiplier (default 0.25)")
    parser.add_argument("--edge-discount", type=float, default=0.02, help="Edge-uncertainty haircut on p (default 0.02)")
    parser.add_argument("--per-trade-cap-pct", type=float, default=0.02, help="Max fraction of bankroll per trade (default 0.02)")
    parser.add_argument("--total-exposure-cap-pct", type=float, default=0.30, help="Max aggregate exposure as fraction of bankroll (default 0.30)")
    parser.add_argument("--current-exposure", type=float, default=0.0, help="Dollars in other open positions (default 0)")
    args = parser.parse_args()

    result = size_position(
        p=args.p,
        price=args.price,
        bankroll=args.bankroll,
        side=args.side,
        multiplier=args.multiplier,
        edge_discount=args.edge_discount,
        per_trade_cap_pct=args.per_trade_cap_pct,
        total_exposure_cap_pct=args.total_exposure_cap_pct,
        current_exposure=args.current_exposure,
    )

    print(f"  Raw Kelly fraction:           {result.raw_fraction:+.4f}")
    print(f"  After edge discount:          {result.discounted_fraction:+.4f}")
    print(f"  After {args.multiplier:.2f} multiplier:    {result.fractional_kelly:+.4f}")
    print(f"  Final fraction (post-caps):   {result.final_fraction:+.4f}")
    print(f"  Dollar stake:                 ${result.dollar_stake:.2f}")
    print(f"  Contract count:               {result.contract_count}")
    if result.binding_cap:
        print(f"  Binding constraint:           {result.binding_cap}")
    if result.raw_fraction <= 0:
        print("\n  No edge detected — do not bet.")


if __name__ == "__main__":
    main()
