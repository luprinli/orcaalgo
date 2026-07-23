"""Panel type-layer validation for qst-ir/0.4."""

from __future__ import annotations

from pydantic import ValidationError

from qst.ir.schema import NodeV04, StrategyIRV04
from qst.panel.model import PanelTypeLayerSpec, parse_panel_type_by_output
from qst.panel.operators import (
    PANEL_OPERATOR_TOKENS,
    WEIGHT_OPERATOR_TOKENS,
)
from qst.validation import Diagnostic

PANEL_OPS_CAPABILITY = "panel_ops"
PANEL_TYPE_CAPABILITY = "panel_type"
PANEL_WEIGHTS_CAPABILITY = "panel_weights"
PANEL_OPERATOR_PREFIXES = ("panel.", "selection.", "weight.")
PANEL_WEIGHT_OPERATOR_PREFIX = "weight."
DEFERRED_WEIGHT_TOKENS = {"weight.equal", "weight.long_short", "weight.normalize_net"}
PR8_PANEL_OPERATOR_ALIASES = {
    "panel.cross_sectional_rank",
    "panel.zscore_by_universe",
    "panel.neutralize_group",
    "selection.top_k",
    "selection.bottom_k",
}
PR8_WEIGHT_SOURCE_TOKENS = {
    "weight.equal_weight",
    "weight.rank_weight",
    "weight.inverse_vol_weight",
}
PR8_WEIGHT_TRANSFORM_TOKENS = {
    "weight.market_neutral_weight",
    "weight.group_neutral_weight",
    "weight.max_weight_clip",
    "weight.normalize_net",
}
PR8_WEIGHT_MIXED_INPUT_TOKENS = {"weight.vol_target_weight"}


def validate_panel_type_layer_v04(ir: StrategyIRV04) -> list[Diagnostic]:
    """Validate WP8d Panel type-layer, Panel operator, and Weight operator declarations."""

    diagnostics: list[Diagnostic] = []
    has_panel_type_capability = PANEL_TYPE_CAPABILITY in ir.capabilities
    has_panel_ops_capability = PANEL_OPS_CAPABILITY in ir.capabilities
    has_panel_weights_capability = PANEL_WEIGHTS_CAPABILITY in ir.capabilities
    weight_output_refs = _weight_output_refs(ir)
    for node in ir.strategy.nodes:
        diagnostics.extend(_validate_node_panel_metadata(node, has_panel_type_capability))
        diagnostics.extend(
            _validate_panel_operator_gate(
                node,
                has_panel_ops_capability,
                has_panel_weights_capability,
                weight_output_refs,
            )
        )
        diagnostics.extend(_validate_panel_state_gate(node))
    return diagnostics


def _validate_node_panel_metadata(node: NodeV04, has_panel_type_capability: bool) -> list[Diagnostic]:
    diagnostics: list[Diagnostic] = []
    panel_outputs = _panel_outputs(node)
    metadata = node.metadata

    if "panel_type" in metadata:
        diagnostics.append(
            _diagnostic(
                "QST_V2_PANEL_TYPE_METADATA_LOCATION",
                node,
                None,
                "Panel type metadata must be output-scoped under metadata.panel_type_by_output.",
            )
        )

    raw_by_output = metadata.get("panel_type_by_output")
    parsed_by_output = {}
    if raw_by_output is not None:
        try:
            parsed_by_output = parse_panel_type_by_output(raw_by_output)
        except (TypeError, ValueError, ValidationError) as exc:
            diagnostics.append(
                _diagnostic(
                    "QST_V2_PANEL_TYPE_METADATA_INVALID",
                    node,
                    None,
                    f"metadata.panel_type_by_output is invalid: {exc}",
                )
            )
            parsed_by_output = {}

    for output_name, output_spec in node.signature.outputs.items():
        if output_name not in parsed_by_output:
            continue
        if output_spec.type.kind != "Panel":
            diagnostics.append(
                _diagnostic(
                    "QST_V2_PANEL_TYPE_METADATA_NON_PANEL_OUTPUT",
                    node,
                    output_name,
                    "Panel type metadata may only be attached to Panel outputs.",
                )
            )

    for output_name in parsed_by_output:
        if output_name not in node.signature.outputs:
            diagnostics.append(
                _diagnostic(
                    "QST_V2_PANEL_TYPE_METADATA_UNKNOWN_OUTPUT",
                    node,
                    output_name,
                    "Panel type metadata references an unknown output port.",
                )
            )

    for output_name in panel_outputs:
        if not has_panel_type_capability:
            diagnostics.append(
                _diagnostic(
                    "QST_V2_PANEL_TYPE_CAPABILITY_REQUIRED",
                    node,
                    output_name,
                    "Panel outputs require the panel_type capability.",
                )
            )
        if output_name not in parsed_by_output:
            diagnostics.append(
                _diagnostic(
                    "QST_V2_PANEL_TYPE_METADATA_REQUIRED",
                    node,
                    output_name,
                    "Panel outputs require metadata.panel_type_by_output for the output port.",
                )
            )
    return diagnostics


