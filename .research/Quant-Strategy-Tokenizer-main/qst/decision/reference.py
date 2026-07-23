"""Deterministic reference helpers for Token System v2 Decision Algebra."""

from __future__ import annotations

from decimal import Decimal
from typing import Any, Literal, TypedDict, cast

from qst.artifacts.decimal_string import (
    normalize_to_canonical,
    validate_decimal_string,
)
from qst.decision.model import (
    FINAL_UNKNOWN_AS_ACCEPT_REASON,
    FINAL_UNKNOWN_AS_REJECT_REASON,
    DecisionCombineResult,
    DecisionFoldPolicyId,
    DecisionKind,
    DecisionMonoidId,
    DecisionScorePolicy,
    DecisionV2,
)
from qst.validation import Diagnostic, ValidationResult


def combine_decisions(
    monoid_id: str,
    decisions: list[DecisionV2 | dict[str, Any]],
    *,
    score_policy: DecisionScorePolicy | None = None,
) -> DecisionCombineResult:
    """Combine decisions with a true monoid."""

    if monoid_id not in _MONOID_PRIORITY:
        return _error("QST_V2_DECISION_UNKNOWN_MONOID", f"Unknown monoid {monoid_id!r}")
    if not decisions:
        return _error("QST_V2_DECISION_EMPTY_INPUT", "Decision combine requires non-empty input.")
    parsed = _parse_decisions(decisions)
    if isinstance(parsed, DecisionCombineResult):
        return parsed

    priority = _MONOID_PRIORITY[monoid_id]
    kind = max((decision.kind for decision in parsed), key=lambda item: priority[item])
    return DecisionCombineResult(
        decision=DecisionV2(
            kind=kind,
            reasons=_merge_reasons(parsed),
            score=_merge_score(parsed, score_policy),
        )
    )


def fold_decisions(
    policy_id: str,
    decisions: list[DecisionV2 | dict[str, Any]],
    *,
    score_policy: DecisionScorePolicy | None = None,
) -> DecisionCombineResult:
    """Fold decisions with a public DecisionFoldPolicy."""

    if not decisions:
        return _error("QST_V2_DECISION_EMPTY_INPUT", "Decision fold requires non-empty input.")
    if policy_id in _MONOID_PRIORITY:
        return combine_decisions(policy_id, decisions, score_policy=score_policy)
    policy = _FOLD_POLICIES.get(cast(DecisionFoldPolicyId, policy_id))
    if policy is None:
        return _error("QST_V2_DECISION_UNKNOWN_FOLD_POLICY", f"Unknown fold policy {policy_id!r}")

    result = combine_decisions(policy["base_monoid"], decisions, score_policy=score_policy)
    if result.decision is None:
        return result
    if result.decision.kind != "unknown":
        return result

    reason = (
        FINAL_UNKNOWN_AS_REJECT_REASON
        if policy["final_unknown"] == "reject"
        else FINAL_UNKNOWN_AS_ACCEPT_REASON
    )
    return DecisionCombineResult(
        decision=DecisionV2(
            kind=policy["final_unknown"],
            reasons=(*result.decision.reasons, reason),
            score=result.decision.score,
        )
    )


def aggregate_decisions(
    aggregator_id: str,
    decisions: list[DecisionV2 | dict[str, Any]],
    *,
    params: dict[str, Any] | None = None,
) -> DecisionCombineResult:
    """Aggregate decisions with a non-monoid aggregator."""

    params = params or {}
    if not decisions:
        return _error("QST_V2_DECISION_EMPTY_INPUT", "Decision aggregate requires non-empty input.")
    parsed = _parse_decisions(decisions)
    if isinstance(parsed, DecisionCombineResult):
        return parsed

    if any(decision.kind == "block" for decision in parsed):
        return _block_result(parsed)
    if aggregator_id == "decision.majority":
        return _majority(parsed)
    if aggregator_id == "decision.quorum":
        return _quorum(parsed, params)
    if aggregator_id == "decision.weighted_vote":
        return _weighted_vote(parsed, params)
    return _error(
        "QST_V2_DECISION_UNKNOWN_AGGREGATOR",
        f"Unknown aggregator {aggregator_id!r}",
    )


