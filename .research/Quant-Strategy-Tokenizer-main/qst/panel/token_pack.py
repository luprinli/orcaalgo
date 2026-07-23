"""TokenPack metadata for WP8c Panel operators."""

from __future__ import annotations

from typing import Literal

from qst.ir import TokenRefV04
from qst.numeric import NumericPolicy, NumericRepresentation
from qst.panel.operators import (
    PANEL_OPERATOR_TOKENS,
    PANEL_OPS_PACK_ID,
    PANEL_OPS_PACK_VERSION,
    PANEL_WEIGHTS_PACK_ID,
    PANEL_WEIGHTS_PACK_VERSION,
    WEIGHT_OPERATOR_TOKENS,
    PanelOperatorName,
    WeightOperatorName,
)
from qst.ports import InputSpec, OutputSpec
from qst.tokens import TokenPackManifestV2, TokenRiskSpec, TokenSpecV2, token_surface
from qst.types import parse_type_spec

PanelOperatorCategory = Literal["panel_operator", "selection_operator"]
WeightOperatorCategory = Literal["weight_operator"]


def panel_ops_token_pack_v2() -> TokenPackManifestV2:
    """Return the WP8c core TokenPack metadata for Panel operators."""

    return TokenPackManifestV2(
        pack_id=PANEL_OPS_PACK_ID,
        version=PANEL_OPS_PACK_VERSION,
        namespaces=("core",),
        tokens=tuple(_panel_operator_spec(name) for name in PANEL_OPERATOR_TOKENS),
        origin_tier="core",
    )


def panel_weights_token_pack_v2() -> TokenPackManifestV2:
    """Return the WP8d core TokenPack metadata for Weight operators."""

    return TokenPackManifestV2(
        pack_id=PANEL_WEIGHTS_PACK_ID,
        version=PANEL_WEIGHTS_PACK_VERSION,
        namespaces=("core",),
        tokens=tuple(_weight_operator_spec(name) for name in WEIGHT_OPERATOR_TOKENS),
        origin_tier="core",
    )


def _panel_operator_spec(name: PanelOperatorName) -> TokenSpecV2:
    return TokenSpecV2(
        token_id=f"core.{name}",
        token_ref=TokenRefV04(
            namespace="core",
            name=name,
            version=1,
            behavior_version=1,
        ),
        version=1,
        behavior_version=1,
        origin_tier="core",
        inputs=_inputs_for(name),
        outputs=_outputs_for(name),
        params_schema=_params_schema(name),
        purity="pure",
        state={
            "panel_operator": True,
            "category": _category_for(name),
            "wp": "WP8c",
            "reference_semantics": "deterministic",
        },
        numeric_policy=_numeric_policy_for(name),
        risk=TokenRiskSpec(risk_level="medium"),
        surface=token_surface(
            family="weight" if name == "selection.to_weights" else "panel",
            category=_category_for(name),
            layer="derived",
            maturity="accepted",
            execution_support="reference_helper",
            temporal="panel operators join input port temporal metadata",
            numeric="semantic_float64" if _numeric_policy_for(name).representation == "float64" else "declared_by_numeric_policy",
            missing_data="UniverseMask=false is out-of-universe; active missing values follow missing_policy",
            failure_mode="validation_result_diagnostics",
            panel="requires panel_type and panel_ops capabilities",
            panel_aware=True,
            deterministic_level=(
                "semantic_float64"
                if _numeric_policy_for(name).representation == "float64"
                else "reference_exact"
            ),
            examples=(f"core.{name}/v1/bv1",),
            common_mistakes=("Panel reference numeric behavior is semantic float64, not bit-exact.",),
        ),
        tests=[
            {
                "kind": "reference_helper",
                "deterministic": True,
            }
        ],
    )


def _weight_operator_spec(name: WeightOperatorName) -> TokenSpecV2:
    return TokenSpecV2(
        token_id=f"core.{name}",
        token_ref=TokenRefV04(
            namespace="core",
            name=name,
            version=1,
            behavior_version=1,
        ),
        version=1,
        behavior_version=1,
        origin_tier="core",
        inputs={"weights": _input("Panel[decimal]")},
        outputs={"weights": _output("Panel[decimal]")},
        params_schema=_weight_params_schema(name),
        purity="pure",
        state={
            "panel_operator": True,
            "category": "weight_operator",
            "wp": "WP8d",
            "reference_semantics": "deterministic",
        },
        numeric_policy=NumericPolicy(
            representation="decimal",
            deterministic_level="semantic",
            reduction_order="fixed_input_order",
            nan_policy="reject",
            inf_policy="reject",
        ),
        risk=TokenRiskSpec(risk_level="medium"),
        surface=token_surface(
            family="weight",
            category="weight_operator",
            layer="derived",
            maturity="accepted",
            execution_support="reference_helper",
            temporal="weight operators join input port temporal metadata",
            numeric="decimal canonical weight arithmetic with semantic reference rules",
            missing_data="UniverseMask=false is out-of-universe; active missing weights follow missing_policy",
            failure_mode="validation_result_diagnostics",
            panel="requires panel_type and panel_weights capabilities; input must be WeightPanel",
            panel_aware=True,
            deterministic_level="reference_exact",
            examples=(f"core.{name}/v1/bv1",),
            common_mistakes=("cap_per_symbol does not redistribute or preserve gross exposure.",),
        ),
        tests=[
            {
                "kind": "reference_helper",
                "deterministic": True,
            }
        ],
    )


def _inputs_for(name: PanelOperatorName) -> dict[str, InputSpec]:
    if name == "panel.mask":
        return {
            "panel": _input("Panel[float]"),
            "mask": _input("Panel[bool]"),
        }
    if name == "panel.residualize":
        return {
            "panel": _input("Panel[float]"),
            "factor": _input("TimeSeries[float]"),
        }
    if name == "selection.to_weights":
        return {"selection": _input("Panel[bool]")}
    return {"panel": _input("Panel[float]")}


