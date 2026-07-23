from __future__ import annotations

import argparse
import json
import sys
from collections import Counter, defaultdict
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from validate_strategy_coverage_matrix import (
    load_custom_token_governance_manifest,
    load_matrix,
    load_reserved_non_goal_boundary_manifest,
    validate_matrix,
    validation_payload,
)

DOGFOOD_MVP_TARGET = 1
DOGFOOD_PUBLICATION_TARGET = 5
HEADLINE_METRIC = "routable_record_coverage_raw"
HEADLINE_METRIC_LABEL = "measured strategy record-layer raw routable coverage"
CORE_RULE_ROW_IDS = {
    "int_020_macd_trend",
    "int_021_atr_filter",
    "int_022_linear_regression_slope",
    "int_081_signal_composition",
    "int_082_decision_long_short_rule",
    "int_083_entry_exit_gate_record",
    "int_084_beta_residual_timeseries",
    "int_085_donchian_volatility_rule",
}
PANEL_FACTOR_WEIGHT_ROW_IDS = {
    "int_041_inverse_vol_weight",
    "int_049_sector_neutral_rank",
    "int_050_beta_neutral_signal",
    "int_086_panel_alias_records",
    "int_087_equal_rank_weight_records",
    "int_088_group_neutral_weight_record",
    "int_089_inverse_volatility_weight_record",
    "int_090_weight_vol_target_wrapper",
}
STATE_GATE_RISK_ROW_IDS = {
    "int_027_min_hold_gate",
    "int_028_max_hold_gate",
    "int_029_trailing_stop_record",
    "int_030_stop_loss_record",
    "int_031_take_profit_record",
    "int_032_rebalance_band",
    "int_035_exposure_cap",
    "int_055_volatility_regime_gate",
    "int_056_drawdown_gate",
    "int_091_state_hold_gate_records",
    "int_092_stop_take_profit_risk_records",
    "int_093_trailing_drawdown_risk_records",
    "int_094_volatility_regime_time_window_records",
    "int_095_rebalance_exposure_turnover_records",
}


@dataclass(frozen=True)
class CheckIssue:
    code: str
    message: str

    def as_dict(self) -> dict[str, str]:
        return {"code": self.code, "message": self.message}


def build_report(matrix: dict[str, Any], *, repo_root: Path | None = None) -> dict[str, Any]:
    validation_issues, validation_summary = validate_matrix(matrix, repo_root=repo_root)
    patterns = [row for row in matrix.get("patterns", []) if isinstance(row, dict)]
    frontier_patterns = [
        row
        for row in patterns
        if row.get("benchmark_group") != "dogfood" and row.get("status", "active") == "active"
    ]
    dogfood_patterns = [row for row in patterns if row.get("benchmark_group") == "dogfood"]

    total_weight = _weight_sum(frontier_patterns)
    supported = _by_class(frontier_patterns, "supported")
    partial = _by_class(frontier_patterns, "partially_supported")
    custom = _by_class(frontier_patterns, "custom_token_required")
    mechanically_reviewed_supported = [
        row
        for row in supported
        if _false_supported(row).get("mechanical_status") == "pass"
    ]

    custom_discount = _custom_discount(matrix)
    direct_weight = _weight_sum(mechanically_reviewed_supported)
    supported_weight = _weight_sum(supported)
    partial_weight = _weight_sum(partial)
    custom_weight = _weight_sum(custom)
    routable_raw_weight = supported_weight + partial_weight + custom_weight
    routable_discounted_weight = supported_weight + partial_weight + (custom_weight * custom_discount)
    routable_denominator = max(routable_raw_weight, 1)
    supported_denominator = max(len(supported), 1)

    benchmark_groups = _benchmark_group_summary(patterns)
    by_classification = _classification_summary(patterns)
    kernel_gaps = _kernel_gap_records(frontier_patterns)
    missing_tokens = _missing_token_records(frontier_patterns)

    metrics = {
        "direct_builtin_coverage": _ratio(direct_weight, total_weight),
        "routable_record_coverage_raw": _ratio(routable_raw_weight, total_weight),
        "routable_record_coverage_discounted": _ratio(routable_discounted_weight, total_weight),
        "custom_token_route_share": _ratio(custom_weight, routable_denominator),
        "false_supported_rate_mechanical": _status_rate(supported, "mechanical_status", "fail", supported_denominator),
        "false_supported_rate_semantic": _status_rate(supported, "semantic_status", "fail", supported_denominator),
        "false_supported_rate_boundary": _status_rate(supported, "boundary_status", "fail", supported_denominator),
        "boundary_false_supported_count": _status_count(supported, "boundary_status", "fail"),
        "kernel_gap_count": len(kernel_gaps),
        "token_bloat_index": _ratio(len({record["name"] for record in missing_tokens}), len(frontier_patterns)),
    }

    report = {
        "coverage_frontier": {
            "schema_version": matrix.get("schema_version"),
            "pattern_count": len(patterns),
            "frontier_pattern_count": len(frontier_patterns),
            "dogfood_pattern_count": len(dogfood_patterns),
            "total_weight": total_weight,
            "benchmark_groups": benchmark_groups,
            "by_classification": by_classification,
            "metrics": metrics,
            "dogfood": _dogfood_summary(dogfood_patterns),
            "dogfood_target": _dogfood_target_summary(dogfood_patterns),
            "custom_token_governance": _custom_token_governance_summary(
                patterns,
                matrix,
                repo_root=repo_root,
            ),
            "reserved_non_goal_boundary": _reserved_non_goal_boundary_summary(
                patterns,
                repo_root=repo_root,
            ),
            "core_rule_token_batch": _core_rule_summary(frontier_patterns),
            "panel_factor_weight_batch": _panel_factor_weight_summary(frontier_patterns),
            "state_gate_risk_batch": _state_gate_risk_summary(frontier_patterns),
            "next_best_expansions": _next_best_expansions(missing_tokens, kernel_gaps),
            "validation": validation_payload(validation_issues, validation_summary)[
                "strategy_coverage_matrix_validation"
            ],
        }
    }
    report["coverage_frontier"]["frontier_gate"] = _frontier_gate_summary(report["coverage_frontier"], matrix)
    check_issues = check_report(report, matrix)
    report["coverage_frontier"]["check"] = {
        "result": "pass" if not check_issues else "fail",
        "issue_count": len(check_issues),
        "issues": [issue.as_dict() for issue in check_issues],
    }
    return report


