"""Default profile policies for Token System v2."""

from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, ConfigDict

from qst.numeric import NumericPolicy

PROFILE_POLICY_SCHEMA_VERSION: Literal["qst-profile-policy/0.4"] = "qst-profile-policy/0.4"

ProfileName = Literal["research", "paper", "pretrade", "production_guarded"]
ProfileAction = Literal["allow", "warning", "error"]
EffectKind = Literal["none", "external_read", "external_write"]
LifecycleState = Literal["active", "deprecated", "known_bug", "blocked"]
CustomTokenRisk = Literal["low", "medium", "high", "unknown"]
CapabilityName = Literal["core", "panel", "custom_token_runtime"]


class ProfileDecision(BaseModel):
    """A deterministic profile policy decision."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    action: ProfileAction
    code: str
    message: str


class ProfilePolicy(BaseModel):
    """Profile-level guardrails used by v2 validation."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-profile-policy/0.4"] = PROFILE_POLICY_SCHEMA_VERSION
    profile: ProfileName
    allow_unsafe_future: bool
    allow_external_read: bool
    allow_external_write: bool = False
    allow_unknown_numeric: bool
    allowed_capabilities: tuple[CapabilityName, ...] = ("core",)

    def decide_unsafe_future(self, *, unsafe_future: bool) -> ProfileDecision:
        """Decide whether future-looking data is allowed."""

        if not unsafe_future:
            return _decision("allow", "unsafe_future_absent", "No unsafe future data declared.")
        if self.allow_unsafe_future:
            return _decision("warning", "unsafe_future_warning", "Unsafe future data is research-only.")
        return _decision("error", "unsafe_future_rejected", "Unsafe future data is not allowed.")

    def decide_external_effect(self, effect: EffectKind) -> ProfileDecision:
        """Decide whether an external effect is allowed."""

        if effect == "none":
            return _decision("allow", "external_effect_absent", "No external effect declared.")
        if effect == "external_write":
            return _decision("error", "external_write_rejected", "External writes are not allowed.")
        if self.allow_external_read:
            return _decision("allow", "external_read_allowed", "External reads are allowed.")
        return _decision("error", "external_read_rejected", "External reads are not allowed.")

    def decide_custom_token_risk(self, risk: CustomTokenRisk) -> ProfileDecision:
        """Decide whether custom token runtime risk is acceptable."""

        if risk == "low":
            return _decision("allow", "custom_token_low_risk", "Low-risk custom token accepted.")
        if self.profile in {"research", "paper"}:
            return _decision("warning", "custom_token_risk_warning", f"Custom token risk={risk}.")
        return _decision("error", "custom_token_risk_rejected", f"Custom token risk={risk}.")

    def decide_unknown_numeric(self, *, unknown_numeric: bool) -> ProfileDecision:
        """Decide whether unknown numeric behavior is acceptable."""

        if not unknown_numeric:
            return _decision("allow", "unknown_numeric_absent", "No unknown numeric behavior declared.")
        if self.allow_unknown_numeric:
            return _decision("warning", "unknown_numeric_warning", "Unknown numeric behavior is research-only.")
        return _decision("error", "unknown_numeric_rejected", "Unknown numeric behavior is not allowed.")

    def decide_numeric_policy(self, numeric_policy: NumericPolicy) -> ProfileDecision:
        """Decide whether a declared numeric policy is acceptable for this profile."""

        if numeric_policy.risk_level == "low":
            return _decision("allow", "numeric_policy_low_risk", "Numeric policy is low risk.")
        if numeric_policy.risk_level == "medium":
            if self.profile in {"research", "paper"}:
                return _decision(
                    "warning",
                    "numeric_policy_medium_risk",
                    "Numeric policy is engine-dependent.",
                )
            return _decision(
                "error",
                "numeric_policy_medium_risk_rejected",
                "Engine-dependent numeric policy is not allowed.",
            )
        if self.allow_unknown_numeric:
            return _decision(
                "warning",
                "numeric_policy_high_risk",
                "Unknown or platform-dependent numeric policy is research-only.",
            )
        return _decision(
            "error",
            "numeric_policy_high_risk_rejected",
            "Unknown or platform-dependent numeric policy is not allowed.",
        )

    def decide_lifecycle(self, lifecycle: LifecycleState) -> ProfileDecision:
        """Decide whether a token lifecycle state is acceptable."""

        if lifecycle == "active":
            return _decision("allow", "lifecycle_active", "Lifecycle is active.")
        if lifecycle == "deprecated":
            return _decision("warning", "lifecycle_deprecated", "Lifecycle is deprecated.")
        if lifecycle == "known_bug" and self.profile in {"research", "paper"}:
            return _decision("warning", "lifecycle_known_bug", "Known bug is research-only.")
        return _decision("error", f"lifecycle_{lifecycle}_rejected", f"Lifecycle {lifecycle} is rejected.")

    def decide_capability(self, capability: CapabilityName) -> ProfileDecision:
        """Decide whether a v0.4 capability is allowed in the current stage."""

        if capability in self.allowed_capabilities:
            return _decision("allow", "capability_allowed", f"Capability {capability} is allowed.")
        return _decision("error", "capability_not_accepted", f"Capability {capability} is not accepted yet.")


def _decision(action: ProfileAction, code: str, message: str) -> ProfileDecision:
    return ProfileDecision(action=action, code=code, message=message)


_DEFAULT_POLICIES: dict[ProfileName, ProfilePolicy] = {
    "research": ProfilePolicy(
        profile="research",
        allow_unsafe_future=True,
        allow_external_read=True,
        allow_unknown_numeric=True,
    ),
    "paper": ProfilePolicy(
        profile="paper",
        allow_unsafe_future=True,
        allow_external_read=True,
        allow_unknown_numeric=True,
    ),
    "pretrade": ProfilePolicy(
        profile="pretrade",
        allow_unsafe_future=False,
        allow_external_read=False,
        allow_unknown_numeric=False,
    ),
    "production_guarded": ProfilePolicy(
        profile="production_guarded",
        allow_unsafe_future=False,
        allow_external_read=False,
        allow_unknown_numeric=False,
    ),
}


def get_profile_policy(profile: ProfileName) -> ProfilePolicy:
    """Return a frozen default profile policy."""

    return _DEFAULT_POLICIES[profile]