def _outputs_for(name: PanelOperatorName) -> dict[str, OutputSpec]:
    if name == "panel.rank":
        return {"ranked": _output("Panel[int]")}
    if name in {"panel.top_k", "panel.bottom_k"}:
        return {"selection": _output("Panel[bool]")}
    if name == "selection.to_weights":
        return {"weights": _output("Panel[decimal]")}
    return {"panel": _output("Panel[float]")}


def _params_schema(name: PanelOperatorName) -> dict[str, object]:
    common_missing = {
        "missing_policy": {
            "enum": ["error_on_missing", "drop_missing"],
            "default": "error_on_missing",
        }
    }
    if name == "panel.rank":
        return {
            "type": "object",
            "additionalProperties": False,
            "properties": {
                **common_missing,
                "axis": {"const": "symbol", "default": "symbol"},
                "order": {"enum": ["descending", "ascending"], "default": "descending"},
                "rank_base": {"const": 1, "default": 1},
                "tie_policy": {"const": "stable_symbol_order", "default": "stable_symbol_order"},
            },
        }
    if name == "panel.zscore":
        return {
            "type": "object",
            "additionalProperties": False,
            "properties": {
                **common_missing,
                "axis": {"const": "symbol", "default": "symbol"},
                "ddof": {"const": 0, "default": 0},
                "zero_variance_policy": {"const": "output_zero", "default": "output_zero"},
            },
        }
    if name in {"panel.top_k", "panel.bottom_k"}:
        return {
            "type": "object",
            "additionalProperties": False,
            "required": ["k"],
            "properties": {
                **common_missing,
                "k": {"type": "integer", "minimum": 1},
                "tie_policy": {"const": "stable_symbol_order", "default": "stable_symbol_order"},
                "selection_size_policy": {"const": "allow_smaller", "default": "allow_smaller"},
            },
        }
    if name == "panel.winsorize":
        return {
            "type": "object",
            "additionalProperties": False,
            "properties": {
                **common_missing,
                "lower_quantile": {"type": "string", "default": "0.01"},
                "upper_quantile": {"type": "string", "default": "0.99"},
                "interpolation": {"const": "nearest_rank", "default": "nearest_rank"},
            },
        }
    if name == "panel.group_demean":
        return {
            "type": "object",
            "additionalProperties": False,
            "required": ["group_spec_ref"],
            "properties": {
                **common_missing,
                "group_spec_ref": {"type": "string", "minLength": 1},
                "missing_group_policy": {
                    "enum": ["error", "drop", "assign_unknown"],
                    "default": "error",
                },
            },
        }
    if name == "panel.residualize":
        return {
            "type": "object",
            "additionalProperties": False,
            "properties": {
                **common_missing,
                "include_intercept": {"const": True, "default": True},
                "min_observations": {"type": "integer", "minimum": 3, "default": 3},
                "insufficient_observations_policy": {
                    "enum": ["unknown", "error"],
                    "default": "unknown",
                },
            },
        }
    if name == "selection.to_weights":
        return {
            "type": "object",
            "additionalProperties": False,
            "properties": {
                "method": {
                    "enum": ["equal_long", "equal_short", "equal_long_short"],
                    "default": "equal_long",
                },
                "weight_kind": {"const": "raw", "default": "raw"},
                "normalized": {"const": False, "default": False},
            },
        }
    return {
        "type": "object",
        "additionalProperties": False,
        "properties": common_missing,
    }


def _weight_params_schema(name: WeightOperatorName) -> dict[str, object]:
    common = {
        "missing_policy": {
            "enum": ["error_on_missing", "drop_missing"],
            "default": "error_on_missing",
        }
    }
    if name == "weight.normalize_gross":
        return {
            "type": "object",
            "additionalProperties": False,
            "properties": {
                **common,
                "target_gross": {"type": "string", "default": "1"},
                "zero_gross_policy": {"enum": ["keep_zero", "error"], "default": "keep_zero"},
            },
        }
    if name == "weight.cap_per_symbol":
        return {
            "type": "object",
            "additionalProperties": False,
            "required": ["max_abs_weight"],
            "properties": {
                **common,
                "max_abs_weight": {"type": "string"},
                "mode": {"const": "clip_no_redistribute", "default": "clip_no_redistribute"},
            },
        }
    return {
        "type": "object",
        "additionalProperties": False,
        "properties": {
            **common,
            "target_net": {"const": "0", "default": "0"},
            "target_gross": {"type": "string", "default": "1"},
            "neutralization_method": {
                "const": "demean_then_gross_normalize",
                "default": "demean_then_gross_normalize",
            },
            "zero_gross_policy": {"enum": ["keep_zero", "error"], "default": "keep_zero"},
        },
    }


def _category_for(name: PanelOperatorName) -> PanelOperatorCategory:
    if name == "selection.to_weights":
        return "selection_operator"
    return "panel_operator"


def _numeric_policy_for(name: PanelOperatorName) -> NumericPolicy:
    representation: NumericRepresentation
    if name == "selection.to_weights":
        representation = "decimal"
    elif name in {"panel.top_k", "panel.bottom_k", "panel.mask"}:
        representation = "object"
    else:
        representation = "float64"
    return NumericPolicy(
        representation=representation,
        deterministic_level="semantic",
        reduction_order="fixed_input_order",
        nan_policy="reject",
        inf_policy="reject",
    )


def _input(type_spec: str) -> InputSpec:
    return InputSpec(type=parse_type_spec(type_spec))


def _output(type_spec: str) -> OutputSpec:
    return OutputSpec(type=parse_type_spec(type_spec))
