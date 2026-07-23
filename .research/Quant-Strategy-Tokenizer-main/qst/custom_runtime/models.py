"""Core models for WP9 custom token runtime boundaries."""

from __future__ import annotations

from pathlib import Path
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field

from qst.hash import hash_v2_payload
from qst.ir import TokenRefV04
from qst.profiles import ProfileName
from qst.validation import Diagnostic, ValidationResult

ApprovalScope = Literal["project", "user"]
AuthorizationStatus = Literal["allowed", "requires_approval", "denied_by_profile"]


class TokenRuntimeContext(BaseModel):
    """Filesystem and profile context for custom token service calls."""

    model_config = ConfigDict(arbitrary_types_allowed=True)

    base_path: Path = Path(".")
    profile: ProfileName = "research"
    run_id: str = "manual"
    current_time_utc: str | None = None


class TokenIntegrityResult(BaseModel):
    """Integrity-only verification outcome.

    This result must not encode whether a human approved execution.
    """

    model_config = ConfigDict(extra="forbid", frozen=True)

    ok: bool = True
    token_ref: TokenRefV04
    token_spec_hash: str = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    token_pack_hash: str = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    implementation_ref_hash: str = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    runtime_environment_hash: str = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    audit_chain_hash: str = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    risk_level: str = "unknown"
    diagnostics: list[Diagnostic] = Field(default_factory=list)

    @classmethod
    def from_diagnostics(
        cls,
        *,
        token_ref: TokenRefV04,
        token_spec_hash: str,
        token_pack_hash: str,
        implementation_ref_hash: str,
        runtime_environment_hash: str,
        audit_chain_hash: str,
        risk_level: str,
        diagnostics: list[Diagnostic],
    ) -> TokenIntegrityResult:
        return cls(
            ok=all(diagnostic.severity != "error" for diagnostic in diagnostics),
            token_ref=token_ref,
            token_spec_hash=token_spec_hash,
            token_pack_hash=token_pack_hash,
            implementation_ref_hash=implementation_ref_hash,
            runtime_environment_hash=runtime_environment_hash,
            audit_chain_hash=audit_chain_hash,
            risk_level=risk_level,
            diagnostics=diagnostics,
        )


class TokenAuthorizationResult(BaseModel):
    """Profile and approval authorization result."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    ok: bool = True
    status: AuthorizationStatus
    profile: ProfileName
    diagnostics: list[Diagnostic] = Field(default_factory=list)
    approval_record_hash: str | None = Field(default=None, pattern=r"^sha256:[0-9a-f]{64}$")


class ApprovalRequest(BaseModel):
    """Request to write a local custom-token approval."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    token_ref: TokenRefV04
    profile: ProfileName
    scope: ApprovalScope = "project"
    approved_by: str = Field(min_length=1)
    allow_token: bool = False
    ack_risk: bool = False
    approved_risk_level: str = "unknown"
    approved_capabilities: tuple[str, ...] = ("custom_token_runtime",)
    token_spec_hash: str = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    token_pack_hash: str = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    implementation_ref_hash: str = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    runtime_environment_hash: str = Field(pattern=r"^sha256:[0-9a-f]{64}$")


class ApprovalRecord(ApprovalRequest):
    """Persisted local approval record."""

    schema_version: Literal["qst-approval-record/0.4"] = "qst-approval-record/0.4"
    approval_id: str


class ExecutionGrant(BaseModel):
    """Short-lived permission to execute one token under one profile."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-execution-grant/0.4"] = "qst-execution-grant/0.4"
    token_ref: TokenRefV04
    token_spec_hash: str = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    token_pack_hash: str = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    implementation_ref_hash: str = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    runtime_environment_hash: str = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    approval_record_hash: str = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    profile: ProfileName
    issued_for_run_id: str
    expires_at: str


class TokenVerifyReport(BaseModel):
    """User-facing verify report with separated integrity and authorization."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    integrity: TokenIntegrityResult
    authorization: TokenAuthorizationResult

    @property
    def ok(self) -> bool:
        return self.integrity.ok and self.authorization.ok


class TokenExecutionResult(BaseModel):
    """Custom token execution result."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    ok: bool = True
    output: Any | None = None
    diagnostics: list[Diagnostic] = Field(default_factory=list)
    audit_chain_hash: str = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    audit_records: list[dict[str, Any]] = Field(default_factory=list)

    @classmethod
    def from_validation(
        cls,
        *,
        output: Any | None,
        validation: ValidationResult,
        audit_chain_hash: str,
        audit_records: list[dict[str, Any]],
    ) -> TokenExecutionResult:
        return cls(
            ok=validation.ok,
            output=output if validation.ok else None,
            diagnostics=validation.diagnostics,
            audit_chain_hash=audit_chain_hash,
            audit_records=audit_records,
        )


def approval_record_hash(record: ApprovalRecord | dict[str, Any]) -> str:
    """Hash approval records as local trust material."""

    parsed = record if isinstance(record, ApprovalRecord) else ApprovalRecord.model_validate(record)
    return hash_v2_payload("approval_record", parsed.model_dump(mode="json"))
