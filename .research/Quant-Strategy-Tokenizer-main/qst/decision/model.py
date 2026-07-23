"""Decision Algebra models for Token System v2 WP7."""

from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator

from qst.artifacts.decimal_string import (
    DecimalString,
    validate_decimal_string,
)
from qst.canonical_json import stable_json_bytes
from qst.validation import ValidationResult

DECISION_SCHEMA_VERSION: Literal["qst-decision/0.4"] = "qst-decision/0.4"
DECISION_MONOID_SCHEMA_VERSION: Literal["qst-decision-monoid/0.4"] = (
    "qst-decision-monoid/0.4"
)
DECISION_FOLD_POLICY_SCHEMA_VERSION: Literal["qst-decision-fold-policy/0.4"] = (
    "qst-decision-fold-policy/0.4"
)
DECISION_AGGREGATOR_SCHEMA_VERSION: Literal["qst-decision-aggregator/0.4"] = (
    "qst-decision-aggregator/0.4"
)

DecisionKind = Literal["accept", "reject", "unknown", "block"]
DecisionScorePolicy = Literal["max_annotation"]
DecisionMonoidId = Literal[
    "decision.unknown_propagating_and",
    "decision.any_accept",
]
DecisionFoldPolicyId = Literal[
    "decision.strict_and",
    "decision.permissive_and",
    "decision.unknown_propagating_and",
    "decision.any_accept",
]
DecisionAggregatorId = Literal[
    "decision.majority",
    "decision.weighted_vote",
    "decision.quorum",
]

FINAL_UNKNOWN_AS_REJECT_REASON = "DECISION_FINALIZED_UNKNOWN_AS_REJECT"
FINAL_UNKNOWN_AS_ACCEPT_REASON = "DECISION_FINALIZED_UNKNOWN_AS_ACCEPT"


class DecisionV2(BaseModel):
    """Canonical v0.4 decision carrier."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-decision/0.4"] = DECISION_SCHEMA_VERSION
    kind: DecisionKind
    reasons: tuple[str, ...] = Field(default_factory=tuple)
    score: DecimalString | None = None

    @field_validator("reasons", mode="before")
    @classmethod
    def _canonicalize_reasons(cls, value: object) -> tuple[str, ...]:
        if value is None:
            return ()
        if isinstance(value, str):
            value = (value,)
        if not isinstance(value, (list, tuple)):
            raise TypeError("Decision reasons must be a list or tuple of strings")
        reasons = []
        for item in value:
            if not isinstance(item, str) or not item:
                raise ValueError("Decision reasons must be non-empty strings")
            reasons.append(item)
        return tuple(sorted(set(reasons)))

    @field_validator("score", mode="before")
    @classmethod
    def _validate_score(cls, value: object) -> object:
        if value is None:
            return None
        if not isinstance(value, str):
            raise TypeError("Decision score must be a canonical DecimalString")
        return validate_decimal_string(value)


class DecisionMonoidSpec(BaseModel):
    """A true monoid over DecisionKind."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-decision-monoid/0.4"] = DECISION_MONOID_SCHEMA_VERSION
    monoid_id: DecisionMonoidId
    identity: DecisionKind
    priority: tuple[DecisionKind, ...]
    score_policy: DecisionScorePolicy | None = None

    @model_validator(mode="after")
    def _validate_priority(self) -> DecisionMonoidSpec:
        if set(self.priority) != {"accept", "reject", "unknown", "block"}:
            raise ValueError("Decision monoid priority must contain all DecisionKind values")
        return self


class DecisionFoldPolicySpec(BaseModel):
    """Public fold policy backed by a true monoid plus finalizer."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-decision-fold-policy/0.4"] = (
        DECISION_FOLD_POLICY_SCHEMA_VERSION
    )
    policy_id: DecisionFoldPolicyId
    base_monoid: DecisionMonoidId
    final_unknown: Literal["accept", "reject", "unknown"]
    score_policy: DecisionScorePolicy | None = None


class DecisionAggregatorSpec(BaseModel):
    """Decision aggregator metadata."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-decision-aggregator/0.4"] = (
        DECISION_AGGREGATOR_SCHEMA_VERSION
    )
    aggregator_id: DecisionAggregatorId
    score_policy: Literal["ignore", "weight_only", "score_times_weight"] = "ignore"
    unknown_policy: Literal["abstain"] = "abstain"
    missing_score_policy: Literal["error", "abstain", "use_default"] = "error"
    missing_weight_policy: Literal["error", "use_one"] = "error"


class DecisionCombineResult(BaseModel):
    """Decision helper result with validation diagnostics."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    decision: DecisionV2 | None = None
    diagnostics: ValidationResult = Field(default_factory=ValidationResult)

    @property
    def ok(self) -> bool:
        """Whether the result has a decision and no error diagnostics."""

        return self.decision is not None and self.diagnostics.ok


def ensure_decision_payload_json(value: Any, *, field_name: str) -> None:
    """Validate canonical JSON compatibility for Decision Algebra metadata."""

    try:
        stable_json_bytes(value)
    except (TypeError, ValueError) as exc:
        raise ValueError(f"{field_name} must be canonical JSON-compatible") from exc
