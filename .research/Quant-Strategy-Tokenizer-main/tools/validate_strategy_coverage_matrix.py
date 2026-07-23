from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import yaml

SCHEMA_VERSION = "qst-strategy-coverage/0.3"
ALLOWED_CLASSIFICATIONS = {
    "supported",
    "partially_supported",
    "custom_token_required",
    "reserved",
    "non_goal",
}
ALLOWED_BENCHMARK_GROUPS = {
    "internal_matrix",
    "external_benchmark",
    "user_submitted",
    "dogfood",
}
REQUIRED_EXTERNAL_CATEGORIES = {
    "indicator_rule",
    "mean_reversion",
    "trend_following",
    "breakout",
    "state_gate",
    "panel_factor",
    "weight_record",
    "custom_signal",
    "custom_model",
    "reserved_event_stream",
    "non_goal_execution",
}
KERNEL_GAP_FIELDS = ("category", "description", "preferred_solution")
MARKET_WEIGHT_FIELDS = ("source_frequency", "implementation_relevance", "user_relevance", "final")
SOURCE_REF_RE = re.compile(r"^docs/reports/external_benchmark_sources\.md#(src-[a-z0-9-]+)$")
SOURCE_HEADING_RE = re.compile(r"^### (src-[a-z0-9-]+)\s*$", re.MULTILINE)
CUSTOM_GOVERNANCE_MANIFEST = Path("tests/coverage_cases/custom_token_governance/custom_token_routes.yaml")
CUSTOM_GOVERNANCE_SCHEMA_VERSION = "qst-custom-token-governance/0.1"
BOUNDARY_MANIFEST = Path("tests/coverage_cases/reserved_non_goal_boundaries/boundary_cases.yaml")
BOUNDARY_SCHEMA_VERSION = "qst-reserved-non-goal-boundary/0.1"
YAML_LOADER = getattr(yaml, "CSafeLoader", yaml.SafeLoader)


@dataclass(frozen=True)
class MatrixIssue:
    severity: str
    code: str
    path: str
    message: str

    def as_dict(self) -> dict[str, str]:
        return {
            "severity": self.severity,
            "code": self.code,
            "path": self.path,
            "message": self.message,
        }


def load_matrix(path: Path) -> dict[str, Any]:
    loaded = _load_yaml_mapping(path)
    if not isinstance(loaded, dict):
        msg = f"{path} must parse to a mapping"
        raise ValueError(msg)
    return loaded


def _load_yaml_mapping(path: Path) -> Any:
    # CSafeLoader is the safe loader implemented by libyaml; fall back to SafeLoader.
    return yaml.load(path.read_text(encoding="utf-8"), Loader=YAML_LOADER)  # noqa: S506


def validation_payload(issues: list[MatrixIssue], summary: dict[str, Any]) -> dict[str, Any]:
    error_count = sum(1 for issue in issues if issue.severity == "error")
    return {
        "strategy_coverage_matrix_validation": {
            "result": "pass" if error_count == 0 else "fail",
            "issue_count": error_count,
            "issues": [issue.as_dict() for issue in issues],
            "summary": summary,
        }
    }


