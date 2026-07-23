from __future__ import annotations

from qst.decision import (
    FINAL_UNKNOWN_AS_ACCEPT_REASON,
    FINAL_UNKNOWN_AS_REJECT_REASON,
    DecisionV2,
    fold_decisions,
)


def _d(kind: str, score: str | None = None) -> DecisionV2:
    return DecisionV2(kind=kind, reasons=[kind], score=score)  # type: ignore[arg-type]


def _kind(policy: str, kinds: list[str]) -> str:
    decision = fold_decisions(policy, [_d(kind) for kind in kinds]).decision
    assert decision is not None
    return decision.kind


def test_strict_and_truth_table() -> None:
    assert _kind("decision.strict_and", ["accept", "accept"]) == "accept"
    assert _kind("decision.strict_and", ["accept", "unknown"]) == "reject"
    assert _kind("decision.strict_and", ["unknown", "unknown"]) == "reject"
    assert _kind("decision.strict_and", ["reject", "unknown"]) == "reject"
    assert _kind("decision.strict_and", ["block", "accept"]) == "block"


def test_permissive_and_truth_table() -> None:
    assert _kind("decision.permissive_and", ["accept", "unknown"]) == "accept"
    assert _kind("decision.permissive_and", ["unknown", "unknown"]) == "accept"
    assert _kind("decision.permissive_and", ["reject", "unknown"]) == "reject"
    assert _kind("decision.permissive_and", ["block", "unknown"]) == "block"


def test_fold_policy_finalizer_appends_deterministic_reason() -> None:
    strict = fold_decisions("decision.strict_and", [_d("unknown")]).decision
    permissive = fold_decisions("decision.permissive_and", [_d("unknown")]).decision

    assert strict is not None
    assert permissive is not None
    assert FINAL_UNKNOWN_AS_REJECT_REASON in strict.reasons
    assert FINAL_UNKNOWN_AS_ACCEPT_REASON in permissive.reasons


def test_fold_policy_score_does_not_change_final_kind() -> None:
    strict = fold_decisions("decision.strict_and", [_d("unknown", score="0.99")]).decision
    permissive = fold_decisions(
        "decision.permissive_and",
        [_d("unknown", score="0.01")],
    ).decision

    assert strict is not None
    assert permissive is not None
    assert strict.kind == "reject"
    assert permissive.kind == "accept"


def test_empty_fold_is_error() -> None:
    result = fold_decisions("decision.strict_and", [])

    assert result.decision is None
    assert not result.diagnostics.ok
    assert result.diagnostics.errors[0].code == "QST_V2_DECISION_EMPTY_INPUT"
