# Select Tokens

prompt_system_version: qst-stage-3c-v0.3.2.3
task_type: authoring
foundation: core/00_FOUNDATION.md

## Use When

Use after intent classification to map behavior to current token refs.

## Inputs

- User request and any attached strategy, diagnostic, report, or code context.
- Repository evidence loaded through the smallest relevant reader or load profile.
- Target profile when validation, execution support, or reserved-design status matters.

## Procedure

1. Load vocabulary evidence from `readers/READ_TOKEN_SYSTEM.md`.
2. For each strategy concept, search only current vocabulary evidence.
3. Select an existing token ref or mark the concept missing.
4. Record maturity and execution_support for every candidate.
5. Attach coverage_row evidence from `docs/reports/strategy_coverage_matrix.yaml` when a
   matching pattern exists.
6. Record missing_tokens, kernel_gap evidence, and custom_token_route_share pressure from
   `docs/reports/strategy_coverage_report.md` when relevant.
7. Check the target profile through `tasks/PROFILE_GATE_REVIEW.md`.
8. Reject executable use of reserved design tokens and reject non_goal runtime requests.
9. Decide whether authoring may proceed.

Reserved design tokens are visible vocabulary boundaries, not executable behavior.

## Output

Return selected token refs, maturity, execution support, profile caveats, and rejected
alternatives:

```yaml
token_selection:
  may_author_gkr:
  selected:
    - concept:
      selected_token_ref:
      family:
      maturity:
      execution_support:
      profile_status:
      input_ports:
      output_ports:
      coverage_row:
      kernel_gap:
      evidence:
      reason:
  rejected_candidates:
    - concept:
      candidate:
      reason:
  missing_tokens:
  kernel_gaps:
  custom_token_route_share:
  reserved_blockers:
  non_goal_blockers:
  custom_token_required:
  profile_warnings:
```

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
- If a token ref is not found in current vocabulary evidence, it must not appear under
  `selected_token_ref`.
