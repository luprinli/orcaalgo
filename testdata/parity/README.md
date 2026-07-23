# Parity Golden Fixtures

Deterministic fixtures for the backtest↔live **parity oracle**
(see `docs/backtest_live_parity_audit_report.md` R7 / §24).

## Layout

```
testdata/parity/
  <SYMBOL>/<generation>/ticks.jsonl   # synthetic tick stream (ReplayEngine format)
```

Currently committed: `SPY/golden/ticks.jsonl` — 480 ticks (240 one-minute bars, two
ticks per minute), a flat ~100.00 base with tiny alternating noise plus a periodic
spike/decay tuned so a mean-reversion strategy (`intraday_mr`, lookback 20, entry
z=2) produces deterministic entry/exit signals.

## Regeneration

The fixture is byte-stable (fixed seed epoch `2026-01-05T14:30:00Z` and a fully
deterministic price path). Regenerate with:

```
go run ./cmd/orca-cli seed-parity-fixture
# or into a custom dir:
go run ./cmd/orca-cli seed-parity-fixture testdata/parity/SPY/golden
```

Regenerate **quarterly** or whenever the fixture strategy's `.gkr.yaml` changes hash.
An intentional change to the fixture must be reviewed by the risk + backtest owners;
an *unintended* golden diff on an unrelated PR is the signal `CI-1` is designed to catch.

## Consumers

- `internal/engine/replay_parity_test.go` — `TestReplayDeterminism_SingleStrategyFilter`
  loads this fixture and asserts two independent live replays produce byte-identical
  signals (the determinism precondition for the parity oracle).
