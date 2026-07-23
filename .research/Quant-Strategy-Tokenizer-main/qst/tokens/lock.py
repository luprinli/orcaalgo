"""Token System v2 lock snapshot helpers.

WP5b records TokenSpec / TokenPack identity material for future qst-ir/0.4
locks. These helpers validate metadata only; they never import or execute
custom token implementation references.
"""

from __future__ import annotations

from collections.abc import Iterable
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field

from qst.hash import (
    implementation_ref_hash_v2,
    runtime_environment_hash_v2,
    token_pack_hash_for_pack_v2,
    token_spec_hash_for_spec_v2,
)
from qst.ir import TokenRefV04
from qst.tokens.pack import TokenPackManifestV2
from qst.tokens.spec import (
    AttestationKind,
    OriginTier,
    RiskLevel,
    TokenSpecV2,
)
from qst.validation import Diagnostic, ValidationResult

HashString = str


class TokenLockEntryV04(BaseModel):
    """One token identity entry recorded in a qst-lock/0.4 snapshot."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    token_ref: TokenRefV04
    token_spec_hash: HashString = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    token_pack_hash: HashString = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    implementation_ref_hash: HashString = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    runtime_environment_hash: HashString = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    origin_tier: OriginTier
    attestation_kind: AttestationKind
    risk_level: RiskLevel = "unknown"


class TokenPackLockDependencyV04(BaseModel):
    """One TokenPack dependency entry recorded in qst-lock/0.4."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    pack_id: str = Field(min_length=1)
    version: str = Field(min_length=1)
    token_pack_hash: HashString = Field(pattern=r"^sha256:[0-9a-f]{64}$")


