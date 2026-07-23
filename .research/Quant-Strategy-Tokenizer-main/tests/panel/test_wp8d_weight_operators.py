from __future__ import annotations

from qst.hash import (
    token_pack_hash_for_pack_v2,
    token_spec_hash_for_spec_v2,
)
from qst.ir import NodeV04, StrategyBodyV04, StrategyIRV04, validate_ir_v04
from qst.panel import (
    WeightPanelValue,
    WeightPoint,
    weight_cap_per_symbol,
    weight_market_neutral,
    weight_normalize_gross,
)
from qst.panel.token_pack import (
    PANEL_WEIGHTS_PACK_ID,
    PANEL_WEIGHTS_PACK_VERSION,
    panel_weights_token_pack_v2,
)
from qst.tokens import TokenRegistryV2

SHA = "sha256:" + "0" * 64


def _weight_metadata() -> dict[str, object]:
    return {
        "kind": "weight_panel",
        "universe_mask": {
            "universe_ref": "artifacts/universe/base.json",
            "members": ["BTC/USDT", "ETH/USDT", "SOL/USDT"],
            "included": ["BTC/USDT", "ETH/USDT", "SOL/USDT"],
        },
        "missing_policy": {"kind": "error_on_missing"},
        "weight_panel": {
            "weight_kind": "raw",
            "weights_ref": "artifacts/weights/raw.json",
            "weights_hash": SHA,
        },
    }


def _source_node(*, output_type: str = "Panel[decimal]", metadata: bool = True) -> NodeV04:
    return NodeV04(
        id="raw",
        token_ref={"namespace": "core", "name": "type.weight_panel", "version": 1, "behavior_version": 1},
        signature={"outputs": {"weights": {"type": output_type}}},
        metadata={"panel_type_by_output": {"weights": _weight_metadata()}} if metadata else {},
    )


def _weight_op_node(
    name: str = "weight.normalize_gross",
    *,
    input_type: str = "Panel[decimal]",
    input_ref: str = "raw.weights",
    output_type: str = "Panel[decimal]",
    metadata: bool = True,
) -> NodeV04:
    return NodeV04(
        id="weight_op",
        token_ref={"namespace": "core", "name": name, "version": 1, "behavior_version": 1},
        inputs={"weights": input_ref},
        signature={
            "inputs": {"weights": {"type": input_type}},
            "outputs": {"weights": {"type": output_type}},
        },
        metadata={"panel_type_by_output": {"weights": _weight_metadata()}} if metadata else {},
    )


def _ir(nodes: list[NodeV04], capabilities: list[str]) -> StrategyIRV04:
    return StrategyIRV04(
        capabilities=capabilities,
        strategy=StrategyBodyV04(id="weight_ops_strategy", nodes=nodes),
    )


def _weights() -> WeightPanelValue:
    return WeightPanelValue(
        rows=(
            WeightPoint(timestamp="t1", symbol="BTC/USDT", weight="0.6"),
            WeightPoint(timestamp="t1", symbol="ETH/USDT", weight="0.2"),
            WeightPoint(timestamp="t1", symbol="SOL/USDT", weight="-0.2"),
        )
    )


def test_panel_weights_capability_is_explicit_but_panel_ops_is_not_required() -> None:
    valid = validate_ir_v04(
        _ir([_source_node(), _weight_op_node()], ["core", "panel_type", "panel_weights"])
    )
    without_type = validate_ir_v04(_ir([_source_node(), _weight_op_node()], ["core", "panel_weights"]))
    without_weights = validate_ir_v04(_ir([_source_node(), _weight_op_node()], ["core", "panel_type"]))

    assert valid.ok
    assert without_type.errors[0].code == "QST_V2_CAPABILITY_PANEL_WEIGHTS_REQUIRES_PANEL_TYPE"
    assert "QST_V2_CAPABILITY_PANEL_WEIGHTS_REQUIRED" in {
        diagnostic.code for diagnostic in without_weights.errors
    }


