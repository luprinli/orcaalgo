---
name: maker-pricing
description: Compute a maker quote that pays the maker fee tier (often zero) instead of the taker fee while still being likely to fill. Use when the user asks "what's the maker price", "quote inside the spread", "should I be a maker or taker", "what limit price should I set", "how do I minimize fees on this trade", or is placing a limit order on a prediction market and wants to optimize the fill-vs-fee tradeoff. Also use when designing a strategy's order-placement layer.
---

# Maker pricing

Decide whether to quote as a maker (rest on the book inside the spread) or take liquidity (cross the spread). Most prediction-market venues charge a fee on takers and zero or a rebate on makers — so maker pricing is the difference between a strategy that's profitable and one that gets eaten by fees.

## Maker vs taker — when each makes sense

**Take liquidity (cross the spread) when:**

- Your edge is large enough that fees and slippage are noise — typically true when expected ROI on the position is well above 5%.
- The information you're trading on is decaying fast (a news event, a model update that other participants will quickly absorb).
- The market is so thin that the next print might be a long time coming.
- You're closing a position and willing to pay to flatten.

**Quote as a maker (rest inside the spread) when:**

- You're systematically harvesting small edges where fees are a meaningful fraction of expected return.
- The market is liquid enough that a quote one tick inside the touch will likely fill within your time horizon.
- You can re-quote when the book moves against you.

## The standard maker quote

The textbook quote is one tick inside the touch. For a YES contract priced in cents:

- If the best ask is 48¢, place a YES buy limit at 47¢.
- If the best bid is 42¢ and you want to sell YES (or buy NO), place a sell limit at 43¢ (equivalently, buy NO at 57¢).

This earns the maker fee (often zero) instead of the taker fee while preserving most of the fill probability of crossing.

Two practical refinements:

1. **Cap the limit at sane bounds.** Never quote at 0 or 100; clamp to a band like 1¢–99¢ to avoid stupid fills if the book moves.
2. **Don't cross.** If `best_ask − tick ≤ best_bid`, there's no inside-the-spread room — either pay to take or skip the trade.

## Adverse selection — the hidden cost of being a maker

Maker quotes are filled disproportionately when the market is moving against you. If the market is fair, your quote fills exactly when someone else's information is "your quote is too generous." This adverse-selection cost is real and is the reason maker rebates exist on most venues.

A reasonable rule of thumb on retail prediction markets: the adverse-selection cost is 1–3¢ on a 50¢ contract, depending on liquidity. Bake this into the edge calculation — your effective edge is `(model_p − ask) − tick − adverse_selection_haircut`, not `(model_p − mid)`.

## Re-quote logic

Resting maker orders go stale. The book will move. A simple discipline:

- After placing a maker order, watch the book.
- If the touch moves by ≥1 tick against you (e.g., your YES bid is at 47¢ and the best ask drops to 46¢), cancel and re-quote at the new touch − 1 tick.
- If the touch moves by ≥1 tick in your favor (e.g., the bid rises through your price), hold — you may have already filled or be about to.
- If your order has been resting for a fixed timeout without filling (say 5 minutes for a slow market), either cancel or upgrade to a more aggressive limit.

The point isn't to chase every tick — it's to avoid leaving stale quotes where the market has already revealed new information.

## Fee math

Always compute the breakeven fee budget *before* placing. For a stake `s` at fill price `c` per contract:

- Expected payoff if YES wins: `s × (1 − c) / c` minus any taker fee on settlement (usually zero).
- Expected loss if YES loses: `s`.
- Required win probability for breakeven: `p_break = c + fee/(1−c)`.

If the fee on entry is 1¢ on a 50¢ contract, your true breakeven probability is not 50%, it's 52%. Many "edges" disappear at this granularity.

## How to respond

When the user asks for a maker quote:

1. Confirm the current best bid and ask (or fetch them if a venue connector is available).
2. Compute the one-tick-inside price for the user's side.
3. Verify the quote doesn't cross.
4. Report the limit price plus the fee saved versus crossing.
5. Note the re-quote rule the user should apply if the book moves.

When the user asks whether to be a maker or taker:

1. Compute the user's expected edge.
2. Compute the maker-vs-taker fee delta.
3. Recommend the maker route if the edge is small (under ~5% expected ROI) and the market is liquid; recommend taker if the edge is large or the information is decaying.

## Reference script

A working quote helper is at `scripts/maker_quote.py`:

```
python scripts/maker_quote.py --side yes --bid 0.42 --ask 0.48
```

It returns the maker limit, the breakeven probability inclusive of fees, and the recommended re-quote thresholds.

## Don't

- Quote at the touch. That's a taker order in disguise — most venues will treat a marketable limit as a take and charge the taker fee.
- Skip the cross check. A limit that crosses the existing book auto-fills as a taker.
- Forget the adverse-selection cost. Maker fills look free; they aren't.
- Hold a stale quote through a meaningful move. You'll either get adversely selected or miss the trade entirely.

## References

- Avellaneda & Stoikov (2008). *High-Frequency Trading in a Limit Order Book.*
- Cont, Stoikov & Talreja (2010). *A Stochastic Model for Order Book Dynamics.*
