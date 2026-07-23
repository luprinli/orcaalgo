from __future__ import annotations

import shutil
from pathlib import Path

from tools.verify_prompt_remote_artifacts import verify_prompt_artifacts

ROOT = Path(__file__).resolve().parents[2]
PROMPT_ROOT = ROOT / "docs" / "agent" / "prompts" / "qst_stage_3c_v0_3_2"


def _copy_repo_subset(tmp_path: Path) -> tuple[Path, Path]:
    repo = tmp_path / "repo"
    prompt = repo / "docs" / "agent" / "prompts" / "qst_stage_3c_v0_3_2"
    shutil.copytree(PROMPT_ROOT, prompt)
    tools = repo / "tools"
    tests = repo / "tests" / "agent_prompts"
    tools.mkdir(parents=True)
    tests.mkdir(parents=True)
    shutil.copy(ROOT / "tools" / "validate_prompt_set.py", tools / "validate_prompt_set.py")
    shutil.copy(
        ROOT / "tests" / "agent_prompts" / "test_validate_prompt_set.py",
        tests / "test_validate_prompt_set.py",
    )
    return repo, prompt


def _issue_paths(result: dict[str, object]) -> set[str]:
    payload = result["prompt_artifact_verification"]
    assert isinstance(payload, dict)
    artifacts = payload["artifacts"]
    assert isinstance(artifacts, list)
    return {
        str(artifact["path"])
        for artifact in artifacts
        if isinstance(artifact, dict) and artifact.get("verdict") == "fail"
    }


def test_prompt_artifact_verifier_passes_for_repo_checkout() -> None:
    result = verify_prompt_artifacts(PROMPT_ROOT)

    assert result["prompt_artifact_verification"]["result"] == "pass"


def test_prompt_artifact_verifier_detects_bad_python(tmp_path: Path, monkeypatch) -> None:
    repo, prompt = _copy_repo_subset(tmp_path)
    monkeypatch.chdir(repo)
    (repo / "tools" / "validate_prompt_set.py").write_text(
        "from __future__ import annotations import json\n",
        encoding="utf-8",
    )
    result = verify_prompt_artifacts(prompt)

    assert result["prompt_artifact_verification"]["result"] == "fail"
    assert "tools/validate_prompt_set.py" in _issue_paths(result)


def test_prompt_artifact_verifier_detects_compressed_markdown(tmp_path: Path, monkeypatch) -> None:
    repo, prompt = _copy_repo_subset(tmp_path)
    monkeypatch.chdir(repo)
    (prompt / "core" / "00_FOUNDATION.md").write_text(
        "# QST Agent Foundation prompt_system_version: qst-stage-3c-v0.3.2.3 compressed",
        encoding="utf-8",
    )
    result = verify_prompt_artifacts(prompt)

    assert result["prompt_artifact_verification"]["result"] == "fail"
    assert "docs/agent/prompts/qst_stage_3c_v0_3_2/core/00_FOUNDATION.md" in _issue_paths(result)


def test_prompt_artifact_verifier_detects_bad_golden_yaml(tmp_path: Path, monkeypatch) -> None:
    repo, prompt = _copy_repo_subset(tmp_path)
    monkeypatch.chdir(repo)
    (prompt / "golden" / "01_ema_cross.intent.yaml").write_text(
        "golden_task: [unterminated\n",
        encoding="utf-8",
    )
    result = verify_prompt_artifacts(prompt)

    assert result["prompt_artifact_verification"]["result"] == "fail"
    assert "docs/agent/prompts/qst_stage_3c_v0_3_2/golden/01_ema_cross.intent.yaml" in _issue_paths(result)