def validate_matrix(
    matrix: dict[str, Any],
    *,
    repo_root: Path | None = None,
) -> tuple[list[MatrixIssue], dict[str, Any]]:
    root = repo_root or Path.cwd()
    issues: list[MatrixIssue] = []

    def add(code: str, path: str, message: str, severity: str = "error") -> None:
        issues.append(MatrixIssue(severity=severity, code=code, path=path, message=message))

    _validate_top_level(matrix, add)
    patterns_raw = matrix.get("patterns")
    patterns = patterns_raw if isinstance(patterns_raw, list) else []
    source_ids = _external_source_ids(root)
    governance_manifest = _load_custom_token_governance_manifest(root)
    governance_routes = _custom_governance_routes(governance_manifest)
    boundary_manifest = _load_reserved_non_goal_boundary_manifest(root)
    boundary_cases = _reserved_non_goal_boundary_cases(boundary_manifest)

    ids: set[str] = set()
    groups: dict[str, int] = {}
    classes: dict[str, int] = {}
    external_categories: set[str] = set()
    dogfood_count = 0

    for index, row_raw in enumerate(patterns):
        path = f"patterns[{index}]"
        if not isinstance(row_raw, dict):
            add("pattern_not_mapping", path, "pattern row must be a mapping")
            continue
        row = row_raw
        row_id = _string(row.get("id"))
        row_path = f"patterns[{row_id or index}]"
        if not row_id:
            add("missing_pattern_id", f"{path}.id", "pattern id is required")
        elif row_id in ids:
            add("duplicate_pattern_id", f"{row_path}.id", f"duplicate pattern id {row_id!r}")
        else:
            ids.add(row_id)

        group = _string(row.get("benchmark_group"))
        classification = _string(row.get("expected_classification"))
        if group not in ALLOWED_BENCHMARK_GROUPS:
            add("invalid_benchmark_group", f"{row_path}.benchmark_group", f"invalid group {group!r}")
        else:
            groups[group] = groups.get(group, 0) + 1
            if group == "external_benchmark":
                category = _string(row.get("category"))
                if category:
                    external_categories.add(category)
                _validate_external_source_ref(row, row_path, source_ids, add)
            if group == "dogfood":
                dogfood_count += 1

        if classification not in ALLOWED_CLASSIFICATIONS:
            add(
                "invalid_classification",
                f"{row_path}.expected_classification",
                f"invalid classification {classification!r}",
            )
        else:
            classes[classification] = classes.get(classification, 0) + 1

        _validate_market_weight(row, row_path, add)
        _validate_classification_contract(row, row_path, classification, add)
        _validate_custom_governance(row, row_path, classification, governance_routes, add)
        _validate_reserved_non_goal_boundary(row, row_path, classification, boundary_cases, add)
        _validate_kernel_gaps(row, row_path, add)

    if groups.get("external_benchmark", 0) < 20:
        add(
            "external_benchmark_count_below_threshold",
            "patterns",
            "external_benchmark must contain at least 20 rows",
        )
    missing_categories = sorted(REQUIRED_EXTERNAL_CATEGORIES - external_categories)
    if missing_categories:
        add(
            "external_benchmark_category_missing",
            "patterns.external_benchmark.category",
            f"missing external categories: {', '.join(missing_categories)}",
        )
    if dogfood_count == 0:
        add("dogfood_missing", "patterns", "at least one dogfood row is required")

    summary = {
        "pattern_count": len(patterns),
        "benchmark_groups": dict(sorted(groups.items())),
        "classifications": dict(sorted(classes.items())),
        "external_categories": sorted(external_categories),
        "dogfood_count": dogfood_count,
    }
    return issues, summary


def load_custom_token_governance_manifest(repo_root: Path | None = None) -> dict[str, Any]:
    """Load the optional PR10 custom-token governance manifest."""
    root = repo_root or Path.cwd()
    return _load_custom_token_governance_manifest(root)


def load_reserved_non_goal_boundary_manifest(repo_root: Path | None = None) -> dict[str, Any]:
    """Load the optional PR11 reserved/non-goal boundary manifest."""
    root = repo_root or Path.cwd()
    return _load_reserved_non_goal_boundary_manifest(root)


def _validate_top_level(matrix: dict[str, Any], add: Any) -> None:
    if matrix.get("schema_version") != SCHEMA_VERSION:
        add(
            "invalid_schema_version",
            "schema_version",
            f"schema_version must be {SCHEMA_VERSION!r}",
        )
    for field in ("metadata", "thresholds", "coverage_policy"):
        if not isinstance(matrix.get(field), dict):
            add("missing_top_level_mapping", field, f"{field} must be present as a mapping")
    if not isinstance(matrix.get("patterns"), list):
        add("missing_patterns_list", "patterns", "patterns must be present as a list")


def _validate_market_weight(row: dict[str, Any], row_path: str, add: Any) -> None:
    weight = row.get("market_weight")
    if not isinstance(weight, dict):
        add("missing_market_weight", f"{row_path}.market_weight", "market_weight is required")
        return
    values: dict[str, int | float] = {}
    for field in MARKET_WEIGHT_FIELDS:
        value = weight.get(field)
        if not isinstance(value, int | float):
            add(
                "invalid_market_weight_field",
                f"{row_path}.market_weight.{field}",
                f"{field} must be numeric",
            )
            return
        values[field] = value
    computed = values["source_frequency"] + values["implementation_relevance"] + values["user_relevance"]
    if computed != values["final"]:
        add(
            "invalid_market_weight_sum",
            f"{row_path}.market_weight.final",
            "source_frequency + implementation_relevance + user_relevance must equal final",
        )