def check_report(report: dict[str, Any], matrix: dict[str, Any]) -> list[CheckIssue]:
    frontier = report["coverage_frontier"]
    metrics = frontier["metrics"]
    thresholds = matrix.get("thresholds", {}) if isinstance(matrix.get("thresholds"), dict) else {}
    issues: list[CheckIssue] = []

    validation = frontier["validation"]
    if validation["result"] != "pass":
        issues.append(CheckIssue("validator_failed", "coverage matrix validator returned errors"))
    custom_max = _number(thresholds.get("custom_token_route_max"), 1.0)
    if metrics["custom_token_route_share"] > custom_max:
        issues.append(CheckIssue("custom_token_share_exceeds_cap", "custom token route share exceeds cap"))
    false_max = _number(thresholds.get("false_supported_max"), 1.0)
    if metrics["false_supported_rate_mechanical"] > false_max:
        issues.append(
            CheckIssue("mechanical_false_supported_exceeds_cap", "mechanical false-supported rate exceeds cap")
        )
    if metrics["boundary_false_supported_count"] > 0:
        issues.append(CheckIssue("boundary_false_supported_present", "boundary false-supported count is non-zero"))
    groups = frontier["benchmark_groups"]
    external_count = groups.get("external_benchmark", {}).get("count", 0)
    if external_count < 20:
        issues.append(CheckIssue("external_benchmark_count_below_threshold", "external benchmark count is below 20"))
    if frontier["dogfood_pattern_count"] < 1:
        issues.append(CheckIssue("dogfood_missing", "dogfood row is missing"))
    if frontier["dogfood_target"]["publication_status"] != "pass":
        issues.append(
            CheckIssue(
                "dogfood_publication_target_missing",
                "dogfood publication target requires at least five dogfood rows",
            )
        )
    custom_governance = frontier["custom_token_governance"]
    if custom_governance["missing_governance_rows"]:
        issues.append(
            CheckIssue(
                "custom_token_governance_missing",
                "custom-token rows are missing governance manifest entries",
            )
        )
    if custom_governance["stale_route_count"] > 0:
        issues.append(
            CheckIssue(
                "stale_custom_token_route_present",
                "stale custom-token route rows remain in the active matrix",
            )
        )
    boundary = frontier["reserved_non_goal_boundary"]
    if boundary["missing_boundary_rows"]:
        issues.append(
            CheckIssue(
                "reserved_non_goal_boundary_missing",
                "reserved/non-goal rows are missing boundary manifest entries",
            )
        )
    if frontier["frontier_gate"]["result"] != "pass":
        issues.append(CheckIssue("frontier_gate_failed", "coverage frontier publication gate failed"))
    return issues


