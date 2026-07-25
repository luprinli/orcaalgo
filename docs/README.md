# `docs/` — Project Documentation

Architecture specifications, technical references, and operations guides.

[↑ Back to Root README](../README.md)

## Current Documents

| File | Content |
|------|---------|
| `frontend-backtest-audit-2026-07-24.md` | Full frontend architecture audit: 23 routes, 4-group/20-item navigation, component catalog, backend capability mapping, duplication analysis, skeleton pages, navigation restructuring plan (3 groups/13 items) |
| `backtest-hub-ui-audit-2026-07-25.md` | BacktestHub component-by-component audit: Runner/History/Detail views, Promote-to-Live wizard, UX workflow analysis, P0–P3 prioritized remediation plan (22 items, 22h total) |
| `capital-pool-live-wiring-plan-2026-07-25.md` | 6-wave implementation plan: RiskPipeline interfaces, CapitalGate/PropFirmGate/SignalGate abstractions, BaseCapitalPool extraction, Engine/LiveEngine/API server wiring, per-account strategy isolation, KillSwitch integration (18h total) |
| `per-strategy-optimization-implementation-2026-07-25.md` | Light optimizer remediation: wiring RunLightOptimize into RunMatrixConcurrent, fixing optimize API handler stub, fixing IVS default-value bug, deterministic symbol selection, scoring metric unification, regression protection |
| `README.md` | This file — documentation index |

## Archived Documents

The following documents from previous audit cycles have been archived (removed from the active tree). Their findings were addressed and superseded by the current documentation set above:

- `executive_summary_2026-07-16.md`
- `PLATFORM_GUARDRAILS.md`
- `database_topology.md`
- `openapi.yaml`
- `frontend_audit_report.md`
- `frontend_remediation_plan.md`
- `test_suite_audit_report.md`
- `test_suite_remediation_plan.md`
- `full_system_audit.md`
- `backtest_live_parity_audit.md`
