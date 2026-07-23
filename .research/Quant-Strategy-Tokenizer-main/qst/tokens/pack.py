"""TokenPack v2 manifest and dependency models."""

from __future__ import annotations

from typing import Any, Literal

from packaging.specifiers import InvalidSpecifier, SpecifierSet
from packaging.version import InvalidVersion, Version
from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator

from qst.tokens.spec import AttestationKind, OriginTier, TokenSpecV2

TOKEN_PACK_SCHEMA_VERSION: Literal["qst-token-pack/0.4"] = "qst-token-pack/0.4"
EmbeddedTokenPolicy = Literal["none", "spec_only", "spec_and_source"]


class TokenPackDependency(BaseModel):
    """Dependency on another TokenPack."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    pack_id: str = Field(min_length=1)
    version_constraint: str = ""
    token_pack_hash: str | None = Field(default=None, pattern=r"^sha256:[0-9a-f]{64}$")

    @field_validator("version_constraint")
    @classmethod
    def _validate_specifier(cls, value: str) -> str:
        try:
            SpecifierSet(value)
        except InvalidSpecifier as exc:
            raise ValueError(f"Invalid TokenPack dependency constraint: {value!r}") from exc
        return value

    def matches(self, version: str) -> bool:
        """Whether a version satisfies this dependency."""

        parsed = parse_version(version)
        return parsed in SpecifierSet(self.version_constraint)


class TokenPackManifestV2(BaseModel):
    """Portable TokenPack manifest used by WP5 registry validation."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-token-pack/0.4"] = TOKEN_PACK_SCHEMA_VERSION
    pack_id: str = Field(min_length=1)
    version: str
    namespaces: tuple[str, ...]
    tokens: tuple[TokenSpecV2, ...] = Field(default_factory=tuple)
    dependencies: tuple[TokenPackDependency, ...] = Field(default_factory=tuple)
    origin_tier: OriginTier
    attestation_kind: AttestationKind = "none"
    embedded_token_policy: EmbeddedTokenPolicy = "none"
    contains_executable_code: bool = False
    embeds_source: bool = False

    @model_validator(mode="before")
    @classmethod
    def _sort_sequences(cls, value: Any) -> Any:
        if not isinstance(value, dict):
            return value
        normalized = dict(value)
        normalized["namespaces"] = sorted(set(normalized.get("namespaces", ())))
        normalized["tokens"] = sorted(
            normalized.get("tokens", ()),
            key=_token_sort_key,
        )
        normalized["dependencies"] = sorted(
            normalized.get("dependencies", ()),
            key=_dependency_sort_key,
        )
        return normalized

    @field_validator("version")
    @classmethod
    def _validate_version(cls, value: str) -> str:
        parse_version(value)
        return value

    @model_validator(mode="after")
    def _validate_manifest(self) -> TokenPackManifestV2:
        if not self.namespaces:
            raise ValueError("TokenPack must declare at least one namespace")
        namespace_set = set(self.namespaces)
        for token in self.tokens:
            if token.token_ref.namespace not in namespace_set:
                raise ValueError(
                    f"Token {token.token_id!r} namespace is not declared by pack {self.pack_id!r}"
                )
        if self.embedded_token_policy == "spec_and_source" and not self.embeds_source:
            raise ValueError("spec_and_source policy requires embeds_source=true")
        if self.embeds_source and self.embedded_token_policy == "none":
            raise ValueError("embeds_source requires an embedding policy")
        return self

    @property
    def parsed_version(self) -> Version:
        """PEP 440 parsed version."""

        return parse_version(self.version)


def parse_version(version: str) -> Version:
    """Parse a PEP 440 version and normalize exceptions."""

    try:
        return Version(version)
    except InvalidVersion as exc:
        raise ValueError(f"Invalid TokenPack version: {version!r}") from exc


def _token_sort_key(value: Any) -> tuple[str, str, int, int]:
    if isinstance(value, TokenSpecV2):
        return value.ref_key
    if isinstance(value, dict):
        token_ref = value.get("token_ref", {})
        if isinstance(token_ref, dict):
            return (
                str(token_ref.get("namespace", "")),
                str(token_ref.get("name", "")),
                _int_or_zero(token_ref.get("version", value.get("version", 0))),
                _int_or_zero(
                    token_ref.get("behavior_version", value.get("behavior_version", 0))
                ),
            )
    return ("", "", 0, 0)


def _dependency_sort_key(value: Any) -> tuple[str, str, str]:
    if isinstance(value, TokenPackDependency):
        return (value.pack_id, value.version_constraint, value.token_pack_hash or "")
    if isinstance(value, dict):
        return (
            str(value.get("pack_id", "")),
            str(value.get("version_constraint", "")),
            str(value.get("token_pack_hash", "")),
        )
    return ("", "", "")


def _int_or_zero(value: object) -> int:
    if isinstance(value, int):
        return value
    if isinstance(value, str):
        try:
            return int(value)
        except ValueError:
            return 0
    return 0