def render_markdown(report: dict[str, Any]) -> str:
    frontier = report["coverage_frontier"]
    metrics = frontier["metrics"]
    lines = [
        "# Strategy Coverage Report",
        "",
        "Generated from `docs/reports/strategy_coverage_matrix.yaml`.",
        "",
        "This report measures the QST strategy record layer only. It does not claim broker,",
        "exchange, live execution, HFT runtime, full backtest engine, production execution,",
        "profitability, or portfolio optimizer coverage.",
        "",
        "## Summary",
        "",
        f"- Pattern count: `{frontier['pattern_count']}`",
        f"- Frontier pattern count: `{frontier['frontier_pattern_count']}`",
        f"- Dogfood pattern count: `{frontier['dogfood_pattern_count']}`",
        f"- Total frontier weight: `{frontier['total_weight']}`",
        f"- Check result: `{frontier['check']['result']}`",
        "",
        "## Benchmark Groups",
        "",
        "| Group | Count | Weight | Supported | Partial | Custom | Reserved | Non-goal |",
        "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |",
    ]
    for group, summary in frontier["benchmark_groups"].items():
        class_counts = summary["class_counts"]
        lines.append(
            "| {group} | {count} | {weight} | {supported} | {partial} | {custom} | {reserved} | {non_goal} |".format(
                group=group,
                count=summary["count"],
                weight=summary["weight"],
                supported=class_counts.get("supported", 0),
                partial=class_counts.get("partially_supported", 0),
                custom=class_counts.get("custom_token_required", 0),
                reserved=class_counts.get("reserved", 0),
                non_goal=class_counts.get("non_goal", 0),
            )
        )
    lines.extend(
        [
            "",
            "## Classification Summary",
            "",
            "| Classification | Count | Weight |",
            "| --- | ---: | ---: |",
        ]
    )
    for classification, summary in frontier["by_classification"].items():
        lines.append(f"| {classification} | {summary['count']} | {summary['weight']} |")

    lines.extend(
        [
            "",
            "## Metrics",
            "",
            "| Metric | Value |",
            "| --- | ---: |",
        ]
    )
    for key, value in metrics.items():
        lines.append(f"| {key} | {_format_metric(value)} |")

    gate = frontier["frontier_gate"]
    headline = gate["headline"]
    lines.extend(
        [
            "",
            "## Frontier Gate",
            "",
            "PR12 applies publication-gate thresholds to the measured coverage frontier.",
            "`measured_frontier` thresholds are reported as measured values, not hardcoded",
            "target percentages.",
            "",
            f"- Gate result: `{gate['result']}`",
            f"- Headline metric: `{headline['metric']}`",
            f"- Headline value: `{_format_percent(headline['value'])}`",
            f"- Headline label: `{headline['label']}`",
            "",
            "| Gate | Threshold | Measured | Result |",
            "| --- | --- | ---: | --- |",
        ]
    )
    for check in gate["checks"]:
        lines.append(
            "| {name} | {threshold} | {measured} | {result} |".format(
                name=check["name"],
                threshold=check["threshold"],
                measured=_format_gate_value(check["measured"]),
                result=check["result"],
            )
        )

    lines.extend(
        [
            "",
            "## Public Statement",
            "",
            _public_statement_text(frontier),
        ]
    )

    custom_governance = frontier["custom_token_governance"]
    lines.extend(
        [
            "",
            "## Custom Token Governance",
            "",
            "PR10 records governance evidence for active custom-token routes. These routes",
            "remain record-layer classification evidence only; verification may inspect",
            "metadata and integrity, but it does not approve, grant, or execute custom code.",
            "",
            f"- Route share: `{_format_metric(metrics['custom_token_route_share'])}`",
            f"- Route cap: `{_format_metric(custom_governance['route_cap'])}`",
            f"- Discount: `{_format_metric(custom_governance['discount'])}` (`{custom_governance['discount_status']}`)",
            f"- Active custom routes: `{custom_governance['active_route_count']}`",
            f"- Missing governance rows: `{len(custom_governance['missing_governance_rows'])}`",
            f"- Stale route findings: `{custom_governance['stale_route_count']}`",
            "",
            "| Row | Reason | Missing tokens | Future built-in candidate | Remain custom route |",
            "| --- | --- | --- | --- | --- |",
        ]
    )
    for row in custom_governance["rows"]:
        lines.append(
            "| {id} | {reason} | {missing_tokens} | {future_builtin_candidate} | {remain_custom_route} |".format(
                id=row["id"],
                reason=row["reason"],
                missing_tokens=", ".join(row["missing_tokens"]) or "none",
                future_builtin_candidate=str(row["future_builtin_candidate"]).lower(),
                remain_custom_route=str(row["remain_custom_route"]).lower(),
            )
        )
    lines.extend(
        [
            "",
            "| Retired stale route | Replacement evidence |",
            "| --- | --- |",
        ]
    )
    for row in custom_governance["stale_route_reviews"]:
        lines.append(f"| {row['pattern_id']} | {row['replacement_evidence']} |")

    boundary = frontier["reserved_non_goal_boundary"]
    lines.extend(
        [
            "",
            "## Reserved / Non-Goal Boundary",
            "",
            "PR11 records audit evidence for reserved design and non-goal rows. Reserved",
            "rows are future design boundaries that require explicit TypeSpec/runtime",
            "work before use. Non-goal rows remain outside QST scope and must not be",
            "weakened into partial, custom-token, or supported classifications.",
            "",
            f"- Reserved rows: `{boundary['reserved_count']}`",
            f"- Non-goal rows: `{boundary['non_goal_count']}`",
            f"- Missing boundary rows: `{len(boundary['missing_boundary_rows'])}`",
            f"- Boundary false-supported count: `{metrics['boundary_false_supported_count']}`",
            "",
            "| Row | Classification | Diagnostic class | Boundary class | Reason |",
            "| --- | --- | --- | --- | --- |",
        ]
    )
    for row in boundary["rows"]:
        lines.append(
            "| {id} | {classification} | {diagnostic_class} | {boundary_class} | {reason} |".format(
                id=row["id"],
                classification=row["classification"],
                diagnostic_class=row["diagnostic_class"],
                boundary_class=row["boundary_class"],
                reason=row["reason"],
            )
        )

    lines.extend(
        [
            "",
            "## Next Best Expansions",
            "",
            "| Family or kernel | Type | Weighted gain | Complexity cost | Coverage efficiency |",
            "| --- | --- | ---: | ---: | ---: |",
        ]
    )
    for expansion in frontier["next_best_expansions"]:
        lines.append(
            "| {family_or_kernel} | {type} | {weighted_gain} | {complexity_cost} | {coverage_efficiency} |".format(
                **expansion
            )
        )

    lines.extend(
        [
            "",
            "## Core rule token batch",
            "",
            "PR6 adds accepted record/reference token coverage for common indicator, signal,",
            "and decision-rule patterns. These rows remain record-layer evidence and do not",
            "claim broad runtime execution, broker/exchange behavior, or profitability.",
            "",
            "| Row | Classification | Mechanical status | Example | Required tokens |",
            "| --- | --- | --- | --- | --- |",
        ]
    )
    for row in frontier["core_rule_token_batch"]["rows"]:
        lines.append(
            "| {id} | {classification} | {mechanical_status} | {example} | {required_tokens} |".format(
                id=row["id"],
                classification=row["classification"],
                mechanical_status=row["mechanical_status"],
                example=row["example"] or "pending",
                required_tokens=", ".join(row["required_tokens"]),
            )
        )

    lines.extend(
        [
            "",
            "## Panel / Factor / Weight Batch",
            "",
            "PR8 adds accepted record/reference token coverage for panel aliases,",
            "factor records, and deterministic weight transforms. These rows remain",
            "record-layer evidence and do not claim optimizer, rebalance, broker,",
            "exchange, live execution, or profitability coverage.",
            "",
            "| Row | Classification | Mechanical status | Example | Required tokens |",
            "| --- | --- | --- | --- | --- |",
        ]
    )
    for row in frontier["panel_factor_weight_batch"]["rows"]:
        lines.append(
            "| {id} | {classification} | {mechanical_status} | {example} | {required_tokens} |".format(
                id=row["id"],
                classification=row["classification"],
                mechanical_status=row["mechanical_status"],
                example=row["example"] or "pending",
                required_tokens=", ".join(row["required_tokens"]),
            )
        )

    lines.extend(
        [
            "",
            "## State / Gate / Risk Batch",
            "",
            "PR9 adds accepted record/reference token coverage for common state, gate,",
            "stop/take-profit, drawdown, exposure, turnover, and rebalance-band records.",
            "These rows remain record-layer evidence and do not claim broker/exchange",
            "execution, live stop orders, backtests, account runtime, or Calendar/EventStream",
            "support.",
            "",
            "| Row | Classification | Mechanical status | Example | Required tokens |",
            "| --- | --- | --- | --- | --- |",
        ]
    )
    for row in frontier["state_gate_risk_batch"]["rows"]:
        lines.append(
            "| {id} | {classification} | {mechanical_status} | {example} | {required_tokens} |".format(
                id=row["id"],
                classification=row["classification"],
                mechanical_status=row["mechanical_status"],
                example=row["example"] or "pending",
                required_tokens=", ".join(row["required_tokens"]),
            )
        )

    lines.extend(
        [
            "",
            "## Dogfood",
            "",
            f"- Status: `{frontier['dogfood']['status']}`",
            f"- Classifications: `{', '.join(frontier['dogfood']['classifications'])}`",
            f"- MVP target: `{frontier['dogfood_target']['mvp_count']} / {frontier['dogfood_target']['mvp_required']}` "
            f"(`{frontier['dogfood_target']['mvp_status']}`)",
            f"- Publication target: `{frontier['dogfood_target']['publication_count']} / "
            f"{frontier['dogfood_target']['publication_required']}` "
            f"(`{frontier['dogfood_target']['publication_status']}`)",
            "",
            "| Row | Status | Classification | Candidate GKR | Evidence report | Limitations |",
            "| --- | --- | --- | --- | --- | --- |",
        ]
    )
    for row in frontier["dogfood"]["rows"]:
        limitations = "; ".join(row["limitations"]) if row["limitations"] else "none"
        lines.append(
            "| {id} | {status} | {classification} | {candidate_gkr} | {report} | {limitations} |".format(
                id=row["id"],
                status=row["status"],
                classification=row["classification"],
                candidate_gkr=row["candidate_gkr"] or "not recorded",
                report=row["report"] or "not recorded",
                limitations=limitations,
            )
        )
    lines.extend(
        [
            "",
            "Dogfood rows remain excluded from headline frontier percentages. The publication",
            "target records breadth evidence for the dogfood set, not runtime execution or",
            "profitability.",
            "",
            "## Validation",
            "",
            f"- Validator result: `{frontier['validation']['result']}`",
            f"- Validator issue count: `{frontier['validation']['issue_count']}`",
            f"- Report check result: `{frontier['check']['result']}`",
            f"- Report check issue count: `{frontier['check']['issue_count']}`",
            "",
        ]
    )
    return "\n".join(lines)


