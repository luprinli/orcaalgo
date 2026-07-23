from __future__ import annotations

from qst.decision import DecisionV2, aggregate_decisions


def _d(kind: str, score: str | None = None) -> DecisionV2:
    return DecisionV2(kind=kind, reasons=[kind], score=score)  # type: ignore[arg-type]


def test_majority_over_known_votes() -> None:
    result = aggregate_decisions(
        "decision.majority",
        [_d("accept"), _d("accept"), _d("reject"), _d("unknown")],
    )

    assert result.decision is not None
    assert result.decision.kind == "accept"


def test_majority_tie_or_no_known_is_unknown() -> None:
    tie = aggregate_decisions("decision.majority", [_d("accept"), _d("reject")])
    none_known = aggregate_decisions("decision.majority", [_d("unknown")])

    assert tie.decision is not None
    assert none_known.decision is not None
    assert tie.decision.kind == "unknown"
    assert none_known.decision.kind == "unknown"


def test_block_veto_applies_to_aggregators() -> None:
    result = aggregate_decisions(
        "decision.quorum",
        [_d("accept"), _d("block")],
        params={"accept_quorum": 1},
    )

    assert result.decision is not None
    assert result.decision.kind == "block"


def test_quorum_accept_reject_and_unknown_paths() -> None:
    accepted = aggregate_decisions(
        "decision.quorum",
        [_d("accept"), _d("reject")],
        params={"accept_quorum": 1, "min_known": 1},
    ).decision
    rejected = aggregate_decisions(
        "decision.quorum",
        [_d("reject"), _d("reject")],
        params={"accept_quorum": 2, "min_known": 2},
    ).decision
    unknown = aggregate_decisions(
        "decision.quorum",
        [_d("reject"), _d("unknown")],
        params={"accept_quorum": 2, "min_known": 2},
    ).decision

    assert accepted is not None
    assert rejected is not None
    assert unknown is not None
    assert accepted.kind == "accept"
    assert rejected.kind == "reject"
    assert unknown.kind == "unknown"


def test_weighted_vote_weight_only_and_zero_margin() -> None:
    accept = aggregate_decisions(
        "decision.weighted_vote",
        [_d("accept"), _d("reject")],
        params={"weights": {"0": "2", "1": "1"}},
    ).decision
    unknown = aggregate_decisions(
        "decision.weighted_vote",
        [_d("accept"), _d("reject")],
        params={"weights": {"0": "1", "1": "1"}},
    ).decision

    assert accept is not None
    assert unknown is not None
    assert accept.kind == "accept"
    assert unknown.kind == "unknown"


def test_weighted_vote_score_times_weight_requires_scores() -> None:
    missing = aggregate_decisions(
        "decision.weighted_vote",
        [_d("accept"), _d("reject", score="0.5")],
        params={"score_policy": "score_times_weight", "weights": {"0": "1", "1": "1"}},
    )
    accepted = aggregate_decisions(
        "decision.weighted_vote",
        [_d("accept", score="0.9"), _d("reject", score="0.5")],
        params={"score_policy": "score_times_weight", "weights": {"0": "1", "1": "1"}},
    )

    assert missing.decision is None
    assert missing.diagnostics.errors[0].code == "QST_V2_DECISION_MISSING_SCORE"
    assert accepted.decision is not None
    assert accepted.decision.kind == "accept"


def test_weighted_vote_missing_weight_is_error() -> None:
    result = aggregate_decisions(
        "decision.weighted_vote",
        [_d("accept")],
        params={"weights": {}},
    )

    assert result.decision is None
    assert result.diagnostics.errors[0].code == "QST_V2_DECISION_MISSING_WEIGHT"


def test_weighted_vote_weight_must_be_canonical_decimal_string() -> None:
    result = aggregate_decisions(
        "decision.weighted_vote",
        [_d("accept")],
        params={"weights": {"0": "1.0"}},
    )

    assert result.decision is None
    assert result.diagnostics.errors[0].code == "QST_V2_DECISION_INVALID_PARAMS"


def test_empty_aggregate_is_error() -> None:
    result = aggregate_decisions("decision.majority", [])

    assert result.decision is None
    assert result.diagnostics.errors[0].code == "QST_V2_DECISION_EMPTY_INPUT"
