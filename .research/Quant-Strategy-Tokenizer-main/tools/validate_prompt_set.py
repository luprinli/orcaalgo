from __future__ import annotations

import json
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import yaml

PROMPT_VERSION = "qst-stage-3c-v0.3.2.3"
ALLOWED_CLASSIFICATIONS = {
    "supported",
    "partially_supported",
    "reserved",
    "custom_token_required",
    "non_goal",
}

REQUIRED_FILES = (
    "README.md",
    "INDEX.md",
    "core/00_FOUNDATION.md",
    "core/01_REPO_CONTEXT_PROTOCOL.md",
    "core/02_INPUT_SECURITY.md",
    "core/03_BEHAVIOR_CORE.md",
    "core/04_REPORT_SCHEMA.md",
    "core/05_ESCALATION_PROTOCOL.md",
    "core/06_HANDOFF_PROTOCOL.md",
    "load_profiles/PROFILE_MINIMAL.md",
    "load_profiles/PROFILE_STRATEGY_AUTHORING.md",
    "load_profiles/PROFILE_TOKEN_WORK.md",
    "load_profiles/PROFILE_HASH_CANONICAL.md",
    "load_profiles/PROFILE_CUSTOM_RUNTIME.md",
    "load_profiles/PROFILE_DOCS_PROMPTS.md",
    "load_profiles/PROFILE_FULL_AUDIT.md",
    "schemas/STRATEGY_INTENT_SCHEMA.md",
    "schemas/VOCABULARY_SNAPSHOT_SCHEMA.md",
    "schemas/REPO_CONTEXT_SCHEMA.md",
    "schemas/MODULE_REPORT_SCHEMA.md",
    "schemas/GOLDEN_TASK_SCHEMA.md",
    "validation/VALIDATE_PROMPT_SET.md",
    "validation/PROMPT_GOLDEN_TEST_PROTOCOL.md",
    "construction/INSTALLATION_PLAN.md",
    "construction/REPLACEMENT_PLAN.md",
    "construction/ROADMAP_BY_MODULE.md",
    "construction/STAGE_3C_PROMPT_ACCEPTANCE.md",
    "construction/STAGE_3C_PROMPT_ACCEPTANCE_EVIDENCE.md",
)

STALE_PATTERNS = (
    (r"\bquant_strategy_tokenizer\b", "stale_import_package"),
    (r"\.qst\.yaml\b", "stale_strategy_suffix"),
    (r"\.qstpkg\b", "stale_package_suffix"),
    (r"\.qsp\b", "stale_package_suffix"),
    (r"\bQST_CURRENT_STATE\b", "stale_current_state"),
    (r"\bcurrent_coverage\b", "hardcoded_current_state"),
    (r"\bhead_commit\b", "hardcoded_current_state"),
    (r"\baccepted:\s*~", "stale_prompt_state"),
    (r"\breserved_design:\s*0\b", "stale_prompt_state"),
    (r"\b(graph_hash|param_hash|instance_hash)\s*=\s*sha256:", "hardcoded_hash_truth"),
    (r"sha256:[0-9a-f]{64}", "hardcoded_hash_truth"),
)

REFERENCE_PATTERN = re.compile(r"`((?:core|load_profiles|readers|tasks|schemas|golden|validation|construction)/[^`]+?)`")
MAX_MARKDOWN_LINE_LENGTH = 240
MIN_MARKDOWN_LINES = 5

