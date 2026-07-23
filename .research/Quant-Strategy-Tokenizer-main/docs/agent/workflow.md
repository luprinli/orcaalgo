# Agent Workflow

1. Inspect `git status --short`.
2. Identify the smallest affected modules and tests.
3. Make scoped changes.
4. Run focused tests.
5. Run `ruff`, `mypy`, and CLI smoke checks before completion.
6. Summarize changed files and any gates that could not be run.