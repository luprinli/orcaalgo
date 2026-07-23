"""Qlib workflow YAML loading without importing or executing Qlib."""

from __future__ import annotations

from collections.abc import Mapping
from datetime import date, datetime
from pathlib import Path
from typing import Any

import yaml

YAML_LOADER = getattr(yaml, "CSafeLoader", yaml.SafeLoader)


def load_workflow_config(path: Path) -> dict[str, Any]:
    """Load a Qlib workflow YAML file as canonical JSON-compatible data."""

    raw = yaml.load(path.read_text(encoding="utf-8"), Loader=YAML_LOADER)  # noqa: S506
    if not isinstance(raw, Mapping):
        raise ValueError("Qlib workflow config must be a YAML mapping")
    normalized = _normalize_yaml_value(raw)
    if not isinstance(normalized, dict):
        raise ValueError("Qlib workflow config must normalize to a mapping")
    return normalized


def _normalize_yaml_value(value: Any) -> Any:
    if isinstance(value, datetime | date):
        return value.isoformat()
    if isinstance(value, Mapping):
        return {str(key): _normalize_yaml_value(item) for key, item in value.items()}
    if isinstance(value, list | tuple):
        return [_normalize_yaml_value(item) for item in value]
    if value is None or isinstance(value, str | int | float | bool):
        return value
    return str(value)

