"""TokenPack package manifest policy helpers for QST."""

from __future__ import annotations

from collections.abc import Iterable
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field, model_validator

from qst.hash import token_pack_hash_for_pack_v2
from qst.tokens.pack import EmbeddedTokenPolicy, TokenPackManifestV2
from qst.validation import Diagnostic, ValidationResult

HashString = str


class TokenPackPackageEntryV04(BaseModel):
    """One TokenPack reference recorded in a GKR package manifest."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    pack_id: str = Field(min_length=1)
    version: str = Field(min_length=1)
    token_pack_hash: HashString = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    embedded: bool = False
    contains_executable_code: bool = False


class TokenPacksPackageSectionV04(BaseModel):
    """Optional TokenPack propagation section for GKR package manifests."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    embedded_policy: EmbeddedTokenPolicy = "none"
    packs: tuple[TokenPackPackageEntryV04, ...] = Field(default_factory=tuple)

    @model_validator(mode="after")
    def _validate_policy(self) -> TokenPacksPackageSectionV04:
        for entry in self.packs:
            if self.embedded_policy == "none" and entry.embedded:
                raise ValueError("embedded_policy=none cannot embed TokenPack material")
            if (
                self.embedded_policy == "spec_only"
                and entry.embedded
                and entry.contains_executable_code
            ):
                raise ValueError("embedded_policy=spec_only cannot embed executable code")
        return self


def token_pack_package_entry_from_pack(
    pack: TokenPackManifestV2,
    *,
    embedded: bool,
) -> TokenPackPackageEntryV04:
    """Build one deterministic GKR package TokenPack manifest entry."""

    return TokenPackPackageEntryV04(
        pack_id=pack.pack_id,
        version=pack.version,
        token_pack_hash=token_pack_hash_for_pack_v2(pack),
        embedded=embedded,
        contains_executable_code=pack.contains_executable_code,
    )


def token_pack_package_section_from_packs(
    packs: Iterable[TokenPackManifestV2],
    *,
    embedded_policy: EmbeddedTokenPolicy,
) -> TokenPacksPackageSectionV04:
    """Build the GKR package TokenPack section from pack metadata."""

    embedded = embedded_policy != "none"
    return TokenPacksPackageSectionV04(
        embedded_policy=embedded_policy,
        packs=tuple(
            sorted(
                (
                    token_pack_package_entry_from_pack(pack, embedded=embedded)
                    for pack in packs
                ),
                key=lambda entry: (entry.pack_id, entry.version, entry.token_pack_hash),
            )
        ),
    )


def verify_token_pack_package_section(
    section: TokenPacksPackageSectionV04,
    available_packs: Iterable[TokenPackManifestV2],
) -> ValidationResult:
    """Verify GKR package TokenPack references without executing embedded source."""

    available = {
        (pack.pack_id, pack.version): token_pack_hash_for_pack_v2(pack)
        for pack in available_packs
    }
    diagnostics: list[Diagnostic] = []
    for entry in section.packs:
        actual_hash = available.get((entry.pack_id, entry.version))
        if actual_hash is None:
            diagnostics.append(
                _diagnostic(
                    "QST_V2_PACKAGE_TOKEN_PACK_MISSING",
                    (
                        f"TokenPack {entry.pack_id} {entry.version} referenced by GKR package "
                        "is not available."
                    ),
                )
            )
            continue
        if actual_hash != entry.token_pack_hash:
            diagnostics.append(
                _diagnostic(
                    "QST_V2_PACKAGE_TOKEN_PACK_HASH_MISMATCH",
                    (
                        f"TokenPack {entry.pack_id} {entry.version} hash mismatch: "
                        f"expected {entry.token_pack_hash}, got {actual_hash}."
                    ),
                )
            )
    return ValidationResult(diagnostics=diagnostics)


def _diagnostic(code: str, message: str, severity: Literal["error", "warning"] = "error") -> Diagnostic:
    return Diagnostic(
        code=code,
        severity=severity,
        phase="package",
        message=message,
    )
