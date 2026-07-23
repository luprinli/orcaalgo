# Roadmap By Module

prompt_system_version: qst-stage-3c-v0.3.2.3

## Purpose

Describe how the Stage 3C prompt pack is installed, replaced, and maintained without
changing QST token, IR, hash, runtime, or strategy semantics.

## Steps

1. Keep one active prompt system under `docs/agent/prompts/qst_stage_3c_v0_3_2`.
2. Validate required files, cross references, golden tasks, and content completeness.
3. Keep CI prompt-validation enabled.
4. Record real command evidence instead of expected-result prose.

## Acceptance

This construction note is accepted only when the prompt validator, agent prompt tests,
and CI prompt-validation job pass.

## Guardrails

- Use current repository evidence before making current-state claims.
- Do not invent token refs, schema fields, capabilities, or runtime behavior.
- Keep reserved design features non-executable and route unsupported behavior explicitly.
- Treat validation, hash stability, and prompt success as engineering evidence only.