def test_weight_operator_rejects_non_weightpanel_input_and_missing_metadata() -> None:
    bad_type = validate_ir_v04(
        _ir(
            [_source_node(output_type="Panel[float]"), _weight_op_node(input_type="Panel[float]")],
            ["core", "panel_type", "panel_weights"],
        )
    )
    missing_metadata = validate_ir_v04(
        _ir([_source_node(metadata=False), _weight_op_node()], ["core", "panel_type", "panel_weights"])
    )
    bad_output = validate_ir_v04(
        _ir(
            [_source_node(), _weight_op_node(output_type="Panel[float]", metadata=False)],
            ["core", "panel_type", "panel_weights"],
        )
    )

    assert "QST_V2_WEIGHT_INPUT_NOT_WEIGHT_PANEL" in {diagnostic.code for diagnostic in bad_type.errors}
    assert "QST_V2_WEIGHT_INPUT_METADATA_REQUIRED" in {
        diagnostic.code for diagnostic in missing_metadata.errors
    }
    assert "QST_V2_WEIGHT_OUTPUT_NOT_WEIGHT_PANEL" in {diagnostic.code for diagnostic in bad_output.errors}
    assert "QST_V2_WEIGHT_OUTPUT_METADATA_REQUIRED" in {
        diagnostic.code for diagnostic in bad_output.errors
    }


def test_deferred_and_unknown_weight_operators_are_stable_diagnostics() -> None:
    deferred = validate_ir_v04(
        _ir([_source_node(), _weight_op_node("weight.equal")], ["core", "panel_type", "panel_weights"])
    )
    unknown = validate_ir_v04(
        _ir(
            [_source_node(), _weight_op_node("weight.factor_neutral")],
            ["core", "panel_type", "panel_weights"],
        )
    )

    assert deferred.errors[0].code == "QST_V2_WEIGHT_OPERATOR_DEFERRED"
    assert unknown.errors[0].code == "QST_V2_WEIGHT_OPERATOR_NOT_ACCEPTED"


def test_normalize_gross_canonicalizes_params_and_handles_zero_gross() -> None:
    normalized = weight_normalize_gross(_weights(), target_gross="1.0")
    zero = WeightPanelValue(rows=(WeightPoint(timestamp="t1", symbol="BTC/USDT", weight="0"),))
    zero_keep = weight_normalize_gross(zero, zero_gross_policy="keep_zero")
    zero_error = weight_normalize_gross(zero, zero_gross_policy="error")

    assert normalized.diagnostics.ok
    assert normalized.trace["target_gross"] == "1"
    assert [(row.symbol, row.weight) for row in normalized.weights.rows] == [
        ("BTC/USDT", "0.6"),
        ("ETH/USDT", "0.2"),
        ("SOL/USDT", "-0.2"),
    ]
    assert normalized.weights.weight_kind == "normalized"
    assert normalized.weights.normalized is True
    assert zero_keep.weights.rows[0].weight == "0"
    assert not zero_error.diagnostics.ok
    assert zero_error.diagnostics.errors[0].code == "QST_V2_WEIGHT_ZERO_GROSS"


def test_cap_per_symbol_clips_zero_cap_and_does_not_renormalize() -> None:
    capped = weight_cap_per_symbol(_weights(), max_abs_weight="0.10")
    all_zero = weight_cap_per_symbol(_weights(), max_abs_weight="0")
    unsupported = weight_cap_per_symbol(_weights(), max_abs_weight="0.1", mode="redistribute")

    assert capped.diagnostics.ok
    assert [(row.symbol, row.weight) for row in capped.weights.rows] == [
        ("BTC/USDT", "0.1"),
        ("ETH/USDT", "0.1"),
        ("SOL/USDT", "-0.1"),
    ]
    assert capped.trace["gross_after"]["t1"] == "0.3"
    assert [row.weight for row in all_zero.weights.rows] == ["0", "0", "0"]
    assert not unsupported.diagnostics.ok
    assert unsupported.diagnostics.errors[0].code == "QST_V2_WEIGHT_CAP_MODE_UNSUPPORTED"


