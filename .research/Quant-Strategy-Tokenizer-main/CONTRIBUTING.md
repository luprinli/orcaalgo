# Contributing

QST uses a current-only public tree. Contributions should preserve the active package name `qst`, the `.gkr.yaml` source suffix, and the internal `qst-ir/0.4` schema identity.

## Local Checks

Run the focused checks before opening changes:

```bash
python -m compileall qst
python -m ruff check .
python -m mypy qst
python -m pytest tests -q
python -m qst.cli vocabulary --check
python -m qst.cli hash examples/strategies/kdj_cross_basic.gkr.yaml
python -m qst.cli hash examples/strategies/kdj_with_ema_filter.gkr.yaml
```

For final verification, also run coverage:

```bash
python -m pytest --cov=qst --cov-fail-under=85 -q
```

## Change Rules

- Do not change canonical or hash semantics without an ADR.
- Do not add broad runtime execution through the CLI.
- Do not make custom-token approval portable.
- Do not introduce business framework imports into `qst/`.
- Keep examples under `examples/` and deterministic reference data under `tests/reference/`.

## Documentation

Update [docs/architecture.md](docs/architecture.md), [docs/security.md](docs/security.md), or [docs/reference.md](docs/reference.md) when behavior, trust boundaries, or public file layout changes.