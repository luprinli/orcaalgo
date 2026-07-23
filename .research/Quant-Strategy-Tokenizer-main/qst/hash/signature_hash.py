"""Signature hash framework for Token System v2."""

from __future__ import annotations

from collections.abc import Mapping
from typing import Any

from qst.hash.common import hash_v2_payload
from qst.ir.schema import TokenRefV04
from qst.panel import PanelTypeLayerSpec, parse_panel_type_by_output
from qst.ports import PortSignature


def signature_hash_v2(payload: Any | None = None) -> str:
    """Hash token or callable signature material."""

    return hash_v2_payload("signature", {} if payload is None else payload)


def signature_hash_for_ports_v2(
    signature: PortSignature | Mapping[str, Any],
    *,
    token_ref: TokenRefV04 | Mapping[str, Any] | None = None,
) -> str:
    """Hash structured v2 PortSignature material."""

    parsed = signature if isinstance(signature, PortSignature) else PortSignature.model_validate(signature)
    payload: dict[str, Any] = {
        "signature": parsed.model_dump(mode="json", exclude_none=True),
    }
    if token_ref is not None:
        parsed_ref = token_ref if isinstance(token_ref, TokenRefV04) else TokenRefV04.model_validate(token_ref)
        payload["token_ref"] = parsed_ref.model_dump(mode="json")
    return signature_hash_v2(payload)


def signature_hash_for_panel_ports_v2(
    signature: PortSignature | Mapping[str, Any],
    panel_type_by_output: Mapping[str, PanelTypeLayerSpec | Mapping[str, Any]],
    *,
    token_ref: TokenRefV04 | Mapping[str, Any] | None = None,
) -> str:
    """Hash structured PortSignature material plus semantic Panel metadata."""

    parsed = signature if isinstance(signature, PortSignature) else PortSignature.model_validate(signature)
    parsed_panel = parse_panel_type_by_output(dict(panel_type_by_output))
    payload: dict[str, Any] = {
        "signature": parsed.model_dump(mode="json", exclude_none=True),
        "panel_type_by_output": {
            output_name: spec.model_dump(mode="json", exclude_none=True)
            for output_name, spec in sorted(parsed_panel.items())
        },
    }
    if token_ref is not None:
        parsed_ref = token_ref if isinstance(token_ref, TokenRefV04) else TokenRefV04.model_validate(token_ref)
        payload["token_ref"] = parsed_ref.model_dump(mode="json")
    return signature_hash_v2(payload)