def test_market_neutral_demeans_then_normalizes_and_target_net_is_zero_only() -> None:
    neutral = weight_market_neutral(_weights(), target_gross="1.00")
    unsupported_net = weight_market_neutral(_weights(), target_net="0.1")

    assert neutral.diagnostics.ok
    assert [(row.symbol, row.weight) for row in neutral.weights.rows] == [
        ("BTC/USDT", "0.5"),
        ("ETH/USDT", "0"),
        ("SOL/USDT", "-0.5"),
    ]
    assert neutral.trace["adjustment"]["t1"] == "0.2"
    assert neutral.trace["gross_after"]["t1"] == "1"
    assert not unsupported_net.diagnostics.ok
    assert unsupported_net.diagnostics.errors[0].code == "QST_V2_WEIGHT_TARGET_NET_UNSUPPORTED"


def test_market_neutral_zero_gross_policy_branches_and_empty_universe() -> None:
    equal = WeightPanelValue(
        rows=(
            WeightPoint(timestamp="t1", symbol="BTC/USDT", weight="0.2"),
            WeightPoint(timestamp="t1", symbol="ETH/USDT", weight="0.2"),
        )
    )
    empty = WeightPanelValue(rows=(WeightPoint(timestamp="t1", symbol="BTC/USDT", weight="0.2", in_universe=False),))
    keep_zero = weight_market_neutral(equal, zero_gross_policy="keep_zero")
    error = weight_market_neutral(equal, zero_gross_policy="error")
    empty_result = weight_market_neutral(empty)

    assert keep_zero.diagnostics.ok
    assert keep_zero.diagnostics.warnings[0].code == "QST_V2_WEIGHT_MARKET_NEUTRAL_ZEROED"
    assert [row.weight for row in keep_zero.weights.rows] == ["0", "0"]
    assert not error.diagnostics.ok
    assert error.diagnostics.errors[0].code == "QST_V2_WEIGHT_MARKET_NEUTRAL_ZERO_GROSS"
    assert not empty_result.diagnostics.ok
    assert empty_result.diagnostics.errors[0].code == "QST_V2_WEIGHT_MARKET_NEUTRAL_EMPTY_UNIVERSE"


def test_gross_and_cap_are_not_simultaneously_guaranteed_by_composition_order() -> None:
    raw = WeightPanelValue(
        rows=(
            WeightPoint(timestamp="t1", symbol="BTC/USDT", weight="0.9"),
            WeightPoint(timestamp="t1", symbol="ETH/USDT", weight="0.1"),
        )
    )
    normalized_then_capped = weight_cap_per_symbol(
        weight_normalize_gross(raw, target_gross="1").weights,
        max_abs_weight="0.4",
    )
    capped_then_normalized = weight_normalize_gross(
        weight_cap_per_symbol(raw, max_abs_weight="0.4").weights,
        target_gross="1",
    )

    assert normalized_then_capped.trace["gross_after"]["t1"] == "0.5"
    assert [row.weight for row in capped_then_normalized.weights.rows] == ["0.8", "0.2"]


def test_panel_weights_token_pack_validates_and_hashes_semantic_params() -> None:
    pack = panel_weights_token_pack_v2()
    registry = TokenRegistryV2.from_packs((pack,))
    cap = next(spec for spec in pack.tokens if spec.token_ref.name == "weight.cap_per_symbol")
    changed = cap.model_copy(
        update={
            "params_schema": {
                **cap.params_schema,
                "properties": {
                    **cap.params_schema["properties"],
                    "mode": {"const": "redistribute"},
                },
            }
        }
    )

    assert pack.pack_id == PANEL_WEIGHTS_PACK_ID
    assert pack.version == PANEL_WEIGHTS_PACK_VERSION
    assert registry.result.ok
    assert [record.spec.token_id for record in registry.records] == [
        "core.weight.cap_per_symbol",
        "core.weight.market_neutral",
        "core.weight.normalize_gross",
    ]
    assert token_pack_hash_for_pack_v2(pack) == token_pack_hash_for_pack_v2(panel_weights_token_pack_v2())
    assert token_spec_hash_for_spec_v2(cap) != token_spec_hash_for_spec_v2(changed)
    for spec in pack.tokens:
        assert spec.numeric_policy.deterministic_level == "semantic"
        assert spec.numeric_policy.deterministic_level != "bit_exact"
