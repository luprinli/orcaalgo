"""TokenPack file loading helpers for WP9."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

import yaml

from qst.tokens import TokenPackManifestV2

TOKEN_PACK_FILENAMES = (
    "tokenpack.json",
    "tokenpack.yaml",
    "tokenpack.yml",
    "manifest.json",
    "manifest.yaml",
    "manifest.yml",
)


def load_token_pack(path: str | Path) -> TokenPackManifestV2:
    """Load a TokenPack manifest from a file or directory."""

    source = Path(path)
    manifest_path = _manifest_path(source)
    raw = _read_mapping(manifest_path)
    return TokenPackManifestV2.model_validate(raw)


def _manifest_path(source: Path) -> Path:
    if source.is_file():
        return source
    for filename in TOKEN_PACK_FILENAMES:
        candidate = source / filename
        if candidate.exists():
            return candidate
    raise FileNotFoundError(f"No TokenPack manifest found under {source}")


def _read_mapping(path: Path) -> dict[str, Any]:
    text = path.read_text(encoding="utf-8")
    if path.suffix.lower() == ".json":
        raw = json.loads(text)
    else:
        raw = yaml.safe_load(text)
    if not isinstance(raw, dict):
        raise TypeError(f"TokenPack manifest must be a mapping: {path}")
    return raw