def render_text(report: dict[str, Any]) -> str:
    frontier = report["coverage_frontier"]
    metrics = frontier["metrics"]
    return "\n".join(
        [
            "QST strategy coverage report",
            f"patterns: {frontier['pattern_count']}",
            f"frontier_patterns: {frontier['frontier_pattern_count']}",
            f"dogfood_patterns: {frontier['dogfood_pattern_count']}",
            f"dogfood_publication_target: {frontier['dogfood_target']['publication_status']}",
            f"check: {frontier['check']['result']}",
            f"frontier_gate: {frontier['frontier_gate']['result']}",
            f"headline_metric: {frontier['frontier_gate']['headline']['metric']}",
            f"headline_value: {_format_metric(frontier['frontier_gate']['headline']['value'])}",
            f"direct_builtin_coverage: {_format_metric(metrics['direct_builtin_coverage'])}",
            f"routable_record_coverage_raw: {_format_metric(metrics['routable_record_coverage_raw'])}",
            f"routable_record_coverage_discounted: {_format_metric(metrics['routable_record_coverage_discounted'])}",
            f"custom_token_route_share: {_format_metric(metrics['custom_token_route_share'])}",
            f"reserved_non_goal_boundary: {frontier['reserved_non_goal_boundary']['status']}",
            f"kernel_gap_count: {metrics['kernel_gap_count']}",
        ]
    )