SEMANTIC_ANCHORS: dict[str, tuple[tuple[str, tuple[str, ...]], ...]] = {
    "core/01_REPO_CONTEXT_PROTOCOL.md": (
        ("git status", ("git status",)),
        ("git rev-parse", ("git rev-parse",)),
        ("pyproject", ("pyproject.toml",)),
        ("vocabulary command", ("qst vocabulary",)),
        ("validation command", ("qst validate",)),
        ("hash command", ("qst hash",)),
        ("canonical command", ("canonicalize",)),
        ("repo context output", ("repo_context",)),
        ("coverage matrix", ("strategy_coverage_matrix", "strategy_coverage_matrix.yaml")),
        ("coverage report tool", ("report_strategy_coverage.py",)),
        ("coverage report check", ("--check",)),
        ("dogfood evidence", ("dogfood_target_set", "dogfood_case")),
    ),
    "tasks/CLASSIFY_STRATEGY_INTENT.md": (
        ("supported classification", ("supported",)),
        ("partially supported classification", ("partially_supported",)),
        ("custom token classification", ("custom_token_required",)),
        ("reserved classification", ("reserved",)),
        ("non-goal classification", ("non_goal",)),
        ("broker boundary", ("broker",)),
        ("exchange boundary", ("exchange",)),
        ("live execution boundary", ("live execution",)),
        ("event stream boundary", ("eventstream", "event stream")),
        ("coverage matrix", ("strategy_coverage_matrix",)),
        ("external benchmark", ("external_benchmark",)),
        ("dogfood case", ("dogfood_case",)),
        ("false-supported risk", ("false_supported_rate",)),
    ),
    "tasks/SELECT_TOKENS.md": (
        ("selected token output", ("selected_token_ref",)),
        ("maturity output", ("maturity",)),
        ("execution support output", ("execution_support",)),
        ("rejected candidates", ("rejected_candidates",)),
        ("missing tokens", ("missing_tokens",)),
        ("coverage row evidence", ("coverage_row", "strategy_coverage_matrix")),
        ("kernel gap routing", ("kernel_gap",)),
        ("custom-token pressure", ("custom_token_route_share",)),
        ("reserved/non-goal rejection", ("reserved", "non_goal")),
    ),
    "tasks/AUTHOR_GKR_STRATEGY.md": (
        ("strategy source", (".gkr.yaml",)),
        ("node plan", ("node_plan",)),
        ("validation command", ("qst validate",)),
        ("hash command", ("qst hash",)),
        ("canonical command", ("canonicalize",)),
        ("repair route", ("repair_gkr_diagnostics",)),
        ("dogfood candidate evidence", ("dogfood candidate", "dogfood_case")),
        ("record-layer boundary", ("record-layer evidence",)),
        ("no runtime claim", ("runtime/backtest/profitability",)),
    ),
    "tasks/REPAIR_GKR_DIAGNOSTICS.md": (
        ("schema diagnostic", ("schema_error",)),
        ("unknown token diagnostic", ("unknown_token",)),
        ("port diagnostic", ("port_error",)),
        ("type diagnostic", ("type_error",)),
        ("temporal diagnostic", ("temporal_error",)),
        ("attempt limit", ("3 attempts", "max 3", "maximum 3")),
        ("missing token route", ("missing_token",)),
        ("kernel gap route", ("kernel_gap",)),
        ("reserved route", ("reserved_design",)),
        ("non-goal route", ("non_goal_runtime",)),
        ("custom-token route", ("custom_token_required",)),
        ("no fake time series", ("time-series fake", "fake with time series")),
    ),
    "tasks/PROFILE_GATE_REVIEW.md": (
        ("custom-token share", ("custom_token_route_share",)),
        ("false-supported rate", ("false_supported_rate",)),
        ("reserved boundary", ("reserved",)),
        ("non-goal boundary", ("non_goal",)),
        ("coverage report", ("strategy_coverage_report",)),
    ),
    "tasks/CUSTOM_TOKEN_ROUTING.md": (
        ("verify boundary", ("verify",)),
        ("approve boundary", ("approve",)),
        ("grant boundary", ("grant",)),
        ("execute boundary", ("execute",)),
        ("no execute boundary", ("must not execute",)),
        ("no import boundary", ("must not import", "do not import")),
        ("custom-token share", ("custom_token_route_share",)),
        ("route cap", ("custom token route cap", "custom-token route cap")),
        ("coverage report", ("strategy_coverage_report",)),
    ),
}


@dataclass(frozen=True)
class Check:
    id: str
    name: str
    issues: tuple[str, ...]

    @property
    def result(self) -> str:
        return "pass" if not self.issues else "fail"

    def to_json(self) -> dict[str, Any]:
        return {
            "id": self.id,
            "name": self.name,
            "result": self.result,
            "issues": list(self.issues),
        }


