from __future__ import annotations

from qst.types import parse_type_spec


def test_panel_type_shell_parses_without_axis_semantics() -> None:
    spec = parse_type_spec("Panel[float]")

    assert spec.kind == "Panel"
    assert spec.value_type is not None
    assert spec.value_type.name == "float"
    assert spec.model_dump(mode="json", exclude_none=True) == {
        "schema_version": "qst-typespec/0.4",
        "kind": "Panel",
        "value_type": "float",
        "axes": [],
        "universe": {"kind": "unspecified"},
        "missing_policy": "unknown",
        "group_spec_ref": "",
        "selection_kind": "none",
        "weight_constraints": {},
        "panel_capability_required": True,
    }
