"""NumericPolicy metadata for Token System v2.

Numeric behavior is explicit, canonical, and hash-bearing metadata.
"""

from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, ConfigDict

NUMERIC_POLICY_SCHEMA_VERSION: Literal["qst-numeric-policy/0.4"] = "qst-numeric-policy/0.4"

NumericRepresentation = Literal[
    "float64",
    "float32",
    "decimal",
    "int64",
    "bool",
    "object",
    "unknown",
]
DeterministicLevel = Literal[
    "semantic",
    "bit_exact",
    "engine_specific",
    "platform_dependent",
    "unknown",
]
ReductionOrder = Literal[
    "fixed_input_order",
    "commutative",
    "engine_default",
    "parallel_nondeterministic",
    "unknown",
]
NanPolicy = Literal["propagate", "reject", "ignore", "coerce_null", "unknown"]
InfPolicy = Literal["reject", "propagate", "coerce_null", "unknown"]
NumericPolicyRisk = Literal["low", "medium", "high"]

DETERMINISTIC_LEVELS: tuple[DeterministicLevel, ...] = (
    "semantic",
    "bit_exact",
    "engine_specific",
    "platform_dependent",
    "unknown",
)


class NumericPolicy(BaseModel):
    """Explicit numeric behavior declaration for v0.4 token behavior material."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-numeric-policy/0.4"] = NUMERIC_POLICY_SCHEMA_VERSION
    representation: NumericRepresentation
    deterministic_level: DeterministicLevel
    reduction_order: ReductionOrder
    nan_policy: NanPolicy
    inf_policy: InfPolicy

    @property
    def has_unknowns(self) -> bool:
        """Whether any part of the numeric policy is declared unknown."""

        return (
            self.representation == "unknown"
            or self.deterministic_level == "unknown"
            or self.reduction_order == "unknown"
            or self.nan_policy == "unknown"
            or self.inf_policy == "unknown"
        )

    @property
    def risk_level(self) -> NumericPolicyRisk:
        """Coarse profile-gate risk level for the numeric policy."""

        if self.has_unknowns or self.deterministic_level == "platform_dependent":
            return "high"
        if (
            self.deterministic_level == "engine_specific"
            or self.reduction_order in {"engine_default", "parallel_nondeterministic"}
        ):
            return "medium"
        return "low"


def semantic_float64_policy() -> NumericPolicy:
    """Return the Wexample policy for deterministic semantic float64 tokens."""

    return NumericPolicy(
        representation="float64",
        deterministic_level="semantic",
        reduction_order="fixed_input_order",
        nan_policy="propagate",
        inf_policy="reject",
    )
