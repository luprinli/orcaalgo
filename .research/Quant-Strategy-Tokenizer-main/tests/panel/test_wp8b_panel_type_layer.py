from __future__ import annotations

import hashlib
import json
from pathlib import Path
from typing import Any

import pytest
from pydantic import ValidationError

from qst.hash import (
    signature_hash_for_panel_ports_v2,
    signature_hash_for_ports_v2,
)
from qst.ir import NodeV04, StrategyBodyV04, StrategyIRV04, validate_ir_v04
from qst.ir.canonical import canonicalize_v04
from qst.panel import (
    GroupSpec,
    PanelMissingPolicy,
    PanelRepresentation,
    SelectionPanelType,
    UniverseMask,
    WeightPanelType,
)
from qst.types import TypeSpec, parse_type_spec

ROOT = Path(__file__).resolve().parents[2]
SCHEMA_DIR = ROOT / "docs" / "schemas"
SHA = "sha256:" + "0" * 64
TYPESPEC_SCHEMA_HASH_WP8A = "sha256:f181ce889c24abc36bb57cc5662b50669331b68f2000e7df0dbe6c42647207fd"
NON_PANEL_SIGNATURE_HASH = "sha256:faa375e10973332b887b9a8c98b1c996e87b08e419af07dd3b3d34431c61477a"


