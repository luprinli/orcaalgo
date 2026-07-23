from __future__ import annotations

import argparse
import json
from dataclasses import dataclass
from pathlib import Path
from typing import Any
from urllib.error import URLError
from urllib.parse import urlparse
from urllib.request import urlopen

import yaml

PROMPT_VERSION = "qst-stage-3c-v0.3.2.3"
MAX_MARKDOWN_LINE_LENGTH = 240
MIN_MARKDOWN_LINES = 5

PYTHON_FILES = (
    "tools/validate_prompt_set.py",
    "tests/agent_prompts/test_validate_prompt_set.py",
)

PROMPT_MARKDOWN_FILES = (
    "README.md",
    "core/00_FOUNDATION.md",
    "tasks/CLASSIFY_STRATEGY_INTENT.md",
    "validation/VALIDATE_PROMPT_SET.md",
    "construction/STAGE_3C_PROMPT_ACCEPTANCE.md",
    "construction/STAGE_3C_PROMPT_ACCEPTANCE_EVIDENCE.md",
)

GOLDEN_YAML_FILES = (
    "golden/01_ema_cross.intent.yaml",
    "golden/12_custom_token_kalman_signal.intent.yaml",
    "golden/13_event_stream_intraday.intent.yaml",
)


@dataclass(frozen=True)
class ArtifactCheck:
    path: str
    source: str
    byte_count: int
    line_count: int
    checks: dict[str, str]
    issues: tuple[str, ...]

    @property
    def verdict(self) -> str:
        return "pass" if not self.issues else "fail"

    def to_json(self) -> dict[str, Any]:
        return {
            "path": self.path,
            "source": self.source,
            "byte_count": self.byte_count,
            "line_count": self.line_count,
            "verdict": self.verdict,
            "checks": self.checks,
            "issues": list(self.issues),
        }


def verify_prompt_artifacts(prompt_root: Path, raw_base: str | None = None) -> dict[str, Any]:
    repo_root = Path.cwd()
    prompt_root = prompt_root.resolve()
    prompt_root_rel = prompt_root.relative_to(repo_root).as_posix()
    checks: list[ArtifactCheck] = []

    for path in PYTHON_FILES:
        checks.append(_check_python(path, _read_artifact(repo_root, path, raw_base)))
    for path in PROMPT_MARKDOWN_FILES:
        artifact_path = f"{prompt_root_rel}/{path}"
        checks.append(_check_markdown(artifact_path, _read_artifact(repo_root, artifact_path, raw_base)))
    for path in GOLDEN_YAML_FILES:
        artifact_path = f"{prompt_root_rel}/{path}"
        checks.append(_check_golden_yaml(artifact_path, _read_artifact(repo_root, artifact_path, raw_base)))

    result = "pass" if all(check.verdict == "pass" for check in checks) else "fail"
    return {
        "prompt_artifact_verification": {
            "result": result,
            "prompt_version": PROMPT_VERSION,
            "source": "raw" if raw_base else "local",
            "raw_base": raw_base,
            "artifacts": [check.to_json() for check in checks],
        }
    }


def _read_artifact(repo_root: Path, path: str, raw_base: str | None) -> tuple[str, str]:
    if raw_base:
        url = f"{raw_base.rstrip('/')}/{path}"
        parsed = urlparse(url)
        if parsed.scheme != "https" or parsed.netloc != "raw.githubusercontent.com":
            return "", f"{url} (unsupported raw URL; expected https://raw.githubusercontent.com)"
        try:
            with urlopen(url, timeout=20) as response:  # noqa: S310
                return response.read().decode("utf-8"), url
        except (OSError, URLError) as exc:
            return "", f"{url} ({type(exc).__name__}: {exc})"
    return (repo_root / path).read_text(encoding="utf-8"), path


