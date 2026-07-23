from __future__ import annotations

from qst.ports import (
    InputSpec,
    OutputSpec,
    PortSignature,
    PortTemporalSpec,
    TemporalRequirement,
)


def test_input_spec_has_type_and_temporal_requirement() -> None:
    spec = InputSpec(
        type="TimeSeries[float]",
        temporal_requirement=TemporalRequirement(
            max_available_at="bar_close",
            allow_unsafe_future=False,
        ),
    )

    assert spec.model_dump(mode="json", exclude_none=True) == {
        "schema_version": "qst-portspec/0.4",
        "type": {"schema_version": "qst-typespec/0.4", "kind": "TimeSeries", "value_type": "float"},
        "temporal_requirement": {
            "schema_version": "qst-port-temporal/0.4",
            "max_available_at": "bar_close",
            "allow_unsafe_future": False,
        },
    }


def test_output_spec_has_type_and_port_temporal() -> None:
    spec = OutputSpec(
        type="TimeSeries[bool]",
        port_temporal=PortTemporalSpec(
            available_at="bar_close",
            latency_bars=0,
            min_history_bars=0,
            unsafe_future=False,
        ),
    )

    assert spec.model_dump(mode="json", exclude_none=True) == {
        "schema_version": "qst-portspec/0.4",
        "type": {"schema_version": "qst-typespec/0.4", "kind": "TimeSeries", "value_type": "bool"},
        "port_temporal": {
            "schema_version": "qst-port-temporal/0.4",
            "available_at": "bar_close",
            "latency_bars": 0,
            "min_history_bars": 0,
            "unsafe_future": False,
        },
    }


def test_type_intrinsic_temporal_port_temporal_and_input_requirement_are_distinct() -> None:
    signature = PortSignature(
        inputs={
            "price": InputSpec(
                type={
                    "kind": "TimeSeries",
                    "value_type": "float",
                    "intrinsic_temporal": {
                        "default_available_at": "bar_close",
                        "default_clock": "bar",
                    },
                },
                temporal_requirement={"max_available_at": "bar_close"},
            )
        },
        outputs={
            "signal": OutputSpec(
                type="TimeSeries[bool]",
                port_temporal={"available_at": "bar_close", "latency_bars": 0},
            )
        },
    )
    payload = signature.model_dump(mode="json", exclude_none=True)

    assert "intrinsic_temporal" in payload["inputs"]["price"]["type"]
    assert "temporal_requirement" in payload["inputs"]["price"]
    assert "port_temporal" in payload["outputs"]["signal"]
    assert "port_temporal" not in payload["inputs"]["price"]["type"]
    assert "temporal_requirement" not in payload["outputs"]["signal"]


def test_empty_port_signature_is_stable() -> None:
    assert PortSignature().model_dump(mode="json") == {
        "schema_version": "qst-portspec/0.4",
        "inputs": {},
        "outputs": {},
    }
