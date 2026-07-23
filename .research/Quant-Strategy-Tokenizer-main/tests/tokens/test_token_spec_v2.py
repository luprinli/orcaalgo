from __future__ import annotations

import pytest

from qst.hash import token_spec_hash_for_spec_v2
from qst.ir import TokenRefV04
from qst.numeric import NumericPolicy, semantic_float64_policy
from qst.token_evolution import TokenLifecycleStatus
from qst.tokens import TOKEN_SPEC_SCHEMA_VERSION, TokenRiskSpec, TokenSpecV2, token_surface


def make_spec(
    *,
    namespace: str = "demo",
    name: str = "identity",
    behavior_version: int = 1,
    numeric_policy: NumericPolicy | None = None,
    lifecycle: TokenLifecycleStatus | None = None,
    attestation_kind: str = "none",
    implementation_ref: dict[str, object] | None = None,
    runtime_environment_ref: dict[str, object] | None = None,
    risk: dict[str, object] | None = None,
) -> TokenSpecV2:
    return TokenSpecV2(
        token_id=f"{namespace}.{name}",
        token_ref=TokenRefV04(
            namespace=namespace,
            name=name,
            version=1,
            behavior_version=behavior_version,
        ),
        version=1,
        behavior_version=behavior_version,
        origin_tier="community_pack",
        attestation_kind=attestation_kind,  # type: ignore[arg-type]
        inputs={"x": {"type": "TimeSeries[float]"}},
        outputs={"y": {"type": "TimeSeries[float]"}},
        numeric_policy=numeric_policy or semantic_float64_policy(),
        lifecycle=lifecycle or TokenLifecycleStatus(),
        implementation_ref=implementation_ref,
        runtime_environment_ref=runtime_environment_ref,
        risk=risk or {},
        surface=token_surface(
            family="math",
            category="identity",
            layer="custom",
            maturity="experimental",
            execution_support="metadata_only",
        ),
    )


def test_token_spec_v2_validates_required_fields() -> None:
    spec = make_spec()

    assert spec.schema_version == TOKEN_SPEC_SCHEMA_VERSION
    assert spec.token_id == "demo.identity"
    assert spec.token_ref.namespace == "demo"
    assert spec.inputs["x"].type.kind == "TimeSeries"
    assert spec.outputs["y"].type.kind == "TimeSeries"
    assert spec.risk.risk_level == "unknown"


def test_token_spec_rejects_inconsistent_token_ref() -> None:
    with pytest.raises(ValueError):
        TokenSpecV2(
            token_id="wrong.identity",
            token_ref=TokenRefV04(namespace="demo", name="identity", version=1, behavior_version=1),
            version=1,
            behavior_version=1,
            origin_tier="community_pack",
            inputs={},
            outputs={},
            numeric_policy=semantic_float64_policy(),
            surface=token_surface(
                family="math",
                category="identity",
                layer="custom",
                maturity="experimental",
                execution_support="metadata_only",
            ),
        )


def test_token_spec_requires_numeric_policy() -> None:
    with pytest.raises(ValueError):
        TokenSpecV2.model_validate(
            {
                "token_id": "demo.identity",
                "token_ref": {
                    "namespace": "demo",
                    "name": "identity",
                    "version": 1,
                    "behavior_version": 1,
                },
                "version": 1,
                "behavior_version": 1,
                "origin_tier": "community_pack",
            }
        )


def test_token_spec_requires_surface_metadata() -> None:
    payload = make_spec().model_dump(mode="json")
    payload.pop("surface")

    with pytest.raises(ValueError):
        TokenSpecV2.model_validate(payload)


def test_token_spec_hash_is_deterministic_and_policy_sensitive() -> None:
    base = make_spec()
    bit_exact = make_spec(
        numeric_policy=NumericPolicy(
            representation="float64",
            deterministic_level="bit_exact",
            reduction_order="fixed_input_order",
            nan_policy="propagate",
            inf_policy="reject",
        )
    )

    assert token_spec_hash_for_spec_v2(base) == token_spec_hash_for_spec_v2(base)
    assert token_spec_hash_for_spec_v2(base) != token_spec_hash_for_spec_v2(bit_exact)


def test_lifecycle_and_behavior_version_affect_token_spec_hash() -> None:
    active = make_spec()
    deprecated = make_spec(lifecycle=TokenLifecycleStatus(lifecycle="deprecated"))
    behavior_v2 = make_spec(behavior_version=2)

    assert token_spec_hash_for_spec_v2(active) != token_spec_hash_for_spec_v2(deprecated)
    assert token_spec_hash_for_spec_v2(active) != token_spec_hash_for_spec_v2(behavior_v2)


def test_attestation_is_structural_claim_only() -> None:
    spec = make_spec(attestation_kind="qst_verified")

    assert spec.attestation_kind == "qst_verified"


def test_token_risk_spec_is_structured_and_hash_sensitive() -> None:
    low = make_spec(risk={"risk_level": "low", "reasons": ["b", "a", "a"], "requires_approval": False})
    high = make_spec(risk={"risk_level": "high", "reasons": ["a", "b"], "requires_approval": True})
    compat_shape = make_spec(risk={"risk_level": "medium", "explanation": ["executes local code"]})

    assert low.risk == TokenRiskSpec(risk_level="low", reasons=("a", "b"), requires_approval=False)
    assert compat_shape.risk.reasons == ("executes local code",)
    assert token_spec_hash_for_spec_v2(low) != token_spec_hash_for_spec_v2(high)
