# Token Family Registry

Stage 3A organizes public tokens by family:

```text
math, bool, compare, data, time, align, window, signal, indicator,
decision, gate, state, panel, weight, risk, optimizer, execution,
event, distribution, continuous_score, factor
```

Each built-in token must declare:

- `surface.family`
- `surface.category`
- `surface.layer`
- `surface.maturity`
- `surface.execution_support`
- `surface.contract`
- `surface.capabilities`
- `surface.agent_metadata`

## Maturity

| Maturity | Research | Paper | Pretrade | Production Guarded |
| --- | --- | --- | --- | --- |
| `accepted` | pass | pass | pass | pass |
| `frozen` | pass | pass | pass | pass |
| `experimental` | warning | warning | error | error |
| `deprecated` | warning | warning | warning | error |
| `reserved_design` | error | error | error | error |

## Execution Support

| Value | Meaning |
| --- | --- |
| `metadata_only` | Recognized and explainable; not executable. |
| `reference_helper` | Deterministic reference helper exists for tests or examples. |
| `runtime_executor` | Formal runtime executor exists. |
| `external_only` | Execution comes from an approved custom token or adapter boundary. |

`accepted` means the token governance and contract are accepted. It does not
imply broad strategy execution.

## Stage 3A.1 Primitive Families

Stage 3A.1 makes the canonical primitive surface explicit:

- `math.*`: arithmetic, reductions, transforms, predicates, conditionals, and
  missing-value transforms.
- `bool.*`: boolean logic and boolean reductions.
- `cmp.*`: numeric comparisons and range comparisons.

Existing `logic.*` and `compare.*` tokens remain accepted aliases for current
demo and reference material. New token authoring should prefer `bool.*` and
`cmp.*`. Cross-over semantics such as `crosses` belong to the `signal` family,
not the comparison family.

Primitive reference helpers are conformance helpers only. They do not create a
broad GKR runtime and are not invoked by validation.

## Stage 3A.2 Series Families

Stage 3A.2 completes the accepted series-token surface for:

- `data.*`: identity, shift, diff, percentage change, and log return.
- `time.*`: session filtering over timestamp time-of-day.
- `align.*`: inner join, left join, forward fill, and missing-row drop.
- `window.*`: trailing max/min/mean/std/sum/count/zscore.
- `signal.*`: cross, threshold, normalization, and simple recursive smoothing.
- `indicator.*`: SMA, EMA, RSI, Bollinger bands, and channel breakout.

All Stage 3A.2 helpers use deterministic reference semantics for conformance
only. Window tokens are trailing-window tokens; centered windows remain unsafe
temporal behavior. `data.shift` with negative periods is an unsafe-future
operation unless a reference test explicitly opts in.

## Stage 3A.3 Decision, Gate, and State Families

Stage 3A.3 completes the accepted decision/state control surface:

- `decision.*`: Decision Algebra monoids, fold policies, aggregators, and
  `decision.lift_bool`.
- `state.*`: delay, accumulate, edge detect, and closed-set FSM reference
  helpers.
- `gate.*`: cooldown, market freeze, circuit breaker, observe period, and slot
  budget gate semantics.

Decision monoids remain limited to `decision.unknown_propagating_and` and
`decision.any_accept`. `decision.strict_and` and `decision.permissive_and` are
fold policies, not monoids. `error` is never a DecisionKind; gate and state
failures use diagnostics. For `state.fsm`, `failure_policy="raise"` means emit
an error diagnostic in the reference helper, not raise a Python exception.

Gate tokens are stateful reference semantics. They are not order management,
portfolio risk enforcement, or broker/exchange execution.

## Stage 3A.4 Panel, Weight, Risk, and Optimizer Families

Stage 3A.4 completes the accepted panel/weight/risk surface:

- `panel.*`: accepted panel reference operators from WP8c.
- `selection.to_weights`: raw WeightPanel conversion, still under reference semantics.
- `weight.*`: accepted WeightPanel constraint transforms from WP8d.
- `risk.*`: deterministic reference helpers for position caps, volatility
  scaling, and turnover clipping.
- `optimizer.mean_variance`: experimental metadata-only solver-backed design
  surface.

Panel float operators declare `semantic_float64`, not bit-exact reproducibility.
Weight operators use canonical decimal reference semantics. Risk helpers do not
create a portfolio engine, optimizer, order planner, or broker risk control.
They only expose deterministic conformance behavior for token contracts.

Solver-backed optimizer tokens remain non-executable until they provide an
explicit solver determinism contract. In Stage 3A.4, optimizer tokens are visible
for vocabulary and agent explanation only.

## Stage 3A.5 Reserved Design and Execution Boundaries

Stage 3A.5 locks vocabulary-visible but non-executable boundaries:

- `event.*`: reserved EventStream design surface.
- `distribution.*`: reserved distribution-model design surface.
- `execution.*`: reserved broker/exchange execution and feedback boundary.
- `plan.*`: accepted metadata-only plan shells; not runtime execution.
- `score.*`: continuous-score transforms; not DecisionKind or execution plans.

Reserved-design tokens must be `metadata_only`, `reserved_only=true`,
`contract.scope=validation_only`, and `deterministic_level=reserved`. They may
be listed by vocabulary and explained to agents, but any strategy node that
references them fails validation with `QST_TOKEN_RESERVED_DESIGN_NOT_EXECUTABLE`
in every profile.

`score.calibrate` remains experimental metadata-only until an accepted reference
helper or runtime contract exists. `score.zscore` remains an accepted reference
helper.

## PR6 Core Rule Token Batch

PR6 extends the accepted record/reference surface for common rule strategies
without changing IR, canonical/hash algorithms, strategy validation semantics,
or public demo behavior.