def _validate_panel_operator_gate(
    node: NodeV04,
    has_panel_ops_capability: bool,
    has_panel_weights_capability: bool,
    weight_output_refs: set[str],
) -> list[Diagnostic]:
    name = _token_name(node)
    if name is None or not name.startswith(PANEL_OPERATOR_PREFIXES):
        return []
    if name.startswith(PANEL_WEIGHT_OPERATOR_PREFIX):
        if name in DEFERRED_WEIGHT_TOKENS and name not in PR8_WEIGHT_TRANSFORM_TOKENS:
            return [
                _diagnostic(
                    "QST_V2_WEIGHT_OPERATOR_DEFERRED",
                    node,
                    None,
                    f"Weight operator token {name!r} is deferred beyond WP8d-v1.",
                )
            ]
        accepted_weight_token = (
            name in WEIGHT_OPERATOR_TOKENS
            or name in PR8_WEIGHT_SOURCE_TOKENS
            or name in PR8_WEIGHT_TRANSFORM_TOKENS
            or name in PR8_WEIGHT_MIXED_INPUT_TOKENS
        )
        if not accepted_weight_token:
            return [
                _diagnostic(
                    "QST_V2_WEIGHT_OPERATOR_NOT_ACCEPTED",
                    node,
                    None,
                    f"Weight operator token {name!r} is not accepted in WP8d.",
                )
            ]
        if not has_panel_weights_capability:
            return [
                _diagnostic(
                    "QST_V2_CAPABILITY_PANEL_WEIGHTS_REQUIRED",
                    node,
                    None,
                    f"Weight operator token {name!r} requires the panel_weights capability.",
                )
            ]
        if name in PR8_WEIGHT_SOURCE_TOKENS:
            return _validate_weight_source_signature(node)
        if name in PR8_WEIGHT_MIXED_INPUT_TOKENS:
            return _validate_weight_mixed_signature(node, weight_output_refs)
        return _validate_weight_operator_signature(node, weight_output_refs)
    if name in PANEL_OPERATOR_TOKENS or name in PR8_PANEL_OPERATOR_ALIASES:
        if has_panel_ops_capability:
            return []
        return [
            _diagnostic(
                "QST_V2_CAPABILITY_PANEL_OPS_REQUIRED",
                node,
                None,
                f"Panel operator token {name!r} requires the panel_ops capability.",
            )
        ]
    return [
        _diagnostic(
            "QST_V2_PANEL_OPERATOR_NOT_ACCEPTED",
            node,
            None,
            f"Panel operator token {name!r} is not accepted in WP8c.",
        )
    ]


def _validate_panel_state_gate(node: NodeV04) -> list[Diagnostic]:
    if _token_name(node) != "state.fsm":
        return []
    if not any(input_spec.type.kind == "Panel" for input_spec in node.signature.inputs.values()):
        return []
    return [
        _diagnostic(
            "QST_V2_PANEL_STATE_AUTOBROADCAST_UNSUPPORTED",
            node,
            None,
            "state.fsm does not auto-broadcast over Panel inputs in WP8c.",
        )
    ]


def _panel_outputs(node: NodeV04) -> list[str]:
    return [
        output_name
        for output_name, output_spec in node.signature.outputs.items()
        if output_spec.type.kind == "Panel"
    ]


def _weight_output_refs(ir: StrategyIRV04) -> set[str]:
    refs: set[str] = set()
    for node in ir.strategy.nodes:
        raw = node.metadata.get("panel_type_by_output")
        if raw is None:
            continue
        try:
            parsed = parse_panel_type_by_output(raw)
        except (TypeError, ValueError, ValidationError):
            continue
        for output_name, spec in parsed.items():
            if spec.kind == "weight_panel":
                refs.add(f"{node.id}.{output_name}")
    return refs


def _validate_weight_source_signature(node: NodeV04) -> list[Diagnostic]:
    diagnostics: list[Diagnostic] = []
    panel_inputs = [
        input_name
        for input_name, input_spec in node.signature.inputs.items()
        if input_spec.type.kind == "Panel"
    ]
    if not panel_inputs:
        diagnostics.append(
            _diagnostic(
                "QST_V2_WEIGHT_SOURCE_INPUT_REQUIRED",
                node,
                None,
                "PR8 weight source tokens require at least one Panel input.",
            )
        )
    diagnostics.extend(_validate_weight_outputs(node))
    return diagnostics


