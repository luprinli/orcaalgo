from __future__ import annotations

import py_compile
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[2]
PROMPT_ROOT = ROOT / "docs" / "agent" / "prompts" / "qst_stage_3c_v0_3_2"
MAX_MARKDOWN_LINE_LENGTH = 240


def test_critical_python_artifacts_are_multiline_and_compile() -> None:
    minimum_lines = {
        ROOT / "tools" / "validate_prompt_set.py": 100,
        ROOT / "tools" / "verify_prompt_remote_artifacts.py": 80,
        ROOT / "tests" / "agent_prompts" / "test_validate_prompt_set.py": 30,
    }

    for path, minimum in minimum_lines.items():
        lines = path.read_text(encoding="utf-8").splitlines()
        assert len(lines) >= minimum, f"{path}: only {len(lines)} lines, expected >= {minimum}"
        py_compile.compile(str(path), doraise=True)


def test_ci_workflow_is_multiline_yaml_with_prompt_validation_job() -> None:
    path = ROOT / ".github" / "workflows" / "ci.yml"
    lines = path.read_text(encoding="utf-8").splitlines()
    assert len(lines) >= 20, f"{path}: only {len(lines)} lines, expected >= 20"

    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    assert isinstance(data, dict)
    jobs = data.get("jobs")
    assert isinstance(jobs, dict)
    assert "prompt-validation" in jobs


def test_prompt_markdown_files_are_multiline_and_readable() -> None:
    bad: list[str] = []
    for path in sorted(PROMPT_ROOT.rglob("*.md")):
        lines = path.read_text(encoding="utf-8").splitlines()
        if len(lines) < 5:
            bad.append(f"{path}: only {len(lines)} lines")
        if not any(line.startswith("# ") for line in lines):
            bad.append(f"{path}: missing H1")
        if "prompt_system_version: qst-stage-3c-v0.3.2.3" not in "\n".join(lines):
            bad.append(f"{path}: missing prompt_system_version")
        for line_number, line in enumerate(lines, start=1):
            if len(line) > MAX_MARKDOWN_LINE_LENGTH:
                bad.append(f"{path}:{line_number}: line length {len(line)}")

    assert not bad, "\n".join(bad)


def test_golden_yaml_files_are_multiline_parseable_mappings() -> None:
    bad: list[str] = []
    for path in sorted((PROMPT_ROOT / "golden").glob("*.yaml")):
        lines = path.read_text(encoding="utf-8").splitlines()
        if len(lines) < 5:
            bad.append(f"{path}: only {len(lines)} lines")
            continue
        data = yaml.safe_load(path.read_text(encoding="utf-8"))
        if not isinstance(data, dict):
            bad.append(f"{path}: root is not mapping")
            continue
        golden_task = data.get("golden_task")
        if not isinstance(golden_task, dict):
            bad.append(f"{path}: missing golden_task mapping")

    assert not bad, "\n".join(bad)


def test_runbook_critical_artifact_line_counts() -> None:
    checks = {
        ROOT / "tools" / "validate_prompt_set.py": 100,
        ROOT / "tests" / "agent_prompts" / "test_validate_prompt_set.py": 30,
        ROOT / ".github" / "workflows" / "ci.yml": 20,
        PROMPT_ROOT / "core" / "00_FOUNDATION.md": 20,
        PROMPT_ROOT / "golden" / "01_ema_cross.intent.yaml": 20,
    }

    for path, minimum in checks.items():
        lines = path.read_text(encoding="utf-8").splitlines()
        assert len(lines) >= minimum, f"{path}: only {len(lines)} lines, expected >= {minimum}"