def _benchmark_group_summary(patterns: list[dict[str, Any]]) -> dict[str, Any]:
    grouped: dict[str, list[dict[str, Any]]] = defaultdict(list)
    for row in patterns:
        grouped[str(row.get("benchmark_group", ""))].append(row)
    summary = {}
    for group in sorted(grouped):
        rows = grouped[group]
        class_counts = Counter(str(row.get("expected_classification", "")) for row in rows)
        summary[group] = {
            "count": len(rows),
            "weight": _weight_sum(rows),
            "class_counts": dict(sorted(class_counts.items())),
        }
    return summary


def _classification_summary(patterns: list[dict[str, Any]]) -> dict[str, Any]:
    grouped: dict[str, list[dict[str, Any]]] = defaultdict(list)
    for row in patterns:
        grouped[str(row.get("expected_classification", ""))].append(row)
    summary = {}
    for classification in sorted(grouped):
        rows = grouped[classification]
        summary[classification] = {"count": len(rows), "weight": _weight_sum(rows)}
    return summary


def _dogfood_summary(rows: list[dict[str, Any]]) -> dict[str, Any]:
    statuses = sorted({str(row.get("status", "")) for row in rows if row.get("status")})
    classifications = sorted({str(row.get("expected_classification", "")) for row in rows})
    return {
        "count": len(rows),
        "status": ", ".join(statuses) if statuses else "missing",
        "classifications": classifications,
        "weight": _weight_sum(rows),
        "rows": [_dogfood_row_summary(row) for row in rows],
    }


def _custom_token_governance_summary(
    rows: list[dict[str, Any]],
    matrix: dict[str, Any],
    *,
    repo_root: Path | None,
) -> dict[str, Any]:
    manifest = load_custom_token_governance_manifest(repo_root)
    route_rows = [row for row in rows if row.get("expected_classification") == "custom_token_required"]
    routes = {
        str(route.get("pattern_id")): route
        for route in manifest.get("routes", [])
        if isinstance(route, dict) and route.get("pattern_id")
    }
    stale_reviews = [
        review
        for review in manifest.get("stale_route_reviews", [])
        if isinstance(review, dict) and review.get("pattern_id")
    ]
    thresholds = matrix.get("thresholds") if isinstance(matrix.get("thresholds"), dict) else {}
    policy = matrix.get("coverage_policy") if isinstance(matrix.get("coverage_policy"), dict) else {}
    discount = policy.get("custom_token_discount") if isinstance(policy.get("custom_token_discount"), dict) else {}
    summaries = []
    missing_governance = []
    for row in sorted(route_rows, key=lambda item: str(item.get("id", ""))):
        row_id = str(row.get("id", ""))
        route = row.get("custom_token_route") if isinstance(row.get("custom_token_route"), dict) else {}
        governance = routes.get(row_id)
        gaps = row.get("gaps") if isinstance(row.get("gaps"), dict) else {}
        if not governance:
            missing_governance.append(row_id)
        summaries.append(
            {
                "id": row_id,
                "reason": str(route.get("reason", "")),
                "input_ports": [str(item) for item in route.get("input_ports", [])]
                if isinstance(route.get("input_ports"), list)
                else [],
                "output_ports": [str(item) for item in route.get("output_ports", [])]
                if isinstance(route.get("output_ports"), list)
                else [],
                "missing_tokens": [str(item) for item in gaps.get("missing_tokens", [])]
                if isinstance(gaps.get("missing_tokens"), list)
                else [],
                "missing_types": [str(item) for item in gaps.get("missing_types", [])]
                if isinstance(gaps.get("missing_types"), list)
                else [],
                "future_builtin_candidate": bool(governance.get("future_builtin_candidate"))
                if governance
                else False,
                "remain_custom_route": bool(governance.get("remain_custom_route")) if governance else False,
            }
        )
    active_row_ids = {str(row.get("id", "")) for row in route_rows}
    stale_route_count = sum(1 for row_id in active_row_ids if row_id.startswith("int_040"))
    return {
        "manifest_schema_version": manifest.get("schema_version", ""),
        "route_cap": _number(thresholds.get("custom_token_route_max"), 0.0),
        "discount": _number(discount.get("value"), 0.5),
        "discount_status": str(discount.get("status", "")),
        "active_route_count": len(route_rows),
        "missing_governance_rows": missing_governance,
        "stale_route_count": stale_route_count,
        "rows": summaries,
        "stale_route_reviews": [
            {
                "pattern_id": str(review.get("pattern_id", "")),
                "replacement_evidence": ", ".join(str(item) for item in review.get("replacement_evidence", []))
                if isinstance(review.get("replacement_evidence"), list)
                else "",
            }
            for review in stale_reviews
        ],
    }