def _validate_classification_contract(
    row: dict[str, Any],
    row_path: str,
    classification: str,
    add: Any,
) -> None:
    evidence = row.get("evidence")
    evidence_map = evidence if isinstance(evidence, dict) else {}
    boundary = row.get("boundary")
    boundary_map = boundary if isinstance(boundary, dict) else {}
    false_supported = row.get("false_supported")
    false_map = false_supported if isinstance(false_supported, dict) else {}

    if classification == "supported":
        required_tokens = evidence_map.get("required_tokens")
        if not _non_empty_list(required_tokens):
            add(
                "supported_missing_required_tokens",
                f"{row_path}.evidence.required_tokens",
                "supported rows must list required tokens",
            )
        examples = evidence_map.get("examples")
        mechanical_status = _string(false_map.get("mechanical_status"))
        if not _non_empty_list(examples) and mechanical_status != "pending":
            add(
                "supported_without_example_not_pending",
                f"{row_path}.false_supported.mechanical_status",
                "supported rows without examples must be explicitly pending",
            )
    elif classification == "custom_token_required":
        custom_route = row.get("custom_token_route")
        route = custom_route if isinstance(custom_route, dict) else {}
        if not _string(route.get("reason")):
            add(
                "custom_route_missing_reason",
                f"{row_path}.custom_token_route.reason",
                "custom-token rows must include a route reason",
            )
        if not _non_empty_list(route.get("input_ports")):
            add(
                "custom_route_missing_input_ports",
                f"{row_path}.custom_token_route.input_ports",
                "custom-token rows must include input ports",
            )
        if not _non_empty_list(route.get("output_ports")):
            add(
                "custom_route_missing_output_ports",
                f"{row_path}.custom_token_route.output_ports",
                "custom-token rows must include output ports",
            )
    elif classification == "reserved":
        if not _string(boundary_map.get("reserved_reason")):
            add(
                "reserved_missing_reserved_reason",
                f"{row_path}.boundary.reserved_reason",
                "reserved rows must include reserved_reason",
            )
    elif classification == "non_goal":
        if not _string(boundary_map.get("non_goal_reason")):
            add(
                "non_goal_missing_non_goal_reason",
                f"{row_path}.boundary.non_goal_reason",
                "non_goal rows must include non_goal_reason",
            )


def _validate_custom_governance(
    row: dict[str, Any],
    row_path: str,
    classification: str,
    governance_routes: dict[str, dict[str, Any]],
    add: Any,
) -> None:
    if classification != "custom_token_required":
        return
    row_id = _string(row.get("id"))
    route = row.get("custom_token_route")
    route_map = route if isinstance(route, dict) else {}
    governance = governance_routes.get(row_id)
    if not isinstance(governance, dict):
        add(
            "custom_route_governance_missing",
            f"{row_path}.custom_token_route",
            "custom-token rows must be listed in the custom-token governance manifest",
        )
        return
    if not _string(governance.get("reason")):
        add(
            "custom_governance_missing_reason",
            f"{row_path}.custom_token_governance.reason",
            "custom-token governance route must include a reason",
        )
    if not _non_empty_list(governance.get("input_ports")):
        add(
            "custom_governance_missing_input_ports",
            f"{row_path}.custom_token_governance.input_ports",
            "custom-token governance route must include input ports",
        )
    if not _non_empty_list(governance.get("output_ports")):
        add(
            "custom_governance_missing_output_ports",
            f"{row_path}.custom_token_governance.output_ports",
            "custom-token governance route must include output ports",
        )
    if governance.get("remain_custom_route") is not True:
        add(
            "custom_governance_not_marked_active",
            f"{row_path}.custom_token_governance.remain_custom_route",
            "active custom-token routes must be explicitly marked remain_custom_route: true",
        )
    if governance.get("execution_boundary") != "not_approved_not_granted_not_executed":
        add(
            "custom_governance_execution_boundary_missing",
            f"{row_path}.custom_token_governance.execution_boundary",
            "custom-token governance must preserve the no approval/grant/execution boundary",
        )
    if _non_empty_list(route_map.get("input_ports")) and governance.get("input_ports") != route_map.get("input_ports"):
        add(
            "custom_governance_input_ports_mismatch",
            f"{row_path}.custom_token_governance.input_ports",
            "custom-token governance input ports must match matrix route ports",
        )
    if _non_empty_list(route_map.get("output_ports")) and governance.get("output_ports") != route_map.get("output_ports"):
        add(
            "custom_governance_output_ports_mismatch",
            f"{row_path}.custom_token_governance.output_ports",
            "custom-token governance output ports must match matrix route ports",
        )


