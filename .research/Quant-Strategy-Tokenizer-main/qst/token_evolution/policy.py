"""TokenEvolutionPolicy shell for Token System v2."""

from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator

from qst.canonical_json import stable_json_bytes

TOKEN_EVOLUTION_POLICY_SCHEMA_VERSION: Literal["qst-token-evolution-policy/0.4"] = (
    "qst-token-evolution-policy/0.4"
)
TOKEN_LIFECYCLE_SCHEMA_VERSION: Literal["qst-token-lifecycle/0.4"] = "qst-token-lifecycle/0.4"

LifecycleState = Literal["active", "deprecated", "known_bug", "blocked"]


class TokenLifecycleStatus(BaseModel):
    """Lifecycle status hash material for a specific token behavior version."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-token-lifecycle/0.4"] = TOKEN_LIFECYCLE_SCHEMA_VERSION
    lifecycle: LifecycleState = "active"
    reason: str | None = None
    replacement_token_ref: dict[str, Any] | None = None

    @field_validator("replacement_token_ref")
    @classmethod
    def _validate_replacement_ref(cls, value: dict[str, Any] | None) -> dict[str, Any] | None:
        if value is None:
            return value
        try:
            stable_json_bytes(value)
        except (TypeError, ValueError) as exc:
            raise ValueError("replacement_token_ref must be canonical JSON-compatible") from exc
        return value


class TokenEvolutionPolicy(BaseModel):
    """Repository-wide behavior evolution policy for v0.4 TokenSpec work."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-token-evolution-policy/0.4"] = (
        TOKEN_EVOLUTION_POLICY_SCHEMA_VERSION
    )
    behavior_version_never_silent: bool = True
    output_changing_bugfix_bumps_behavior_version: bool = True
    old_behavior_version_remains_verifiable: bool = True
    known_bug_emits_audit_warning: bool = True
    deprecated_emits_audit_warning: bool = True
    new_recipes_may_default_to_deprecated: bool = False
    blocked_token_hard_error_profiles: tuple[str, ...] = Field(
        default=("pretrade", "production_guarded")
    )


def default_token_evolution_policy() -> TokenEvolutionPolicy:
    """Return the accepted Wdefault token evolution policy."""

    return TokenEvolutionPolicy()
