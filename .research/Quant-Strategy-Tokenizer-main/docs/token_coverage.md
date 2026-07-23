# Token Coverage

Stage 3A adds conformance gates for the public built-in vocabulary:

- every built-in TokenSpec has `surface` metadata;
- accepted tokens have temporal, numeric, missing-data, and failure-mode contracts;
- experimental, deprecated, and reserved-design tokens are gated by profile;
- built-in TokenPacks are returned by `builtin_token_packs()` in deterministic order;
- all public demos have validation artifacts and graph/param/instance hash sentinels.

Stage 3A.1 adds concrete primitive coverage for `math.*`, `bool.*`, and
`cmp.*` token contracts. The helper implementations are deterministic reference
semantics for tests and demos; they are not a broad GKR execution runtime.

The primitive coverage fixes:

- numeric domain errors for divide-by-zero, negative square root, non-positive
  logarithm, non-finite input, and bool-as-float input;
- boolean truth tables and explicit empty-input policies for boolean folds;
- comparison truth tables with inclusive range boundaries by default.

Stage 3A.2 adds accepted data/time/align/window/signal/indicator coverage.
Series reference helpers use `Sequence[tuple[str, value]]` test material,
canonical timestamp ordering, and duplicate-timestamp diagnostics. Numeric
series reject bool, NaN, and Infinity. The helpers are intentionally separate
from strategy validation and do not create a broad runtime.

The series coverage fixes:

- unsafe future behavior for negative `data.shift` periods;
- return-transform domain errors for zero percentage-change denominators and
  non-positive log-return inputs;
- trailing-window behavior, `min_periods`, population standard deviation, and
  zero-variance z-score output;
- cross/threshold signal truth tables and current-bar-safe channel breakout.

Stage 3A.3 adds decision/gate/state conformance coverage:

- `decision.lift_bool` and Decision Algebra facade delegation;
- monoid/fold/aggregator classification for decision tokens;
- state helper facade coverage that preserves existing traces and diagnostics;
- gate reference fixtures for cooldown, market freeze, circuit breaker, observe
  period, and slot budget behavior.

Gate helpers output `DecisionV2` values only. Diagnostics remain diagnostics and
do not become a DecisionKind.

Stage 3A.4 adds panel/weight/risk/optimizer conformance coverage:

- panel and weight token surface contracts preserve accepted WP8 semantics;
- panel float operators are explicitly semantic float64 and make no bit-exact
  claim;
- weight operators use canonical decimal reference arithmetic;
- risk helpers cover position-cap blocking, volatility-target scaling, and
  turnover-cap clipping without optimization, redistribution, order planning, or
  broker execution;
- optimizer tokens are experimental, solver-backed, and metadata-only until a
  deterministic solver contract is accepted.

Stage 3A.5 adds reserved-design and execution-boundary coverage:

- event, distribution, and execution boundary tokens are vocabulary-visible but
  non-executable;
- every reserved-design token is metadata-only, reserved-only, validation-only,
  and rejected by strategy validation in every profile;
- accepted `plan.*` tokens remain plan-shape metadata shells and do not expose a
  runtime/reference facade;
- continuous-score tokens remain separate from DecisionKind, order planning, and
  risk execution.

Stage 3A.6 locks demo and acceptance coverage:

- all 12 public examples validate through both Python API and CLI;
- all 12 examples have expected diagnostics plus graph/param/instance hash
  sentinels;
- the accepted full-trace set is exactly `01_ema_cross`,
  `08_market_neutral_rank`, and `12_custom_token_kalman_signal`;
- demo artifacts are conformance evidence, not runtime execution logs.

PR6 adds a core rule token batch for common indicator, signal, and decision-rule
records:

- `indicator.macd`, `indicator.atr`, `indicator.donchian_channel`,
  `indicator.volatility`, `indicator.linear_regression_slope`,
  `indicator.beta`, and `indicator.residual` now have accepted reference-helper
  coverage.
- `signal.greater_than`, `signal.less_than`, boolean signal composition,
  band/breakout triggers, z-score reversion triggers, and rank top/bottom-k
  selection helpers are covered by token conformance tests.
