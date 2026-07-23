from __future__ import annotations

import hashlib
import json
from pathlib import Path
from typing import Any

import pytest
from jsonschema import Draft202012Validator, ValidationError

from qst.ir import StrategyBodyV04, StrategyIRV04
from qst.types import TypeSpec, parse_type_spec

ROOT = Path(__file__).resolve().parents[2]
SCHEMA_DIR = ROOT / "docs" / "schemas"

PANEL_SHELL_FIELDS = {
    "axes",
    "universe",
    "missing_policy",
    "group_spec_ref",
    "selection_kind",
    "weight_constraints",
    "panel_capability_required",
}

EXPECTED_SCHEMA_HASHES = {
    "panel_representation-0.4.schema.json": (
        "sha256:ed2fea8886b9a229b71b86ebaf5b4a717cf14e8e0cf4b23fcaebac192df66588"
    ),
    "panel_universe_mask-0.4.schema.json": (
        "sha256:f171e4f4c1a0702835606c1100fc2bf0e93a31a632bb89c00aada8a0329f1e32"
    ),
    "panel_missing_policy-0.4.schema.json": (
        "sha256:7ce719f73763b0d974d58bdb0d0ca2550c0b74cdbe93dff5d68bdbee23cbcbb6"
    ),
    "panel_group_spec-0.4.schema.json": (
        "sha256:b44d24f02e29cdc73a22becacbe392f761b24c734c0c31e2ea13364131209b4f"
    ),
    "panel_selection_weight-0.4.schema.json": (
        "sha256:baff36b907bdf7fafe400bf881866af2b87444d3e3fc78cd14906837edac9155"
    ),
    "panel_temporal_state-0.4.schema.json": (
        "sha256:1cedea73f3d006e533aa3586455fb88e6f64924e313b90a914353bfe39566362"
    ),
    "type_spec-0.4.schema.json": (
        "sha256:f181ce889c24abc36bb57cc5662b50669331b68f2000e7df0dbe6c42647207fd"
    ),
}

SHA = "sha256:" + "0" * 64


def _load_schema(name: str) -> dict[str, Any]:
    return json.loads((SCHEMA_DIR / name).read_text(encoding="utf-8"))


