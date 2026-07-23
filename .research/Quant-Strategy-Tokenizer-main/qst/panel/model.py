"""Panel type-layer data models.

WP8b makes Panel type semantics explicit without changing TypeSpec shape or
adding Panel runtime behavior.
"""

from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, model_validator

from qst.artifacts.decimal_string import DecimalString
from qst.artifacts.safety import POSIXRelativePath
from qst.hash.common import HashString

PANEL_REPRESENTATION_SCHEMA_VERSION: Literal["qst-panel-representation/0.4"] = (
    "qst-panel-representation/0.4"
)
UNIVERSE_MASK_SCHEMA_VERSION: Literal["qst-panel-universe-mask/0.4"] = (
    "qst-panel-universe-mask/0.4"
)
MISSING_POLICY_SCHEMA_VERSION: Literal["qst-panel-missing-policy/0.4"] = (
    "qst-panel-missing-policy/0.4"
)
GROUP_SPEC_SCHEMA_VERSION: Literal["qst-panel-group-spec/0.4"] = "qst-panel-group-spec/0.4"
PANEL_SELECTION_WEIGHT_SCHEMA_VERSION: Literal["qst-panel-selection-weight/0.4"] = (
    "qst-panel-selection-weight/0.4"
)
PANEL_TYPE_LAYER_SCHEMA_VERSION: Literal["qst-panel-type-layer/0.4"] = (
    "qst-panel-type-layer/0.4"
)

MissingPolicyKind = Literal["error_on_missing", "drop_missing"]
MissingGroupPolicy = Literal["error", "drop", "assign_unknown"]
PanelLayerKind = Literal["panel", "selection_panel", "weight_panel"]
GroupSpecKind = Literal["static_mapping", "field_ref"]
SelectionKind = Literal["long_only", "short_only", "long_short", "ranked"]
SelectionSide = Literal["long", "short", "both"]
WeightKind = Literal["raw", "normalized"]


class PanelRepresentation(BaseModel):
    """Panel representation model."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-panel-representation/0.4"] = PANEL_REPRESENTATION_SCHEMA_VERSION
    kind: Literal["sparse_logical"] = "sparse_logical"
    universe_mask_required: Literal[True] = True


class UniverseMask(BaseModel):
    """Sparse logical universe membership mask."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-panel-universe-mask/0.4"] = UNIVERSE_MASK_SCHEMA_VERSION
    representation: Literal["sparse_logical"] = "sparse_logical"
    universe_ref: POSIXRelativePath
    members: list[str] = Field(default_factory=list)
    included: list[str] = Field(default_factory=list)
    false_semantics: Literal["out_of_universe_not_missing"] = "out_of_universe_not_missing"

    @model_validator(mode="before")
    @classmethod
    def _sort_members(cls, value: Any) -> Any:
        if not isinstance(value, dict):
            return value
        value = dict(value)
        for field in ("members", "included"):
            items = value.get(field, [])
            if len(items) != len(set(items)):
                raise ValueError(f"UniverseMask {field} must be unique")
            value[field] = sorted(items)
        return value

    @model_validator(mode="after")
    def _included_subset(self) -> UniverseMask:
        missing = sorted(set(self.included) - set(self.members))
        if missing:
            raise ValueError(f"UniverseMask included members must be in members: {missing}")
        return self


class PanelMissingPolicy(BaseModel):
    """Policy for values missing inside the active universe."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-panel-missing-policy/0.4"] = MISSING_POLICY_SCHEMA_VERSION
    kind: MissingPolicyKind = "error_on_missing"
    applies_when: Literal["universe_mask_true_value_missing"] = "universe_mask_true_value_missing"
    nullable_decimal_string: Literal[False] = False


class GroupSpec(BaseModel):
    """Panel group specification."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-panel-group-spec/0.4"] = GROUP_SPEC_SCHEMA_VERSION
    kind: GroupSpecKind
    group_id: str = Field(min_length=1)
    mapping_ref: POSIXRelativePath | None = None
    mapping_hash: HashString | None = None
    field_path: str | None = None
    missing_group_policy: MissingGroupPolicy = "error"
    group_label_type: Literal["string"] = "string"

    @model_validator(mode="after")
    def _validate_shape(self) -> GroupSpec:
        if self.kind == "static_mapping":
            if self.mapping_ref is None or self.mapping_hash is None:
                raise ValueError("static_mapping GroupSpec requires mapping_ref and mapping_hash")
            if not self.mapping_ref.endswith(".json"):
                raise ValueError("static_mapping GroupSpec mapping_ref must point to a JSON file")
            if self.field_path is not None:
                raise ValueError("static_mapping GroupSpec does not accept field_path")
        if self.kind == "field_ref":
            if self.field_path is None:
                raise ValueError("field_ref GroupSpec requires field_path")
            if not self.field_path.startswith("universe.metadata."):
                raise ValueError("field_ref GroupSpec field_path must start with universe.metadata.")
            if self.mapping_ref is not None or self.mapping_hash is not None:
                raise ValueError("field_ref GroupSpec does not accept mapping_ref or mapping_hash")
        return self


