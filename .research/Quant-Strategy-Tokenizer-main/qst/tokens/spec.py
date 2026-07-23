"""TokenSpec v2 model for qst-ir/0.4 token metadata."""

from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator

from qst.canonical_json import stable_json_bytes
from qst.ir import TokenRefV04
from qst.numeric import NumericPolicy
from qst.ports import InputSpec, OutputSpec
from qst.token_evolution import TokenLifecycleStatus
from qst.tokens.surface import TokenSurfaceSpec

TOKEN_SPEC_SCHEMA_VERSION: Literal["qst-token-spec/0.4"] = "qst-token-spec/0.4"

OriginTier = Literal[
    "core",
    "verified_pack",
    "community_pack",
    "user_local",
    "experimental",
    "unsafe",
]
AttestationKind = Literal["none", "local_tests_passed", "qst_verified", "signed_pack"]
TokenPurityV2 = Literal[
    "pure",
    "contextual_read",
    "external_read",
    "external_write",
    "forbidden",
]
RiskLevel = Literal["low", "medium", "high", "unknown"]


class TokenRiskSpec(BaseModel):
    """Structured risk metadata for v0.4 token specs."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    risk_level: RiskLevel = "unknown"
    reasons: tuple[str, ...] = ()
    requires_approval: bool = True

    @model_validator(mode="before")
    @classmethod
    def _coerce_compat_shape(cls, value: Any) -> Any:
        if isinstance(value, dict) and "explanation" in value and "reasons" not in value:
            coerced = dict(value)
            explanation = coerced.pop("explanation")
            coerced["reasons"] = explanation
            return coerced
        return value

    @field_validator("reasons", mode="after")
    @classmethod
    def _sort_reasons(cls, value: tuple[str, ...]) -> tuple[str, ...]:
        return tuple(sorted(dict.fromkeys(value)))


class TokenSpecV2(BaseModel):
    """Serializable v0.4 token specification.

    WP5 stores implementation references as canonical metadata only. It never
    imports or executes custom token code.
    """

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-token-spec/0.4"] = TOKEN_SPEC_SCHEMA_VERSION
    token_id: str = Field(min_length=3)
    token_ref: TokenRefV04
    version: int = Field(ge=1)
    behavior_version: int = Field(ge=1)
    origin_tier: OriginTier
    attestation_kind: AttestationKind = "none"
    inputs: dict[str, InputSpec] = Field(default_factory=dict)
    outputs: dict[str, OutputSpec] = Field(default_factory=dict)
    params_schema: dict[str, Any] = Field(default_factory=dict)
    purity: TokenPurityV2 = "pure"
    state: dict[str, Any] = Field(default_factory=dict)
    numeric_policy: NumericPolicy
    implementation_ref: dict[str, Any] | None = None
    runtime_environment_ref: dict[str, Any] | None = None
    lifecycle: TokenLifecycleStatus = Field(default_factory=TokenLifecycleStatus)
    risk: TokenRiskSpec = Field(default_factory=TokenRiskSpec)
    surface: TokenSurfaceSpec
    tests: list[dict[str, Any]] = Field(default_factory=list)
    dependencies: list[TokenRefV04] = Field(default_factory=list)

    @field_validator(
        "params_schema",
        "state",
        "implementation_ref",
        "runtime_environment_ref",
        mode="after",
    )
    @classmethod
    def _validate_json_mapping(cls, value: dict[str, Any] | None) -> dict[str, Any] | None:
        if value is not None:
            _ensure_json(value, field_name="TokenSpecV2 mapping")
        return value

    @field_validator("tests", mode="after")
    @classmethod
    def _validate_tests(cls, value: list[dict[str, Any]]) -> list[dict[str, Any]]:
        _ensure_json(value, field_name="TokenSpecV2 tests")
        return value

    @model_validator(mode="after")
    def _validate_identity(self) -> TokenSpecV2:
        expected_token_id = f"{self.token_ref.namespace}.{self.token_ref.name}"
        if self.token_id != expected_token_id:
            raise ValueError(
                f"token_id {self.token_id!r} must match token_ref {expected_token_id!r}"
            )
        if "." not in self.token_id:
            raise ValueError("token_id must be namespace-qualified")
        if self.version != self.token_ref.version:
            raise ValueError("version must match token_ref.version")
        if self.behavior_version != self.token_ref.behavior_version:
            raise ValueError("behavior_version must match token_ref.behavior_version")
        return self

    @property
    def ref_key(self) -> tuple[str, str, int, int]:
        """Canonical token reference key."""

        return (
            self.token_ref.namespace,
            self.token_ref.name,
            self.token_ref.version,
            self.token_ref.behavior_version,
        )


def _ensure_json(value: Any, *, field_name: str) -> None:
    try:
        stable_json_bytes(value)
    except (TypeError, ValueError) as exc:
        raise ValueError(f"{field_name} must be canonical JSON-compatible") from exc