def _validate_weight_mixed_signature(node: NodeV04, weight_output_refs: set[str]) -> list[Diagnostic]:
    diagnostics: list[Diagnostic] = []
    weights_input = node.signature.inputs.get("weights")
    if weights_input is None or weights_input.type.kind != "Panel":
        diagnostics.append(
            _diagnostic(
                "QST_V2_WEIGHT_INPUT_NOT_WEIGHT_PANEL",
                node,
                "weights",
                "Mixed weight transforms require a weights Panel[decimal] input.",
            )
        )
    elif weights_input.type.value_type is None or weights_input.type.value_type.name != "decimal":
        diagnostics.append(
            _diagnostic(
                "QST_V2_WEIGHT_INPUT_NOT_WEIGHT_PANEL",
                node,
                "weights",
                "Mixed weight transform weights input must use Panel[decimal].",
            )
        )
    raw_ref = node.inputs.get("weights")
    if not isinstance(raw_ref, str) or raw_ref not in weight_output_refs:
        diagnostics.append(
            _diagnostic(
                "QST_V2_WEIGHT_INPUT_METADATA_REQUIRED",
                node,
                "weights",
                "Mixed weight transform weights input must reference weight_panel metadata.",
            )
        )
    diagnostics.extend(_validate_weight_outputs(node))
    return diagnostics


def _validate_weight_operator_signature(node: NodeV04, weight_output_refs: set[str]) -> list[Diagnostic]:
    diagnostics: list[Diagnostic] = []
    weight_inputs = [
        input_name
        for input_name, input_spec in node.signature.inputs.items()
        if input_spec.type.kind == "Panel"
    ]
    if not weight_inputs:
        diagnostics.append(
            _diagnostic(
                "QST_V2_WEIGHT_INPUT_NOT_WEIGHT_PANEL",
                node,
                None,
                "Weight operators require a Panel[decimal] WeightPanel input.",
            )
        )
    for input_name in weight_inputs:
        input_spec = node.signature.inputs[input_name]
        if input_spec.type.value_type is None or input_spec.type.value_type.name != "decimal":
            diagnostics.append(
                _diagnostic(
                    "QST_V2_WEIGHT_INPUT_NOT_WEIGHT_PANEL",
                    node,
                    input_name,
                    "Weight operator Panel inputs must use Panel[decimal].",
                )
            )
        raw_ref = node.inputs.get(input_name)
        if not isinstance(raw_ref, str) or raw_ref not in weight_output_refs:
            diagnostics.append(
                _diagnostic(
                    "QST_V2_WEIGHT_INPUT_METADATA_REQUIRED",
                    node,
                    input_name,
                    "Weight operator inputs must reference an output with weight_panel type metadata.",
                )
            )

    diagnostics.extend(_validate_weight_outputs(node))
    return diagnostics


def _validate_weight_outputs(node: NodeV04) -> list[Diagnostic]:
    diagnostics: list[Diagnostic] = []
    parsed_outputs: dict[str, PanelTypeLayerSpec] = {}
    raw_outputs = node.metadata.get("panel_type_by_output")
    if raw_outputs is not None:
        try:
            parsed_outputs = parse_panel_type_by_output(raw_outputs)
        except (TypeError, ValueError, ValidationError):
            parsed_outputs = {}
    for output_name, output_spec in node.signature.outputs.items():
        if output_spec.type.kind != "Panel":
            continue
        if output_spec.type.value_type is None or output_spec.type.value_type.name != "decimal":
            diagnostics.append(
                _diagnostic(
                    "QST_V2_WEIGHT_OUTPUT_NOT_WEIGHT_PANEL",
                    node,
                    output_name,
                    "Weight operator outputs must use Panel[decimal].",
                )
            )
        if output_name not in parsed_outputs or parsed_outputs[output_name].kind != "weight_panel":
            diagnostics.append(
                _diagnostic(
                    "QST_V2_WEIGHT_OUTPUT_METADATA_REQUIRED",
                    node,
                    output_name,
                    "Weight operator outputs must declare weight_panel type metadata.",
                )
            )
    return diagnostics


def _token_name(node: NodeV04) -> str | None:
    if node.token_ref is not None:
        return node.token_ref.name
    return node.token


def _diagnostic(code: str, node: NodeV04, port: str | None, message: str) -> Diagnostic:
    return Diagnostic(
        code=code,
        severity="error",
        phase="signature",
        message=message,
        node_id=node.id,
        port=port,
    )
