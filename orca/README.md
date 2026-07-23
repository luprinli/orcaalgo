# `orca/` — Python Domain Core

The Python layer is the **canonical authority** for strategy definitions, mathematical models, and quality assurance pipelines. It is never invoked on the hot path — Go calls Python via `os/exec` subprocess for validation, calibration, pre-flight, and attribution.

[↑ Back to Root README](../README.md)

## Sub-Packages

### `orca/models/` — Frozen Pydantic v2 Domain Models

Immutable models with `ConfigDict(frozen=True, extra="forbid")`. All prices use `Decimal` for fixed-point compliance.

| Module | Key Types | Purpose |
|--------|-----------|---------|
| `strategy.py` | `StrategyIRV04`, `Node`, `TokenRef`, `PortSignature`, `TemporalRule` | GKR strategy IR schema (qst-ir/0.4) |
| `trade.py` | `TradeSignal`, `Order`, `Fill`, `Position` | Trade lifecycle using `Decimal` prices |
| `risk.py` | `RiskSnapshot`, `KillSwitchState`, `BreachCondition`, `DrawdownLevel` | FTMO-compliant risk telemetry |

```python
from orca.models.strategy import StrategyIRV04, Node, TokenRef
from orca.models.trade import TradeSignal, Order
from orca.models.risk import RiskSnapshot, DrawdownLevel
```

### `orca/ir/` — GKR Strategy Intermediate Representation

Declarative strategy definitions with multi-profile validation.

| Module | Key Functions | Purpose |
|--------|---------------|---------|
| `loader.py` | `load_ir(path)`, `save_ir(ir, path)` | YAML ↔ Pydantic roundtrip |
| `validator.py` | `validate_ir(ir, profile)` | 4-tier validation (research/paper/pretrade/production_guarded) |
| `canonical.py` | `canonicalize_ir(ir)` | Deterministic serialization for hashing |
| `diagnostics.py` | `Diagnostic` dataclass | Structured error/warning/info reporting |
| `compiler.py` | `compile_all()` | GKR YAML → Go strategy source code |
| `params_export.py` | `export_params_json()` | Extract strategy params for Go consumption |

```bash
orca validate configs/strategies/intraday_mr.gkr.yaml
python -m orca.ir.compiler          # Generate strategy Go config from GKR YAMLs
```

### `orca/math/` — Canonical Mathematical Functions

