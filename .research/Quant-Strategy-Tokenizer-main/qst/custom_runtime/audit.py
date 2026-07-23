"""Audit records for Token System v2 custom token runtime."""

from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, model_validator

from qst.hash import audit_chain_hash_v2

AuditAction = Literal[
    "verify_integrity",
    "check_authorization",
    "approve",
    "execute",
]


class AuditRecord(BaseModel):
    """One deterministic custom-token audit event.

    ``recorded_at`` may be stored for humans, but it is excluded from audit
    chain hash material.
    """

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-custom-token-audit/0.4"] = "qst-custom-token-audit/0.4"
    action: AuditAction
    token_ref: dict[str, Any]
    profile: str
    outcome: str
    diagnostics: list[dict[str, Any]] = Field(default_factory=list)
    hashes: dict[str, str] = Field(default_factory=dict)
    recorded_at: str | None = None

    @model_validator(mode="before")
    @classmethod
    def _sort_payload(cls, value: object) -> object:
        if not isinstance(value, dict):
            return value
        normalized = dict(value)
        normalized["diagnostics"] = sorted(
            normalized.get("diagnostics", []),
            key=lambda item: (str(item.get("code", "")), str(item.get("message", ""))),
        )
        normalized["hashes"] = dict(sorted(normalized.get("hashes", {}).items()))
        return normalized

    def hash_material(self) -> dict[str, Any]:
        """Canonical material used by audit_chain_hash_v2."""

        return self.model_dump(mode="json", exclude={"recorded_at"})


def audit_chain_hash_for_records(records: list[AuditRecord] | tuple[AuditRecord, ...]) -> str:
    """Hash ordered audit JSONL material without wall-clock timestamps."""

    return audit_chain_hash_v2([record.hash_material() for record in records])
