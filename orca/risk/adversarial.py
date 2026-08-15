#!/usr/bin/env python3
"""
Adversarial News Simulation Tool
=================================
Injects adversarial news events into a backtest replay for STRESS TESTING.
THIS IS NOT A DATA SUBSTITUTE. Use real market data for backtesting.
See orca-fetch --source=stooq for production data loading.

Usage:
    python adversarial_sim.py --backtest-results results.json --news-events events.csv
    python adversarial_sim.py --generate-events --output test_events.csv

Dependencies:
    pip install pandas numpy
"""

import argparse
import csv
import json
import random
import sys
from datetime import datetime, timedelta

import pandas as pd

EVENT_TEMPLATES = [
    ("Fake: {symbol} earnings miss by 20%", "negative", 0.85),
    ("Fake: {symbol} CEO resigns amid scandal", "negative", 0.90),
    ("Fake: {symbol} receives unexpected upgrade", "positive", 0.75),
    ("Fake: Regulatory probe launched into {symbol}", "negative", 0.80),
    ("Fake: {symbol} announces massive buyback", "positive", 0.70),
    ("Fake: {symbol} data breach exposes millions", "negative", 0.95),
    ("Fake: {symbol} beats earnings by 15%", "positive", 0.65),
    ("Fake: Major competitor challenges {symbol} patent", "negative", 0.72),
]


def generate_events(
    symbols: list[str],
    start_date: str,
    end_date: str,
    num_events: int = 20,
    output: str = "adversarial_events.csv",
) -> None:
    start = datetime.fromisoformat(start_date)
    end = datetime.fromisoformat(end_date)
    total_days = (end - start).days

    events = []
    for _ in range(num_events):
        symbol = random.choice(symbols)
        template, sentiment, confidence = random.choice(EVENT_TEMPLATES)
        headline = template.format(symbol=symbol)

        day_offset = random.randint(0, total_days)
        event_time = start + timedelta(days=day_offset)
        event_time = event_time.replace(
            hour=random.randint(9, 15),
            minute=random.randint(0, 59),
        )

        corroborated = random.random() < 0.3  # 30% corroborated

        events.append(
            {
                "timestamp": event_time.isoformat(),
                "symbol": symbol,
                "headline": headline,
                "sentiment": sentiment,
                "sentiment_score": -1.0 if sentiment == "negative" else 1.0,
                "confidence": confidence,
                "was_corroborated": corroborated,
                "is_adversarial": True,
            }
        )

    events.sort(key=lambda e: str(e["timestamp"]))

    with open(output, "w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=events[0].keys())
        writer.writeheader()
        writer.writerows(events)

    print(f"Generated {len(events)} adversarial events → {output}")


def inject_events(
    backtest_results: str,
    events_file: str,
    guardrail_enabled: bool = True,
) -> dict:
    with open(backtest_results) as f:
        results = json.load(f)

    events_df = pd.read_csv(events_file)
    events_df["timestamp"] = pd.to_datetime(events_df["timestamp"])

    trades = results.get("trades", [])
    if not trades:
        print("No trades in backtest results; loading from equity curve")
        trades = results.get("equity_curve", [])

    rejected_count = 0
    total_impact = 0.0

    for _, event in events_df.iterrows():
        event_time = pd.Timestamp(event["timestamp"])

        relevant_trades = [
            t
            for t in trades
            if t.get("symbol", "") == event["symbol"]
            and abs(
                (pd.Timestamp(t.get("entry_time", t.get("time", ""))) - event_time).total_seconds()
            )
            < 3600
        ]

        if guardrail_enabled and not event["was_corroborated"]:
            rejected_count += len(relevant_trades)

        for t in relevant_trades:
            total_impact += t.get("PnL", 0) or t.get("pnl", 0)

    print("\nAdversarial Simulation Results:")
    print(f"  Total events:        {len(events_df)}")
    print(f"  Guardrail enabled:   {guardrail_enabled}")
    print(f"  Trades rejected:     {rejected_count}")
    print(f"  Total P&L impact:    ${total_impact:,.2f}")

    if guardrail_enabled:
        print(f"  Guardrail prevented {rejected_count} potentially harmful trades")

    return {
        "total_events": len(events_df),
        "guardrail_enabled": guardrail_enabled,
        "trades_rejected": rejected_count,
        "pnl_impact": total_impact,
    }


def main():
    parser = argparse.ArgumentParser(description="Adversarial news simulation for Orca Core")
    parser.add_argument("--generate-events", action="store_true", help="Generate synthetic events")
    parser.add_argument("--backtest-results", help="Path to backtest results JSON")
    parser.add_argument("--news-events", default="adversarial_events.csv", help="Events CSV file")
    parser.add_argument(
        "--output", default="adversarial_events.csv", help="Output for generated events"
    )
    parser.add_argument("--symbols", default="SPY,AAPL,MSFT,TSLA", help="Comma-separated symbols")
    parser.add_argument("--start", default="2024-01-01", help="Start date")
    parser.add_argument("--end", default="2025-12-31", help="End date")
    parser.add_argument("--num-events", type=int, default=30, help="Number of events to generate")
    parser.add_argument("--no-guardrail", action="store_true", help="Disable guardrail simulation")
    args = parser.parse_args()

    if args.generate_events:
        symbols = [s.strip() for s in args.symbols.split(",")]
        generate_events(symbols, args.start, args.end, args.num_events, args.output)
    elif args.backtest_results:
        inject_events(
            args.backtest_results,
            args.news_events,
            guardrail_enabled=not args.no_guardrail,
        )
    else:
        print("Either --generate-events or --backtest-results is required")
        sys.exit(1)


if __name__ == "__main__":
    main()