Indicator additions:

- `indicator.rolling_mean`, `indicator.rolling_std`, and
  `indicator.rolling_zscore` as accepted aliases over trailing window
  reference semantics.
- `indicator.macd`, with EMA fast/slow/signal windows and deterministic
  semantic-float64 output.
- `indicator.bollinger_band`, `indicator.atr`, `indicator.donchian_channel`,
  and `indicator.volatility` for accepted volatility/channel records.
- `indicator.linear_regression_slope`, `indicator.beta`, and
  `indicator.residual` for trailing OLS reference records with stable
  diagnostics for insufficient observations and zero-variance inputs.

Signal additions:

- numeric and boolean rule composition through `signal.greater_than`,
  `signal.less_than`, `signal.and`, `signal.or`, `signal.not`,
  `signal.between`, and `signal.outside_band`;
- breakout and mean-reversion triggers through `signal.breakout_up`,
  `signal.breakout_down`, and `signal.zscore_revert`;
- cross-sectional selection records through `signal.rank_top_k` and
  `signal.rank_bottom_k`.

Decision additions:

- `decision.long_flat`, `decision.long_short`,
  `decision.entry_exit_to_position`, `decision.signal_to_decision`,
  `decision.rank_to_selection`, `decision.selection_to_weight`, and
  `decision.gate_decision`.

These decision helpers preserve existing `DecisionKind` values. Long/short side
and position intent are represented through reasons or state records, not by
adding new decision kinds. `decision.gate_decision` keeps `block` dominant and
otherwise preserves the input decision.

All PR6 additions are `accepted` with `execution_support=reference_helper`.
They are conformance/reference helpers only; they do not create broad GKR
runtime execution, a backtester, broker/exchange behavior, optimizer execution,
or production execution. `indicator.kdj` remains outside PR6 and stays on the
custom-token route.

## PR8 Panel / Factor / Weight Batch

PR8 extends the accepted record/reference surface for panel aliases, factor
records, and deterministic weight transforms. It does not change panel runtime
semantics, create a rebalance engine, or introduce broker/exchange execution.

Panel and selection additions:

- `panel.cross_sectional_rank`, `panel.zscore_by_universe`, and
  `panel.neutralize_group` as canonical Section 21 aliases over accepted
  `panel.rank`, `panel.zscore`, and `panel.group_demean` semantics.
- `selection.top_k` and `selection.bottom_k` as selection aliases over
  accepted top/bottom-k panel reference behavior.

Factor additions:

- `factor.sector_neutral_rank` ranks explicitly supplied group-neutral records.
  The helper requires group metadata and never infers sector or instrument
  classification.
- `factor.residualize` delegates to accepted residualization semantics.
- `factor.beta_neutral_signal` records residualized/beta-aware signal evidence.
  It is not a complete beta-neutral portfolio construction engine.

Weight additions:

- `weight.equal_weight`, `weight.rank_weight`, `weight.inverse_vol_weight`,
  `weight.vol_target_weight`, `weight.market_neutral_weight`,
  `weight.group_neutral_weight`, `weight.max_weight_clip`, and
  `weight.normalize_net`.

All PR8 additions are `accepted` with `execution_support=reference_helper`.
Weight helpers reject bool and non-finite numeric material. Group-neutral factor
and weight helpers require explicit group metadata. These helpers are
conformance/reference helpers only and do not imply optimizer execution,
rebalance scheduling, order routing, live trading, backtesting, or profitability.

## PR9 State / Gate / Risk Batch

PR9 extends accepted record/reference coverage for common state, gate, and risk
controls. It does not add broker/exchange execution, live stop orders, position
lifecycle runtime, a rebalance engine, EventStream, Calendar TypeSpec,
backtesting, or optimizer execution.

Gate additions:

- `gate.volatility_regime` blocks decisions when realized volatility exceeds an
  explicit threshold.
- `gate.drawdown` blocks decisions when an explicit drawdown record breaches the
  configured limit.
- `gate.time_window` records UTC time-of-day gating with inclusive `HH:MM`
  boundaries. Calendar semantics remain deferred.
- `gate.rebalance` records threshold/band rebalance intent only.
- `gate.min_hold` and `gate.max_hold` record hold-age gates without claiming
  position lifecycle or order-management runtime.

Risk additions:

- `risk.stop_loss_record`, `risk.take_profit_record`, and
  `risk.trailing_stop_record` record deterministic stop/take-profit thresholds
  without placing broker-side orders.
- `risk.max_drawdown_record` records deterministic equity-curve drawdown
  gating without account runtime.
- `risk.volatility_target_record`, `risk.exposure_cap_record`, and
  `risk.turnover_limit_record` expose record-layer aliases for accepted
  volatility-target, exposure-cap, and turnover-limit semantics.

All PR9 additions are `accepted` with `execution_support=reference_helper`.
Gate helpers return `DecisionV2` records where `block` is a decision kind and
errors remain diagnostics. Risk helpers reject bool and non-finite numeric
material and never imply broker, exchange, backtest, live execution, or
portfolio/account runtime support.

## PR10 Custom-Token Governance

PR10 does not add token refs. It records governance evidence for
`custom_token_required` coverage rows and enforces that each route has a reason,
input/output ports, and verify/approve/grant/execute separation.

Governance evidence:

- `docs/reports/custom_token_governance_review.md`
- `tests/coverage_cases/custom_token_governance/custom_token_routes.yaml`

`int_040_net_normalize` is no longer a custom-token route because
`weight.normalize_net` is accepted PR8 record evidence. `indicator.kdj`, built-in
Kalman signal, pair-spread model, score calibration, ML-score, panel-factor, and
sentiment routes remain custom-token or future-token candidates.
