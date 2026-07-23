from __future__ import annotations

from qst.types import IntrinsicTemporalSpec, TypeSpec, parse_type_spec


def test_timeseries_float_with_intrinsic_temporal() -> None:
    spec = TypeSpec(
        kind="TimeSeries",
        value_type="float",
        intrinsic_temporal=IntrinsicTemporalSpec(
            default_available_at="bar_close",
            default_clock="bar",
        ),
    )

    assert spec.model_dump(mode="json", exclude_none=True) == {
        "schema_version": "qst-typespec/0.4",
        "kind": "TimeSeries",
        "value_type": "float",
        "intrinsic_temporal": {
            "schema_version": "qst-typespec/0.4",
            "default_available_at": "bar_close",
            "default_clock": "bar",
        },
    }


def test_timeseries_shorthand_canonical_dump() -> None:
    spec = parse_type_spec("TimeSeries[float]")

    assert spec.model_dump(mode="json", exclude_none=True) == {
        "schema_version": "qst-typespec/0.4",
        "kind": "TimeSeries",
        "value_type": "float",
    }


def test_intrinsic_temporal_defaults_are_stable() -> None:
    temporal = IntrinsicTemporalSpec()

    assert temporal.model_dump(mode="json") == {
        "schema_version": "qst-typespec/0.4",
        "default_available_at": "bar_close",
        "default_clock": "bar",
    }
