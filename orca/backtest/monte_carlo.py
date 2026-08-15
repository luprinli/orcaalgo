#!/usr/bin/env python3
"""
Monte Carlo Prop Firm Pass Probability Tool
============================================
Runs N simulated trading periods with randomly shuffled trade sequences
and the FTMO consistency rule applied.

Usage:
    python monte_carlo.py --trades 200 --simulations 1000 --monthly-target 10.0

Dependencies:
    pip install numpy
"""

import argparse
import json
import random

import numpy as np


def generate_random_trades(
    num_trades: int,
    rng: np.random.Generator,
    win_rate: float = 0.55,
    avg_win: float = 1.0,
    avg_loss: float = 0.8,
) -> list[float]:
    pnls = []
    for _ in range(num_trades):
        if random.random() < win_rate:
            pnls.append(abs(rng.normal(avg_win, avg_win * 0.3)))
        else:
            pnls.append(-abs(rng.normal(avg_loss, avg_loss * 0.3)))
    return pnls


def resample_actual_trades(
    actual_pnls: list[float], num_trades: int, block_len: int = 7
) -> list[float]:
    if len(actual_pnls) == 0:
        return []
    if num_trades <= 0:
        num_trades = len(actual_pnls)
    if block_len <= 1 or block_len > len(actual_pnls) // 4:
        block_len = 1
    n = len(actual_pnls)
    sampled = []
    idx = 0
    while idx < num_trades:
        start = random.randint(0, n - 1)
        for j in range(block_len):
            if idx >= num_trades:
                break
            sampled.append(actual_pnls[(start + j) % n])
            idx += 1
    return sampled


def apply_consistency_rule(
    daily_pnls: list[float],
    starting_balance: float,
    monthly_target_pct: float = 10.0,
    consistency_threshold_pct: float = 30.0,
) -> tuple[bool, float]:
    daily_limit_pct = 2.0
    max_drawdown_pct = 10.0

    balance = starting_balance
    peak = starting_balance

    for daily_pnl in daily_pnls:
        pnl_pct = (daily_pnl / starting_balance) * 100

        if abs(pnl_pct) > daily_limit_pct:
            return False, balance

        monthly_target = starting_balance * (monthly_target_pct / 100)
        if daily_pnl > monthly_target * (consistency_threshold_pct / 100):
            balance += daily_pnl * 0.5
        else:
            balance += daily_pnl

        if balance > peak:
            peak = balance

        drawdown = (peak - balance) / peak * 100
        if drawdown > max_drawdown_pct:
            return False, balance

    return True, balance


