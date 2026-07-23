"""Adapter manifest artifact model."""

from __future__ import annotations

from typing import Any

from packaging.specifiers import SpecifierSet
from packaging.version import Version
from pydantic import BaseModel, ConfigDict, Field, field_validator


class AdapterManifest(BaseModel):
    """Manifest for a QST adapter package."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    adapter_id: str
    adapter_version: str
    qst_core_compatible: str
    implements_ports: list[str]
    supported_engines: list[str] = Field(default_factory=list)
    capabilities: dict[str, Any] = Field(default_factory=dict)
    known_limitations: list[str] = Field(default_factory=list)

    @field_validator("qst_core_compatible")
    @classmethod
    def validate_specifier(cls, value: str) -> str:
        try:
            SpecifierSet(value)
        except Exception as exc:
            raise ValueError(f"Not a valid PEP 440 SpecifierSet: {value!r}") from exc
        return value

    @field_validator("adapter_version")
    @classmethod
    def validate_version(cls, value: str) -> str:
        try:
            Version(value)
        except Exception as exc:
            raise ValueError(f"Not a valid PEP 440 version: {value!r}") from exc
        return value