def is_monoid_id(identifier: str) -> bool:
    """Whether an id is registered as a true monoid."""

    return identifier in _MONOID_PRIORITY


def is_fold_policy_id(identifier: str) -> bool:
    """Whether an id is registered as a fold policy or monoid-backed fold."""

    return identifier in _FOLD_POLICIES or identifier in _MONOID_PRIORITY


def is_aggregator_id(identifier: str) -> bool:
    """Whether an id is registered as an aggregator."""

    return identifier in _AGGREGATOR_IDS


def _parse_decisions(
    decisions: list[DecisionV2 | dict[str, Any]],
) -> list[DecisionV2] | DecisionCombineResult:
    parsed: list[DecisionV2] = []
    for index, decision in enumerate(decisions):
        try:
            parsed.append(
                decision if isinstance(decision, DecisionV2) else DecisionV2.model_validate(decision)
            )
        except ValueError as exc:
            return _error(
                "QST_V2_DECISION_INVALID_INPUT",
                f"Decision at index {index} is invalid: {exc}",
            )
    return parsed


def _majority(decisions: list[DecisionV2]) -> DecisionCombineResult:
    accepts = sum(1 for decision in decisions if decision.kind == "accept")
    rejects = sum(1 for decision in decisions if decision.kind == "reject")
    if accepts > rejects:
        kind: DecisionKind = "accept"
    elif rejects > accepts:
        kind = "reject"
    else:
        kind = "unknown"
    return DecisionCombineResult(decision=DecisionV2(kind=kind, reasons=_merge_reasons(decisions)))


def _quorum(decisions: list[DecisionV2], params: dict[str, Any]) -> DecisionCombineResult:
    accept_quorum = params.get("accept_quorum")
    min_known = params.get("min_known")
    if not isinstance(accept_quorum, int) or accept_quorum <= 0:
        return _error("QST_V2_DECISION_INVALID_PARAMS", "accept_quorum must be a positive integer.")
    if min_known is not None and (
        not isinstance(min_known, int) or min_known < accept_quorum
    ):
        return _error(
            "QST_V2_DECISION_INVALID_PARAMS",
            "min_known must be an integer greater than or equal to accept_quorum.",
        )

    accepts = sum(1 for decision in decisions if decision.kind == "accept")
    known = sum(1 for decision in decisions if decision.kind in {"accept", "reject"})
    if accepts >= accept_quorum:
        kind: DecisionKind = "accept"
    elif min_known is not None and known >= min_known:
        kind = "reject"
    else:
        kind = "unknown"
    return DecisionCombineResult(decision=DecisionV2(kind=kind, reasons=_merge_reasons(decisions)))


