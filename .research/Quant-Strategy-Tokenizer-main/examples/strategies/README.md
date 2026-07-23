# Public Strategy Examples

These examples are editable Graph Kernel Record sources. They are validation,
hash, and reference-evidence examples; they are not broad runtime execution
recipes.

Run any example with:

```bash
qst validate examples/strategies/01_ema_cross/strategy.gkr.yaml
qst hash examples/strategies/01_ema_cross/strategy.gkr.yaml
```

Reference diagnostics and hash sentinels live under
`tests/reference/strategies/<case>/`. Full trace artifacts are intentionally
limited to one scalar/time-series case, one panel/weight case, and one custom
token case.

| Case | Focus | Token families | Full trace |
| --- | --- | --- | --- |
| `01_ema_cross` | EMA cross with decision bridge | data, indicator, signal, decision | yes |
| `02_rsi_reversal` | RSI threshold reversal | indicator, signal, decision | no |
| `03_bollinger_mean_reversion` | Bollinger mean reversion shell | indicator, signal, decision | no |
| `04_breakout_channel` | Channel breakout shell | indicator, signal, decision | no |
| `05_cooldown_trend_following` | Stateful cooldown gate | indicator, signal, decision, gate, state | no |
| `06_circuit_breaker_mean_reversion` | Circuit breaker gate | indicator, signal, decision, gate, state | no |
| `07_topk_momentum_panel` | Panel top-k selection | panel, selection, weight | no |
| `08_market_neutral_rank` | Market-neutral panel weights | panel, selection, weight | yes |
| `09_btc_residual_meanrev` | BTC residual mean reversion | panel, indicator, selection, weight | no |
| `10_volatility_target_weight` | Volatility-target weight shell | weight, risk | no |
| `11_turnover_constrained_rebalance` | Turnover-constrained weight shell | weight, risk | no |
| `12_custom_token_kalman_signal` | Approval-bound custom token boundary | custom_runtime, signal, decision | yes |
