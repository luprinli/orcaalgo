"""Token spec hash framework for Token System v2."""

from __future__ import annotations

from typing import TYPE_CHECKING, Any

from qst.hash.common import hash_v2_payload

if TYPE_CHECKING:
    from qst.tokens.spec import TokenSpecV2


def token_spec_hash_v2(payload: Any | None = None) -> str:
    """Hash TokenSpec v2 material once WP5 defines it."""

    return hash_v2_payload("token_spec", {} if payload is None else payload)


def token_spec_hash_for_spec_v2(spec: TokenSpecV2 | dict[str, Any]) -> str:
    """Hash canonical TokenSpec v2 material."""

    from qst.tokens.spec import TokenSpecV2

    if not isinstance(spec, TokenSpecV2):
        spec = TokenSpecV2.model_validate(spec)
    return token_spec_hash_v2(spec.model_dump(mode="json"))
