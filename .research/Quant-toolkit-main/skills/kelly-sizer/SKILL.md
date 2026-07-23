---
name: kelly-sizer
description: Compute optimal bet sizing on a prediction-market contract using the Kelly criterion. Use when the user asks "how much should I bet", "Kelly size this trade", "what's my optimal stake", "fractional Kelly", "is this position too big", or supplies a (win-probability, contract-price) pair and wants a recommended stake. Also use when sizing a portfolio of independent bets or deciding what Kelly multiplier to run at a given bankroll size.
---

# Kelly sizer

Compute the Kelly-optimal stake for a prediction-market position, then apply the standard practical adjustments (fractional multiplier, edge-uncertainty discount, hard caps).

## What Kelly answers — and what it doesn't

Kelly answers one question: given a known true win probability `p` and a known contract price `c`, what fraction of bankroll maximizes the long-run geometric growth rate of the portfolio? It does not answer:

- "Is my model right?" — that's a calibration question, not a sizing question.
- "Am I overexposed across correlated bets?" — Kelly applied per-bet under-prices correlation risk.
- "Can I survive the drawdown?" — full Kelly produces drawdowns most retail traders won't stomach.

Treat the output as an upper bound and apply a fractional multiplier.

## Formula

For a binary contract priced at `c ∈ (0, 1)` per unit payoff with true win probability `p`, the Kelly fraction of bankroll to stake is:

```
f* = (p - c) / (1 - c)        if betting YES  (long the contract)
f* = (q - (1-c)) / c          if betting NO   (short the contract)
                              where q = 1 - p
```

`f*` represents the *fraction of bankroll* to risk, not the dollar stake. The dollar stake is `f* × bankroll`.

If `f* ≤ 0`, there is no edge — do not bet. This is the most important check; an overconfident model produces phantom edges and full Kelly catastrophically over-bets them.

## Fractional Kelly — the only Kelly worth running

Full Kelly is mathematically optimal under three assumptions almost no retail trader satisfies: known true probability, no model risk, infinite-time horizon. Each violation argues for a smaller multiplier.

Standard practice is to run between ⅛ and ½ Kelly. Quarter Kelly is the most defensible default. It gives up roughly 25% of the optimal growth rate in exchange for cutting expected drawdown by more than half. The mathematical justification: fractional Kelly with multiplier `k` is equivalent to maximizing a negative-power utility, which encodes loss aversion an actual person actually has.

Recommend quarter Kelly unless the user has empirical evidence their model is exceptionally well-calibrated. Bias smaller (⅛) for new strategies, larger (½) only after months of validated calibration.

## Edge-uncertainty discount

The biggest practical risk is not that you're under-betting — it's that your edge estimate is wrong. Mean-error sensitivity is roughly 20× variance sensitivity in Kelly's growth-rate calculus. A 5-point overestimate in `p` is catastrophic; a 5-point underestimate in variance is benign.

Apply a discount: subtract a calibration-buffer from `p` before sizing. A conservative default is to use `p - 0.02` for well-validated models and `p - 0.05` for newer ones. If the user has a posterior distribution over `p`, use the lower decile.

## Caps and discipline

Kelly says nothing about per-trade dollar caps. Real-world constraints that should always apply on top:

- **Per-trade cap** — never risk more than a fixed dollar amount on a single contract, regardless of what Kelly recommends. Common defaults: 1–2% of bankroll.
- **Per-market cap** — limit concentration on a single event or correlated cluster of events.
- **Total exposure cap** — limit aggregate open exposure as a fraction of bankroll. 20–40% is a reasonable band.
- **Per-day cap** — limit the number of fills per day to keep variance manageable.

When Kelly recommends more than the cap, the cap wins. Report both numbers so the user sees the gap.

## How to respond

When the user asks for sizing:

1. Confirm the inputs you have: bankroll, win probability, contract price, side (YES or NO).
2. Compute raw Kelly `f*` and dollar stake.
3. Apply the user's chosen Kelly multiplier (default ¼).
4. Apply edge-uncertainty discount if the user hasn't already done so.
5. Apply caps and report the binding constraint.
6. State the recommended stake in both dollars and contract count.

If any input is missing, ask for it. Do not invent a default win probability.

## Reference script

A working CLI is included at `scripts/kelly.py`:

```
python scripts/kelly.py --p 0.62 --price 0.48 --bankroll 1000 --multiplier 0.25
```

It outputs the raw fraction, the discounted fraction, the dollar stake, and the binding cap if any. Read it before recommending — the math is the same as this skill describes, but the script handles NO-side, integer contract rounding, and the cap arithmetic.

## Don't

- Recommend full Kelly. Even if the user asks for it, push back; it's almost always wrong for retail bankrolls.
- Ignore correlation. If the user is sizing a contract that's correlated with an open position, halve the recommendation as a placeholder until they can compute joint Kelly properly.
- Round up. Always round contract counts down to enforce the cap.

## References

- Kelly, J.L. (1956). *A New Interpretation of Information Rate.* Bell System Technical Journal.
- Thorp, E.O. (2006). *The Kelly Criterion in Blackjack, Sports Betting, and the Stock Market.*
- MacLean, Thorp & Ziemba (2010). *The Kelly Capital Growth Investment Criterion: Theory and Practice.*