def _validate_reserved_non_goal_boundary(
    row: dict[str, Any],
    row_path: str,
    classification: str,
    boundary_cases: dict[str, dict[str, Any]],
    add: Any,
) -> None:
    if classification not in {"reserved", "non_goal"}:
        return
    row_id = _string(row.get("id"))
    boundary = row.get("boundary")
    boundary_map = boundary if isinstance(boundary, dict) else {}
    gaps = row.get("gaps")
    gaps_map = gaps if isinstance(gaps, dict) else {}
    case = boundary_cases.get(row_id)
    if not isinstance(case, dict):
        add(
            "reserved_non_goal_boundary_case_missing",
            f"{row_path}.boundary",
            "reserved/non-goal rows must be listed in the boundary manifest",
        )
        return
    if case.get("classification") != classification:
        add(
            "reserved_non_goal_boundary_classification_mismatch",
            f"{row_path}.boundary.classification",
            "boundary manifest classification must match matrix classification",
        )
    if not _string(case.get("diagnostic_class")):
        add(
            "reserved_non_goal_boundary_diagnostic_missing",
            f"{row_path}.boundary.diagnostic_class",
            "boundary manifest case must include diagnostic_class",
        )
    if not _string(case.get("boundary_class")):
        add(
            "reserved_non_goal_boundary_class_missing",
            f"{row_path}.boundary.boundary_class",
            "boundary manifest case must include boundary_class",
        )
    forbidden = case.get("must_not_reclassify_as")
    if not isinstance(forbidden, list) or "supported" not in forbidden:
        add(
            "reserved_non_goal_boundary_forbidden_reclassification_missing",
            f"{row_path}.boundary.must_not_reclassify_as",
            "boundary manifest case must forbid supported reclassification",
        )
    if classification == "reserved":
        if not _string(boundary_map.get("reserved_reason")):
            add(
                "reserved_boundary_reason_missing",
                f"{row_path}.boundary.reserved_reason",
                "reserved boundary manifest case requires matrix reserved_reason",
            )
        if case.get("future_stage_allowed") is not True:
            add(
                "reserved_boundary_future_stage_missing",
                f"{row_path}.boundary.future_stage_allowed",
                "reserved rows must explicitly mark future_stage_allowed: true",
            )
    if classification == "non_goal":
        if not _string(boundary_map.get("non_goal_reason")):
            add(
                "non_goal_boundary_reason_missing",
                f"{row_path}.boundary.non_goal_reason",
                "non-goal boundary manifest case requires matrix non_goal_reason",
            )
        if case.get("future_stage_allowed") is not False:
            add(
                "non_goal_boundary_future_stage_allowed",
                f"{row_path}.boundary.future_stage_allowed",
                "non-goal rows must explicitly mark future_stage_allowed: false",
            )
        for forbidden_class in ("partially_supported", "custom_token_required"):
            if forbidden_class not in forbidden:
                add(
                    "non_goal_boundary_forbidden_reclassification_missing",
                    f"{row_path}.boundary.must_not_reclassify_as",
                    "non-goal rows must forbid partial/custom weakening",
                )
                break
    missing_types = gaps_map.get("missing_types")
    case_missing_types = case.get("missing_types")
    if isinstance(missing_types, list) and missing_types:
        if not isinstance(case_missing_types, list) or set(case_missing_types) != set(missing_types):
            add(
                "reserved_non_goal_boundary_missing_types_mismatch",
                f"{row_path}.boundary.missing_types",
                "boundary manifest missing_types must match matrix gap missing_types",
            )


def _validate_kernel_gaps(row: dict[str, Any], row_path: str, add: Any) -> None:
    gaps = row.get("gaps")
    if not isinstance(gaps, dict):
        return
    kernel_gaps = gaps.get("kernel_gaps")
    if kernel_gaps in (None, []):
        return
    if not isinstance(kernel_gaps, list):
        add("kernel_gaps_not_list", f"{row_path}.gaps.kernel_gaps", "kernel_gaps must be a list")
        return
    for gap_index, gap_raw in enumerate(kernel_gaps):
        gap_path = f"{row_path}.gaps.kernel_gaps[{gap_index}]"
        if not isinstance(gap_raw, dict):
            add("kernel_gap_not_mapping", gap_path, "kernel gap must be a mapping")
            continue
        for field in KERNEL_GAP_FIELDS:
            if not _string(gap_raw.get(field)):
                add(
                    "kernel_gap_missing_field",
                    f"{gap_path}.{field}",
                    f"kernel gap must include {field}",
                )


