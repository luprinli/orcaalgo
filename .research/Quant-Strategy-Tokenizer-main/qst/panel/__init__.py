"""Panel type-layer and reference operator models for Token System v2."""

from typing import Any

from qst.panel.model import (
    GROUP_SPEC_SCHEMA_VERSION,
    MISSING_POLICY_SCHEMA_VERSION,
    PANEL_REPRESENTATION_SCHEMA_VERSION,
    PANEL_SELECTION_WEIGHT_SCHEMA_VERSION,
    PANEL_TYPE_LAYER_SCHEMA_VERSION,
    UNIVERSE_MASK_SCHEMA_VERSION,
    GroupSpec,
    PanelMissingPolicy,
    PanelRepresentation,
    PanelTypeLayerSpec,
    SelectionPanelType,
    UniverseMask,
    UniverseMaskRef,
    WeightConstraints,
    WeightPanelType,
    parse_panel_type_by_output,
)
from qst.panel.operators import (
    PANEL_OPERATOR_TOKENS,
    PANEL_OPS_PACK_ID,
    PANEL_OPS_PACK_VERSION,
    PANEL_WEIGHTS_PACK_ID,
    PANEL_WEIGHTS_PACK_VERSION,
    WEIGHT_OPERATOR_TOKENS,
    PanelOperatorResult,
    PanelPoint,
    PanelValue,
    SelectionPanelValue,
    SelectionPoint,
    WeightPanelValue,
    WeightPoint,
    panel_bottom_k,
    panel_demean,
    panel_group_demean,
    panel_mask,
    panel_rank,
    panel_residualize,
    panel_top_k,
    panel_winsorize,
    panel_zscore,
    selection_to_weights,
    weight_cap_per_symbol,
    weight_market_neutral,
    weight_normalize_gross,
)


def panel_ops_token_pack_v2() -> Any:
    """Return the WP8c Panel operators TokenPack without importing it eagerly."""

    from qst.panel.token_pack import panel_ops_token_pack_v2 as _impl

    return _impl()


def panel_weights_token_pack_v2() -> Any:
    """Return the WP8d Weight operators TokenPack without importing it eagerly."""

    from qst.panel.token_pack import panel_weights_token_pack_v2 as _impl

    return _impl()


__all__ = [
    "GROUP_SPEC_SCHEMA_VERSION",
    "MISSING_POLICY_SCHEMA_VERSION",
    "PANEL_OPERATOR_TOKENS",
    "PANEL_OPS_PACK_ID",
    "PANEL_OPS_PACK_VERSION",
    "PANEL_REPRESENTATION_SCHEMA_VERSION",
    "PANEL_SELECTION_WEIGHT_SCHEMA_VERSION",
    "PANEL_TYPE_LAYER_SCHEMA_VERSION",
    "PANEL_WEIGHTS_PACK_ID",
    "PANEL_WEIGHTS_PACK_VERSION",
    "UNIVERSE_MASK_SCHEMA_VERSION",
    "WEIGHT_OPERATOR_TOKENS",
    "GroupSpec",
    "PanelMissingPolicy",
    "PanelOperatorResult",
    "PanelPoint",
    "PanelRepresentation",
    "PanelTypeLayerSpec",
    "PanelValue",
    "SelectionPanelType",
    "SelectionPanelValue",
    "SelectionPoint",
    "UniverseMask",
    "UniverseMaskRef",
    "WeightConstraints",
    "WeightPanelType",
    "WeightPanelValue",
    "WeightPoint",
    "panel_bottom_k",
    "panel_demean",
    "panel_group_demean",
    "panel_mask",
    "panel_ops_token_pack_v2",
    "panel_rank",
    "panel_residualize",
    "panel_top_k",
    "panel_weights_token_pack_v2",
    "panel_winsorize",
    "panel_zscore",
    "parse_panel_type_by_output",
    "selection_to_weights",
    "weight_cap_per_symbol",
    "weight_market_neutral",
    "weight_normalize_gross",
]
