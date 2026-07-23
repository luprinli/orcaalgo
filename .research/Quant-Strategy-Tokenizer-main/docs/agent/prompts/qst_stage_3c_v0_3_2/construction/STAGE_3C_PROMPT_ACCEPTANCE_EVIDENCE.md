# Stage 3C Prompt Acceptance Evidence

prompt_system_version: qst-stage-3c-v0.3.2.3
construction_type: acceptance_evidence

This evidence records the Stage 3C prompt-pack validation gates. It is intentionally
focused on commands and verifier outcomes rather than rendered browser snippets.

The directory name `qst_stage_3c_v0_3_2` is the stable pack directory.
The active internal patch prompt-system version is `qst-stage-3c-v0.3.2.3`.
Previous version: `qst-stage-3c-v0.3.2.2`.

## Local Evidence

Recorded command results:

| Command | Exit code | Stdout excerpt | Stderr excerpt | Verdict |
| --- | ---: | --- | --- | --- |
| `python -m py_compile tools/validate_prompt_set.py` | `0` | no output | no output | pass |
| `python -m py_compile tests/agent_prompts/test_validate_prompt_set.py` | `0` | no output | no output | pass |
| `python tools/validate_prompt_set.py docs/agent/prompts/qst_stage_3c_v0_3_2` | `0` | `"result": "pass"`; all validator checks pass, including content completeness | no output | pass |
| `python tools/verify_prompt_remote_artifacts.py docs/agent/prompts/qst_stage_3c_v0_3_2` | `0` | `"result": "pass"`; local artifact format checks pass for Python, Markdown, and golden YAML | no output | pass |
| `python -m pytest tests/agent_prompts -q` | `0` | `27 passed` | no output | pass |
| `python -m pytest tests/agent_prompts/test_prompt_semantic_anchors.py -q` | `0` | semantic anchor tests pass | no output | pass |
| `python -m ruff check .` | `0` | `All checks passed!` | no output | pass |
| `python -m mypy qst` | `0` | `Success: no issues found in 105 source files` | no output | pass |
| `python -m pytest tests -q` | `0` | `441 passed` | no output | pass |
| `python -m pytest --cov=qst --cov-fail-under=85 -q` | `0` | `441 passed`; total coverage remains above the configured floor | no output | pass |
| `python -m qst.cli vocabulary --check` | `0` | `"ok": true`; zero diagnostics | no output | pass |

## Coverage Frontier PR5 Evidence

This patch aligns the prompt pack with Coverage Frontier evidence without changing QST
runtime, token, IR, canonical, hash, schema, example, or matrix semantics.

| Command | Exit code | Stdout excerpt | Stderr excerpt | Verdict |
| --- | ---: | --- | --- | --- |
| `python tools/validate_strategy_coverage_matrix.py docs/reports/strategy_coverage_matrix.yaml` | `0` | validation `pass`; `pattern_count: 105`; `dogfood_count: 5` | no output | pass |
| `python tools/report_strategy_coverage.py docs/reports/strategy_coverage_matrix.yaml --check` | `0` | `dogfood_publication_target: pass`; `custom_token_route_share: 0.1732`; `kernel_gap_count: 20` | no output | pass |
| `python -m pytest tests/coverage_cases -q` | `0` | `24 passed` | no output | pass |

New semantic anchors require the active prompt pack to mention:

- `strategy_coverage_matrix`
- `external_benchmark`
- `dogfood_case`
- `custom_token_route_share`
- `false_supported_rate`
- `kernel_gap`
- reserved/non-goal routing

Additional artifact-format gates passed locally:

- `tools/validate_prompt_set.py`, `tools/verify_prompt_remote_artifacts.py`, and
  `tests/agent_prompts/test_validate_prompt_set.py` compile as Python.
- semantic anchor validation passes for repo context, strategy classification, token
  selection, GKR authoring, diagnostic repair, and custom-token routing prompts.
- `.github/workflows/ci.yml` parses as YAML and contains a `prompt-validation` job.
- every golden intent YAML file parses as a mapping and contains `golden_task`.
- critical artifact line-count checks pass for the validator, validator tests, CI workflow,
  `core/00_FOUNDATION.md`, and `golden/01_ema_cross.intent.yaml`.

## CI Evidence

The CI workflow includes a dedicated `prompt-validation` job. It runs:

```bash
python -m py_compile tools/validate_prompt_set.py
python -m py_compile tests/agent_prompts/test_validate_prompt_set.py
python tools/validate_prompt_set.py docs/agent/prompts/qst_stage_3c_v0_3_2
python tools/verify_prompt_remote_artifacts.py docs/agent/prompts/qst_stage_3c_v0_3_2
python -m pytest tests/agent_prompts -q
```

CI must pass this job before Stage 3C prompt-pack acceptance is considered complete. The
validator now rejects prompt files that are merely present but lack required sections or
minimum useful content, and it rejects low-information prompts missing operational
semantic anchors.

## External Post-Push Evidence

The raw-artifact verifier can be run against a commit-SHA raw URL after push:

```bash
python tools/verify_prompt_remote_artifacts.py docs/agent/prompts/qst_stage_3c_v0_3_2 --raw-base https://raw.githubusercontent.com/waswrsis/Quant-Strategy-Tokenizer/<commit-sha>
```

The verifier records byte count, line count, compile/parse checks, and verdict for:

- `tools/validate_prompt_set.py`
- `tests/agent_prompts/test_validate_prompt_set.py`
- prompt-pack `README.md`
- `core/00_FOUNDATION.md`
- `tasks/CLASSIFY_STRATEGY_INTENT.md`
- `validation/VALIDATE_PROMPT_SET.md`
- Stage 3C acceptance docs
- the three complete golden task YAML files

## Recorded Local Artifact Evidence

The local artifact verifier passed on the updated prompt pack. It confirmed:

- `tools/validate_prompt_set.py` compiles as Python.
- `tests/agent_prompts/test_validate_prompt_set.py` compiles as Python.
- prompt-pack `README.md`, `core/00_FOUNDATION.md`, `tasks/CLASSIFY_STRATEGY_INTENT.md`,
  `validation/VALIDATE_PROMPT_SET.md`, and Stage 3C construction docs have H1 headings,
  prompt-system version markers, enough lines, and acceptable line lengths.
- `golden/01_ema_cross.intent.yaml`, `golden/12_custom_token_kalman_signal.intent.yaml`,
  and `golden/13_event_stream_intraday.intent.yaml` parse as YAML mappings with expected
  golden task structure.

Reviewers can rerun the raw verifier against the final commit SHA using the command above.

## Acceptance Decision

Stage 3C prompt pack acceptance requires all local prompt gates, CI prompt gates, and raw-artifact evidence checks to pass.

Current committed evidence status: `local-pass-not-remote-verified`.

This file records local repair evidence and the required evidence format. CI enforces the
local gate. The exact commit-SHA raw verifier is used after push as public repository
evidence and must be reported outside this self-referential file.