def validate_prompt_set(root: Path) -> dict[str, Any]:
    root = root.resolve()
    checks = [
        _check_required_files(root),
        _check_version_consistency(root),
        _check_markdown_readability(root),
        _check_content_completeness(root),
        _check_stale_information(root),
        _check_cross_references(root),
        _check_load_profiles(root),
        _check_classification_vocabulary(root),
        _check_custom_token_separation(root),
        _check_reserved_design_rule(root),
        _check_golden_yaml_schema(root),
        _check_golden_tasks(root),
        _check_semantic_anchors(root),
        _check_operator_manual_boundary(root),
    ]
    result = "pass" if all(check.result == "pass" for check in checks) else "fail"
    return {
        "prompt_validation": {
            "version": PROMPT_VERSION,
            "result": result,
            "checks": [check.to_json() for check in checks],
        }
    }


def _check_required_files(root: Path) -> Check:
    issues = [f"missing required file: {path}" for path in REQUIRED_FILES if not (root / path).is_file()]
    return Check("required_files", "required prompt files exist", tuple(issues))


def _check_version_consistency(root: Path) -> Check:
    issues: list[str] = []
    for path in _prompt_markdown_files(root):
        text = path.read_text(encoding="utf-8")
        if f"prompt_system_version: {PROMPT_VERSION}" not in text:
            issues.append(f"{_rel(root, path)} missing prompt_system_version {PROMPT_VERSION}")
    return Check("version", "prompt version consistency", tuple(issues))


def _check_markdown_readability(root: Path) -> Check:
    issues: list[str] = []
    for path in _prompt_markdown_files(root):
        lines = path.read_text(encoding="utf-8").splitlines()
        rel_path = _rel(root, path)
        if not any(line.startswith("# ") for line in lines):
            issues.append(f"{rel_path} missing H1 heading")
        if len(lines) < MIN_MARKDOWN_LINES:
            issues.append(f"{rel_path} appears compressed: only {len(lines)} lines")
        for line_number, line in enumerate(lines, start=1):
            if len(line) > MAX_MARKDOWN_LINE_LENGTH:
                issues.append(
                    f"{rel_path}:{line_number} line length {len(line)} exceeds {MAX_MARKDOWN_LINE_LENGTH}"
                )
    return Check("markdown_readability", "Markdown prompts are readable multi-line files", tuple(issues))


def _check_content_completeness(root: Path) -> Check:
    issues: list[str] = []
    groups: tuple[tuple[str, tuple[str, ...], int], ...] = (
        ("core", ("## Purpose", "## Operating Rules", "## Required Output"), 90),
        ("load_profiles", ("## Use When", "## Load Order", "## Stop Conditions", "## Output"), 90),
        ("readers", ("## Purpose", "## Read", "## Extract", "## Report"), 90),
        ("tasks", ("## Use When", "## Inputs", "## Procedure", "## Output", "## Guardrails"), 90),
        ("schemas", ("## Purpose", "## Required Fields", "## Validation Rules", "## Output"), 80),
        ("validation", ("## Purpose", "## Required Checks", "## Execution", "## Output"), 80),
    )
    for directory, required_sections, minimum_words in groups:
        for path in sorted((root / directory).glob("*.md")):
            text = path.read_text(encoding="utf-8")
            rel_path = _rel(root, path)
            for section in required_sections:
                if section not in text:
                    issues.append(f"{rel_path} missing content section {section}")
            word_count = len(re.findall(r"[A-Za-z0-9_]+", text))
            if word_count < minimum_words:
                issues.append(f"{rel_path} has only {word_count} words; minimum is {minimum_words}")
    return Check("content_completeness", "prompt files contain required sections and useful content", tuple(issues))


def _check_stale_information(root: Path) -> Check:
    issues: list[str] = []
    for path in _prompt_text_files(root):
        text = path.read_text(encoding="utf-8")
        for pattern, code in STALE_PATTERNS:
            if re.search(pattern, text):
                issues.append(f"{_rel(root, path)} contains {code}: {pattern}")
    return Check("stale_information", "no stale current-state or hash truth", tuple(issues))


def _check_cross_references(root: Path) -> Check:
    issues: list[str] = []
    for path in _prompt_markdown_files(root):
        for ref in REFERENCE_PATTERN.findall(path.read_text(encoding="utf-8")):
            if not (root / ref).is_file():
                issues.append(f"{_rel(root, path)} references missing file {ref}")
    return Check("cross_references", "cross-reference integrity", tuple(sorted(issues)))


