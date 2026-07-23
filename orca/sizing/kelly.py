from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class KellyResult:
    raw_kelly: float
    discounted_p: float
    fractional_kelly: float
    per_trade_cap: float
    exposure_limit: float
    final_allocation: float
    contracts: int


def kelly_fraction_binary(p: float, price: float, side: str = "yes") -> float:
    if not 0 < price < 1:
        raise ValueError(f"Price must be in (0, 1), got {price}")
    if not 0 <= p <= 1:
        raise ValueError(f"Probability must be in [0, 1], got {p}")

    if side.lower() == "yes":
        return (p - price) / (1 - price)
    elif side.lower() == "no":
        q = 1 - p
        no_price = 1 - price
        return (q - no_price) / (1 - no_price)
    else:
        raise ValueError(f"Side must be 'yes' or 'no', got '{side}'")


def kelly_fraction_continuous(p_win: float, win_loss_ratio: float = 1.0) -> float:
    if not 0 <= p_win <= 1:
        raise ValueError(f"p_win must be in [0, 1], got {p_win}")
    if win_loss_ratio <= 0:
        raise ValueError(f"win_loss_ratio must be positive, got {win_loss_ratio}")
    return (p_win * win_loss_ratio - (1 - p_win)) / win_loss_ratio


def kelly_with_attenuators(
    p: float,
    price: float,
    side: str = "yes",
    multiplier: float = 0.25,
    edge_discount: float = 0.02,
    per_trade_cap_pct: float = 0.02,
    total_exposure_cap_pct: float = 0.30,
    current_exposure_pct: float = 0.0,
) -> KellyResult:
    side_str = str(side).lower()
    if side_str == "yes":
        p_discounted = max(p - edge_discount, 0.0)
    elif side_str == "no":
        p_discounted = min(p + edge_discount, 1.0)

    raw = kelly_fraction_binary(p_discounted, price, side)
    fractional = raw * multiplier

    per_trade_limit = min(fractional, per_trade_cap_pct)
    headroom = max(total_exposure_cap_pct - current_exposure_pct, 0.0)
    exposure_limit = min(per_trade_limit, headroom)
    final_alloc = max(0.0, exposure_limit)

    return KellyResult(
        raw_kelly=raw,
        discounted_p=p_discounted,
        fractional_kelly=fractional,
        per_trade_cap=per_trade_limit,
        exposure_limit=exposure_limit,
        final_allocation=final_alloc,
        contracts=0,
    )
