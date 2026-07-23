---
name: drawdown-monitor
description: Monitor a strategy's running drawdown and halt or de-risk when it exceeds configured thresholds. Use when the user asks "what's my drawdown", "set up drawdown halt", "should I stop trading", "DD monitor", "max drawdown", "circuit breaker", or has a live trading process and wants automatic risk-off behavior. Also use to compute the historical maximum drawdown of a backtest before scaling up live size.
---

# Drawdown monitor

Track running drawdown against high-water-mark and trigger one of three responses (warn, de-risk, halt) when configured thresholds are crossed. Drawdown is the most operationally useful risk metric for a live bot — it's the thing that ends careers when it gets large, and the easiest to monitor in real time.

## Definitions

**High-water-mark (HWM)** — the maximum bankroll value observed since the strategy started (or since the last reset).

**Current drawdown (DD)** — fractional loss from HWM:

```
DD = (HWM − current_bankroll) / HWM
```

DD is non-negative and equals zero at any new HWM. Always reported as a positive number (e.g., "DD 12%" means down 12% from HWM).

**Maximum drawdown (MDD)** — the largest DD value observed over the strategy's history. Backward-looking statistic.

**Drawdown duration** — number of days the strategy has been below HWM. A shallow but long drawdown can be more demoralizing than a deep-fast one even if MDD is smaller.

## Why drawdown matters more than P&L volatility

Volatility (variance of returns) doesn't tell you what it feels like to be in the drawdown. A strategy with low daily variance but a one-time 30% loss is operationally worse than one with high daily variance and a 15% max loss. People close out positions during drawdowns; the drawdown itself, not the variance, is what produces capitulation.

Practical implication: budget drawdown explicitly. A strategy you can stomach a 25% drawdown on is a different strategy than one you can stomach 5%.

## The three thresholds

A useful monitor has three levels, escalating in severity:

### Warn (e.g., DD ≥ 5%)

- Notify the user. Do not change behavior.
- Increase logging detail to make post-mortem easier.
- Force a P&L attribution run to surface what's leaking.

This is informational. Many strategies dip into a 5% drawdown routinely; the alert is to make sure you notice if it's about to get worse.

### De-risk (e.g., DD ≥ 10%)

- Cut the Kelly multiplier by half (e.g., from ¼ to ⅛).
- Tighten edge thresholds by some fraction (e.g., add 2 percentage points to the minimum edge).
- Continue trading but at reduced size.
- Notify the user, ask for acknowledgement to continue.

The point of de-risking before halting is to ride out drawdowns that are within the strategy's expected envelope without giving back unrealized P&L on the way down.

### Halt (e.g., DD ≥ 20%)

- Stop placing new orders.
- Cancel resting maker quotes.
- Let existing positions ride to settlement (forced flattening at a bad time is usually worse than waiting).
- Require explicit user re-arm before trading resumes.

Halt is the circuit breaker. It exists to prevent a single bad week from compounding into account-ending losses while you investigate.

## Setting the thresholds

The right thresholds depend on the strategy's expected drawdown envelope, which you should know from the backtest:

- Look at the backtest's historical MDD. Multiply by ~1.5x to account for the fact that out-of-sample is typically worse than in-sample.
- Set `halt` at that level or modestly below it. You want to halt before you exceed the backtest envelope, not after.
- Set `de-risk` at roughly 50% of `halt`.
- Set `warn` at roughly 25% of `halt`.

For a strategy with backtested MDD of 15%: `warn` at 5%, `de-risk` at 10%, `halt` at 20%.

If the backtest didn't include a regime stress test, halve all thresholds. Untested strategies should halt sooner.

## Daily vs. rolling

There are several windows you can compute DD over:

- **All-time** — from inception. The most conservative; once you've had a deep drawdown, it stays as your reference forever.
- **Trailing 90-day** — resets HWM if no new high in 90 days. More forgiving; lets the strategy "reset" after a structural P&L step-down.
- **Rolling 30-day** — short-window. Useful for catching regime breaks but produces noisy alerts.

A reasonable default is to monitor all-time and 30-day in parallel, alert on either, and use all-time for halt logic.

## Recovery and re-arm

Once halted, the strategy stays halted until the user re-arms it. To re-arm responsibly:

1. Investigate what caused the drawdown. Run a calibration audit and a P&L attribution.
2. Decide whether the strategy needs a fix (filter change, recalibration) or whether the drawdown was within normal variance.
3. If a fix is required, backtest it before re-arming.
4. Re-arm at a *lower* size than before (cut by 50%). Run for at least 30 trades at the lower size before scaling back up.
5. Reset HWM only if there's an explicit reason (e.g., a structural model change makes pre-change P&L not comparable).

## How to respond

When the user asks for drawdown status:

1. Compute current DD against all-time HWM.
2. Compute trailing-30-day DD.
3. Compare against configured thresholds; report which level is active (`clear`, `warn`, `de-risk`, `halt`).
4. Show the trailing P&L chart in text form (e.g., last 14 days of daily P&L).
5. If a non-clear level is active, recommend the corresponding action.

When the user asks to configure thresholds:

1. Ask for the backtested MDD (or run a backtest to estimate it).
2. Suggest the three thresholds per the rule above.
3. Confirm and write them to the bot's config.

## Reference script

A working monitor is at `scripts/drawdown.py`:

```
python scripts/drawdown.py --ledger trades.sqlite --hwm-window all
```

It reads a settled-trade ledger, computes HWM and current DD, and emits the threshold verdict. Run it as part of the bot's startup or as a scheduled task.

## Don't

- Reset HWM after a deep drawdown just because the run looks bad. The reference matters; resetting hides risk.
- Use volatility-based limits as a substitute for DD limits. They measure different things.
- Halt without notifying. The user needs to know the strategy stopped, not discover it after a quiet day.
- Re-arm at full size after a halt. Even after a fix, treat re-arm as a new strategy on probation.

## References

- Magdon-Ismail & Atiya (2004). *Maximum Drawdown.* Risk Magazine.
- Chekhlov, A., Uryasev, S., Zabarankin, M. (2005). *Drawdown Measure in Portfolio Optimization.*
