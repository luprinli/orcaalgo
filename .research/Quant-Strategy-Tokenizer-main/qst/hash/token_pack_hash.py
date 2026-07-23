"""Token pack hash framework for Token System v2."""

from __future__ import annotations

from typing import TYPE_CHECKING, Any

from qst.hash.common import hash_v2_payload

if TYPE_CHECKING:
    from qst.tokens.pack import TokenPackManifestV2


def token_pack_hash_v2(payload: Any | None = None) -> str:
    """Hash TokenPack material once WP5 defines it."""

    return hash_v2_payload("token_pack", {} if payload is None else payload)


def token_pack_hash_for_pack_v2(pack: TokenPackManifestV2 | dict[str, Any]) -> str:
    """Hash canonical TokenPack v2 material."""

    from qst.tokens.pack import TokenPackManifestV2

    if not isinstance(pack, TokenPackManifestV2):
        pack = TokenPackManifestV2.model_validate(pack)
    return token_pack_hash_v2(pack.model_dump(mode="json"))
