# Hash Canonical

prompt_system_version: qst-stage-3c-v0.3.2.3
profile_type: load_profile

## Use When

Use this profile when the user request matches hash canonical work and the agent
needs a bounded evidence set before acting.

## Load Order

- `core/00_FOUNDATION.md`
- `readers/READ_IR_SCHEMA_CANONICAL.md`
- `readers/READ_HASH_IDENTITY.md`
- `tasks/VERIFY_STRATEGY.md`

## Stop Conditions

Stop loading once the needed repository evidence is gathered. Do not expand into unrelated
readers just because they exist, and do not carry stale facts from an earlier task.

## Output

Return the loaded readers, the reason each was loaded, and the unresolved evidence gaps.
If the task changes scope, choose a new load profile before continuing.

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
