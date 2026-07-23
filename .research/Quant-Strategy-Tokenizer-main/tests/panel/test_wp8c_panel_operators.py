from __future__ import annotations

import pytest
from pydantic import ValidationError

from qst.hash import (
    signature_hash_for_panel_ports_v2,
    signature_hash_for_ports_v2,
    token_pack_hash_for_pack_v2,
    token_spec_hash_for_spec_v2,
)
from qst.ir import NodeV04, StrategyBodyV04, StrategyIRV04, validate_ir_v04
from qst.panel import (
    PanelPoint,
    PanelValue,
    SelectionPanelValue,
    SelectionPoint,
    panel_bottom_k,
    panel_group_demean,
    panel_mask,
    panel_rank,
    panel_residualize,
    panel_top_k,
    panel_winsorize,
    panel_zscore,
    selection_to_weights,
)
from qst.panel.token_pack import (
    PANEL_OPS_PACK_ID,
    PANEL_OPS_PACK_VERSION,
    panel_ops_token_pack_v2,
)
from qst.tokens import TokenRegistryV2

SHA = "sha256:" + "0" * 64
NON_PANEL_SIGNATURE_HASH = "sha256:faa375e10973332b887b9a8c98b1c996e87b08e419af07dd3b3d34431c61477a"


def _panel_metadata(kind: str = "panel") -> dict[str, object]:
    base: dict[str, object] = {
        "kind": kind,
        "universe_mask": {
            "universe_ref": "artifacts/universe/base.json",
            "members": ["BTC/USDT", "ETH/USDT", "SOL/USDT"],
            "included": ["BTC/USDT", "ETH/USDT", "SOL/USDT"],
        },
        "missing_policy": {"kind": "error_on_missing"},
    }
    if kind == "weight_panel":
        base["weight_panel"] = {
            "weight_kind": "raw",
            "weights_ref": "artifacts/weights/raw.json",
            "weights_hash": SHA,
        }
    return base


def _panel_operator_node(name: str = "panel.rank") -> NodeV04:
    return NodeV04(
        id="panel_op",
        token_ref={"namespace": "core", "name": name, "version": 1, "behavior_version": 1},
        signature={"outputs": {"ranked": {"type": "Panel[int]"}}},
        metadata={"panel_type_by_output": {"ranked": _panel_metadata()}},
    )


def _ir(node: NodeV04, capabilities: list[str]) -> StrategyIRV04:
    return StrategyIRV04(
        capabilities=capabilities,
        strategy=StrategyBodyV04(id="panel_ops_strategy", nodes=[node]),
    )


def _sample_panel() -> PanelValue:
    return PanelValue(
        rows=(
            PanelPoint(timestamp="2026-05-15T00:00:00Z", symbol="BTC/USDT", value="3"),
            PanelPoint(timestamp="2026-05-15T00:00:00Z", symbol="ETH/USDT", value="3"),
            PanelPoint(timestamp="2026-05-15T00:00:00Z", symbol="SOL/USDT", value="1"),
            PanelPoint(timestamp="2026-05-15T00:01:00Z", symbol="BTC/USDT", value="2"),
            PanelPoint(timestamp="2026-05-15T00:01:00Z", symbol="ETH/USDT", value="4"),
        )
    )


def test_panel_ops_capability_is_explicit_and_future_capabilities_stay_rejected() -> None:
    assert validate_ir_v04(_ir(_panel_operator_node(), ["core", "panel_type", "panel_ops"])).ok

    no_type = validate_ir_v04(_ir(_panel_operator_node(), ["core", "panel_ops"]))
    no_ops = validate_ir_v04(_ir(_panel_operator_node(), ["core", "panel_type"]))
    with pytest.raises(ValidationError):
        _ir(_panel_operator_node(), ["core", "panel"])
    weights = validate_ir_v04(
        _ir(_panel_operator_node("weight.equal"), ["core", "panel_type", "panel_ops"])
    )
    recipes = validate_ir_v04(_ir(_panel_operator_node(), ["core", "panel_type", "panel_recipes"]))

    assert "QST_V2_CAPABILITY_PANEL_OPS_REQUIRES_PANEL_TYPE" in {
        diagnostic.code for diagnostic in no_type.errors
    }
    assert "QST_V2_CAPABILITY_PANEL_OPS_REQUIRED" in {diagnostic.code for diagnostic in no_ops.errors}
    assert "QST_V2_WEIGHT_OPERATOR_DEFERRED" in {
        diagnostic.code for diagnostic in weights.errors
    }
    assert recipes.errors[0].code == "capability_not_accepted"