def _validate_external_source_ref(
    row: dict[str, Any],
    row_path: str,
    source_ids: set[str],
    add: Any,
) -> None:
    evidence_refs: list[str] = []
    market_weight = row.get("market_weight")
    if isinstance(market_weight, dict) and isinstance(market_weight.get("evidence"), list):
        evidence_refs.extend(str(item) for item in market_weight["evidence"])
    evidence = row.get("evidence")
    if isinstance(evidence, dict):
        for value in evidence.values():
            if isinstance(value, list):
                evidence_refs.extend(str(item) for item in value)
    matched = [SOURCE_REF_RE.match(ref) for ref in evidence_refs]
    source_refs = [match.group(1) for match in matched if match]
    if not source_refs:
        add(
            "external_source_ref_missing",
            f"{row_path}.market_weight.evidence",
            "external benchmark rows must reference docs/reports/external_benchmark_sources.md#src-*",
        )
        return
    for source_ref in source_refs:
        if source_ref not in source_ids:
            add(
                "external_source_anchor_missing",
                f"{row_path}.market_weight.evidence",
                f"source anchor {source_ref!r} is not present in external_benchmark_sources.md",
            )


def _external_source_ids(repo_root: Path) -> set[str]:
    path = repo_root / "docs" / "reports" / "external_benchmark_sources.md"
    if not path.exists():
        return set()
    return set(SOURCE_HEADING_RE.findall(path.read_text(encoding="utf-8")))


def _load_custom_token_governance_manifest(repo_root: Path) -> dict[str, Any]:
    path = repo_root / CUSTOM_GOVERNANCE_MANIFEST
    if not path.exists():
        return {}
    loaded = _load_yaml_mapping(path)
    return loaded if isinstance(loaded, dict) else {}


def _custom_governance_routes(manifest: dict[str, Any]) -> dict[str, dict[str, Any]]:
    if manifest.get("schema_version") != CUSTOM_GOVERNANCE_SCHEMA_VERSION:
        return {}
    routes = manifest.get("routes")
    if not isinstance(routes, list):
        return {}
    result: dict[str, dict[str, Any]] = {}
    for route in routes:
        if not isinstance(route, dict):
            continue
        pattern_id = _string(route.get("pattern_id"))
        if pattern_id:
            result[pattern_id] = route
    return result


def _load_reserved_non_goal_boundary_manifest(repo_root: Path) -> dict[str, Any]:
    path = repo_root / BOUNDARY_MANIFEST
    if not path.exists():
        return {}
    loaded = _load_yaml_mapping(path)
    return loaded if isinstance(loaded, dict) else {}


def _reserved_non_goal_boundary_cases(manifest: dict[str, Any]) -> dict[str, dict[str, Any]]:
    if manifest.get("schema_version") != BOUNDARY_SCHEMA_VERSION:
        return {}
    cases = manifest.get("cases")
    if not isinstance(cases, list):
        return {}
    result: dict[str, dict[str, Any]] = {}
    for case in cases:
        if not isinstance(case, dict):
            continue
        pattern_id = _string(case.get("pattern_id"))
        if pattern_id:
            result[pattern_id] = case
    return result


def _string(value: object) -> str:
    return value if isinstance(value, str) else ""


def _non_empty_list(value: object) -> bool:
    return isinstance(value, list) and len(value) > 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Validate a QST strategy coverage matrix.")
    parser.add_argument("matrix", type=Path)
    args = parser.parse_args(argv)

    try:
        matrix = load_matrix(args.matrix)
        issues, summary = validate_matrix(matrix, repo_root=Path.cwd())
    except Exception as exc:
        issues = [
            MatrixIssue(
                severity="error",
                code="matrix_load_failed",
                path=str(args.matrix),
                message=str(exc),
            )
        ]
        summary = {}

    payload = validation_payload(issues, summary)
    print(json.dumps(payload, indent=2, sort_keys=True))
    result = payload["strategy_coverage_matrix_validation"]["result"]
    return 0 if result == "pass" else 1


if __name__ == "__main__":
    sys.exit(main())
