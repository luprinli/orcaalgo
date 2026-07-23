from __future__ import annotations

from qst.hash import signature_hash_for_ports_v2, signature_hash_v2
from qst.ports import PortSignature


def test_signature_hash_supports_empty_payload() -> None:
    assert signature_hash_v2() == signature_hash_v2({})


def test_signature_hash_is_deterministic() -> None:
    left = signature_hash_v2({"outputs": ["b", "a"], "inputs": {"x": "series"}})
    right = signature_hash_v2({"inputs": {"x": "series"}, "outputs": ["b", "a"]})

    assert left == right


def test_signature_hash_includes_structured_port_signature() -> None:
    signature = PortSignature(
        inputs={"price": {"type": "TimeSeries[float]"}},
        outputs={"signal": {"type": "TimeSeries[bool]"}},
    )
    changed = PortSignature(
        inputs={"price": {"type": "TimeSeries[float]"}},
        outputs={"signal": {"type": "TimeSeries[float]"}},
    )

    assert signature_hash_for_ports_v2(signature) != signature_hash_for_ports_v2(changed)


def test_signature_hash_for_ports_is_order_stable() -> None:
    left = {
        "inputs": {
            "b": {"type": "Scalar[int]"},
            "a": {"type": "Scalar[float]"},
        },
        "outputs": {
            "z": {"type": "Decision"},
            "c": {"type": "Plan"},
        },
    }
    right = {
        "outputs": {
            "c": {"type": "Plan"},
            "z": {"type": "Decision"},
        },
        "inputs": {
            "a": {"type": "Scalar[float]"},
            "b": {"type": "Scalar[int]"},
        },
    }

    assert signature_hash_for_ports_v2(left) == signature_hash_for_ports_v2(right)
