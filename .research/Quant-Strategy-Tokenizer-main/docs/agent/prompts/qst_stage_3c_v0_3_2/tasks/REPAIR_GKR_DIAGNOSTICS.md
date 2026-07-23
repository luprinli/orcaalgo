# Repair Gkr Diagnostics

prompt_system_version: qst-stage-3c-v0.3.2.3
task_type: repair
foundation: core/00_FOUNDATION.md

## Use When

Use when a strategy fails validation or produces unexpected diagnostics.

## Inputs

- User request and any attached strategy, diagnostic, report, or code context.
- Repository evidence loaded through the smallest relevant reader or load profile.
- Target profile when validation, execution support, or reserved-design status matters.

## Procedure

1. Preserve the original strategy intent.
2. Classify every diagnostic.
3. Decide whether each diagnostic is repairable.
4. Apply the smallest repair.
5. Re-run validation.
6. Stop after 3 attempts.
7. Escalate if repair requires a new token, new type-system feature, validator bypass,
   profile weakening, reserved design execution, runtime, broker, or exchange capability.
8. Route unrepaired diagnostics into missing_token, kernel_gap, reserved_design,
   non_goal_runtime, or custom_token_required evidence for coverage follow-up.

## Diagnostic Classes

Classify each diagnostic as one of:

- schema_error
- unknown_token
- profile_blocked_token
- reserved_design_token
- port_error
- type_error
- temporal_error
- capability_error
- custom_token_required
- non_goal_runtime_required
- missing_token
- kernel_gap
- reserved_design
- non_goal_runtime

## Output

Return failure cause, patch summary, command evidence, and final blockers:

```yaml
diagnostic_repair:
  attempts:
    - number:
      diagnostics_before:
      change:
      diagnostics_after:
      command:
  final_status:
  unrepaired_blockers:
  escalation:
  coverage_routing:
    missing_token:
    kernel_gap:
    reserved_design:
    non_goal_runtime:
    custom_token_required:
```

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
- Do not make more than 3 attempts before reporting unrepaired blockers.
- Do not repair reserved design or EventStream gaps by using a time-series fake.
