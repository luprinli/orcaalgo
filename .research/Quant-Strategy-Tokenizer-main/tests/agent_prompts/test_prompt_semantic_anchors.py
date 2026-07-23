from __future__ import annotations

from pathlib import Path

from tools.validate_prompt_set import validate_prompt_set

ROOT = Path(__file__).resolve().parents[2]
PROMPT_ROOT = ROOT / "docs" / "agent" / "prompts" / "qst_stage_3c_v0_3_2"


def _text(relative_path: str) -> str:
    return (PROMPT_ROOT / relative_path).read_text(encoding="utf-8").lower()


def test_prompt_validator_enforces_semantic_anchors() -> None:
    result = validate_prompt_set(PROMPT_ROOT)
    checks = result["prompt_validation"]["checks"]
    semantic = next(check for check in checks if check["id"] == "semantic_anchors")

    assert semantic["result"] == "pass"


def test_repo_context_protocol_has_operational_commands() -> None:
    text = _text("core/01_REPO_CONTEXT_PROTOCOL.md")

    for anchor in (
        "git status",
        "git rev-parse",
        "pyproject.toml",
        "qst vocabulary",
        "qst validate",
        "qst hash",
        "canonicalize",
        "examples/strategies",
        "tests/reference",
        "repo_context",
        "strategy_coverage_matrix",
        "report_strategy_coverage.py --check",
        "dogfood_case",
    ):
        assert anchor in text


def test_classify_intent_has_boundary_rules() -> None:
    text = _text("tasks/CLASSIFY_STRATEGY_INTENT.md")

    for anchor in (
        "supported",
        "partially_supported",
        "custom_token_required",
        "reserved",
        "non_goal",
        "broker",
        "exchange",
        "live execution",
        "eventstream",
        "strategy_coverage_matrix",
        "external_benchmark",
        "dogfood_case",
        "false_supported_rate",
    ):
        assert anchor in text


def test_select_tokens_has_selection_output_fields() -> None:
    text = _text("tasks/SELECT_TOKENS.md")

    for anchor in (
        "selected_token_ref",
        "maturity",
        "execution_support",
        "profile_status",
        "rejected_candidates",
        "missing_tokens",
        "coverage_row",
        "kernel_gap",
        "custom_token_route_share",
        "non_goal",
    ):
        assert anchor in text


def test_author_gkr_has_validate_hash_canonical_flow() -> None:
    text = _text("tasks/AUTHOR_GKR_STRATEGY.md")

    for anchor in (
        ".gkr.yaml",
        "node_plan",
        "qst validate",
        "qst hash",
        "canonicalize",
        "repair_gkr_diagnostics",
        "dogfood_case",
        "record-layer evidence",
        "runtime/backtest/profitability",
    ):
        assert anchor in text


def test_repair_diagnostics_has_classes_and_attempt_limit() -> None:
    text = _text("tasks/REPAIR_GKR_DIAGNOSTICS.md")

    for anchor in (
        "schema_error",
        "unknown_token",
        "port_error",
        "type_error",
        "temporal_error",
        "3 attempts",
        "missing_token",
        "kernel_gap",
        "reserved_design",
        "non_goal_runtime",
        "custom_token_required",
        "time-series fake",
    ):
        assert anchor in text


def test_custom_token_routing_preserves_boundary_terms() -> None:
    text = _text("tasks/CUSTOM_TOKEN_ROUTING.md")

    for anchor in (
        "verify",
        "approve",
        "grant",
        "execute",
        "must not execute",
        "do not import",
        "custom_token_route_share",
        "custom token route cap",
        "strategy_coverage_report",
    ):
        assert anchor in text


def test_profile_gate_review_has_coverage_frontier_controls() -> None:
    text = _text("tasks/PROFILE_GATE_REVIEW.md")

    for anchor in (
        "strategy_coverage_report",
        "custom_token_route_share",
        "false_supported_rate",
        "reserved",
        "non_goal",
    ):
        assert anchor in text
