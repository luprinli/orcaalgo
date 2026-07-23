from __future__ import annotations

import pytest

from qst.hash import token_pack_hash_for_pack_v2
from qst.tokens import TokenPackDependency, TokenPackManifestV2
from tests.tokens.test_token_spec_v2 import make_spec


def make_pack(
    pack_id: str = "demo-pack",
    *,
    version: str = "1.0.0",
    namespace: str = "demo",
    tokens: tuple[object, ...] | None = None,
    dependencies: tuple[TokenPackDependency, ...] = (),
    origin_tier: str = "community_pack",
) -> TokenPackManifestV2:
    return TokenPackManifestV2(
        pack_id=pack_id,
        version=version,
        namespaces=(namespace,),
        tokens=tokens if tokens is not None else (make_spec(namespace=namespace),),
        dependencies=dependencies,
        origin_tier=origin_tier,  # type: ignore[arg-type]
    )


def test_token_pack_validates_pep440_version_and_namespaces() -> None:
    pack = make_pack()

    assert pack.version == "1.0.0"
    assert pack.namespaces == ("demo",)

    with pytest.raises(ValueError):
        make_pack(version="not a version")

    with pytest.raises(ValueError):
        make_pack(namespace="other", tokens=(make_spec(namespace="demo"),))


def test_token_pack_sorts_tokens_and_dependencies() -> None:
    alpha = make_spec(name="alpha")
    zed = make_spec(name="zed")
    pack = make_pack(
        tokens=(zed, alpha),
        dependencies=(
            TokenPackDependency(pack_id="z-pack", version_constraint=">=1"),
            TokenPackDependency(pack_id="a-pack", version_constraint=">=1"),
        ),
    )

    assert [token.token_ref.name for token in pack.tokens] == ["alpha", "zed"]
    assert [dependency.pack_id for dependency in pack.dependencies] == ["a-pack", "z-pack"]


def test_token_pack_hash_is_deterministic_and_sensitive() -> None:
    pack = make_pack()
    with_dep = make_pack(
        dependencies=(TokenPackDependency(pack_id="base", version_constraint=">=1.0"),)
    )

    assert token_pack_hash_for_pack_v2(pack) == token_pack_hash_for_pack_v2(pack)
    assert token_pack_hash_for_pack_v2(pack) != token_pack_hash_for_pack_v2(with_dep)


def test_embedded_source_requires_policy() -> None:
    with pytest.raises(ValueError):
        TokenPackManifestV2(
            pack_id="bad-pack",
            version="1.0.0",
            namespaces=("bad",),
            tokens=(),
            origin_tier="community_pack",
            embeds_source=True,
            embedded_token_policy="none",
        )


def test_dependency_constraint_is_validated() -> None:
    with pytest.raises(ValueError):
        TokenPackDependency(pack_id="base", version_constraint="=>1")