class TokenLockSnapshotV04(BaseModel):
    """Token-related qst-lock/0.4 material."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    tokens: tuple[TokenLockEntryV04, ...] = Field(default_factory=tuple)
    token_pack_dependencies: tuple[TokenPackLockDependencyV04, ...] = Field(default_factory=tuple)


def token_lock_entry_from_spec(
    spec: TokenSpecV2,
    pack: TokenPackManifestV2,
) -> TokenLockEntryV04:
    """Build a deterministic qst-lock token entry from a TokenSpec and pack."""

    return TokenLockEntryV04(
        token_ref=spec.token_ref,
        token_spec_hash=token_spec_hash_for_spec_v2(spec),
        token_pack_hash=token_pack_hash_for_pack_v2(pack),
        implementation_ref_hash=implementation_ref_hash_v2(spec.implementation_ref),
        runtime_environment_hash=runtime_environment_hash_v2(spec.runtime_environment_ref),
        origin_tier=spec.origin_tier,
        attestation_kind=spec.attestation_kind,
        risk_level=_risk_level(spec),
    )


def token_pack_lock_dependency_from_pack(
    pack: TokenPackManifestV2,
) -> TokenPackLockDependencyV04:
    """Build a deterministic qst-lock TokenPack dependency entry."""

    return TokenPackLockDependencyV04(
        pack_id=pack.pack_id,
        version=pack.version,
        token_pack_hash=token_pack_hash_for_pack_v2(pack),
    )


def verify_token_lock_snapshot(
    snapshot: TokenLockSnapshotV04,
    available_packs: Iterable[TokenPackManifestV2],
) -> ValidationResult:
    """Verify qst-lock token metadata against available TokenPack metadata.

    Verification checks hashes and metadata only. It does not execute
    ``implementation_ref`` or inspect external source trees.
    """

    packs = tuple(available_packs)
    diagnostics: list[Diagnostic] = []
    packs_by_identity = {(pack.pack_id, pack.version): pack for pack in packs}

    for dependency in snapshot.token_pack_dependencies:
        pack = packs_by_identity.get((dependency.pack_id, dependency.version))
        if pack is None:
            diagnostics.append(
                _diagnostic(
                    "QST_V2_LOCK_TOKEN_PACK_MISSING",
                    (
                        f"TokenPack {dependency.pack_id} {dependency.version} required by lock "
                        "is not available."
                    ),
                )
            )
            continue
        actual_hash = token_pack_hash_for_pack_v2(pack)
        if actual_hash != dependency.token_pack_hash:
            diagnostics.append(
                _diagnostic(
                    "QST_V2_LOCK_TOKEN_PACK_HASH_MISMATCH",
                    (
                        f"TokenPack {dependency.pack_id} {dependency.version} hash mismatch: "
                        f"expected {dependency.token_pack_hash}, got {actual_hash}."
                    ),
                )
            )

    records = _pack_token_records(packs)
    for entry in snapshot.tokens:
        candidates = records.get(_ref_key(entry.token_ref), ())
        if not candidates:
            diagnostics.append(
                _diagnostic(
                    "QST_V2_LOCK_TOKEN_PACK_MISSING",
                    f"Token {_qualified_name(entry.token_ref)} required by lock is not available.",
                )
            )
            continue

        matching_pack = [record for record in candidates if record.pack_hash == entry.token_pack_hash]
        if not matching_pack:
            diagnostics.append(
                _diagnostic(
                    "QST_V2_LOCK_TOKEN_PACK_HASH_MISMATCH",
                    f"Token {_qualified_name(entry.token_ref)} has no available pack with locked hash.",
                )
            )
            continue

        matching_spec = [record for record in matching_pack if record.spec_hash == entry.token_spec_hash]
        if not matching_spec:
            diagnostics.append(
                _diagnostic(
                    "QST_V2_LOCK_TOKEN_SPEC_HASH_MISMATCH",
                    f"Token {_qualified_name(entry.token_ref)} TokenSpec hash does not match lock.",
                )
            )
            continue

        record = matching_spec[0]
        implementation_hash = implementation_ref_hash_v2(record.spec.implementation_ref)
        if implementation_hash != entry.implementation_ref_hash:
            diagnostics.append(
                _diagnostic(
                    "QST_V2_LOCK_IMPLEMENTATION_REF_HASH_MISMATCH",
                    f"Token {_qualified_name(entry.token_ref)} implementation_ref hash mismatch.",
                )
            )

        runtime_hash = runtime_environment_hash_v2(record.spec.runtime_environment_ref)
        if runtime_hash != entry.runtime_environment_hash:
            diagnostics.append(
                _diagnostic(
                    "QST_V2_LOCK_RUNTIME_ENVIRONMENT_HASH_MISMATCH",
                    f"Token {_qualified_name(entry.token_ref)} runtime_environment_ref hash mismatch.",
                )
            )

    return ValidationResult(diagnostics=diagnostics)


class _TokenRecord(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    spec: TokenSpecV2
    pack_hash: str
    spec_hash: str


def _pack_token_records(
    packs: tuple[TokenPackManifestV2, ...],
) -> dict[tuple[str, str, int, int], tuple[_TokenRecord, ...]]:
    records: dict[tuple[str, str, int, int], list[_TokenRecord]] = {}
    for pack in packs:
        pack_hash = token_pack_hash_for_pack_v2(pack)
        for spec in pack.tokens:
            records.setdefault(spec.ref_key, []).append(
                _TokenRecord(
                    spec=spec,
                    pack_hash=pack_hash,
                    spec_hash=token_spec_hash_for_spec_v2(spec),
                )
            )
    return {
        key: tuple(sorted(value, key=lambda record: (record.pack_hash, record.spec_hash)))
        for key, value in records.items()
    }


def _ref_key(ref: TokenRefV04) -> tuple[str, str, int, int]:
    return (ref.namespace, ref.name, ref.version, ref.behavior_version)


def _qualified_name(ref: TokenRefV04) -> str:
    return f"{ref.namespace}.{ref.name}/v{ref.version}/bv{ref.behavior_version}"


def _risk_level(spec: TokenSpecV2) -> RiskLevel:
    value = spec.risk.risk_level
    if value == "low":
        return "low"
    if value == "medium":
        return "medium"
    if value == "high":
        return "high"
    if value == "unknown":
        return "unknown"
    return "unknown"


def _diagnostic(code: str, message: str, severity: Literal["error", "warning"] = "error") -> Diagnostic:
    return Diagnostic(
        code=code,
        severity=severity,
        phase="package",
        message=message,
    )