def _check_load_profiles(root: Path) -> Check:
    issues: list[str] = []
    for path in sorted((root / "load_profiles").glob("*.md")):
        refs = REFERENCE_PATTERN.findall(path.read_text(encoding="utf-8"))
        if not refs:
            issues.append(f"{_rel(root, path)} does not list load files")
        for ref in refs:
            if not (root / ref).is_file():
                issues.append(f"{_rel(root, path)} load reference missing: {ref}")
    return Check("load_profiles", "load profile file existence", tuple(sorted(issues)))


def _check_classification_vocabulary(root: Path) -> Check:
    issues: list[str] = []
    classifying_files = [
        root / "tasks" / "CLASSIFY_STRATEGY_INTENT.md",
        root / "schemas" / "STRATEGY_INTENT_SCHEMA.md",
    ]
    for path in classifying_files:
        text = path.read_text(encoding="utf-8") if path.is_file() else ""
        for classification in sorted(ALLOWED_CLASSIFICATIONS):
            if classification not in text:
                issues.append(f"{_rel(root, path)} missing classification {classification}")
    for path, task in _golden_tasks(root):
        classification = _nested(task, ("golden_task", "expected", "classification"))
        if classification is not None and classification not in ALLOWED_CLASSIFICATIONS:
            issues.append(f"{_rel(root, path)} has invalid classification {classification!r}")
    return Check("classification_vocabulary", "classification vocabulary consistency", tuple(issues))


def _check_custom_token_separation(root: Path) -> Check:
    path = root / "tasks" / "CUSTOM_TOKEN_ROUTING.md"
    text = path.read_text(encoding="utf-8") if path.is_file() else ""
    required = ("verify", "approve", "execute", "must not execute code")
    issues = [f"{_rel(root, path)} missing custom token boundary phrase: {item}" for item in required if item not in text]
    return Check("custom_token_separation", "custom token verify/approve/execute separation", tuple(issues))


def _check_reserved_design_rule(root: Path) -> Check:
    candidates = [
        root / "core" / "00_FOUNDATION.md",
        root / "tasks" / "SELECT_TOKENS.md",
        root / "tasks" / "PROFILE_GATE_REVIEW.md",
    ]
    joined = "\n".join(path.read_text(encoding="utf-8") for path in candidates if path.is_file())
    issues = []
    if "reserved-design" not in joined and "reserved design" not in joined:
        issues.append("reserved-design rule is not stated")
    if "executable" not in joined:
        issues.append("reserved-design non-executable boundary is not stated")
    return Check("reserved_design", "reserved design cannot be executable", tuple(issues))


def _check_golden_yaml_schema(root: Path) -> Check:
    tasks, load_issues = _load_golden_tasks(root)
    issues = list(load_issues)
    for path, task in tasks:
        payload = task.get("golden_task")
        if not isinstance(payload, dict):
            issues.append(f"{_rel(root, path)} missing golden_task mapping")
            continue
        if not isinstance(payload.get("id"), str) or not payload["id"]:
            issues.append(f"{_rel(root, path)} missing string golden_task.id")
        expected = payload.get("expected")
        if not isinstance(expected, dict):
            issues.append(f"{_rel(root, path)} missing expected mapping")
        elif expected.get("classification") not in ALLOWED_CLASSIFICATIONS:
            issues.append(f"{_rel(root, path)} invalid or missing expected.classification")
    return Check("golden_yaml_schema", "golden YAML parses and follows minimum schema", tuple(issues))