def _reserved_non_goal_boundary_summary(
    rows: list[dict[str, Any]],
    *,
    repo_root: Path | None,
) -> dict[str, Any]:
    manifest = load_reserved_non_goal_boundary_manifest(repo_root)
    boundary_rows = [
        row
        for row in rows
        if row.get("expected_classification") in {"reserved", "non_goal"}
    ]
    cases = {
        str(case.get("pattern_id")): case
        for case in manifest.get("cases", [])
        if isinstance(case, dict) and case.get("pattern_id")
    }
    summaries = []
    missing_rows = []
    for row in sorted(boundary_rows, key=lambda item: str(item.get("id", ""))):
        row_id = str(row.get("id", ""))
        classification = str(row.get("expected_classification", ""))
        boundary = row.get("boundary") if isinstance(row.get("boundary"), dict) else {}
        case = cases.get(row_id)
        if not case:
            missing_rows.append(row_id)
        if classification == "reserved":
            reason = str(boundary.get("reserved_reason", ""))
        else:
            reason = str(boundary.get("non_goal_reason", ""))
        summaries.append(
            {
                "id": row_id,
                "classification": classification,
                "diagnostic_class": str(case.get("diagnostic_class", "")) if case else "",
                "boundary_class": str(case.get("boundary_class", "")) if case else "",
                "reason": reason,
                "future_stage_allowed": bool(case.get("future_stage_allowed")) if case else False,
            }
        )
    reserved_count = sum(1 for row in boundary_rows if row.get("expected_classification") == "reserved")
    non_goal_count = sum(1 for row in boundary_rows if row.get("expected_classification") == "non_goal")
    diagnostic_classes = sorted(
        {
            str(case.get("diagnostic_class", ""))
            for case in cases.values()
            if case.get("diagnostic_class")
        }
    )
    return {
        "manifest_schema_version": str(manifest.get("schema_version", "")),
        "status": "pass" if not missing_rows else "fail",
        "reserved_count": reserved_count,
        "non_goal_count": non_goal_count,
        "missing_boundary_rows": missing_rows,
        "diagnostic_classes": diagnostic_classes,
        "rows": summaries,
    }


def _frontier_gate_summary(frontier: dict[str, Any], matrix: dict[str, Any]) -> dict[str, Any]:
    metrics = frontier["metrics"]
    thresholds = matrix.get("thresholds") if isinstance(matrix.get("thresholds"), dict) else {}
    group_counts = frontier["benchmark_groups"]
    validation = frontier["validation"]
    custom_governance = frontier["custom_token_governance"]
    boundary = frontier["reserved_non_goal_boundary"]

    frontier_count = max(int(frontier.get("frontier_pattern_count", 0)), 1)
    classified_count = 0
    for group_name, summary in group_counts.items():
        if group_name == "dogfood":
            continue
        class_counts = summary.get("class_counts", {})
        if isinstance(class_counts, dict):
            classified_count += sum(int(count) for count in class_counts.values())
    intent_routing = _ratio(classified_count, frontier_count)
    reserved_non_goal_false_positive_rate = _ratio(
        int(metrics["boundary_false_supported_count"]),
        frontier_count,
    )
    checks = [
        _gate_check("intent_routing", thresholds.get("intent_routing_min", 0.0), intent_routing, minimum=True),
        _gate_check(
            "direct_builtin",
            thresholds.get("direct_builtin_min", "measured_frontier"),
            metrics["direct_builtin_coverage"],
            minimum=True,
        ),
        _gate_check(
            "routable_record_raw",
            thresholds.get("routable_record_min", "measured_frontier"),
            metrics["routable_record_coverage_raw"],
            minimum=True,
        ),
        _gate_check(
            "custom_token_route_share",
            thresholds.get("custom_token_route_max", 1.0),
            metrics["custom_token_route_share"],
            minimum=False,
        ),
        _gate_check(
            "mechanical_false_supported_rate",
            thresholds.get("false_supported_max", 1.0),
            metrics["false_supported_rate_mechanical"],
            minimum=False,
        ),
        _gate_check(
            "semantic_false_supported_rate",
            thresholds.get("false_supported_max", 1.0),
            metrics["false_supported_rate_semantic"],
            minimum=False,
        ),
        _gate_check(
            "boundary_false_supported_rate",
            thresholds.get("false_supported_max", 1.0),
            metrics["false_supported_rate_boundary"],
            minimum=False,
        ),
        _gate_check(
            "reserved_non_goal_false_positive_rate",
            thresholds.get("reserved_non_goal_false_positive_max", 1.0),
            reserved_non_goal_false_positive_rate,
            minimum=False,
        ),
        _minimum_count_gate_check(
            "external_benchmark_rows",
            20,
            group_counts.get("external_benchmark", {}).get("count", 0),
        ),
        _exact_gate_check("dogfood_publication_target", "pass", frontier["dogfood_target"]["publication_status"]),
        _exact_gate_check("matrix_validator", "pass", validation["result"]),
        _exact_gate_check("custom_governance_manifest", 0, len(custom_governance["missing_governance_rows"])),
        _exact_gate_check("reserved_non_goal_boundary_manifest", 0, len(boundary["missing_boundary_rows"])),
    ]
    issues = [
        {
            "code": f"{check['name']}_failed",
            "message": f"{check['name']} gate failed",
        }
        for check in checks
        if check["result"] != "pass"
    ]
    headline_value = float(metrics[HEADLINE_METRIC])
    return {
        "result": "pass" if not issues else "fail",
        "headline": {
            "metric": HEADLINE_METRIC,
            "label": HEADLINE_METRIC_LABEL,
            "value": headline_value,
            "percent": _round(headline_value * 100),
        },
        "threshold_policy": {
            "intent_routing_min": thresholds.get("intent_routing_min"),
            "direct_builtin_min": thresholds.get("direct_builtin_min"),
            "routable_record_min": thresholds.get("routable_record_min"),
            "custom_token_route_max": thresholds.get("custom_token_route_max"),
            "false_supported_max": thresholds.get("false_supported_max"),
            "reserved_non_goal_false_positive_max": thresholds.get("reserved_non_goal_false_positive_max"),
        },
        "checks": checks,
        "issues": issues,
        "group_split": {
            "internal_matrix": group_counts.get("internal_matrix", {}),
            "external_benchmark": group_counts.get("external_benchmark", {}),
            "dogfood": group_counts.get("dogfood", {}),
        },
        "public_statement": _public_statement_text(frontier),
    }