def _weighted_vote(decisions: list[DecisionV2], params: dict[str, Any]) -> DecisionCombineResult:
    score_policy = params.get("score_policy", "weight_only")
    missing_weight_policy = params.get("missing_weight_policy", "error")
    missing_score_policy = params.get("missing_score_policy", "error")
    weights = params.get("weights", {})
    if score_policy not in {"weight_only", "score_times_weight"}:
        return _error("QST_V2_DECISION_INVALID_PARAMS", "Unsupported weighted_vote score_policy.")
    if missing_weight_policy not in {"error", "use_one"}:
        return _error(
            "QST_V2_DECISION_INVALID_PARAMS",
            "Unsupported weighted_vote missing_weight_policy.",
        )
    if missing_score_policy not in {"error", "abstain", "use_default"}:
        return _error(
            "QST_V2_DECISION_INVALID_PARAMS",
            "Unsupported weighted_vote missing_score_policy.",
        )
    if not isinstance(weights, dict):
        return _error("QST_V2_DECISION_INVALID_PARAMS", "weights must be a mapping.")

    margin = Decimal("0")
    for index, decision in enumerate(decisions):
        if decision.kind == "unknown":
            continue
        if decision.kind not in {"accept", "reject"}:
            continue
        try:
            weight = _weight_for_index(weights, index, missing_weight_policy)
        except ValueError as exc:
            return _error("QST_V2_DECISION_INVALID_PARAMS", str(exc))
        if weight is None:
            return _error(
                "QST_V2_DECISION_MISSING_WEIGHT",
                f"Missing weight for decision index {index}.",
            )
        contribution = weight
        if score_policy == "score_times_weight":
            score = _score_for_decision(decision, missing_score_policy)
            if score is None:
                return _error(
                    "QST_V2_DECISION_MISSING_SCORE",
                    f"Missing score for decision index {index}.",
                )
            if score < Decimal("0") or score > Decimal("1"):
                return _error(
                    "QST_V2_DECISION_SCORE_OUT_OF_RANGE",
                    "score_times_weight requires score in [0, 1].",
                )
            contribution *= score
        margin += contribution if decision.kind == "accept" else -contribution

    if margin > 0:
        kind: DecisionKind = "accept"
    elif margin < 0:
        kind = "reject"
    else:
        kind = "unknown"
    return DecisionCombineResult(decision=DecisionV2(kind=kind, reasons=_merge_reasons(decisions)))


def _block_result(decisions: list[DecisionV2]) -> DecisionCombineResult:
    return DecisionCombineResult(decision=DecisionV2(kind="block", reasons=_merge_reasons(decisions)))


def _weight_for_index(
    weights: dict[Any, Any],
    index: int,
    missing_weight_policy: object,
) -> Decimal | None:
    raw = weights.get(str(index), weights.get(index))
    if raw is None:
        return Decimal("1") if missing_weight_policy == "use_one" else None
    if not isinstance(raw, str):
        raise ValueError("weights must be canonical DecimalString values.")
    validate_decimal_string(raw)
    return Decimal(str(raw))


def _score_for_decision(
    decision: DecisionV2,
    missing_score_policy: object,
) -> Decimal | None:
    if decision.score is not None:
        return Decimal(decision.score)
    if missing_score_policy == "use_default":
        return Decimal("1")
    if missing_score_policy == "abstain":
        return Decimal("0")
    return None


def _merge_reasons(decisions: list[DecisionV2]) -> tuple[str, ...]:
    return tuple(sorted({reason for decision in decisions for reason in decision.reasons}))


def _merge_score(
    decisions: list[DecisionV2],
    score_policy: DecisionScorePolicy | None,
) -> str | None:
    if score_policy is None:
        return None
    scores = [Decimal(decision.score) for decision in decisions if decision.score is not None]
    if not scores:
        return None
    return normalize_to_canonical(max(scores))


def _error(code: str, message: str) -> DecisionCombineResult:
    return DecisionCombineResult(
        diagnostics=ValidationResult(
            diagnostics=[
                Diagnostic(
                    code=code,
                    severity="error",
                    phase="schema",
                    message=message,
                )
            ]
        )
    )


_MONOID_PRIORITY: dict[DecisionMonoidId, dict[DecisionKind, int]] = {
    "decision.unknown_propagating_and": {
        "accept": 0,
        "unknown": 1,
        "reject": 2,
        "block": 3,
    },
    "decision.any_accept": {
        "reject": 0,
        "unknown": 1,
        "accept": 2,
        "block": 3,
    },
}

class _FoldPolicyData(TypedDict):
    base_monoid: DecisionMonoidId
    final_unknown: Literal["accept", "reject"]


_FOLD_POLICIES: dict[DecisionFoldPolicyId, _FoldPolicyData] = {
    "decision.strict_and": {
        "base_monoid": "decision.unknown_propagating_and",
        "final_unknown": "reject",
    },
    "decision.permissive_and": {
        "base_monoid": "decision.unknown_propagating_and",
        "final_unknown": "accept",
    },
}

_AGGREGATOR_IDS = {
    "decision.majority",
    "decision.weighted_vote",
    "decision.quorum",
}