def _check_golden_tasks(root: Path) -> Check:
    issues: list[str] = []
    complete: list[str] = []
    required_complete = {
        "01_ema_cross": "supported",
        "12_custom_token_kalman_signal": "custom_token_required",
        "13_event_stream_intraday": "reserved",
    }
    tasks, load_issues = _load_golden_tasks(root)
    issues.extend(load_issues)
    for path, task in tasks:
        payload = task.get("golden_task", {}) if isinstance(task, dict) else {}
        task_id = payload.get("id")
        if not task_id:
            issues.append(f"{_rel(root, path)} missing golden_task.id")
            continue
        status = payload.get("status", "complete")
        classification = _nested(task, ("golden_task", "expected", "classification"))
        if classification not in ALLOWED_CLASSIFICATIONS:
            issues.append(f"{_rel(root, path)} invalid or missing classification")
        if status != "skeleton":
            missing = [
                field
                for field in (
                    "intent",
                    "expected",
                    "forbidden_behavior",
                    "acceptance",
                )
                if field not in payload
            ]
            if missing:
                issues.append(f"{_rel(root, path)} incomplete golden task: {', '.join(missing)}")
            else:
                complete.append(str(task_id))
    for task_id, classification in required_complete.items():
        if task_id not in complete:
            issues.append(f"missing complete golden task {task_id}")
        actual = next(
            (
                _nested(task, ("golden_task", "expected", "classification"))
                for _, task in tasks
                if _nested(task, ("golden_task", "id")) == task_id
            ),
            None,
        )
        if actual != classification:
            issues.append(f"golden task {task_id} classification {actual!r} != {classification!r}")
    if len(complete) < 3:
        issues.append("fewer than 3 complete golden tasks")
    return Check("golden_tasks", "golden task minimum", tuple(issues))


def _check_operator_manual_boundary(root: Path) -> Check:
    agent_dir = root.parent.parent
    operator_manual = agent_dir / "operator_manual.md"
    readme = agent_dir / "README.md"
    if not operator_manual.exists() or not readme.exists():
        return Check("operator_manual", "operator manual is not active prompt", ())
    text = readme.read_text(encoding="utf-8").lower()
    issues = []
    if "operator_manual.md" in text and "active prompt system" in text:
        issues.append("operator_manual.md appears in active prompt system wording")
    return Check("operator_manual", "operator manual is reference-only", tuple(issues))


def _check_semantic_anchors(root: Path) -> Check:
    issues: list[str] = []
    for rel_path, groups in SEMANTIC_ANCHORS.items():
        path = root / rel_path
        if not path.is_file():
            issues.append(f"{rel_path} missing semantic anchor target file")
            continue
        text = path.read_text(encoding="utf-8").lower()
        for group_name, anchors in groups:
            if not any(anchor.lower() in text for anchor in anchors):
                issues.append(f"{rel_path} missing semantic anchor group {group_name}")
    return Check("semantic_anchors", "operational semantic anchors are present", tuple(issues))


def _prompt_markdown_files(root: Path) -> list[Path]:
    return sorted(path for path in root.rglob("*.md") if path.is_file())


def _prompt_text_files(root: Path) -> list[Path]:
    return sorted(path for path in root.rglob("*") if path.is_file() and path.suffix in {".md", ".yaml", ".yml"})


def _golden_tasks(root: Path) -> list[tuple[Path, dict[str, Any]]]:
    return _load_golden_tasks(root)[0]


def _load_golden_tasks(root: Path) -> tuple[list[tuple[Path, dict[str, Any]]], list[str]]:
    tasks: list[tuple[Path, dict[str, Any]]] = []
    issues: list[str] = []
    for path in sorted((root / "golden").glob("*.yaml")):
        try:
            loaded = yaml.safe_load(path.read_text(encoding="utf-8"))
        except yaml.YAMLError as exc:
            issues.append(f"{_rel(root, path)} YAML parse error: {exc}")
            continue
        if not isinstance(loaded, dict):
            issues.append(f"{_rel(root, path)} YAML root must be a mapping")
            loaded = {}
        tasks.append((path, loaded))
    return tasks, issues


def _nested(value: dict[str, Any], keys: tuple[str, ...]) -> Any:
    current: Any = value
    for key in keys:
        if not isinstance(current, dict):
            return None
        current = current.get(key)
    return current


def _rel(root: Path, path: Path) -> str:
    try:
        return path.resolve().relative_to(root).as_posix()
    except ValueError:
        return path.as_posix()


def main(argv: list[str] | None = None) -> int:
    args = sys.argv[1:] if argv is None else argv
    if len(args) != 1:
        print("usage: python tools/validate_prompt_set.py <prompt-root>", file=sys.stderr)
        return 2
    result = validate_prompt_set(Path(args[0]))
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0 if result["prompt_validation"]["result"] == "pass" else 1


if __name__ == "__main__":
    raise SystemExit(main())
