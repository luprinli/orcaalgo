"""Base artifact models."""

from __future__ import annotations

from typing import Any

from pydantic import BaseModel, ConfigDict, Field, model_validator

from qst.artifacts.safety import POSIXRelativePath
from qst.hash.common import HashString


class ProvenanceChain(BaseModel):
    """Content lineage for a artifact."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    parent_artifacts: list[HashString] = Field(default_factory=list)
    operation: str | None = None
    notes: list[str] = Field(default_factory=list)


class AdapterIdentity(BaseModel):
    """Identity of the adapter that produced an artifact."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    adapter_id: str
    adapter_version: str


class QSTArtifact(BaseModel):
    """Common fields shared by artifact records."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    artifact_version: str
    artifact_id: HashString | None = None
    provenance: ProvenanceChain = Field(default_factory=ProvenanceChain)
    metadata: dict[str, Any] = Field(default_factory=dict)
    raw_payload_ref: POSIXRelativePath | None = None
    raw_payload_hash: HashString | None = None

    @model_validator(mode="after")
    def validate_raw_payload_pairing(self) -> QSTArtifact:
        if self.raw_payload_ref is not None and self.raw_payload_hash is None:
            raise ValueError("raw_payload_ref set but raw_payload_hash is null")
        return self