def test_deferred_panel_select_is_not_accepted() -> None:
    result = validate_ir_v04(
        _ir(_panel_operator_node("panel.select"), ["core", "panel_type", "panel_ops"])
    )

    assert not result.ok
    assert result.errors[0].code == "QST_V2_PANEL_OPERATOR_NOT_ACCEPTED"


def test_panel_rank_is_deterministic_and_uses_symbol_order_tie_break() -> None:
    result = panel_rank(_sample_panel())

    assert result.diagnostics.ok
    assert [(row.timestamp, row.symbol, row.value) for row in result.panel.rows] == [
        ("2026-05-15T00:00:00Z", "BTC/USDT", "1"),
        ("2026-05-15T00:00:00Z", "ETH/USDT", "2"),
        ("2026-05-15T00:00:00Z", "SOL/USDT", "3"),
        ("2026-05-15T00:01:00Z", "BTC/USDT", "2"),
        ("2026-05-15T00:01:00Z", "ETH/USDT", "1"),
    ]
    assert result.trace["tie_policy"] == "stable_symbol_order"


def test_missing_active_universe_cell_is_error_but_mask_false_is_out_of_universe() -> None:
    missing = PanelValue(
        rows=(
            PanelPoint(timestamp="t1", symbol="BTC/USDT", value=None, in_universe=True),
            PanelPoint(timestamp="t1", symbol="ETH/USDT", value="1", in_universe=False),
        )
    )
    masked = panel_mask(
        PanelValue(rows=(PanelPoint(timestamp="t1", symbol="BTC/USDT", value="1"),)),
        SelectionPanelValue(rows=(SelectionPoint(timestamp="t1", symbol="BTC/USDT", selected=False),)),
    )
    ranked = panel_rank(missing)

    assert masked.panel.rows[0].in_universe is False
    assert not ranked.diagnostics.ok
    assert ranked.diagnostics.errors[0].code == "QST_V2_PANEL_MISSING_VALUE"


def test_zscore_zero_variance_outputs_zero() -> None:
    panel = PanelValue(
        rows=(
            PanelPoint(timestamp="t1", symbol="BTC/USDT", value="2"),
            PanelPoint(timestamp="t1", symbol="ETH/USDT", value="2"),
        )
    )
    result = panel_zscore(panel)

    assert result.diagnostics.ok
    assert [row.value for row in result.panel.rows] == ["0", "0"]
    assert result.trace["zero_variance_policy"] == "output_zero"


def test_top_bottom_k_allow_smaller_and_trace_counts() -> None:
    top = panel_top_k(_sample_panel(), k=10)
    bottom = panel_bottom_k(_sample_panel(), k=1)

    assert [row.selected for row in top.selection.rows if row.timestamp == "2026-05-15T00:00:00Z"] == [
        True,
        True,
        True,
    ]
    assert top.trace["actual_selected"]["2026-05-15T00:00:00Z"] == 3
    assert [
        (row.symbol, row.selected)
        for row in bottom.selection.rows
        if row.timestamp == "2026-05-15T00:00:00Z"
    ] == [("BTC/USDT", False), ("ETH/USDT", False), ("SOL/USDT", True)]


def test_group_demean_requires_group_material_and_demeans_by_group() -> None:
    result = panel_group_demean(
        _sample_panel(),
        groups={"BTC/USDT": "major", "ETH/USDT": "major", "SOL/USDT": "alt"},
    )
    missing_group = panel_group_demean(_sample_panel(), groups={"BTC/USDT": "major"})

    assert result.diagnostics.ok
    assert [(row.symbol, row.value) for row in result.panel.rows if row.timestamp == "2026-05-15T00:00:00Z"] == [
        ("BTC/USDT", "0"),
        ("ETH/USDT", "0"),
        ("SOL/USDT", "0"),
    ]
    assert not missing_group.diagnostics.ok
    assert missing_group.diagnostics.errors[0].code == "QST_V2_PANEL_GROUP_MISSING"


def test_winsorize_uses_nearest_rank_index_formula() -> None:
    panel = PanelValue(
        rows=(
            PanelPoint(timestamp="t1", symbol="A", value="1"),
            PanelPoint(timestamp="t1", symbol="B", value="2"),
            PanelPoint(timestamp="t1", symbol="C", value="3"),
            PanelPoint(timestamp="t1", symbol="D", value="4"),
        )
    )
    result = panel_winsorize(panel, lower_quantile="0.25", upper_quantile="0.75")

    assert result.diagnostics.ok
    assert [row.value for row in result.panel.rows] == ["1", "2", "3", "3"]
    assert result.trace["interpolation"] == "nearest_rank"