def _gate_check(name: str, threshold: object, measured: object, *, minimum: bool) -> dict[str, Any]:
    measured_number = _number(measured, 0.0)
    if threshold == "measured_frontier":
        result = "pass"
        threshold_mode = "measured_frontier"
    else:
        threshold_number = _number(threshold, 0.0 if minimum else 1.0)
        if minimum:
            result = "pass" if measured_number >= threshold_number else "fail"
            threshold_mode = "minimum"
        else:
            result = "pass" if measured_number <= threshold_number else "fail"
            threshold_mode = "maximum"
    return {
        "name": name,
        "threshold": threshold,
        "measured": measured_number,
        "result": result,
        "threshold_mode": threshold_mode,
    }


def _exact_gate_check(name: str, expected: object, measured: object) -> dict[str, Any]:
    result = "pass" if measured == expected else "fail"
    return {
        "name": name,
        "threshold": expected,
        "measured": measured,
        "result": result,
        "threshold_mode": "exact",
    }


def _minimum_count_gate_check(name: str, expected: int, measured: object) -> dict[str, Any]:
    measured_count = int(measured) if isinstance(measured, int) else 0
    return {
        "name": name,
        "threshold": expected,
        "measured": measured_count,
        "result": "pass" if measured_count >= expected else "fail",
        "threshold_mode": "minimum",
    }


def _core_rule_summary(rows: list[dict[str, Any]]) -> dict[str, Any]:
    selected = [row for row in rows if row.get("id") in CORE_RULE_ROW_IDS]
    selected.sort(key=lambda row: str(row.get("id", "")))
    return {
        "count": len(selected),
        "rows": [_core_rule_row_summary(row) for row in selected],
    }


def _panel_factor_weight_summary(rows: list[dict[str, Any]]) -> dict[str, Any]:
    selected = [row for row in rows if row.get("id") in PANEL_FACTOR_WEIGHT_ROW_IDS]
    selected.sort(key=lambda row: str(row.get("id", "")))
    return {
        "count": len(selected),
        "rows": [_core_rule_row_summary(row) for row in selected],
    }


def _state_gate_risk_summary(rows: list[dict[str, Any]]) -> dict[str, Any]:
    selected = [row for row in rows if row.get("id") in STATE_GATE_RISK_ROW_IDS]
    selected.sort(key=lambda row: str(row.get("id", "")))
    return {
        "count": len(selected),
        "rows": [_core_rule_row_summary(row) for row in selected],
    }


def _core_rule_row_summary(row: dict[str, Any]) -> dict[str, Any]:
    evidence = row.get("evidence")
    evidence_map = evidence if isinstance(evidence, dict) else {}
    false_supported = row.get("false_supported")
    false_map = false_supported if isinstance(false_supported, dict) else {}
    examples = evidence_map.get("examples")
    required_tokens = evidence_map.get("required_tokens")
    return {
        "id": str(row.get("id", "")),
        "classification": str(row.get("expected_classification", "")),
        "mechanical_status": str(false_map.get("mechanical_status", "")),
        "example": str(examples[0]) if isinstance(examples, list) and examples else "",
        "required_tokens": [str(token) for token in required_tokens]
        if isinstance(required_tokens, list)
        else [],
    }


def _dogfood_target_summary(rows: list[dict[str, Any]]) -> dict[str, Any]:
    count = len(rows)
    return {
        "mvp_required": DOGFOOD_MVP_TARGET,
        "mvp_count": count,
        "mvp_status": "pass" if count >= DOGFOOD_MVP_TARGET else "fail",
        "publication_required": DOGFOOD_PUBLICATION_TARGET,
        "publication_count": count,
        "publication_status": "pass" if count >= DOGFOOD_PUBLICATION_TARGET else "fail",
        "headline_frontier_policy": "excluded",
    }


def _dogfood_row_summary(row: dict[str, Any]) -> dict[str, Any]:
    evidence = row.get("dogfood_evidence")
    evidence_map = evidence if isinstance(evidence, dict) else {}
    boundary = row.get("boundary")
    boundary_map = boundary if isinstance(boundary, dict) else {}
    limitations = boundary_map.get("limitations")
    return {
        "id": str(row.get("id", "")),
        "status": str(row.get("status", "")),
        "classification": str(row.get("expected_classification", "")),
        "candidate_gkr": _optional_string(evidence_map.get("candidate_gkr")),
        "intent_fixture": _optional_string(evidence_map.get("intent_fixture")),
        "report": _optional_string(evidence_map.get("report")),
        "limitations": [str(item) for item in limitations] if isinstance(limitations, list) else [],
    }


def _missing_token_records(patterns: list[dict[str, Any]]) -> list[dict[str, Any]]:
    records: list[dict[str, Any]] = []
    for row in patterns:
        gaps = row.get("gaps")
        if not isinstance(gaps, dict):
            continue
        for token in gaps.get("missing_tokens", []) or []:
            if isinstance(token, str) and token:
                records.append({"name": token, "pattern_id": row.get("id"), "weight": _row_weight(row)})
    return records


