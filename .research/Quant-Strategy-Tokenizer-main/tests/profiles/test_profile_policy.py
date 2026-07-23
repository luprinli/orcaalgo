from __future__ import annotations

from typing import cast

from qst.numeric import NumericPolicy, semantic_float64_policy
from qst.profiles import PROFILE_POLICY_SCHEMA_VERSION, get_profile_policy
from qst.profiles.policy import ProfileName


def test_default_profiles_are_stable() -> None:
    for profile in ("research", "paper", "pretrade", "production_guarded"):
        policy = get_profile_policy(cast(ProfileName, profile))
        assert policy.schema_version == PROFILE_POLICY_SCHEMA_VERSION
        assert policy.profile == profile
        assert policy.allowed_capabilities == ("core",)


def test_unsafe_future_policy() -> None:
    assert get_profile_policy("research").decide_unsafe_future(unsafe_future=True).action == "warning"
    assert get_profile_policy("pretrade").decide_unsafe_future(unsafe_future=True).action == "error"


def test_external_effect_policy() -> None:
    assert get_profile_policy("paper").decide_external_effect("external_read").action == "allow"
    assert get_profile_policy("pretrade").decide_external_effect("external_read").action == "error"
    assert get_profile_policy("research").decide_external_effect("external_write").action == "error"


def test_custom_token_risk_policy() -> None:
    assert get_profile_policy("research").decide_custom_token_risk("unknown").action == "warning"
    assert get_profile_policy("production_guarded").decide_custom_token_risk("medium").action == "error"


def test_unknown_numeric_lifecycle_and_capability_policy() -> None:
    policy = get_profile_policy("pretrade")

    assert policy.decide_unknown_numeric(unknown_numeric=True).action == "error"
    assert policy.decide_lifecycle("deprecated").action == "warning"
    assert policy.decide_lifecycle("known_bug").action == "error"
    assert policy.decide_lifecycle("blocked").action == "error"
    assert policy.decide_capability("core").action == "allow"
    assert policy.decide_capability("panel").action == "error"


def test_numeric_policy_profile_decision() -> None:
    unknown_policy = NumericPolicy(
        representation="unknown",
        deterministic_level="unknown",
        reduction_order="unknown",
        nan_policy="unknown",
        inf_policy="unknown",
    )

    assert get_profile_policy("research").decide_numeric_policy(semantic_float64_policy()).action == "allow"
    assert get_profile_policy("research").decide_numeric_policy(unknown_policy).action == "warning"
    assert get_profile_policy("pretrade").decide_numeric_policy(unknown_policy).action == "error"
