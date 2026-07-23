# Gate Review

prompt_system_version: qst-stage-3c-v0.3.2.3
task_type: review
foundation: core/00_FOUNDATION.md

## Use When

Use when maturity or execution support changes across profiles.

## Inputs

- User request and any attached strategy, diagnostic, report, or code context.
- Repository evidence loaded through the smallest relevant reader or load profile.
- Target profile when validation, execution support, or reserved-design status matters.

## Procedure

1. Read the repository profile policy and current validator behavior.
2. For every selected token, record maturity and execution_support.
3. Apply the target profile.
4. Read `docs/reports/strategy_coverage_report.md` when coverage-frontier evidence is
   relevant, including custom_token_route_share and false_supported_rate.
5. Identify custom token authorization requirements and whether the route would push the
   request beyond the custom token route cap.
6. Reject reserved-design executable use and non_goal runtime requests.
7. Output pass, warning, or error.

Reserved design tokens are visible vocabulary boundaries, not executable behavior.

## Default Profile Matrix

Use only if current repo evidence does not define a stricter rule:

| maturity | research | paper | pretrade | production_guarded |
| --- | --- | --- | --- | --- |
| accepted | pass | pass | pass | pass |
| frozen | pass | pass | pass | pass |
| experimental | warning | warning | error | error |
| reserved_design | error | error | error | error |
| deprecated | warning | warning | warning | error |

## Output

Return a profile matrix and whether the request is allowed:

```yaml
profile_gate:
  target_profile:
  verdict:
  token_findings:
    - token_ref:
      maturity:
      execution_support:
      profile_status:
      evidence:
  blockers:
  warnings:
  coverage_frontier:
    strategy_coverage_report:
    custom_token_route_share:
    false_supported_rate:
    reserved:
    non_goal:
  required_changes:
```

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
