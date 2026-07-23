"""Token surface and contract metadata for the public token vocabulary."""

from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator

TokenFamily = Literal[
    "math",
    "bool",
    "compare",
    "data",
    "time",
    "align",
    "window",
    "signal",
    "indicator",
    "factor",
    "decision",
    "gate",
    "state",
    "panel",
    "weight",
    "risk",
    "optimizer",
    "execution",
    "event",
    "distribution",
    "continuous_score",
]
TokenLayer = Literal["primitive", "derived", "recipe", "custom", "external"]
TokenMaturity = Literal["accepted", "experimental", "reserved_design", "frozen", "deprecated"]
ExecutionSupport = Literal["metadata_only", "reference_helper", "runtime_executor", "external_only"]
ContractScope = Literal["validation_only", "reference_semantics", "execution_semantics"]
DeterminismContract = Literal[
    "reference_exact",
    "semantic_float64",
    "annotation_only",
    "external",
    "reserved",
]
ProfileName = Literal["research", "paper", "pretrade", "production_guarded"]
TOKEN_SURFACE_SCHEMA_VERSION: Literal["qst-token-surface/0.4"] = "qst-token-surface/0.4"


class TokenCapabilityMetadata(BaseModel):
    """Capability flags that describe a token's public execution surface."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    stateful: bool = False
    panel_aware: bool = False
    solver_backed: bool = False
    external_code: bool = False
    reserved_only: bool = False
    deterministic_level: DeterminismContract = "reference_exact"


class AgentTokenMetadata(BaseModel):
    """Agent-facing token hints. These hints are explanatory, not executable policy."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    examples: tuple[str, ...] = ()
    common_mistakes: tuple[str, ...] = ()
    usage_notes: tuple[str, ...] = ()
    profile_caveats: tuple[str, ...] = ()

    @field_validator("examples", "common_mistakes", "usage_notes", "profile_caveats", mode="after")
    @classmethod
    def _stable_unique(cls, value: tuple[str, ...]) -> tuple[str, ...]:
        return tuple(sorted(dict.fromkeys(value)))


class SolverContractSpec(BaseModel):
    """Determinism contract for solver-backed tokens."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    solver_required: bool = False
    deterministic_contract: str = "not_solver_backed"
    bit_exact_claim: bool = False


class TokenContractSpec(BaseModel):
    """Hash-bearing token behavior contract."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    scope: ContractScope = "validation_only"
    temporal: str = "declared_by_ports"
    numeric: str = "declared_by_numeric_policy"
    missing_data: str = "reject_or_declared_by_token"
    failure_mode: str = "diagnostic_error"
    state: str | None = None
    panel: str | None = None
    solver: SolverContractSpec | None = None
    supported_profiles: tuple[ProfileName, ...] = (
        "research",
        "paper",
        "pretrade",
        "production_guarded",
    )

    @field_validator("supported_profiles", mode="after")
    @classmethod
    def _sort_profiles(cls, value: tuple[ProfileName, ...]) -> tuple[ProfileName, ...]:
        order = ("research", "paper", "pretrade", "production_guarded")
        return tuple(profile for profile in order if profile in set(value))


class TokenSurfaceSpec(BaseModel):
    """Public token family, maturity, and contract metadata."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-token-surface/0.4"] = TOKEN_SURFACE_SCHEMA_VERSION
    family: TokenFamily = "math"
    category: str = "general"
    layer: TokenLayer = "primitive"
    maturity: TokenMaturity = "experimental"
    execution_support: ExecutionSupport = "metadata_only"
    contract: TokenContractSpec = Field(default_factory=TokenContractSpec)
    capabilities: TokenCapabilityMetadata = Field(default_factory=TokenCapabilityMetadata)
    agent_metadata: AgentTokenMetadata = Field(default_factory=AgentTokenMetadata)


def default_token_surface() -> TokenSurfaceSpec:
    """Compatibility default for community/custom specs that predate Stage 3A."""

    return TokenSurfaceSpec(
        family="math",
        category="unspecified",
        layer="custom",
        maturity="experimental",
        execution_support="metadata_only",
        contract=TokenContractSpec(scope="validation_only"),
        capabilities=TokenCapabilityMetadata(deterministic_level="annotation_only"),
    )


def token_surface(
    *,
    family: TokenFamily,
    category: str,
    layer: TokenLayer = "derived",
    maturity: TokenMaturity = "accepted",
    execution_support: ExecutionSupport = "reference_helper",
    contract_scope: ContractScope = "reference_semantics",
    temporal: str = "declared_by_ports",
    numeric: str = "declared_by_numeric_policy",
    missing_data: str = "reject_or_declared_by_token",
    failure_mode: str = "diagnostic_error",
    state: str | None = None,
    panel: str | None = None,
    solver: SolverContractSpec | None = None,
    stateful: bool = False,
    panel_aware: bool = False,
    solver_backed: bool = False,
    external_code: bool = False,
    reserved_only: bool = False,
    deterministic_level: DeterminismContract = "reference_exact",
    examples: tuple[str, ...] = (),
    common_mistakes: tuple[str, ...] = (),
    usage_notes: tuple[str, ...] = (),
    profile_caveats: tuple[str, ...] = (),
) -> TokenSurfaceSpec:
    """Build a canonical token surface with explicit contract fields."""

    return TokenSurfaceSpec(
        family=family,
        category=category,
        layer=layer,
        maturity=maturity,
        execution_support=execution_support,
        contract=TokenContractSpec(
            scope=contract_scope,
            temporal=temporal,
            numeric=numeric,
            missing_data=missing_data,
            failure_mode=failure_mode,
            state=state,
            panel=panel,
            solver=solver,
        ),
        capabilities=TokenCapabilityMetadata(
            stateful=stateful,
            panel_aware=panel_aware,
            solver_backed=solver_backed,
            external_code=external_code,
            reserved_only=reserved_only,
            deterministic_level=deterministic_level,
        ),
        agent_metadata=AgentTokenMetadata(
            examples=examples,
            common_mistakes=common_mistakes,
            usage_notes=usage_notes,
            profile_caveats=profile_caveats,
        ),
    )
