from __future__ import annotations

import pytest

from qst.numeric import (
    NUMERIC_POLICY_SCHEMA_VERSION,
    NumericPolicy,
    semantic_float64_policy,
)


def test_semantic_float64_policy_is_stable() -> None:
    policy = semantic_float64_policy()

    assert policy.schema_version == NUMERIC_POLICY_SCHEMA_VERSION
    assert policy.representation == "float64"
    assert policy.deterministic_level == "semantic"
    assert policy.reduction_order == "fixed_input_order"
    assert policy.nan_policy == "propagate"
    assert policy.inf_policy == "reject"
    assert policy.risk_level == "low"


def test_unknown_numeric_policy_is_high_risk() -> None:
    policy = NumericPolicy(
        representation="unknown",
        deterministic_level="unknown",
        reduction_order="unknown",
        nan_policy="unknown",
        inf_policy="unknown",
    )

    assert policy.has_unknowns
    assert policy.risk_level == "high"


def test_platform_dependent_policy_is_high_risk() -> None:
    policy = NumericPolicy(
        representation="float64",
        deterministic_level="platform_dependent",
        reduction_order="fixed_input_order",
        nan_policy="propagate",
        inf_policy="reject",
    )

    assert not policy.has_unknowns
    assert policy.risk_level == "high"


def test_numeric_policy_requires_explicit_fields() -> None:
    with pytest.raises(ValueError):
        NumericPolicy.model_validate({"representation": "float64"})