def _check_python(path: str, artifact: tuple[str, str]) -> ArtifactCheck:
    text, source = artifact
    issues: list[str] = []
    checks: dict[str, str] = {}
    try:
        compile(text, path, "exec")
        checks["compile"] = "pass"
    except SyntaxError as exc:
        checks["compile"] = "fail"
        issues.append(f"Python compile failed: {exc.msg} at line {exc.lineno}")
    return _artifact_check(path, source, text, checks, issues)


def _check_markdown(path: str, artifact: tuple[str, str]) -> ArtifactCheck:
    text, source = artifact
    lines = text.splitlines()
    issues: list[str] = []
    checks: dict[str, str] = {}

    if any(line.startswith("# ") for line in lines):
        checks["h1"] = "pass"
    else:
        checks["h1"] = "fail"
        issues.append("missing H1 heading")
    if f"prompt_system_version: {PROMPT_VERSION}" in text:
        checks["prompt_version"] = "pass"
    else:
        checks["prompt_version"] = "fail"
        issues.append(f"missing prompt version {PROMPT_VERSION}")
    if len(lines) >= MIN_MARKDOWN_LINES:
        checks["minimum_lines"] = "pass"
    else:
        checks["minimum_lines"] = "fail"
        issues.append(f"only {len(lines)} lines; minimum is {MIN_MARKDOWN_LINES}")

    long_lines = [
        f"line {line_number} length {len(line)}"
        for line_number, line in enumerate(lines, start=1)
        if len(line) > MAX_MARKDOWN_LINE_LENGTH
    ]
    if long_lines:
        checks["line_length"] = "fail"
        issues.extend(long_lines)
    else:
        checks["line_length"] = "pass"
    return _artifact_check(path, source, text, checks, issues)


def _check_golden_yaml(path: str, artifact: tuple[str, str]) -> ArtifactCheck:
    text, source = artifact
    issues: list[str] = []
    checks: dict[str, str] = {}
    try:
        loaded = yaml.safe_load(text)
        checks["yaml_parse"] = "pass"
    except yaml.YAMLError as exc:
        checks["yaml_parse"] = "fail"
        return _artifact_check(path, source, text, checks, (f"YAML parse failed: {exc}",))

    if isinstance(loaded, dict):
        checks["root_mapping"] = "pass"
    else:
        checks["root_mapping"] = "fail"
        issues.append("YAML root is not a mapping")
        loaded = {}
    golden_task = loaded.get("golden_task")
    if isinstance(golden_task, dict):
        checks["golden_task_mapping"] = "pass"
    else:
        checks["golden_task_mapping"] = "fail"
        issues.append("missing golden_task mapping")
        golden_task = {}
    if isinstance(golden_task.get("id"), str) and golden_task.get("id"):
        checks["golden_task_id"] = "pass"
    else:
        checks["golden_task_id"] = "fail"
        issues.append("missing golden_task.id")
    if isinstance(golden_task.get("expected"), dict):
        checks["expected_mapping"] = "pass"
    else:
        checks["expected_mapping"] = "fail"
        issues.append("missing expected mapping")
    return _artifact_check(path, source, text, checks, issues)


def _artifact_check(
    path: str,
    source: str,
    text: str,
    checks: dict[str, str],
    issues: list[str] | tuple[str, ...],
) -> ArtifactCheck:
    return ArtifactCheck(
        path=path,
        source=source,
        byte_count=len(text.encode("utf-8")),
        line_count=text.count("\n") + 1 if text else 0,
        checks=checks,
        issues=tuple(issues),
    )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Verify Stage 3C prompt artifacts locally or from raw URLs.")
    parser.add_argument("prompt_root", type=Path)
    parser.add_argument("--raw-base", help="Raw URL base ending at a repo ref, for example .../<sha>")
    args = parser.parse_args(argv)
    result = verify_prompt_artifacts(args.prompt_root, args.raw_base)
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0 if result["prompt_artifact_verification"]["result"] == "pass" else 1


if __name__ == "__main__":
    raise SystemExit(main())