def test_residualize_single_factor_insufficient_observations_yields_unknown_warning() -> None:
    panel = PanelValue(
        rows=(
            PanelPoint(timestamp="t1", symbol="BTC/USDT", value="1"),
            PanelPoint(timestamp="t2", symbol="BTC/USDT", value="2"),
        )
    )
    result = panel_residualize(panel, factor={"t1": "1", "t2": "2"})

    assert result.diagnostics.ok
    assert result.diagnostics.warnings[0].code == "QST_V2_PANEL_RESIDUALIZE_INSUFFICIENT_OBSERVATIONS"
    assert [row.value for row in result.panel.rows] == [None, None]
    assert result.trace["min_observations"] == 3


def test_selection_to_weights_outputs_raw_equal_weights() -> None:
    selection = SelectionPanelValue(
        selection_kind="long_short",
        rows=(
            SelectionPoint(timestamp="t1", symbol="BTC/USDT", selected=True, side="long"),
            SelectionPoint(timestamp="t1", symbol="ETH/USDT", selected=True, side="short"),
            SelectionPoint(timestamp="t1", symbol="SOL/USDT", selected=True, side="long"),
        ),
    )
    long_weights = selection_to_weights(selection)
    long_short_weights = selection_to_weights(selection, method="equal_long_short")

    assert long_weights.weights.normalized is False
    assert [(row.symbol, row.weight) for row in long_weights.weights.rows] == [
        ("BTC/USDT", "0.3333333333333333"),
        ("ETH/USDT", "0.3333333333333333"),
        ("SOL/USDT", "0.3333333333333333"),
    ]
    assert [(row.symbol, row.weight) for row in long_short_weights.weights.rows] == [
        ("BTC/USDT", "0.5"),
        ("ETH/USDT", "-1"),
        ("SOL/USDT", "0.5"),
    ]


def test_equal_long_short_rejects_side_both() -> None:
    selection = SelectionPanelValue(
        selection_kind="long_short",
        rows=(SelectionPoint(timestamp="t1", symbol="BTC/USDT", selected=True, side="both"),),
    )

    result = selection_to_weights(selection, method="equal_long_short")

    assert result.weights is None
    assert [diagnostic.code for diagnostic in result.diagnostics.diagnostics] == [
        "QST_V2_SELECTION_SIDE_BOTH_UNSUPPORTED"
    ]


def test_panel_ops_token_pack_validates_and_hashes_semantic_params() -> None:
    pack = panel_ops_token_pack_v2()
    registry = TokenRegistryV2.from_packs((pack,))
    zscore = next(spec for spec in pack.tokens if spec.token_ref.name == "panel.zscore")
    changed = zscore.model_copy(
        update={
            "params_schema": {
                **zscore.params_schema,
                "properties": {
                    **zscore.params_schema["properties"],
                    "zero_variance_policy": {"const": "error"},
                },
            }
        }
    )

    assert pack.pack_id == PANEL_OPS_PACK_ID
    assert pack.version == PANEL_OPS_PACK_VERSION
    assert registry.result.ok
    assert [record.spec.token_id for record in registry.records] == [
        f"core.{name}" for name in sorted(spec.token_ref.name for spec in pack.tokens)
    ]
    assert token_pack_hash_for_pack_v2(pack) == token_pack_hash_for_pack_v2(panel_ops_token_pack_v2())
    assert token_spec_hash_for_spec_v2(zscore) != token_spec_hash_for_spec_v2(changed)
    assert {spec.state["category"] for spec in pack.tokens} == {"panel_operator", "selection_operator"}
    for spec in pack.tokens:
        if spec.numeric_policy.representation == "float64":
            assert spec.numeric_policy.deterministic_level == "semantic"
        assert spec.numeric_policy.deterministic_level != "bit_exact"


def test_panel_operator_metadata_affects_panel_signature_hash_but_not_non_panel_hash() -> None:
    signature = {"outputs": {"panel": {"type": "Panel[float]"}}}
    metadata = {"panel": _panel_metadata()}
    changed = {"panel": {**_panel_metadata(), "missing_policy": {"kind": "drop_missing"}}}

    assert signature_hash_for_panel_ports_v2(signature, metadata) != signature_hash_for_panel_ports_v2(
        signature, changed
    )
    assert (
        signature_hash_for_ports_v2({"outputs": {"value": {"type": "Scalar[float]"}}})
        == NON_PANEL_SIGNATURE_HASH
    )
