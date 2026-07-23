---
name: market-scanner
description: Scan open prediction-market contracts for mispricings between your model's probability and the market mid, applying disagreement filters, side-specific edge thresholds, and extreme-price guards. Use when the user asks "scan markets", "find mispricings", "what should I bet on", "what are the opportunities", "show me edges", "what's actionable right now", or wants a list of candidate trades ranked by expected value. Also use as the front half of a trading loop that ends in sizing + maker pricing.
---

# Market scanner

Surface candidate trades from a universe of open prediction-market contracts. The scanner's job is to produce a *short list* of actionable opportunities — not to bet on everything that looks slightly mispriced.

## Why most scanners over-fire

A naive scanner with a single `|edge| > threshold` filter will produce dozens of bad trades for every good one. The reasons:

- **Crowd information at extreme prices** — a market at 2¢ usually reflects information the model isn't entitled to override. Most "edges" at the extremes are model errors, not market errors.
- **Model disagreement at moderate prices** — when multiple model sources disagree, the ensemble's mean is unreliable. Filtering on agreement removes the worst of these.
- **Asymmetric calibration** — many models are systematically biased on one side. A symmetric edge threshold lets the biased side through.
- **Stale prices** — markets that haven't traded in hours have stale midpoints. Edge against a stale mid is fictitious.

A good scanner applies several stacked filters that each remove a specific failure mode. The goal is *high precision*, not high recall — false positives are expensive (real money) and false negatives are cheap (you miss one bet of many similar opportunities).

## Standard filter stack

Apply in this order. Markets that fail any filter are dropped.

1. **Liquidity / freshness** — must have a current bid and ask, with the most recent trade or quote within a recent window (minutes, not hours). Stale books produce phantom edges.

2. **Extreme-price guard** — drop markets where the YES price is in the extreme bands at either end. The exact bounds are strategy-specific, but always have non-trivial bounds. The market's pricing at the extremes carries information your model can't safely override.

3. **Model disagreement** — if the model is an ensemble, compute the inter-model spread. Drop markets where the spread exceeds a threshold. High disagreement means the ensemble mean is uncertain enough that your edge estimate is unreliable.

4. **Edge threshold (asymmetric)** — compute `edge = model_p − ask` for YES and `edge = (1 − model_p) − (1 − bid)` for NO. Apply *separate* thresholds for YES and NO, because calibration is often asymmetric. A common pattern is a tighter threshold on the side the model is known to over-predict.

5. **Side-cost floor** — drop trades where the side-cost (price you pay per contract on the side you're taking) is below a floor. Tiny contract prices look like high-ROI bets but their hit rate must be commensurately tiny, which is hard for any model to estimate reliably.

6. **Maker pricing feasibility** — verify a one-tick-inside-the-spread quote is possible. If the spread is one tick wide or inverted, the only option is to take, which changes the trade's expected ROI net of fees.

After the filters, rank surviving candidates by expected ROI (after fees and adverse-selection haircut), not raw edge. Expected ROI accounts for the cost of the position; raw edge does not.

## Computing edge

For a YES trade at maker price `c_yes`:

```
expected_pnl_per_dollar = (model_p − c_yes) / c_yes − fee_pct − adverse_selection_pct
```

For a NO trade at maker price `c_no = 1 − bid_yes − tick`:

```
expected_pnl_per_dollar = ((1 − model_p) − c_no) / c_no − fee_pct − adverse_selection_pct
```

Use this for ranking. A trade with 10¢ raw edge at 5¢ contract price is not necessarily better than a trade with 8¢ edge at 50¢ — the former has 2x leverage but also 10x the path variance and is more sensitive to model error.

## How to respond

When the user asks for a scan:

1. Confirm the universe to scan (which venue, which categories, time bounds).
2. Confirm the filters and thresholds — or use the user's configured defaults.
3. Pull the open markets and the corresponding model outputs.
4. Apply the filter stack in order, recording the drop reason for each rejection (so the user can audit the funnel).
5. Rank survivors by expected ROI net of fees and adverse selection.
6. Output the top N candidates with: ticker, side, current book, model probability, edge, expected ROI, recommended maker quote (call out to `maker-pricing`), recommended Kelly size (call out to `kelly-sizer`).

If zero candidates survive, *report the funnel* — how many markets started, how many fell out at each stage. That's the diagnostic the user needs to know whether to relax filters or accept that the universe is quiet.

## Sizing handoff

A scanner is not a sizer. After identifying candidates, hand off to the sizing skill (`kelly-sizer`) to determine stake. Do not bypass that step — Kelly will catch cases where the edge looks fine but the side-cost makes the math unattractive.

## Reference template

A working scaffold is at `scripts/scanner_template.py`. It accepts a list of `Market` rows (ticker, bid, ask, last_trade_ts) plus a forecast function, applies the filter stack, and outputs ranked candidates. Adapt the data adapter to your venue.

## Don't

- Use a single symmetric `|edge| > X` filter. It's the single most common reason retail scanners over-fire.
- Skip the funnel diagnostics. When the user wonders why the scanner produced nothing, the answer is in the per-filter drop counts.
- Output more than ~10 candidates. If the scanner produces 50, the filters are too loose.
- Bypass extreme-price and stale-quote guards. They're cheap to compute and exist for empirical reasons.

## References

- Bartlett, R., O'Hara, M. (2026). *Adverse Selection in Event Contract Markets* (Kalshi case study).
- Wolfers, J., Zitzewitz, E. (2004). *Prediction Markets.*