def run_simulation(
    num_trades: int,
    num_days: int,
    num_simulations: int,
    starting_balance: float,
    monthly_target_pct: float,
    consistency_threshold_pct: float,
    rng: np.random.Generator,
    actual_pnls: list[float] | None = None,
) -> dict:
    passes = 0
    final_balances = []

    for i in range(num_simulations):
        if actual_pnls and len(actual_pnls) > 0:
            trades = resample_actual_trades(actual_pnls, num_trades)
        else:
            trades = generate_random_trades(num_trades, rng)

        trades_per_day = max(1, num_trades // num_days)
        daily_pnls = []
        for d in range(num_days):
            start_idx = d * trades_per_day
            end_idx = min(start_idx + trades_per_day, num_trades)
            daily_pnl = sum(trades[start_idx:end_idx])
            daily_pnls.append(daily_pnl)

        passed, final_balance = apply_consistency_rule(
            daily_pnls, starting_balance, monthly_target_pct, consistency_threshold_pct
        )

        if passed:
            passes += 1
            final_balances.append(final_balance)

        if (i + 1) % 100 == 0:
            import sys

            print(
                f"  Simulation {i + 1}/{num_simulations} (pass rate: {passes / (i + 1) * 100:.1f}%)",
                file=sys.stderr,
                flush=True,
            )

    pass_probability = (passes / num_simulations) * 100
    avg_final_balance = np.mean(final_balances) if final_balances else 0
    median_final = np.median(final_balances) if final_balances else 0

    return {
        "simulations": num_simulations,
        "passes": passes,
        "pass_probability_pct": pass_probability,
        "avg_final_balance": float(avg_final_balance),
        "median_final_balance": float(median_final),
        "monthly_target_pct": monthly_target_pct,
        "consistency_threshold_pct": consistency_threshold_pct,
        "passed_threshold": pass_probability > 80.0,
    }


def main():
    parser = argparse.ArgumentParser(description="Monte Carlo prop firm pass probability")
    parser.add_argument("--trades", type=int, default=200, help="Total number of trades")
    parser.add_argument("--num-trades", type=int, default=None, help="Total number of trades (alt)")
    parser.add_argument("--days", type=int, default=20, help="Trading days in period")
    parser.add_argument("--simulations", type=int, default=1000, help="Number of Monte Carlo runs")
    parser.add_argument("--capital", type=float, default=100000.0, help="Starting balance")
    parser.add_argument(
        "--starting-balance", type=float, default=None, help="Starting balance (alt)"
    )
    parser.add_argument(
        "--monthly-target", type=float, default=10.0, help="Monthly profit target (pct)"
    )
    parser.add_argument(
        "--profit-target", type=float, default=None, help="Monthly profit target (alt)"
    )
    parser.add_argument(
        "--consistency-threshold",
        type=float,
        default=30.0,
        help="Consistency outlier threshold (pct)",
    )
    parser.add_argument(
        "--daily-loss-limit", type=float, default=5.0, help="Daily loss limit (pct)"
    )
    parser.add_argument("--max-drawdown", type=float, default=10.0, help="Max drawdown (pct)")
    parser.add_argument("--win-rate", type=float, default=0.55, help="Win rate (0-1)")
    parser.add_argument("--avg-win", type=float, default=1.0, help="Average win amount")
    parser.add_argument("--avg-loss", type=float, default=0.8, help="Average loss amount")
    parser.add_argument("--seed", type=int, default=42, help="Random seed")
    parser.add_argument("--json", action="store_true", help="Output JSON to stdout only")
    parser.add_argument(
        "--trade-pnls",
        type=str,
        default=None,
        help="JSON array of actual trade PnLs for resampling",
    )
    args = parser.parse_args()

    random.seed(args.seed)
    rng = np.random.default_rng(args.seed)

    num_trades = args.num_trades if args.num_trades is not None else args.trades
    capital = args.starting_balance if args.starting_balance is not None else args.capital
    monthly_target = args.profit_target if args.profit_target is not None else args.monthly_target

    actual_pnls = None
    if args.trade_pnls:
        try:
            actual_pnls = json.loads(args.trade_pnls)
        except (json.JSONDecodeError, ValueError):
            pass

    if args.win_rate > 1.0:
        args.win_rate /= 100.0

    if not args.json:
        print(f"Running {args.simulations:,} Monte Carlo simulations...")
        print(f"  Trades: {num_trades}, Days: {args.days}")
        print(f"  Capital: ${capital:,.0f}")
        print(f"  Monthly target: {monthly_target}%")
        print(f"  Consistency threshold: {args.consistency_threshold}%")
        if actual_pnls:
            print(f"  Using {len(actual_pnls)} actual trade PnLs for resampling")
        print()

    result = run_simulation(
        num_trades=num_trades,
        num_days=args.days,
        num_simulations=args.simulations,
        starting_balance=capital,
        monthly_target_pct=monthly_target,
        consistency_threshold_pct=args.consistency_threshold,
        rng=rng,
        actual_pnls=actual_pnls,
    )

    if args.json:
        print(
            json.dumps(
                {
                    "pass_probability": result["pass_probability_pct"],
                    "avg_final_balance": result["avg_final_balance"],
                    "median_final_balance": result["median_final_balance"],
                    "bust_probability": 100.0 - result["pass_probability_pct"],
                    "passes": result["passes"],
                    "simulations": args.simulations,
                }
            )
        )
        return

    print(f"\n{'=' * 50}")
    print("Monte Carlo Results:")
    print(f"  Pass probability:     {result['pass_probability_pct']:.1f}%")
    print(f"  Bust probability:     {100.0 - result['pass_probability_pct']:.1f}%")
    print(f"  Avg final balance:    ${result['avg_final_balance']:,.0f}")
    print(f"  Median final balance: ${result['median_final_balance']:,.0f}")
    print(f"  Threshold met (>80%): {result['passed_threshold']}")
    print(f"{'=' * 50}")

    if result["passed_threshold"]:
        print("\nPASS: Probability exceeds 80% threshold for prop firm qualification")
    else:
        print("\nFAIL: Probability below 80% — review strategy parameters")


if __name__ == "__main__":
    main()
