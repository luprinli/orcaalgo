from __future__ import annotations

import pytest

from qst.token_evolution import (
    TOKEN_EVOLUTION_POLICY_SCHEMA_VERSION,
    TOKEN_LIFECYCLE_SCHEMA_VERSION,
    TokenLifecycleStatus,
    default_token_evolution_policy,
)


def test_default_token_evolution_policy_matches_wp4_rules() -> None:
    policy = default_token_evolution_policy()

    assert policy.schema_version == TOKEN_EVOLUTION_POLICY_SCHEMA_VERSION
    assert policy.behavior_version_never_silent
    assert policy.output_changing_bugfix_bumps_behavior_version
    assert policy.old_behavior_version_remains_verifiable
    assert policy.known_bug_emits_audit_warning
    assert policy.deprecated_emits_audit_warning
    assert not policy.new_recipes_may_default_to_deprecated
    assert policy.blocked_token_hard_error_profiles == ("pretrade", "production_guarded")


def test_lifecycle_status_is_stable_hash_material() -> None:
    lifecycle = TokenLifecycleStatus(lifecycle="deprecated", reason="superseded")

    assert lifecycle.schema_version == TOKEN_LIFECYCLE_SCHEMA_VERSION
    assert lifecycle.lifecycle == "deprecated"
    assert lifecycle.reason == "superseded"


def test_replacement_token_ref_must_be_canonical_json() -> None:
    with pytest.raises(ValueError):
        TokenLifecycleStatus(
            lifecycle="deprecated",
            replacement_token_ref={1: "not a string key"},
        )