- `decision.long_flat`, `decision.long_short`, entry/exit position records,
  signal-to-decision mapping, selection-to-weight routing, and gate-decision
  dominance are covered without adding new `DecisionKind` values.

PR6 moves the MACD, ATR, and linear-regression-slope coverage rows from
custom-token route to supported record evidence, and adds five internal matrix
rows for signal composition, long/short decision rules, entry/exit gate records,
beta/residual time-series records, and Donchian/volatility rule records.

PR6 candidate GKR files live under `tests/coverage_cases/core_rule/` with hash
sentinels. They are coverage-case evidence only and do not change the accepted
12 public demo set under `examples/strategies/`.

`indicator.kdj` remains custom-token-required because it is outside the selected
PR6 Section 19 core rule batch.

PR8 adds accepted panel/factor/weight record coverage:

- `panel.cross_sectional_rank`, `panel.zscore_by_universe`, and
  `panel.neutralize_group` provide canonical aliases over existing panel rank,
  z-score, and group-neutralization helpers.
- `selection.top_k` and `selection.bottom_k` expose selection aliases for
  panel top/bottom-k reference semantics.
- `factor.sector_neutral_rank`, `factor.residualize`, and
  `factor.beta_neutral_signal` provide explicit factor records. Group or sector
  metadata must be supplied by the strategy record; QST does not infer
  instrument sector classifications.
- `weight.equal_weight`, `weight.rank_weight`, `weight.inverse_vol_weight`,
  `weight.vol_target_weight`, `weight.market_neutral_weight`,
  `weight.group_neutral_weight`, `weight.max_weight_clip`, and
  `weight.normalize_net` provide deterministic weight record transforms.

PR8 moves inverse-vol weighting, sector-neutral rank, and beta-neutral signal
coverage to supported record evidence where explicit metadata and candidate GKR
files exist. Candidate GKR files live under
`tests/coverage_cases/panel_factor_weight/` with hash sentinels. These are
coverage-case records only; they do not change the accepted 12 public demos and
do not claim a rebalance engine, optimizer, broker/exchange execution,
backtesting, or profitability.

PR10 also retires the stale `int_040_net_normalize` custom-token route because
`weight.normalize_net` is now accepted record/reference evidence. The custom
governance review records this as a stale-route cleanup, not as a new token.

PR9 adds accepted state/gate/risk record coverage:

- `gate.volatility_regime`, `gate.drawdown`, `gate.time_window`,
  `gate.rebalance`, `gate.min_hold`, and `gate.max_hold` now have accepted
  reference-helper records.
- `risk.stop_loss_record`, `risk.take_profit_record`,
  `risk.trailing_stop_record`, `risk.max_drawdown_record`,
  `risk.volatility_target_record`, `risk.exposure_cap_record`, and
  `risk.turnover_limit_record` now have accepted reference-helper records.

PR9 moves min/max hold gates, stop/take-profit/trailing records, rebalance-band
records, exposure caps, volatility-regime gates, and drawdown gates to supported
record evidence. Candidate GKR files live under
`tests/coverage_cases/state_gate_risk/` with hash sentinels. The evidence is
record-layer only: it does not add broker/exchange execution, live stop orders,
position lifecycle runtime, a rebalance engine, Calendar/EventStream TypeSpec,
backtesting, or profitability claims.

## Custom-Token Governance Coverage

PR10 adds governance evidence for active `custom_token_required` rows:

- `tests/coverage_cases/custom_token_governance/custom_token_routes.yaml`
- `docs/reports/custom_token_governance_review.md`

KDJ, Kalman, score calibration, pair-spread, ML-score, panel-factor, sentiment,
and dogfood custom-ML rows remain custom-token routes unless a future token batch
adds accepted built-in semantics. The governance evidence preserves verify,
approve, grant, execute, and output-validation separation and does not execute
custom code.

## Hash Impact

| Object | Stage 3A impact |
| --- | --- |
| TokenSpec hash | Changes when surface or contract metadata changes. |
| TokenPack hash | Changes when contained TokenSpec hashes change. |
| Strategy graph hash | Unchanged unless graph topology or token refs change. |
| Strategy param hash | Unchanged unless parameters change. |
| Strategy instance hash | Derived from graph and param hashes in the current implementation. |

Reference demo hash sentinels live under `tests/reference/strategies/`.