All probability-emitting models must use these implementations. **Never reimplement in Go** (Hard Prohibition #1).

| Module | Functions | Purpose |
|--------|-----------|---------|
| `brier.py` | `brier_score()`, `murphy_decomposition()` | Brier score + Murphy reliability/resolution/uncertainty decomposition |
| `platt.py` | `platt_scale()` | Platt scaling for probability calibration (Nelder-Mead optimization) |
| `wilson.py` | `wilson_ci()` | Wilson confidence intervals for hit rate bounds |

```python
from orca.math.brier import brier_score, murphy_decomposition
from orca.math.platt import platt_scale
from orca.math.wilson import wilson_ci
```

### `orca/sizing/` — Position Sizing

Kelly criterion with all three attenuators per spec §3.1.3. **Fractional Kelly (k=0.25) is mandatory in production** (Hard Prohibition #6).

| Module | Functions | Purpose |
|--------|-----------|---------|
| `kelly.py` | `kelly_with_attenuators()`, `kelly_fraction_binary()`, `kelly_fraction_continuous()` | Kelly sizing with edge discount, fractional multiplier, per-trade cap, exposure headroom |
| `volatility.py` | `ewma_volatility()`, `vol_adjusted_size()`, `diversification_scaling()`, `diversification_weights()` | EWMA volatility estimation, volatility-adjusted sizing, multi-asset diversification |

```python
from orca.sizing.kelly import kelly_with_attenuators
from orca.sizing.volatility import ewma_volatility, vol_adjusted_size, diversification_weights

result = kelly_with_attenuators(p=0.60, price=0.50, side="yes")
print(f"Allocation: {result.final_allocation:.2%}")  # 0.50%
```

### `orca/calibration/` — Quarterly Calibration Audit

MECE calibration audit using Brier/Murphy decomposition, per-side segmentation, and Platt scaling.

| Module | Key Functions | Purpose |
|--------|---------------|---------|
| `audit.py` | `run_calibration_audit(trades)` | Full audit: overall + per-side segments |
| `platt.py` | `platt_calibrate_segment()` | Platt scaling with min cohort gating |

```bash
orca calibrate --since 90d --output calibration_report.json
```

### `orca/preflight/` — Pre-Deployment Checklist

12-point checklist gates all live deployments (Hard Prohibition #4).

| Module | Key Functions | Purpose |
|--------|---------------|---------|
| `checklist.py` | `run_preflight_checks()` | Config exists, GKR validated, env vars, package integrity, numpy/scipy available |

```bash
orca preflight          # Standard check
orca preflight --strict # Block deployment on warnings
```

### `orca/attribution/` — PnL Attribution

Multi-dimensional PnL slicing with Wilson CI bounds.

| Module | Key Functions | Purpose |
|--------|---------------|---------|
| `slicer.py` | `attribute_pnl(trades)` | 4-dim attribution: by side, price bucket, edge bucket |

```bash
orca attribute --since 90d --output attribution_report.json
```

### `orca/train/` — Model Training

| Module | Key Functions | Purpose |
|--------|---------------|---------|
| `hmm.py` | `train_hmm()`, `export_params_json()` | Train 4-state Gaussian HMM, export to JSON |

```bash
python -m orca.train.hmm  # Train HMM and generate configs/hmm_params.json
```

### `orca/backtest/` — Monte Carlo Simulation

| Module | Key Functions | Purpose |
|--------|---------------|---------|
| `monte_carlo.py` | `run_simulation()` | FTMO prop-firm pass-probability via Monte Carlo |

```bash
python -m orca.backtest.monte_carlo --simulations 10000 --json
```

### `orca/risk/` — Adversarial Testing

| Module | Key Functions | Purpose |
|--------|---------------|---------|
| `adversarial.py` | `generate_events()`, `inject_events()` | Fake news injection for guardrail effectiveness testing |

### `orca/ports/` — Temporal Validation

| Module | Key Functions | Purpose |
|--------|---------------|---------|
| `temporal.py` | `trace_temporal_validation()` | Look-ahead prevention, temporal contract validation |

### `orca/hash/` — Content-Addressable Hashing

Three-layer deterministic hashing for strategy graph, parameters, and instances.

| Module | Key Functions | Purpose |
|--------|---------------|---------|
| `common.py` | `stable_json_bytes()`, `hash_v2()` | Canonical JSON + SHA-256 |
| `graph.py` | `graph_hash_v2()`, `param_hash_v2()`, `instance_hash_v2()` | Three-layer strategy hashing |
| `verify.py` | `verify_graph_hash()`, `verify_instance_hash()` | Hash verification |

## CLI

```bash
orca validate <path>       # Validate .gkr.yaml against production_guarded profile
orca calibrate --since 90d # Run calibration audit
orca preflight             # Pre-deployment checklist
orca attribute --since 90d # PnL attribution
```

## Dependencies

- `pydantic>=2.5` — Immutable domain models
- `pyyaml>=6.0` — Strategy config parsing
- `numpy>=1.24` — Numerical computing
- `scipy>=1.10` — Platt scaling optimization, HMM
- `typer>=0.9` — CLI framework
- `structlog>=23` — Structured logging
- `hmmlearn>=0.3` — HMM training (optional, for training pipeline)
- `scikit-learn>=1.0` — Preprocessing (optional, for training pipeline)

## Testing

```bash
pytest tests/ -v                    # 201 tests
pytest tests/guardian/ -v           # Critical path smoke tests
pytest tests/adversarial/ -v        # Kill-switch resilience tests
```
