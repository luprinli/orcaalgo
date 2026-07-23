# QST

Archived agent-ready research prototype for typed quantitative strategy records.

QST turns strategy intent into deterministic Graph Kernel Records (GKR): typed
`.gkr.yaml` files that can be validated, canonicalized, hashed, reviewed, and
handed off to agents or downstream systems.

It is not a trading bot, broker adapter, exchange adapter, backtester, optimizer
runtime, or production execution engine.

## Status

This repository is closed as an **archived agent-ready research prototype**.

The final tree includes:

- typed `qst-ir/0.4` strategy records
- deterministic `qst-canonical/0.4` canonical JSON
- graph, parameter, and instance hashes
- token surface governance and conformance tests
- Coverage Frontier v0.3 evidence and publication gate
- active agent prompt pack and handoff docs
- a Qlib partial workflow adapter proof

## Install

```bash
pip install -e ".[dev]"
```

## Quick Check

```bash
python -m qst.cli vocabulary --check
python -m qst.cli validate examples/strategies/kdj_cross_basic.gkr.yaml
python -m qst.cli hash examples/strategies/kdj_cross_basic.gkr.yaml
python -m qst.cli canonicalize examples/strategies/kdj_cross_basic.gkr.yaml --output .local_audit/kdj.canonical.json
```

## Qlib Adapter Proof

QST includes a partial Qlib workflow YAML importer:

```bash
python -m qst.cli adapter qlib import examples/adapters/qlib/workflow_config_lightgbm_alpha158.yaml --output .local_audit/qlib_lightgbm_alpha158.gkr.yaml --coverage .local_audit/qlib_lightgbm_alpha158.coverage.json
python -m qst.cli validate .local_audit/qlib_lightgbm_alpha158.gkr.yaml
python -m qst.cli hash .local_audit/qlib_lightgbm_alpha158.gkr.yaml
python -m qst.cli canonicalize .local_audit/qlib_lightgbm_alpha158.gkr.yaml --output .local_audit/qlib_lightgbm_alpha158.canonical.json
```

The adapter is record-layer evidence only. It does not import Qlib, run qrun,
train models, run inference, execute backtests, connect to brokers or exchanges,
or claim lossless Qlib conversion.

## What It Provides

- **Record layer:** stable `.gkr.yaml` strategy records with typed nodes,
  token refs, capabilities, and port signatures.
- **Identity layer:** canonical JSON plus deterministic graph, parameter, and
  instance hashes.
- **Token layer:** accepted, experimental, reserved, and custom-token surfaces
  with maturity and execution-support metadata.
- **Evidence layer:** public demos, reference diagnostics, hash sentinels,
  coverage matrix, coverage report, and dogfood cases.
- **Agent layer:** active prompt pack, takeover prompt, playbook, usage guide,
  and secondary development guides.

## Boundaries

QST deliberately does not provide:

- live trading
- broker or exchange integration
- full backtesting
- Qlib runtime replacement
- qrun execution
- model training or inference execution
- lossless Qlib conversion
- arbitrary Python strategy parsing
- production portfolio optimization
- profitability claims

Custom-token execution is explicit and approval-bound. Verification does not
import or execute user code; execution requires integrity verification, local
approval, an execution grant, and output validation.

## Handoff

Start here:

- [Final Handoff](docs/FINAL_HANDOFF.md)
- [Final Scope](docs/FINAL_SCOPE.md)
- [Final Acceptance](docs/FINAL_ACCEPTANCE.md)
- [Agent Takeover Prompt](docs/agent/AGENT_TAKEOVER_PROMPT.md)
- [Agent Playbook](docs/agent/AGENT_PLAYBOOK.md)
- [Usage Guide](docs/agent/USAGE_GUIDE.md)

## Key Paths

```text
qst/                         Python package and CLI
examples/strategies/         12 public GKR strategy examples
examples/adapters/qlib/      Qlib partial workflow adapter examples
examples/custom_token/       Custom-token reference example
tests/reference/             Deterministic reference fixtures and traces
tests/adapters/qlib/         Qlib adapter proof tests
docs/agent/                  Agent handoff and prompt guidance
docs/adapters/               Adapter boundary and Qlib adapter guide
docs/reports/                Coverage Frontier and acceptance reports
```

## Documentation

- [Architecture](docs/architecture.md)
- [Security](docs/security.md)
- [Reference](docs/reference.md)
- [Token Family Registry](docs/token_family_registry.md)
- [Token Coverage](docs/token_coverage.md)
- [Coverage Report](docs/reports/strategy_coverage_report.md)
- [Qlib Adapter Boundary](docs/adapters/QLIB_ADAPTER_BOUNDARY.md)
- [Agent Guidance](docs/agent/README.md)

## Project Identity

- Python import package: `qst`
- CLI: `qst`
- Distribution name: `quant-strategy-tokenizer`
- Editable strategy source: `.gkr.yaml`
- Packaged record suffix: `.gkr`
- IR: `qst-ir/0.4`
- Canonical schema: `qst-canonical/0.4`