def _schema_hash(schema: dict[str, Any]) -> str:
    data = json.dumps(schema, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode(
        "utf-8"
    )
    return "sha256:" + hashlib.sha256(data).hexdigest()


def _validate(schema_name: str, instance: dict[str, Any]) -> None:
    Draft202012Validator(_load_schema(schema_name)).validate(instance)


def _schema_version_constants(schema: dict[str, Any]) -> list[str]:
    constants: list[str] = []
    if "schema_version" in schema.get("properties", {}):
        constants.append(schema["properties"]["schema_version"]["const"])
    for branch in schema.get("oneOf", []):
        if "schema_version" in branch.get("properties", {}):
            constants.append(branch["properties"]["schema_version"]["const"])
    return constants


def test_wp8a_schemas_are_valid_and_hash_pinned() -> None:
    for name, expected_hash in EXPECTED_SCHEMA_HASHES.items():
        schema = _load_schema(name)

        Draft202012Validator.check_schema(schema)

        assert schema["$id"].startswith("https://qst.local/schemas/")
        assert _schema_version_constants(schema)
        assert _schema_hash(schema) == expected_hash


def test_panel_representation_schema_accepts_sparse_logical() -> None:
    _validate(
        "panel_representation-0.4.schema.json",
        {
            "schema_version": "qst-panel-representation/0.4",
            "kind": "sparse_logical",
            "universe_mask_required": True,
        },
    )


def test_universe_mask_false_is_out_of_universe_not_missing() -> None:
    instance = {
        "schema_version": "qst-panel-universe-mask/0.4",
        "representation": "sparse_logical",
        "universe_ref": "artifacts/universe/btc_eth.json",
        "members": ["BTC/USDT", "ETH/USDT"],
        "included": ["BTC/USDT"],
        "false_semantics": "out_of_universe_not_missing",
    }

    _validate("panel_universe_mask-0.4.schema.json", instance)

    assert "ETH/USDT" not in instance["included"]
    assert instance["false_semantics"] == "out_of_universe_not_missing"


def test_missing_policy_excludes_sparse_representation_and_nullable_decimal() -> None:
    schema = _load_schema("panel_missing_policy-0.4.schema.json")

    assert "sparse_logical" not in schema["properties"]["kind"]["enum"]
    assert schema["properties"]["kind"]["default"] == "error_on_missing"

    _validate(
        "panel_missing_policy-0.4.schema.json",
        {
            "schema_version": "qst-panel-missing-policy/0.4",
            "kind": "error_on_missing",
            "applies_when": "universe_mask_true_value_missing",
            "nullable_decimal_string": False,
        },
    )

    with pytest.raises(ValidationError):
        _validate(
            "panel_missing_policy-0.4.schema.json",
            {
                "schema_version": "qst-panel-missing-policy/0.4",
                "kind": "sparse_logical",
                "applies_when": "universe_mask_true_value_missing",
            },
        )
    with pytest.raises(ValidationError):
        _validate(
            "panel_missing_policy-0.4.schema.json",
            {
                "schema_version": "qst-panel-missing-policy/0.4",
                "kind": "propagate_missing",
                "applies_when": "universe_mask_true_value_missing",
                "nullable_decimal_string": False,
            },
        )
    with pytest.raises(ValidationError):
        _validate(
            "panel_missing_policy-0.4.schema.json",
            {
                "schema_version": "qst-panel-missing-policy/0.4",
                "kind": "drop_missing",
                "applies_when": "universe_mask_true_value_missing",
                "nullable_decimal_string": True,
            },
        )


def test_group_spec_static_mapping_and_field_ref_contracts() -> None:
    schema = _load_schema("panel_group_spec-0.4.schema.json")
    for branch in schema["oneOf"]:
        assert branch["properties"]["missing_group_policy"]["default"] == "error"

    _validate(
        "panel_group_spec-0.4.schema.json",
        {
            "schema_version": "qst-panel-group-spec/0.4",
            "kind": "static_mapping",
            "group_id": "sector",
            "mapping_ref": "artifacts/groups/sector_map.json",
            "mapping_hash": SHA,
            "missing_group_policy": "error",
            "group_label_type": "string",
        },
    )
    _validate(
        "panel_group_spec-0.4.schema.json",
        {
            "schema_version": "qst-panel-group-spec/0.4",
            "kind": "field_ref",
            "group_id": "sector",
            "field_path": "universe.metadata.sector",
            "missing_group_policy": "error",
            "group_label_type": "string",
        },
    )

    with pytest.raises(ValidationError):
        _validate(
            "panel_group_spec-0.4.schema.json",
            {
                "schema_version": "qst-panel-group-spec/0.4",
                "kind": "dynamic_mapping",
                "group_id": "sector",
                "missing_group_policy": "error",
                "group_label_type": "string",
            },
        )
    with pytest.raises(ValidationError):
        _validate(
            "panel_group_spec-0.4.schema.json",
            {
                "schema_version": "qst-panel-group-spec/0.4",
                "kind": "static_mapping",
                "group_id": "sector",
                "mapping_ref": "artifacts/groups/sector_map.json",
                "missing_group_policy": "error",
                "group_label_type": "string",
            },
        )
    with pytest.raises(ValidationError):
        _validate(
            "panel_group_spec-0.4.schema.json",
            {
                "schema_version": "qst-panel-group-spec/0.4",
                "kind": "field_ref",
                "group_id": "sector",
                "missing_group_policy": "error",
                "group_label_type": "string",
            },
        )


def test_selection_panel_and_weight_panel_are_distinct_wire_concepts() -> None:
    selection = {
        "schema_version": "qst-panel-selection-weight/0.4",
        "kind": "selection_panel",
        "selection_kind": "long_only",
        "selected": {"universe_mask_ref": "artifacts/masks/top.json", "universe_mask_hash": SHA},
        "side": "long",
        "score_ref": None,
    }
    weight = {
        "schema_version": "qst-panel-selection-weight/0.4",
        "kind": "weight_panel",
        "weight_kind": "raw",
        "weights_ref": "artifacts/weights/raw.json",
        "weights_hash": SHA,
        "gross_exposure": None,
        "net_exposure": None,
        "weight_constraints": {
            "max_abs_weight_per_symbol": None,
            "gross_target": None,
            "net_target": None,
        },
    }

    _validate("panel_selection_weight-0.4.schema.json", selection)
    _validate("panel_selection_weight-0.4.schema.json", weight)

    invalid_selection = {**selection, "weight_kind": "raw"}
    with pytest.raises(ValidationError):
        _validate("panel_selection_weight-0.4.schema.json", invalid_selection)


def test_temporal_join_residualize_and_panel_state_boundaries() -> None:
    instance = {
        "schema_version": "qst-panel-temporal-state/0.4",
        "temporal_join": {
            "unsafe_future": "any_input_unsafe_future",
            "available_at": "max_available_at_inputs",
            "latency_bars": "max_input_latency_bars",
            "min_history_bars": "max_input_min_history_or_operator_required_history",
        },
        "residualize_v1": {
            "panel": "Panel[float]",
            "factor": "TimeSeries[float]",
            "output": "Panel[float]",
            "scope": "single_factor_only",
        },
        "panel_state": {
            "panel_state_shell_allowed": True,
            "state_fsm_auto_broadcast": False,
        },
    }

    _validate("panel_temporal_state-0.4.schema.json", instance)

    assert instance["residualize_v1"] == {
        "panel": "Panel[float]",
        "factor": "TimeSeries[float]",
        "output": "Panel[float]",
        "scope": "single_factor_only",
    }
    with pytest.raises(ValidationError):
        _validate(
            "panel_temporal_state-0.4.schema.json",
            {
                **instance,
                "panel_state": {
                    "panel_state_shell_allowed": True,
                    "state_fsm_auto_broadcast": True,
                },
            },
        )


def test_typespec_panel_shell_field_set_is_frozen() -> None:
    model_fields = set(TypeSpec.model_fields)
    schema_properties = set(_load_schema("type_spec-0.4.schema.json")["properties"])

    assert PANEL_SHELL_FIELDS <= model_fields
    assert PANEL_SHELL_FIELDS <= schema_properties
    assert {
        field
        for field in model_fields
        if field
        in {
            "axes",
            "universe",
            "missing_policy",
            "group_spec_ref",
            "selection_kind",
            "weight_constraints",
            "panel_capability_required",
        }
    } == PANEL_SHELL_FIELDS

    panel = parse_type_spec("Panel[float]")
    assert panel.model_dump(mode="json", exclude_none=True) == {
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


def test_panel_state_nested_shorthand_remains_schema_level_only() -> None:
    with pytest.raises(ValueError, match="Unsupported TypeSpec shorthand"):
        parse_type_spec("Panel[State[string]]")


def test_panel_capability_remains_rejected() -> None:
    with pytest.raises(ValueError):
        StrategyIRV04(capabilities=["core", "panel"], strategy=StrategyBodyV04(id="panel_gate"))
