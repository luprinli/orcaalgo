"""TokenPack metadata for WP7 Decision Algebra."""

from __future__ import annotations

from typing import Literal

from qst.ir import TokenRefV04
from qst.numeric import NumericPolicy
from qst.ports import InputSpec, OutputSpec
from qst.tokens import TokenPackManifestV2, TokenRiskSpec, TokenSpecV2, token_surface
from qst.types import parse_type_spec

DECISION_ALGEBRA_PACK_ID = "qst-tokenpack-decision-algebra"
DECISION_ALGEBRA_PACK_VERSION = "0.1.0"

DecisionAlgebraCategory = Literal["fold_policy", "monoid", "aggregator"]


def decision_algebra_token_pack_v2() -> TokenPackManifestV2:
    """Return the WP7 core TokenPack metadata for Decision Algebra."""

    return TokenPackManifestV2(
        pack_id=DECISION_ALGEBRA_PACK_ID,
        version=DECISION_ALGEBRA_PACK_VERSION,
        namespaces=("core",),
        tokens=(
            _decision_token_spec("decision.any_accept", "monoid"),
            _decision_token_spec("decision.majority", "aggregator"),
            _decision_token_spec("decision.permissive_and", "fold_policy"),
            _decision_token_spec("decision.quorum", "aggregator"),
            _decision_token_spec("decision.strict_and", "fold_policy"),
            _decision_token_spec("decision.unknown_propagating_and", "monoid"),
            _decision_token_spec("decision.weighted_vote", "aggregator"),
        ),
        origin_tier="core",
    )


def _decision_token_spec(name: str, category: DecisionAlgebraCategory) -> TokenSpecV2:
    return TokenSpecV2(
        token_id=f"core.{name}",
        token_ref=TokenRefV04(
            namespace="core",
            name=name,
            version=1,
            behavior_version=1,
        ),
        version=1,
        behavior_version=1,
        origin_tier="core",
        inputs={"decisions": InputSpec(type=parse_type_spec("EventStream[object]"))},
        outputs={"decision": OutputSpec(type=parse_type_spec("Decision"))},
        params_schema=_params_schema(category),
        purity="pure",
        state={
            "decision_algebra": True,
            "category": category,
            "wp": "WP7",
        },
        numeric_policy=NumericPolicy(
            representation="decimal",
            deterministic_level="semantic",
            reduction_order="fixed_input_order",
            nan_policy="reject",
            inf_policy="reject",
        ),
        risk=TokenRiskSpec(risk_level="low"),
        surface=token_surface(
            family="decision",
            category=category,
            layer="primitive" if category == "monoid" else "derived",
            maturity="accepted",
            execution_support="reference_helper",
            temporal="inherits_input_decision_events",
            numeric="score annotation ignored unless score_policy is declared",
            missing_data="unknown DecisionKind handled by policy",
            failure_mode="validation_result_diagnostics",
            examples=(f"core.{name}/v1/bv1",),
            common_mistakes=("Do not treat diagnostics as DecisionKind error.",),
        ),
        tests=[
            {
                "kind": "reference_helper",
                "deterministic": True,
            }
        ],
    )


def _params_schema(category: DecisionAlgebraCategory) -> dict[str, object]:
    if category == "aggregator":
        return {
            "type": "object",
            "additionalProperties": True,
            "properties": {
                "score_policy": {
                    "enum": ["ignore", "weight_only", "score_times_weight"],
                },
                "unknown_policy": {"enum": ["abstain"]},
            },
        }
    return {
        "type": "object",
        "additionalProperties": False,
        "properties": {
            "score_policy": {"enum": ["max_annotation", None]},
        },
    }
