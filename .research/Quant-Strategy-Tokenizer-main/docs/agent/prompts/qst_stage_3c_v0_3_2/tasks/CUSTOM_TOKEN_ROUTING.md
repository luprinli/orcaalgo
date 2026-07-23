# Custom Token Routing

prompt_system_version: qst-stage-3c-v0.3.2.3
task_type: security
foundation: core/00_FOUNDATION.md

## Use When

Use for any request involving custom Python entry points or external code.

## Inputs

- User request and any attached strategy, diagnostic, report, or code context.
- Repository evidence loaded through the smallest relevant reader or load profile.
- Target profile when validation, execution support, or reserved-design status matters.

## Procedure

1. Restate why custom token routing is needed.
2. Verify whether the request is metadata-only, approval, grant, or execution.
3. Check `docs/reports/strategy_coverage_report.md` for custom_token_route_share and the
   custom token route cap before recommending more custom-token surface area.
4. For verification, inspect metadata and integrity records only.
5. Do not import custom Python during verification.
6. Do not execute custom Python during verification.
7. Do not approve trust or create grants unless the user explicitly requested that step.
8. For execution, require explicit user request, verification pass, profile authorization,
   local approval, execution grant, current UTC time, run id, user-approved inputs, and
   output validation.
9. Report sandbox status and residual risks.

Boundary words: verify, approve, execute, and must not execute code during verification.

## Output

Return verify, approve, grant, execute, and audit status separately:

```yaml
custom_token_routing:
  custom_token_needed:
  reason:
  verify:
    command:
    status:
    code_executed: false
  coverage_frontier:
    strategy_coverage_report:
    custom_token_route_share:
    custom_token_route_cap:
  approve:
    required:
    status:
  grant:
    required:
    status:
  execute:
    requested_by_user:
    allowed:
    status:
  sandbox_status:
  residual_risks:
```

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
- Verification must not execute or import custom code; approval, grant, and execution are
  distinct steps.
