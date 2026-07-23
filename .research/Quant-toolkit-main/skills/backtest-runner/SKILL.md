---
name: backtest-runner
description: Run a historical simulation of a trading strategy on prediction-market data and report ROI, hit rate, Brier score, and per-cohort performance. Use when the user asks "run a backtest", "test this strategy on history", "see if the cap raise pencils out", "validate against historical prices", "would this rule have made money", or supplies a strategy spec + historical data and wants empirical performance numbers. Also use as the gate before promoting any code change that affects sizing, filters, or model weights to live trading.
---

# Backtest runner

Simulate a trading strategy on historical market data with realistic execution assumptions. The output should be honest enough that the user can trust it as a go/no-go signal for shipping a change to production.

## What "honest backtest" means

Most retail backtests overstate edge by 20–200%. The common ways they lie:

- **Look-ahead bias** — using information that wasn't available at decision time (e.g., the day's settlement price to filter trades).
- **Survivorship** — using only markets that resolved, dropping ones that got canceled or never settled.
- **Optimistic execution** — assuming you got filled at the midpoint when the realistic fill was at the offer.
- **Ignoring fees** — most prediction markets have taker fees that wipe out small edges.
- **Cherry-picked time windows** — testing on a regime that suited the strategy, ignoring others.
- **Free parameter tuning** — choosing thresholds *on the test data*, then reporting test performance.

A backtest is only useful if the user trusts it. Every shortcut above produces a number the user *shouldn't* trust. Bake the guardrails in by default.

## Methodology

Use walk-forward simulation:

1. For each historical date `d` in the test window, build the *exact* state of the world as of `d` — model outputs from data available by time `T_decision(d)`, market prices as of `T_decision(d)`, no settlement data from `d` or later.
2. Apply the strategy's filters to produce a candidate trade list at time `T_decision(d)`.
3. Apply sizing logic (Kelly + caps) using the bankroll as of the start of `d`.
4. Simulate execution at the realistic fill price for `T_decision(d)`:
   - If the user's strategy quotes as a maker, fill at one tick inside the ask (for YES) or one tick inside the bid (for NO). Assume *fill probability < 100%* unless the user has empirical evidence otherwise — see the dropoff section.
   - If the strategy is a taker, fill at the ask (YES) or 1−bid (NO).
5. Settle the trade using the actual outcome data and update bankroll.
6. Record the trade with all decision-time features so attribution is possible later.

Never use settlement information from date `d` to filter or size trades placed on date `d`. The only acceptable use of `d`-or-later data is computing the realized outcome at settlement time.

## Execution realism

The single biggest source of backtest inflation is execution assumptions. Calibrate them:

- **Maker fill probability** — quotes inside the spread don't always fill. A reasonable starting model is 60–80% fill rate at one tick inside the touch, dropping to 20–40% at two ticks. If you don't know, use 70% and note the sensitivity.
- **Adverse selection** — the trades that *do* fill are disproportionately the ones where the market moves against you. Apply a 2–5 bps adverse-selection cost to maker fills.
- **Commission / fees** — apply the *taker* fee even on maker fills until the user confirms their venue has a zero-fee maker tier.
- **Slippage on settlement** — settlement is at face value (0 or 1), so no slippage there. But if the strategy includes any early-close logic, model the spread at exit.

## What to report

A useful backtest report contains:

1. Headline ROI on capital deployed and net P&L.
2. Hit rate and average win / average loss.
3. Brier score on the model's predictions (separate from P&L — even a perfect model can lose money if the market is more accurate).
4. Per-cohort slices: by side (YES vs NO), by price bucket (e.g., 0–30¢, 30–50¢, 50–70¢, 70–100¢), by model-agreement level if applicable, by cohort (market category or category proxy).
5. Maximum drawdown and longest losing streak.
6. Trade count, with confidence intervals on ROI (Wilson or bootstrap).
7. Per-day exposure histogram — was the cap binding? On how many days?

Always report the n. A 5% ROI on 30 trades is noise; a 5% ROI on 3,000 trades is real.

## Statistical honesty

Treat the backtest like a hypothesis test:

- Pre-register the strategy and parameters before looking at the test period's results. If you tuned parameters on the test data, the result is biased.
- Compute a confidence interval. Wilson interval on hit rate, bootstrap on ROI. If the lower bound crosses zero, do not ship.
- Holdout discipline: keep the most recent 20–30% of data as a true out-of-sample window. Look at it once, at the end. If the strategy underperforms there, do not ship even if the in-sample numbers are good.

## How to respond

When the user asks for a backtest:

1. Confirm the strategy specification: filters, sizing rule, side handling, execution assumptions.
2. Confirm the data: source, time window, what's available at decision time vs. settlement.
3. Confirm the bankroll, the caps, and the multiplier.
4. Run the simulation (preferably as a one-shot script, not in-conversation).
5. Report the headline numbers, the slices, the CIs, and the verdict.
6. Be willing to say "no edge" if the data says so. Most backtested strategies don't ship.

## Reference template

The skeleton at `scripts/backtest_template.py` is a starting harness — it walks dates, applies filters, sizes positions, simulates fills, and writes a CSV with one row per trade. Adapt it to the user's specific strategy. The template is intentionally generic; do not assume a specific market structure or data source.

## Don't

- Run a backtest on data the user has already eyeballed.
- Report ROI without trade count and CI.
- Use the same data for training the model and testing the strategy.
- Hide the assumptions. If you guessed at fill probability, say so.

## References

- Bailey & López de Prado (2014). *The Deflated Sharpe Ratio.*
- López de Prado, M. (2018). *Advances in Financial Machine Learning.* Chapter on backtesting overfitting.