def _kernel_gap_records(patterns: list[dict[str, Any]]) -> list[dict[str, Any]]:
    records: list[dict[str, Any]] = []
    for row in patterns:
        gaps = row.get("gaps")
        if not isinstance(gaps, dict):
            continue
        kernel_gaps = gaps.get("kernel_gaps") or []
        if not isinstance(kernel_gaps, list):
            continue
        for gap in kernel_gaps:
            if isinstance(gap, dict) and gap.get("category"):
                records.append(
                    {
                        "name": str(gap["category"]),
                        "pattern_id": row.get("id"),
                        "weight": _row_weight(row),
                    }
                )
    return records


def _next_best_expansions(
    missing_tokens: list[dict[str, Any]],
    kernel_gaps: list[dict[str, Any]],
) -> list[dict[str, Any]]:
    aggregate: dict[tuple[str, str], float] = defaultdict(float)
    for record in missing_tokens:
        aggregate[(record["name"], "token")] += float(record["weight"])
    for record in kernel_gaps:
        aggregate[(record["name"], "kernel")] += float(record["weight"])

    expansions = []
    for (name, kind), gain in aggregate.items():
        cost = 1 if kind == "token" else 3
        expansions.append(
            {
                "family_or_kernel": name,
                "type": kind,
                "weighted_gain": _round(gain),
                "complexity_cost": cost,
                "coverage_efficiency": _round(gain / cost),
            }
        )
    return sorted(
        expansions,
        key=lambda item: (
            -float(item["coverage_efficiency"]),
            -float(item["weighted_gain"]),
            str(item["family_or_kernel"]),
        ),
    )[:10]


def _by_class(patterns: list[dict[str, Any]], classification: str) -> list[dict[str, Any]]:
    return [row for row in patterns if row.get("expected_classification") == classification]


def _false_supported(row: dict[str, Any]) -> dict[str, Any]:
    value = row.get("false_supported")
    return value if isinstance(value, dict) else {}


def _status_count(rows: list[dict[str, Any]], field: str, value: str) -> int:
    return sum(1 for row in rows if _false_supported(row).get(field) == value)


def _status_rate(rows: list[dict[str, Any]], field: str, value: str, denominator: int) -> float:
    return _round(_status_count(rows, field, value) / denominator)


def _weight_sum(patterns: list[dict[str, Any]]) -> float:
    return _round(sum(_row_weight(row) for row in patterns))


def _row_weight(row: dict[str, Any]) -> float:
    weight = row.get("market_weight")
    if isinstance(weight, dict):
        return _number(weight.get("final"), 0.0)
    return 0.0


def _custom_discount(matrix: dict[str, Any]) -> float:
    policy = matrix.get("coverage_policy")
    if not isinstance(policy, dict):
        return 0.5
    discount = policy.get("custom_token_discount")
    if not isinstance(discount, dict):
        return 0.5
    return _number(discount.get("value"), 0.5)


def _number(value: object, default: float) -> float:
    return float(value) if isinstance(value, int | float) else default


def _optional_string(value: object) -> str:
    return value if isinstance(value, str) else ""


def _ratio(numerator: float, denominator: float) -> float:
    if denominator <= 0:
        return 0.0
    return _round(numerator / denominator)


def _round(value: float) -> float:
    return round(float(value), 4)


def _format_metric(value: object) -> str:
    if isinstance(value, float):
        return f"{value:.4f}"
    return str(value)


def _format_percent(value: object) -> str:
    return f"{(_number(value, 0.0) * 100):.2f}%"


def _format_gate_value(value: object) -> str:
    if isinstance(value, float):
        return _format_metric(value)
    return str(value)


def _public_statement_text(frontier: dict[str, Any]) -> str:
    metrics = frontier["metrics"]
    headline_value = metrics[HEADLINE_METRIC]
    return (
        "QST has reached a measured strategy record-layer raw routable coverage "
        f"frontier of {_format_percent(headline_value)} on the current Coverage "
        "Frontier v0.3 benchmark. This headline includes direct built-in GKR support, "
        "partial records, and bounded custom-token-required routes. Direct built-in "
        f"coverage is {_format_percent(metrics['direct_builtin_coverage'])}; discounted "
        "routable record coverage is "
        f"{_format_percent(metrics['routable_record_coverage_discounted'])}. This does "
        "not include runtime, backtest, broker, exchange, HFT, optimizer execution, "
        "profitability, production execution, or live trading coverage."
    )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Report QST strategy coverage metrics.")
    parser.add_argument("matrix", type=Path)
    parser.add_argument("--check", action="store_true")
    parser.add_argument("--json", action="store_true", dest="json_output")
    parser.add_argument("--markdown", type=Path)
    args = parser.parse_args(argv)

    try:
        matrix = load_matrix(args.matrix)
        report = build_report(matrix, repo_root=Path.cwd())
    except Exception as exc:
        print(f"failed to build coverage report: {exc}", file=sys.stderr)
        return 1

    if args.markdown:
        args.markdown.write_text(render_markdown(report), encoding="utf-8")
    if args.json_output:
        print(json.dumps(report, indent=2, sort_keys=True))
    elif not args.markdown:
        print(render_text(report))
    else:
        print(f"wrote {args.markdown}")

    if args.check and report["coverage_frontier"]["check"]["result"] != "pass":
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
