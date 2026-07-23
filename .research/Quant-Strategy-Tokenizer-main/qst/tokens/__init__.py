"""Token System v2 TokenSpec, TokenPack, and registry models."""

from qst.tokens.lock import (
    TokenLockEntryV04,
    TokenLockSnapshotV04,
    TokenPackLockDependencyV04,
    token_lock_entry_from_spec,
    token_pack_lock_dependency_from_pack,
    verify_token_lock_snapshot,
)
from qst.tokens.maturity import validate_token_maturity_for_profile
from qst.tokens.pack import (
    TOKEN_PACK_SCHEMA_VERSION,
    TokenPackDependency,
    TokenPackManifestV2,
)
from qst.tokens.package_policy import (
    TokenPackPackageEntryV04,
    TokenPacksPackageSectionV04,
    token_pack_package_entry_from_pack,
    token_pack_package_section_from_packs,
    verify_token_pack_package_section,
)
from qst.tokens.reference import (
    TokenReferenceError,
    TokenReferenceResult,
    evaluate_align_token,
    evaluate_bool_token,
    evaluate_channel_breakout_token,
    evaluate_cmp_token,
    evaluate_data_token,
    evaluate_decision_token,
    evaluate_factor_token,
    evaluate_gate_token,
    evaluate_indicator_token,
    evaluate_math_token,
    evaluate_panel_token,
    evaluate_risk_token,
    evaluate_signal_token,
    evaluate_state_token,
    evaluate_time_token,
    evaluate_weight_token,
    evaluate_window_token,
)
from qst.tokens.registry import (
    RegistryTokenRecord,
    TokenPackDependencyResolution,
    TokenRegistryV2,
    validate_token_pack_dependencies,
)
from qst.tokens.spec import (
    TOKEN_SPEC_SCHEMA_VERSION,
    AttestationKind,
    OriginTier,
    RiskLevel,
    TokenRiskSpec,
    TokenSpecV2,
)
from qst.tokens.surface import (
    AgentTokenMetadata,
    ContractScope,
    DeterminismContract,
    ExecutionSupport,
    SolverContractSpec,
    TokenCapabilityMetadata,
    TokenContractSpec,
    TokenFamily,
    TokenLayer,
    TokenMaturity,
    TokenSurfaceSpec,
    token_surface,
)

__all__ = [
    "TOKEN_PACK_SCHEMA_VERSION",
    "TOKEN_SPEC_SCHEMA_VERSION",
    "AgentTokenMetadata",
    "AttestationKind",
    "ContractScope",
    "DeterminismContract",
    "ExecutionSupport",
    "OriginTier",
    "RegistryTokenRecord",
    "RiskLevel",
    "SolverContractSpec",
    "TokenCapabilityMetadata",
    "TokenContractSpec",
    "TokenFamily",
    "TokenLayer",
    "TokenLockEntryV04",
    "TokenLockSnapshotV04",
    "TokenMaturity",
    "TokenPackDependency",
    "TokenPackDependencyResolution",
    "TokenPackLockDependencyV04",
    "TokenPackManifestV2",
    "TokenPackPackageEntryV04",
    "TokenPacksPackageSectionV04",
    "TokenReferenceError",
    "TokenReferenceResult",
    "TokenRegistryV2",
    "TokenRiskSpec",
    "TokenSpecV2",
    "TokenSurfaceSpec",
    "builtin_token_packs",
    "evaluate_align_token",
    "evaluate_bool_token",
    "evaluate_channel_breakout_token",
    "evaluate_cmp_token",
    "evaluate_data_token",
    "evaluate_decision_token",
    "evaluate_factor_token",
    "evaluate_gate_token",
    "evaluate_indicator_token",
    "evaluate_math_token",
    "evaluate_panel_token",
    "evaluate_risk_token",
    "evaluate_signal_token",
    "evaluate_state_token",
    "evaluate_time_token",
    "evaluate_weight_token",
    "evaluate_window_token",
    "token_lock_entry_from_spec",
    "token_pack_lock_dependency_from_pack",
    "token_pack_package_entry_from_pack",
    "token_pack_package_section_from_packs",
    "token_surface",
    "validate_token_maturity_for_profile",
    "validate_token_pack_dependencies",
    "verify_token_lock_snapshot",
    "verify_token_pack_package_section",
]


def builtin_token_packs() -> tuple[TokenPackManifestV2, ...]:
    """Return built-in TokenPacks in deterministic public vocabulary order."""

    from qst.tokens.builtin import builtin_token_packs as _impl

    return _impl()
