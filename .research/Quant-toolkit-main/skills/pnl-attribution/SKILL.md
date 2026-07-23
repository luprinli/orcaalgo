---
name: pnl-attribution
description: Slice realized P&L by side, price bucket, model agreement, cohort, and time to find where the strategy is making money vs leaking it. Use when the user asks "where am I losing money", "slice my P&L", "P&L attribution", "are the new filters earning", "is the YES side back to negative", "kpi report", or supplies a trade ledger and wants a breakdown by decision dimension. Also use after any production change (filter tighten, sizing bump, model refit) to confirm the change is doing what it was supposed to do.
---

# P&L attribution

Break down realized P&L across the dimensions of a trading decision (side, price, model output, cohort, time) so the user can see *which slices* are earning and which are leaking. Aggregate P&L hides everything that matters.

## Why slice

A strategy with a flat aggregate P&L is almost never *uniformly flat*. It's the sum of profitable slices and unprofitable slices that happen to cancel. The unprofitable ones are tomorrow's improvement; you can only find them with attribution.

A useful slice has three properties:

1. It corresponds to a *decision the strategy makes* — filter threshold, side, sizing tier — so you can act on the finding.
2. It has enough trades to be statistically meaningful, typically n ≥ 30 per cell.
3. It's measurable from the trade ledger without joining new data.

## Standard slices

Always run at minimum:

- **By side** — YES vs NO. Asymmetric performance is common (models often systematically miscalibrate one tail).
- **By entry price bucket** — 0–30¢, 30–50¢, 50–70¢, 70–100¢. The strategy may be profitable in one bucket and a money-fire in another.
- **By cohort** — whatever categorical splits the data has (market category, ticker prefix, geographic region, day-of-week if relevant).
- **By model output / edge** — bucket trades by `|edge|` or by raw model probability and check ROI per bucket. Big-edge bets should at least *trend* better than small-edge ones; if they don't, the model is the problem.
- **By time period** — month-over-month or week-over-week. Catches regime shifts.

Two-way slices are where the real findings live. Side × price-bucket frequently reveals that the strategy is profitable on NO 50–69¢ but losing on YES 60–80¢, which is something you can act on. Always run the obvious two-ways.

## How to read an attribution report

For each slice, report:

1. **Trade count** — the n. Anything below 30 is a hint, not a verdict.
2. **Hit rate** — wins / total. Useful but doesn't capture stake-weighted reality.
3. **ROI on capital deployed** — net P&L / total cost. The headline number.
4. **Net P&L** — absolute dollars. Helps weigh relative importance.
5. **Wilson confidence interval** on the hit rate. If the interval brackets the breakeven probability (typically ~50%), the slice is noise.

Sort by net P&L for the "where is the money" view and by ROI for the "where is the leverage" view. The two answers often disagree — the biggest loss might be a high-volume slice with -2% ROI; the worst ROI might be a low-volume slice with -50%.

## Common findings and what to do

- **One side systematically negative** — apply asymmetric filters (different edge threshold or sizing multiplier per side). Or investigate model calibration on that side (see `calibration-audit`).
- **One price bucket negative** — restrict the strategy to bid outside that bucket, or apply a side-specific edge threshold within it.
- **Big-edge bets underperform small-edge bets** — the model is overconfident at the extremes. Either calibrate (Platt scaling, see `emos-bias-correction`) or cap the maximum edge the strategy will act on.
- **One cohort negative** — exclude the cohort, or apply a cohort-specific calibration layer.
- **Recent month worse than older months** — regime shift. Investigate before scaling up.

## Sample-size discipline

Attribution is the easiest way to fool yourself. Two common mistakes:

1. **Acting on n < 30 slices.** A 60% hit rate on 10 trades has a Wilson interval that easily covers 30%–80%. Resist the urge to call this an "edge" or a "leak."
2. **Multiple-comparison inflation.** If you slice 20 ways and pick the worst one, you've just performed an unadjusted significance test with 20 hypotheses. Use a Bonferroni-style cutoff or, more simply, only act on findings that are robust across multiple natural cuts (e.g., the leak is visible in side × price AND in side × cohort, not just one of them).

## When to re-run

Re-run attribution:

- After every production change.
- Every ~30 settled trades.
- Whenever the aggregate P&L moves significantly out of trend.
- Before any decision to bump trade size or relax filters.

## How to respond

When the user asks for attribution:

1. Confirm the ledger source (path to a SQLite or CSV with one row per settled trade).
2. Run the standard slices: side, price bucket, side × price bucket, edge bucket, cohort.
3. Report each with n, ROI, net P&L, hit rate, and CI.
4. Highlight the top three actionable findings (with n ≥ 30 for action; flag promising but underpowered slices as "watch").
5. Propose a concrete next step for each actionable finding — usually a filter change, a sizing change, or a calibration follow-up.

## Reference script

A working slicer is at `scripts/attribute.py`:

```
python scripts/attribute.py --db trades.sqlite --since 2026-01-01
```

It expects a `trades` table with columns including `side`, `fill_price`, `forecast_p`, `outcome`, `pnl`, `cohort`, `placed_at`. Adapt the schema map at the top of the file to your ledger.

## Don't

- Confuse slice ROI with strategy ROI. A 50%-ROI slice on 5% of capital deployed adds 2.5% to the strategy.
- Average ROI across slices unweighted. Cost-weight everything.
- Drop the n. The reader needs to know how much to trust each number.

## References

- López de Prado, M. (2018). *Advances in Financial Machine Learning.* Chapter on backtest attribution.
- Wilson, E.B. (1927). *Probable inference, the law of succession, and statistical inference.* (for the CI formula)
