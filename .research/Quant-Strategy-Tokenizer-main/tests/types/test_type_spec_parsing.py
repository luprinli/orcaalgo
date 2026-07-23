from __future__ import annotations

import pytest
from pydantic import ValidationError

from qst.types import TypeSpec, ValueType, parse_type_spec


def test_shorthand_timeseries_float_parses_to_structured_type() -> None:
    spec = parse_type_spec("TimeSeries[float]")

    assert spec.kind == "TimeSeries"
    assert spec.value_type == ValueType("float")
    assert spec.model_dump(mode="json", exclude_none=True) == {
        "schema_version": "qst-typespec/0.4",
        "kind": "TimeSeries",
        "value_type": "float",
    }


def test_bare_decision_and_plan_parse_without_value_type() -> None:
    assert parse_type_spec("Decision").model_dump(mode="json", exclude_none=True) == {
        "schema_version": "qst-typespec/0.4",
        "kind": "Decision"
    }
    assert parse_type_spec("Plan").model_dump(mode="json", exclude_none=True) == {
        "schema_version": "qst-typespec/0.4",
        "kind": "Plan",
    }


def test_structured_model_validation() -> None:
    spec = TypeSpec.model_validate({"kind": "Scalar", "value_type": "decimal"})

    assert spec.kind == "Scalar"
    assert spec.value_type is not None
    assert spec.value_type.name == "decimal"


def test_invalid_shorthand_rejected() -> None:
    with pytest.raises(ValidationError):
        TypeSpec.model_validate("BadType[float]")


def test_typed_kind_requires_value_type() -> None:
    with pytest.raises(ValidationError):
        TypeSpec(kind="TimeSeries")


def test_decision_rejects_value_type() -> None:
    with pytest.raises(ValidationError):
        TypeSpec.model_validate({"kind": "Decision", "value_type": "bool"})
