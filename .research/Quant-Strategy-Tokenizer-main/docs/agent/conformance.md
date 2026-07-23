# Agent Conformance

A change is conformant when it preserves:

- `qst-ir/0.4` and `qst-canonical/0.4` identity unless explicitly changed.
- Deterministic canonical JSON.
- Stable hash material boundaries.
- Local-only custom-token approval.
- `.gkr.yaml` public strategy inputs.
- No business framework imports in `qst/`.