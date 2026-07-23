# Strategy Authoring

prompt_system_version: qst-stage-3c-v0.3.2.3
profile_type: load_profile

## Use When

Use this profile when the user request matches strategy authoring work and the agent
needs a bounded evidence set before acting.

## Load Order

Always load:

- `core/00_FOUNDATION.md`
- `core/01_REPO_CONTEXT_PROTOCOL.md`
- `core/02_INPUT_SECURITY.md`
- `core/03_BEHAVIOR_CORE.md`
- `core/04_REPORT_SCHEMA.md`
- `core/05_ESCALATION_PROTOCOL.md`

Then load readers:

- `readers/READ_TOKEN_SYSTEM.md`
- `readers/READ_VALIDATION_DIAGNOSTICS.md`
- `readers/READ_EXAMPLES_STRATEGIES.md`
- `readers/READ_IR_SCHEMA_CANONICAL.md`
- `readers/READ_PROFILES.md`

Then load tasks:

- `tasks/CLASSIFY_STRATEGY_INTENT.md`
- `tasks/SELECT_TOKENS.md`
- `tasks/AUTHOR_GKR_STRATEGY.md`
- `tasks/REPAIR_GKR_DIAGNOSTICS.md`
- `tasks/PROFILE_GATE_REVIEW.md`

Load `tasks/CUSTOM_TOKEN_ROUTING.md` only if classification returns
`custom_token_required`.

## Mandatory Precondition

Before token selection or GKR authoring:

- load `core/01_REPO_CONTEXT_PROTOCOL.md`
- produce `repo_context`
- record commands not run and why
- record coverage-frontier evidence when strategy coverage, benchmark, dogfood, token
  expansion, reserved boundary, or custom-token routing decisions are involved

If `repo_context.token_surface` is unavailable, stop after classification and report the
missing evidence instead of inventing token refs.

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
