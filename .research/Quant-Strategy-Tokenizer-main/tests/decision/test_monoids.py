from __future__ import annotations

import itertools

from qst.decision import (
    DecisionV2,
    combine_decisions,
    is_aggregator_id,
    is_fold_policy_id,
    is_monoid_id,
)

MONOIDS = ("decision.unknown_propagating_and", "decision.any_accept")
KINDS = ("accept", "reject", "unknown", "block")
IDENTITY = {
    "decision.unknown_propagating_and": "accept",
    "decision.any_accept": "reject",
}


def _d(kind: str, reason: str | None = None, score: str | None = None) -> DecisionV2:
    return DecisionV2(kind=kind, reasons=[reason or kind], score=score)  # type: ignore[arg-type]


def test_only_true_monoids_are_registered_as_monoids() -> None:
    assert is_monoid_id("decision.unknown_propagating_and")
    assert is_monoid_id("decision.any_accept")
    assert not is_monoid_id("decision.strict_and")
    assert not is_monoid_id("decision.permissive_and")
    assert not is_monoid_id("decision.majority")
    assert is_fold_policy_id("decision.strict_and")
    assert is_aggregator_id("decision.majority")


def test_monoid_identity_laws() -> None:
    for monoid in MONOIDS:
        identity = _d(IDENTITY[monoid], "identity")
        for kind in KINDS:
            decision = _d(kind)
            left = combine_decisions(monoid, [identity, decision]).decision
            right = combine_decisions(monoid, [decision, identity]).decision

            assert left is not None
            assert right is not None
            assert left.kind == kind
            assert right.kind == kind


def test_monoid_associativity_laws_for_kind() -> None:
    for monoid in MONOIDS:
        for a, b, c in itertools.product(KINDS, repeat=3):
            left_ab = combine_decisions(monoid, [_d(a), _d(b)]).decision
            right_bc = combine_decisions(monoid, [_d(b), _d(c)]).decision
            assert left_ab is not None
            assert right_bc is not None

            left = combine_decisions(monoid, [left_ab, _d(c)]).decision
            right = combine_decisions(monoid, [_d(a), right_bc]).decision

            assert left is not None
            assert right is not None
            assert left.kind == right.kind


def test_score_is_ignored_by_default_and_annotation_policy_does_not_change_kind() -> None:
    default = combine_decisions(
        "decision.unknown_propagating_and",
        [_d("accept", score="0.9"), _d("unknown", score="0.1")],
    ).decision
    annotated = combine_decisions(
        "decision.unknown_propagating_and",
        [_d("accept", score="0.9"), _d("unknown", score="0.1")],
        score_policy="max_annotation",
    ).decision

    assert default is not None
    assert annotated is not None
    assert default.kind == "unknown"
    assert default.score is None
    assert annotated.kind == "unknown"
    assert annotated.score == "0.9"


def test_empty_combine_is_error_not_identity() -> None:
    result = combine_decisions("decision.any_accept", [])

    assert result.decision is None
    assert not result.diagnostics.ok
    assert result.diagnostics.errors[0].code == "QST_V2_DECISION_EMPTY_INPUT"