def _schema_hash(name: str) -> str:
    schema = json.loads((SCHEMA_DIR / name).read_text(encoding="utf-8"))
    data = json.dumps(schema, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode(
        "utf-8"
    )
    return "sha256:" + hashlib.sha256(data).hexdigest()


def _panel_metadata(
    *,
    missing_policy: str = "error_on_missing",
    universe_ref: str = "artifacts/universe/base.json",
    group_mapping_ref: str | None = None,
) -> dict[str, Any]:
    group_spec = None
    if group_mapping_ref is not None:
        group_spec = {
            "kind": "static_mapping",
            "group_id": "sector",
            "mapping_ref": group_mapping_ref,
            "mapping_hash": SHA,
        }
    return {
        "panel": {
            "universe_mask": {
                "universe_ref": universe_ref,
                "members": ["ETH/USDT", "BTC/USDT"],
                "included": ["BTC/USDT"],
            },
            "missing_policy": {"kind": missing_policy},
            **({"group_spec": group_spec} if group_spec is not None else {}),
        }
    }


def _panel_node(metadata: dict[str, Any] | None = None) -> NodeV04:
    return NodeV04(
        id="panel_node",
        token_ref={
            "namespace": "core",
            "name": "type.panel_shell",
            "version": 1,
            "behavior_version": 1,
        },
        signature={
            "outputs": {
                "panel": {
                    "type": "Panel[float]",
                }
            }
        },
        metadata={"panel_type_by_output": _panel_metadata()} if metadata is None else metadata,
    )


def _ir_with_node(node: NodeV04, *, capabilities: list[str] | None = None) -> StrategyIRV04:
    return StrategyIRV04(
        capabilities=capabilities or ["core", "panel_type"],
        strategy=StrategyBodyV04(id="panel_type_layer", nodes=[node]),
    )


def test_schema_correction_adr_and_ir_schema_reserve_granular_capabilities() -> None:
    schema = json.loads((SCHEMA_DIR / "ir-0.4.schema.json").read_text(encoding="utf-8"))
    capabilities = set(schema["properties"]["capabilities"]["items"]["enum"])

    assert {"panel_type", "panel_ops", "panel_weights", "panel_recipes"} <= capabilities
    assert "panel" not in capabilities


def test_panel_type_models_validate_wp8a_semantics() -> None:
    representation = PanelRepresentation()
    mask = UniverseMask(
        universe_ref="artifacts/universe/base.json",
        members=["ETH/USDT", "BTC/USDT"],
        included=["BTC/USDT"],
    )
    missing_policy = PanelMissingPolicy(kind="drop_missing")
    group = GroupSpec(
        kind="static_mapping",
        group_id="sector",
        mapping_ref="artifacts/groups/sector.json",
        mapping_hash=SHA,
    )
    selection = SelectionPanelType(
        selection_kind="long_only",
        selected={"universe_mask_ref": "artifacts/masks/top.json", "universe_mask_hash": SHA},
        side="long",
    )
    weight = WeightPanelType(
        weight_kind="raw",
        weights_ref="artifacts/weights/raw.json",
        weights_hash=SHA,
        gross_exposure="1",
        weight_constraints={"gross_target": "1"},
    )

    assert representation.kind == "sparse_logical"
    assert mask.members == ["BTC/USDT", "ETH/USDT"]
    assert mask.false_semantics == "out_of_universe_not_missing"
    assert missing_policy.applies_when == "universe_mask_true_value_missing"
    assert group.missing_group_policy == "error"
    assert selection.kind == "selection_panel"
    assert weight.kind == "weight_panel"


def test_panel_type_models_reject_dynamic_group_and_bad_missing_shapes() -> None:
    with pytest.raises(ValidationError):
        GroupSpec.model_validate({"kind": "dynamic_mapping", "group_id": "sector"})
    with pytest.raises(ValidationError):
        GroupSpec.model_validate({"kind": "static_mapping", "group_id": "sector", "mapping_hash": SHA})
    with pytest.raises(ValidationError):
        PanelMissingPolicy(kind="error_on_missing", nullable_decimal_string=True)


def test_panel_type_capability_validates_and_panel_ops_requires_panel_type() -> None:
    valid = validate_ir_v04(_ir_with_node(_panel_node()))
    rejected = {
        capability: validate_ir_v04(_ir_with_node(_panel_node(), capabilities=["core", capability]))
        for capability in ["panel_ops", "panel_weights", "panel_recipes"]
    }

    assert valid.ok
    with pytest.raises(ValidationError):
        _ir_with_node(_panel_node(), capabilities=["core", "panel"])
    assert rejected["panel_ops"].errors[0].code == "QST_V2_CAPABILITY_PANEL_OPS_REQUIRES_PANEL_TYPE"
    assert rejected["panel_weights"].errors[0].code == "QST_V2_CAPABILITY_PANEL_WEIGHTS_REQUIRES_PANEL_TYPE"
    for capability, result in rejected.items():
        assert not result.ok
        if capability in {"panel_ops", "panel_weights"}:
            continue
        assert result.errors[0].code == "capability_not_accepted", capability

    custom_runtime = validate_ir_v04(
        StrategyIRV04(
            capabilities=["core", "custom_token_runtime"],
            strategy=StrategyBodyV04(id="custom_runtime_only"),
        )
    )
    assert custom_runtime.ok


def test_panel_output_requires_panel_type_capability_and_output_metadata() -> None:
    no_capability = validate_ir_v04(_ir_with_node(_panel_node(), capabilities=["core"]))
    no_metadata = validate_ir_v04(_ir_with_node(_panel_node(metadata={}), capabilities=["core", "panel_type"]))

    assert not no_capability.ok
    assert [diagnostic.code for diagnostic in no_capability.errors] == [
        "QST_V2_PANEL_TYPE_CAPABILITY_REQUIRED"
    ]
    assert not no_metadata.ok
    assert [diagnostic.code for diagnostic in no_metadata.errors] == [
        "QST_V2_PANEL_TYPE_METADATA_REQUIRED"
    ]


def test_single_panel_type_metadata_location_is_rejected() -> None:
    result = validate_ir_v04(
        _ir_with_node(
            _panel_node(
                metadata={
                    "panel_type": _panel_metadata()["panel"],
                    "panel_type_by_output": _panel_metadata(),
                }
            )
        )
    )

    assert not result.ok
    assert result.errors[0].code == "QST_V2_PANEL_TYPE_METADATA_LOCATION"


def test_panel_type_metadata_must_match_outputs_and_panel_outputs() -> None:
    unknown_output = validate_ir_v04(
        _ir_with_node(
            _panel_node(
                metadata={
                    "panel_type_by_output": {
                        **_panel_metadata(),
                        "missing_output": _panel_metadata()["panel"],
                    }
                }
            )
        )
    )
    non_panel_output = validate_ir_v04(
        _ir_with_node(
            NodeV04(
                id="scalar_node",
                signature={"outputs": {"value": {"type": "Scalar[float]"}}},
                metadata={"panel_type_by_output": {"value": _panel_metadata()["panel"]}},
            )
        )
    )

    assert "QST_V2_PANEL_TYPE_METADATA_UNKNOWN_OUTPUT" in {
        diagnostic.code for diagnostic in unknown_output.errors
    }
    assert "QST_V2_PANEL_TYPE_METADATA_NON_PANEL_OUTPUT" in {
        diagnostic.code for diagnostic in non_panel_output.errors
    }


def test_panel_operator_requires_ops_capability_and_state_autobroadcast_remains_rejected() -> None:
    panel_operator = validate_ir_v04(
        _ir_with_node(
            NodeV04(
                id="op",
                token_ref={
                    "namespace": "core",
                    "name": "panel.rank",
                    "version": 1,
                    "behavior_version": 1,
                },
                signature={"outputs": {"ranked": {"type": "Panel[float]"}}},
                metadata={"panel_type_by_output": {"ranked": _panel_metadata()["panel"]}},
            )
        )
    )
    state_panel = validate_ir_v04(
        _ir_with_node(
            NodeV04(
                id="fsm",
                token_ref={
                    "namespace": "core",
                    "name": "state.fsm",
                    "version": 1,
                    "behavior_version": 1,
                },
                signature={
                    "inputs": {"events": {"type": "Panel[string]"}},
                    "outputs": {"state": {"type": "State[string]"}},
                },
            )
        )
    )

    assert "QST_V2_CAPABILITY_PANEL_OPS_REQUIRED" in {
        diagnostic.code for diagnostic in panel_operator.errors
    }
    assert "QST_V2_PANEL_STATE_AUTOBROADCAST_UNSUPPORTED" in {
        diagnostic.code for diagnostic in state_panel.errors
    }


def test_panel_semantic_metadata_enters_signature_hash() -> None:
    signature = {"outputs": {"panel": {"type": "Panel[float]"}}}
    base = signature_hash_for_panel_ports_v2(signature, _panel_metadata())
    changed_missing_policy = signature_hash_for_panel_ports_v2(
        signature, _panel_metadata(missing_policy="drop_missing")
    )
    changed_universe = signature_hash_for_panel_ports_v2(
        signature, _panel_metadata(universe_ref="artifacts/universe/other.json")
    )
    changed_group = signature_hash_for_panel_ports_v2(
        signature, _panel_metadata(group_mapping_ref="artifacts/groups/sector.json")
    )

    assert base != changed_missing_policy
    assert base != changed_universe
    assert base != changed_group


def test_unrelated_metadata_does_not_enter_panel_signature_hash() -> None:
    signature = {"outputs": {"panel": {"type": "Panel[float]"}}}
    metadata_a = {"panel_type_by_output": _panel_metadata(), "note": "a"}
    metadata_b = {"panel_type_by_output": _panel_metadata(), "note": "b"}

    assert signature_hash_for_panel_ports_v2(signature, metadata_a["panel_type_by_output"]) == (
        signature_hash_for_panel_ports_v2(signature, metadata_b["panel_type_by_output"])
    )


def test_non_panel_signature_hash_remains_pinned() -> None:
    assert (
        signature_hash_for_ports_v2({"outputs": {"value": {"type": "Scalar[float]"}}})
        == NON_PANEL_SIGNATURE_HASH
    )


def test_typespec_schema_and_model_shape_remain_unchanged_from_wp8a() -> None:
    schema = json.loads((SCHEMA_DIR / "type_spec-0.4.schema.json").read_text(encoding="utf-8"))
    panel_fields = {
        "axes",
        "universe",
        "missing_policy",
        "group_spec_ref",
        "selection_kind",
        "weight_constraints",
        "panel_capability_required",
    }

    assert _schema_hash("type_spec-0.4.schema.json") == TYPESPEC_SCHEMA_HASH_WP8A
    assert panel_fields <= set(TypeSpec.model_fields)
    assert schema["properties"]["missing_policy"]["enum"] == ["unknown", "dense", "sparse"]
    assert schema["properties"]["selection_kind"]["enum"] == ["none", "static", "dynamic", "weighted"]
    assert parse_type_spec("Panel[float]").missing_policy == "unknown"


def test_already_supported_panel_value_types_parse_without_extending_values() -> None:
    assert parse_type_spec("Panel[float]").value_type.name == "float"
    assert parse_type_spec("Panel[decimal]").value_type.name == "decimal"


def test_panel_type_capability_survives_canonicalization() -> None:
    canonical = canonicalize_v04(_ir_with_node(_panel_node()))

    assert canonical.capabilities == ["core", "panel_type"]
    assert canonical.strategy.nodes[0].metadata["panel_type_by_output"]["panel"]["universe_mask"][
        "universe_ref"
    ] == "artifacts/universe/base.json"