class UniverseMaskRef(BaseModel):
    """Reference to a persisted UniverseMask."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    universe_mask_ref: POSIXRelativePath
    universe_mask_hash: HashString


class SelectionPanelType(BaseModel):
    """Wire-level selection panel type metadata."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-panel-selection-weight/0.4"] = PANEL_SELECTION_WEIGHT_SCHEMA_VERSION
    kind: Literal["selection_panel"] = "selection_panel"
    selection_kind: SelectionKind
    selected: UniverseMaskRef
    side: SelectionSide | None = None
    score_ref: POSIXRelativePath | None = None


class WeightConstraints(BaseModel):
    """Raw or normalized weight constraints."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    max_abs_weight_per_symbol: DecimalString | None = None
    gross_target: DecimalString | None = None
    net_target: DecimalString | None = None


class WeightPanelType(BaseModel):
    """Wire-level weight panel type metadata."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-panel-selection-weight/0.4"] = PANEL_SELECTION_WEIGHT_SCHEMA_VERSION
    kind: Literal["weight_panel"] = "weight_panel"
    weight_kind: WeightKind
    weights_ref: POSIXRelativePath
    weights_hash: HashString
    gross_exposure: DecimalString | None = None
    net_exposure: DecimalString | None = None
    weight_constraints: WeightConstraints = Field(default_factory=WeightConstraints)


class PanelTypeLayerSpec(BaseModel):
    """Semantic, output-scoped Panel type metadata."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-panel-type-layer/0.4"] = PANEL_TYPE_LAYER_SCHEMA_VERSION
    kind: PanelLayerKind = "panel"
    representation: PanelRepresentation = Field(default_factory=PanelRepresentation)
    universe_mask: UniverseMask
    missing_policy: PanelMissingPolicy = Field(default_factory=PanelMissingPolicy)
    group_spec: GroupSpec | None = None
    selection_panel: SelectionPanelType | None = None
    weight_panel: WeightPanelType | None = None

    @model_validator(mode="after")
    def _validate_kind_payload(self) -> PanelTypeLayerSpec:
        if self.kind == "panel":
            if self.selection_panel is not None or self.weight_panel is not None:
                raise ValueError("panel metadata does not accept selection_panel or weight_panel payload")
        if self.kind == "selection_panel":
            if self.selection_panel is None:
                raise ValueError("selection_panel metadata requires selection_panel payload")
            if self.weight_panel is not None:
                raise ValueError("selection_panel metadata does not accept weight_panel payload")
        if self.kind == "weight_panel":
            if self.weight_panel is None:
                raise ValueError("weight_panel metadata requires weight_panel payload")
            if self.selection_panel is not None:
                raise ValueError("weight_panel metadata does not accept selection_panel payload")
        return self


def parse_panel_type_by_output(value: Any) -> dict[str, PanelTypeLayerSpec]:
    """Parse output-scoped Panel type metadata."""

    if not isinstance(value, dict):
        raise ValueError("panel_type_by_output must be an object keyed by output port name")
    parsed: dict[str, PanelTypeLayerSpec] = {}
    for output_name, spec in sorted(value.items()):
        if not isinstance(output_name, str) or not output_name:
            raise ValueError("panel_type_by_output keys must be non-empty output names")
        parsed[output_name] = (
            spec if isinstance(spec, PanelTypeLayerSpec) else PanelTypeLayerSpec.model_validate(spec)
        )
    return parsed
