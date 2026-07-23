"""Core verify/approve/execute service for WP9 custom tokens."""

from __future__ import annotations

import importlib
import json
from datetime import UTC, datetime, timedelta
from pathlib import Path
from typing import Any

from pydantic import BaseModel, ConfigDict

from qst.artifacts.decimal_string import validate_decimal_string
from qst.canonical_json import stable_json_bytes
from qst.custom_runtime.audit import (
    AuditRecord,
    audit_chain_hash_for_records,
)
from qst.custom_runtime.implementation import (
    ImplementationRef,
    RuntimeEnvironmentRef,
    implementation_ref_hash_for_ref,
    resolve_implementation_hash,
    runtime_environment_ref_hash_for_ref,
)
from qst.custom_runtime.models import (
    ApprovalRecord,
    ApprovalRequest,
    ExecutionGrant,
    TokenAuthorizationResult,
    TokenExecutionResult,
    TokenIntegrityResult,
    TokenRuntimeContext,
    approval_record_hash,
)
from qst.hash import (
    token_pack_hash_for_pack_v2,
    token_spec_hash_for_spec_v2,
)
from qst.ir import TokenRefV04
from qst.profiles import ProfileName
from qst.tokens import TokenPackManifestV2, TokenSpecV2
from qst.validation import Diagnostic, ValidationResult

DEFAULT_EXECUTION_GRANT_TTL_SECONDS = 900


class ApprovalStore(BaseModel):
    """Small deterministic approval store.

    The default in-memory store is useful for tests and agent calls. ``load`` /
    ``save`` support project-local JSON files such as ``.qst/approvals.json``.
    """

    model_config = ConfigDict(extra="forbid", frozen=True)

    records: tuple[ApprovalRecord, ...] = ()

    @classmethod
    def load(cls, path: str | Path) -> ApprovalStore:
        source = Path(path)
        if not source.exists():
            return cls()
        raw = json.loads(source.read_text(encoding="utf-8"))
        if not isinstance(raw, dict):
            raise TypeError(f"Approval store must be a JSON object: {source}")
        return cls.model_validate(raw)

    def save(self, path: str | Path) -> None:
        target = Path(path)
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(stable_json_bytes(self.model_dump(mode="json")))

    def add(self, record: ApprovalRecord) -> ApprovalStore:
        retained = [
            item for item in self.records if _approval_key(item) != _approval_key(record)
        ]
        retained.append(record)
        return ApprovalStore(records=tuple(sorted(retained, key=_approval_sort_key)))

    def revoke(self, token_ref: TokenRefV04, *, profile: ProfileName | None = None) -> ApprovalStore:
        retained = [
            record
            for record in self.records
            if not (
                _ref_key(record.token_ref) == _ref_key(token_ref)
                and (profile is None or record.profile == profile)
            )
        ]
        return ApprovalStore(records=tuple(retained))

    def find(self, integrity: TokenIntegrityResult, *, profile: ProfileName) -> ApprovalRecord | None:
        for record in self.records:
            if not record.allow_token or not record.ack_risk:
                continue
            if record.profile != profile:
                continue
            if _ref_key(record.token_ref) != _ref_key(integrity.token_ref):
                continue
            if (
                record.token_spec_hash == integrity.token_spec_hash
                and record.token_pack_hash == integrity.token_pack_hash
                and record.implementation_ref_hash == integrity.implementation_ref_hash
                and record.runtime_environment_hash == integrity.runtime_environment_hash
            ):
                return record
        return None


class TokenRuntimeService:
    """Core service enforcing WP9 custom-token boundaries."""

    def verify_integrity(
        self,
        pack: TokenPackManifestV2,
        token_ref: TokenRefV04,
        *,
        context: TokenRuntimeContext | None = None,
        audit_records: list[AuditRecord] | None = None,
    ) -> TokenIntegrityResult:
        return verify_integrity(pack, token_ref, context=context, audit_records=audit_records)

    def check_authorization(
        self,
        integrity: TokenIntegrityResult,
        *,
        profile: ProfileName,
        approval_store: ApprovalStore | None = None,
        allow_token: bool = False,
        ack_risk: bool = False,
    ) -> TokenAuthorizationResult:
        return check_authorization(
            integrity,
            profile=profile,
            approval_store=approval_store,
            allow_token=allow_token,
            ack_risk=ack_risk,
        )

    def approve_token_pack(
        self,
        request: ApprovalRequest,
        *,
        approval_store: ApprovalStore | None = None,
    ) -> tuple[ApprovalRecord, ApprovalStore]:
        return approve_token_pack(request, approval_store=approval_store)

    def issue_execution_grant(
        self,
        integrity: TokenIntegrityResult,
        authorization: TokenAuthorizationResult,
        *,
        run_id: str,
        issued_at_utc: str,
        ttl_seconds: int = DEFAULT_EXECUTION_GRANT_TTL_SECONDS,
    ) -> ExecutionGrant:
        return issue_execution_grant(
            integrity,
            authorization,
            run_id=run_id,
            issued_at_utc=issued_at_utc,
            ttl_seconds=ttl_seconds,
        )

    def execute_custom_token(
        self,
        pack: TokenPackManifestV2,
        token_ref: TokenRefV04,
        *,
        inputs: dict[str, Any],
        grant: ExecutionGrant,
        context: TokenRuntimeContext | None = None,
        approval_store: ApprovalStore | None = None,
    ) -> TokenExecutionResult:
        return execute_custom_token(
            pack,
            token_ref,
            inputs=inputs,
            grant=grant,
            context=context,
            approval_store=approval_store,
        )


def verify_integrity(
    pack: TokenPackManifestV2,
    token_ref: TokenRefV04,
    *,
    context: TokenRuntimeContext | None = None,
    audit_records: list[AuditRecord] | None = None,
) -> TokenIntegrityResult:
    """Verify custom-token integrity without importing custom code."""

    ctx = context or TokenRuntimeContext()
    pack_hash = token_pack_hash_for_pack_v2(pack)
    spec = _find_spec(pack, token_ref)
    spec_hash = token_spec_hash_for_spec_v2(spec) if spec is not None else "sha256:" + "0" * 64
    diagnostics: list[Diagnostic] = []

    if spec is None:
        diagnostics.append(_diagnostic("QST_V2_TOKEN_REF_NOT_FOUND", "package", "Token ref not found."))
        return TokenIntegrityResult.from_diagnostics(
            token_ref=token_ref,
            token_spec_hash=spec_hash,
            token_pack_hash=pack_hash,
            implementation_ref_hash=implementation_ref_hash_for_ref(None),
            runtime_environment_hash=runtime_environment_ref_hash_for_ref(None),
            audit_chain_hash=audit_chain_hash_for_records(audit_records or []),
            risk_level="unknown",
            diagnostics=diagnostics,
        )

    implementation_ref = _parse_implementation_ref(spec)
    runtime_ref = _parse_runtime_ref(spec)
    implementation_hash = implementation_ref_hash_for_ref(implementation_ref)
    if implementation_ref is not None:
        try:
            resolved_hash, code = resolve_implementation_hash(implementation_ref, base_path=ctx.base_path)
        except Exception as exc:
            resolved_hash = implementation_hash
            code = "QST_V2_IMPLEMENTATION_REF_HASH_UNRESOLVED"
            diagnostics.append(_diagnostic(code, "package", f"{type(exc).__name__}: {exc}"))
        else:
            if code is not None:
                diagnostics.append(_diagnostic(code, "package", "Implementation reference is not reproducible."))
            if implementation_ref.expected_hash is not None and resolved_hash != implementation_ref.expected_hash:
                diagnostics.append(
                    _diagnostic(
                        "QST_V2_IMPLEMENTATION_REF_HASH_MISMATCH",
                        "package",
                        "implementation_ref expected_hash does not match resolved hash.",
                    )
                )
            if implementation_ref.expected_hash is None and implementation_ref.kind != "spec_only":
                diagnostics.append(
                    _diagnostic(
                        "QST_V2_IMPLEMENTATION_REF_HASH_RECORDED",
                        "package",
                        "Implementation reference hash is recorded but not pinned.",
                        severity="warning",
                    )
                )
            implementation_hash = implementation_ref.expected_hash or resolved_hash

    runtime_hash = runtime_environment_ref_hash_for_ref(runtime_ref)
    if runtime_ref is not None and runtime_ref.reproducibility_level == "environment_recorded":
        diagnostics.append(
            _diagnostic(
                "QST_V2_RUNTIME_ENVIRONMENT_RECORDED_ONLY",
                "package",
                "Runtime environment is recorded but not replayable.",
                severity="warning",
            )
        )

    risk_level = _risk_level(spec)
    if risk_level in {"medium", "high", "unknown"}:
        diagnostics.append(
            _diagnostic(
                "QST_V2_CUSTOM_TOKEN_RISK_VISIBLE",
                "profile",
                f"Custom token risk={risk_level}.",
                severity="warning",
            )
        )

    return TokenIntegrityResult.from_diagnostics(
        token_ref=token_ref,
        token_spec_hash=spec_hash,
        token_pack_hash=pack_hash,
        implementation_ref_hash=implementation_hash,
        runtime_environment_hash=runtime_hash,
        audit_chain_hash=audit_chain_hash_for_records(audit_records or []),
        risk_level=risk_level,
        diagnostics=diagnostics,
    )


def check_authorization(
    integrity: TokenIntegrityResult,
    *,
    profile: ProfileName,
    approval_store: ApprovalStore | None = None,
    allow_token: bool = False,
    ack_risk: bool = False,
) -> TokenAuthorizationResult:
    """Check profile/approval authorization separately from integrity."""

    diagnostics: list[Diagnostic] = []
    store = approval_store or ApprovalStore()
    approval = store.find(integrity, profile=profile)
    approval_hash = approval_record_hash(approval) if approval is not None else None

    if not integrity.ok:
        diagnostics.append(
            _diagnostic(
                "QST_V2_CUSTOM_TOKEN_INTEGRITY_REQUIRED",
                "profile",
                "Integrity verification must pass before authorization.",
            )
        )
        return _authorization("denied_by_profile", profile, diagnostics, approval_hash)

    if approval is not None:
        return _authorization("allowed", profile, diagnostics, approval_hash)

    if profile in {"research", "paper"} and allow_token and ack_risk:
        diagnostics.append(
            _diagnostic(
                "QST_V2_CUSTOM_TOKEN_RUNTIME_APPROVAL_REQUIRED",
                "profile",
                "Execution requires a persisted local ApprovalRecord.",
                severity="warning",
            )
        )
        return _authorization("requires_approval", profile, diagnostics, None)

    if profile == "pretrade" and allow_token and ack_risk:
        diagnostics.append(
            _diagnostic(
                "QST_V2_CUSTOM_TOKEN_RUNTIME_APPROVAL_REQUIRED",
                "profile",
                "Pretrade execution requires a profile-bound ApprovalRecord.",
                severity="warning",
            )
        )
        return _authorization("requires_approval", profile, diagnostics, None)

    code = (
        "QST_V2_CUSTOM_TOKEN_PRETRADE_REQUIRES_APPROVAL"
        if profile == "pretrade"
        else "QST_V2_CUSTOM_TOKEN_RUNTIME_REQUIRES_APPROVAL"
    )
    diagnostics.append(
        _diagnostic(
            code,
            "profile",
            f"Custom token execution is not authorized for profile {profile}.",
        )
    )
    status = "requires_approval" if profile in {"research", "paper", "pretrade"} else "denied_by_profile"
    return _authorization(status, profile, diagnostics, None)


def approve_token_pack(
    request: ApprovalRequest,
    *,
    approval_store: ApprovalStore | None = None,
) -> tuple[ApprovalRecord, ApprovalStore]:
    """Persist a local approval record without executing token code."""

    if not request.allow_token or not request.ack_risk:
        raise ValueError("Approval requires allow_token=True and ack_risk=True")
    material = request.model_dump(mode="json")
    approval_id = "approval_" + approval_record_hash(
        {
            "schema_version": "qst-approval-record/0.4",
            "approval_id": "pending",
            **material,
        }
    ).split(":", 1)[1][:16]
    record = ApprovalRecord(approval_id=approval_id, **material)
    store = (approval_store or ApprovalStore()).add(record)
    return record, store


def issue_execution_grant(
    integrity: TokenIntegrityResult,
    authorization: TokenAuthorizationResult,
    *,
    run_id: str,
    issued_at_utc: str,
    ttl_seconds: int = DEFAULT_EXECUTION_GRANT_TTL_SECONDS,
) -> ExecutionGrant:
    """Issue a short-lived execution grant from verified authorization."""

    if not integrity.ok or not authorization.ok or authorization.approval_record_hash is None:
        raise ValueError("ExecutionGrant requires passing integrity and approval-backed authorization")
    issued_at = _parse_utc_timestamp(issued_at_utc)
    if issued_at is None:
        raise ValueError("ExecutionGrant requires a valid UTC issued_at_utc timestamp")
    if isinstance(ttl_seconds, bool) or not isinstance(ttl_seconds, int) or ttl_seconds <= 0:
        raise ValueError("ExecutionGrant ttl_seconds must be a positive integer")
    expires_at = _format_utc_timestamp(issued_at + timedelta(seconds=ttl_seconds))
    return ExecutionGrant(
        token_ref=integrity.token_ref,
        token_spec_hash=integrity.token_spec_hash,
        token_pack_hash=integrity.token_pack_hash,
        implementation_ref_hash=integrity.implementation_ref_hash,
        runtime_environment_hash=integrity.runtime_environment_hash,
        approval_record_hash=authorization.approval_record_hash,
        profile=authorization.profile,
        issued_for_run_id=run_id,
        expires_at=expires_at,
    )


def execute_custom_token(
    pack: TokenPackManifestV2,
    token_ref: TokenRefV04,
    *,
    inputs: dict[str, Any],
    grant: ExecutionGrant,
    context: TokenRuntimeContext | None = None,
    approval_store: ApprovalStore | None = None,
) -> TokenExecutionResult:
    """Execute a custom token only with a valid grant."""

    ctx = context or TokenRuntimeContext()
    integrity = verify_integrity(pack, token_ref, context=ctx)
    authorization = check_authorization(
        integrity,
        profile=grant.profile,
        approval_store=approval_store,
    )
    grant_diagnostics = _validate_grant(
        integrity,
        authorization,
        grant,
        run_id=ctx.run_id,
        current_time_utc=ctx.current_time_utc,
    )
    spec = _find_spec(pack, token_ref)
    if spec is None:
        grant_diagnostics.append(_diagnostic("QST_V2_TOKEN_REF_NOT_FOUND", "runtime", "Token not found."))
    if grant_diagnostics:
        audit = _audit("execute", token_ref, grant.profile, "rejected", grant_diagnostics, integrity)
        return TokenExecutionResult.from_validation(
            output=None,
            validation=ValidationResult(diagnostics=grant_diagnostics),
            audit_chain_hash=audit_chain_hash_for_records([audit]),
            audit_records=[audit.model_dump(mode="json")],
        )

    assert spec is not None
    implementation_ref = _parse_implementation_ref(spec)
    if implementation_ref is None or implementation_ref.python_entrypoint is None:
        entrypoint_diagnostics = [
            _diagnostic(
                "QST_V2_CUSTOM_TOKEN_ENTRYPOINT_MISSING",
                "runtime",
                "Executable custom token requires python_entrypoint.",
            )
        ]
        audit = _audit("execute", token_ref, grant.profile, "rejected", entrypoint_diagnostics, integrity)
        return TokenExecutionResult.from_validation(
            output=None,
            validation=ValidationResult(diagnostics=entrypoint_diagnostics),
            audit_chain_hash=audit_chain_hash_for_records([audit]),
            audit_records=[audit.model_dump(mode="json")],
        )

    diagnostics: list[Diagnostic] = []
    try:
        raw_output = _call_python_entrypoint(implementation_ref, inputs, base_path=ctx.base_path)
    except Exception as exc:
        diagnostics.append(
            _diagnostic(
                "QST_V2_CUSTOM_TOKEN_EXECUTOR_EXCEPTION",
                "runtime",
                f"{type(exc).__name__}: {exc}",
            )
        )
        raw_output = None
    output = _canonical_output(raw_output, spec, diagnostics)
    audit = _audit("execute", token_ref, grant.profile, "ok" if not diagnostics else "error", diagnostics, integrity)
    return TokenExecutionResult.from_validation(
        output=output,
        validation=ValidationResult(diagnostics=diagnostics),
        audit_chain_hash=audit_chain_hash_for_records([audit]),
        audit_records=[audit.model_dump(mode="json")],
    )


def _canonical_output(
    raw_output: Any,
    spec: TokenSpecV2,
    diagnostics: list[Diagnostic],
) -> Any | None:
    if raw_output is None:
        return None
    try:
        stable_json_bytes(raw_output)
    except (TypeError, ValueError) as exc:
        diagnostics.append(
            _diagnostic(
                "QST_V2_CUSTOM_TOKEN_OUTPUT_NOT_CANONICAL_JSON",
                "runtime",
                str(exc),
            )
        )
        return None
    if not isinstance(raw_output, dict):
        diagnostics.append(
            _diagnostic(
                "QST_V2_CUSTOM_TOKEN_OUTPUT_SHAPE_INVALID",
                "runtime",
                "Custom token output must be a mapping keyed by output port.",
            )
        )
        return None
    declared = set(spec.outputs)
    actual = set(raw_output)
    extra = sorted(actual - declared)
    missing = sorted(declared - actual)
    if extra:
        diagnostics.append(
            _diagnostic(
                "QST_V2_CUSTOM_TOKEN_OUTPUT_EXTRA_PORT",
                "runtime",
                f"Undeclared output ports: {extra}.",
            )
        )
    for output_name in missing:
        diagnostics.append(
            _diagnostic(
                "QST_V2_CUSTOM_TOKEN_OUTPUT_SHAPE_INVALID",
                "runtime",
                f"Missing output port {output_name!r}.",
            )
        )
    if extra or missing:
        return None
    for output_name, output_spec in spec.outputs.items():
        _validate_value_for_type(
            raw_output[output_name],
            output_spec.type.kind,
            output_spec.type.value_type.name if output_spec.type.value_type is not None else None,
            diagnostics,
        )
    return raw_output


def _validate_value_for_type(
    value: Any,
    kind: str,
    value_type: str | None,
    diagnostics: list[Diagnostic],
) -> None:
    values = value if kind in {"TimeSeries", "Panel"} and isinstance(value, list) else [value]
    if kind in {"TimeSeries", "Panel"} and not isinstance(value, list):
        diagnostics.append(
            _diagnostic(
                "QST_V2_CUSTOM_TOKEN_OUTPUT_SHAPE_INVALID",
                "runtime",
                f"{kind} output must be a list.",
            )
        )
        return
    for item in values:
        payload = item.get("value") if kind == "Panel" and isinstance(item, dict) and "value" in item else item
        if payload is None:
            continue
        if value_type == "float" and (isinstance(payload, bool) or not isinstance(payload, int | float)):
            diagnostics.append(_diagnostic("QST_V2_CUSTOM_TOKEN_OUTPUT_TYPE_INVALID", "runtime", "Expected float."))
        if value_type == "decimal":
            if not isinstance(payload, str):
                diagnostics.append(
                    _diagnostic(
                        "QST_V2_CUSTOM_TOKEN_OUTPUT_DECIMAL_INVALID",
                        "runtime",
                        "Expected DecimalString.",
                    )
                )
            else:
                try:
                    validate_decimal_string(payload)
                except Exception as exc:
                    diagnostics.append(
                        _diagnostic(
                            "QST_V2_CUSTOM_TOKEN_OUTPUT_DECIMAL_INVALID",
                            "runtime",
                            str(exc),
                        )
                    )
        if value_type == "bool" and not isinstance(payload, bool):
            diagnostics.append(_diagnostic("QST_V2_CUSTOM_TOKEN_OUTPUT_TYPE_INVALID", "runtime", "Expected bool."))
    if kind == "Decision":
        valid = {"accept", "reject", "unknown", "block"}
        kind_value = value.get("kind") if isinstance(value, dict) else None
        if kind_value not in valid:
            diagnostics.append(
                _diagnostic(
                    "QST_V2_CUSTOM_TOKEN_OUTPUT_DECISION_INVALID",
                    "runtime",
                    "Decision output must use accept/reject/unknown/block.",
                )
            )


def _call_python_entrypoint(
    implementation_ref: ImplementationRef,
    inputs: dict[str, Any],
    *,
    base_path: Path,
) -> Any:
    assert implementation_ref.python_entrypoint is not None
    module_name, function_name = implementation_ref.python_entrypoint.split(":", 1)
    source_path = base_path / implementation_ref.path if implementation_ref.path else None
    import sys

    added_path = None
    if source_path is not None and source_path.exists():
        added_path = str(source_path)
        sys.path.insert(0, added_path)
    try:
        module = importlib.import_module(module_name)
        function = getattr(module, function_name)
        return function(inputs)
    finally:
        if added_path is not None and sys.path and sys.path[0] == added_path:
            sys.path.pop(0)


def _validate_grant(
    integrity: TokenIntegrityResult,
    authorization: TokenAuthorizationResult,
    grant: ExecutionGrant,
    *,
    run_id: str,
    current_time_utc: str | None,
) -> list[Diagnostic]:
    diagnostics: list[Diagnostic] = []
    if not integrity.ok:
        diagnostics.append(_diagnostic("QST_V2_EXECUTION_GRANT_INTEGRITY_FAILED", "runtime", "Integrity failed."))
    if not authorization.ok or authorization.approval_record_hash is None:
        diagnostics.append(
            _diagnostic("QST_V2_EXECUTION_GRANT_APPROVAL_REQUIRED", "runtime", "Approval-backed authorization required.")
        )
    checks = {
        "token_spec_hash": integrity.token_spec_hash,
        "token_pack_hash": integrity.token_pack_hash,
        "implementation_ref_hash": integrity.implementation_ref_hash,
        "runtime_environment_hash": integrity.runtime_environment_hash,
        "approval_record_hash": authorization.approval_record_hash,
    }
    for field_name, actual in checks.items():
        if getattr(grant, field_name) != actual:
            diagnostics.append(
                _diagnostic(
                    "QST_V2_EXECUTION_GRANT_HASH_MISMATCH",
                    "runtime",
                    f"ExecutionGrant {field_name} does not match current verification.",
                )
            )
    if _ref_key(grant.token_ref) != _ref_key(integrity.token_ref):
        diagnostics.append(
            _diagnostic("QST_V2_EXECUTION_GRANT_TOKEN_MISMATCH", "runtime", "Grant token_ref mismatch.")
        )
    if grant.issued_for_run_id != run_id:
        diagnostics.append(
            _diagnostic(
                "QST_V2_EXECUTION_GRANT_RUN_ID_MISMATCH",
                "runtime",
                "ExecutionGrant run id does not match the current execution context.",
            )
        )
    expires_at = _parse_utc_timestamp(grant.expires_at)
    current_time = _parse_utc_timestamp(current_time_utc) if current_time_utc is not None else None
    if expires_at is None or current_time is None:
        diagnostics.append(
            _diagnostic(
                "QST_V2_EXECUTION_GRANT_EXPIRED",
                "runtime",
                "ExecutionGrant expiry cannot be verified with the provided timestamp material.",
            )
        )
    elif expires_at <= current_time:
        diagnostics.append(
            _diagnostic(
                "QST_V2_EXECUTION_GRANT_EXPIRED",
                "runtime",
                "ExecutionGrant has expired.",
            )
        )
    return diagnostics


def _parse_utc_timestamp(value: str) -> datetime | None:
    try:
        parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        return None
    if parsed.tzinfo is None or parsed.utcoffset() != timedelta(0):
        return None
    return parsed.astimezone(UTC)


def _format_utc_timestamp(value: datetime) -> str:
    normalized = value.astimezone(UTC).replace(tzinfo=None)
    return normalized.isoformat(timespec="seconds") + "Z"


def _find_spec(pack: TokenPackManifestV2, token_ref: TokenRefV04) -> TokenSpecV2 | None:
    key = _ref_key(token_ref)
    for spec in pack.tokens:
        if spec.ref_key == key:
            return spec
    return None


def _parse_implementation_ref(spec: TokenSpecV2) -> ImplementationRef | None:
    if spec.implementation_ref is None:
        return None
    return ImplementationRef.model_validate(spec.implementation_ref)


def _parse_runtime_ref(spec: TokenSpecV2) -> RuntimeEnvironmentRef | None:
    if spec.runtime_environment_ref is None:
        return None
    return RuntimeEnvironmentRef.model_validate(spec.runtime_environment_ref)


def _risk_level(spec: TokenSpecV2) -> str:
    value = spec.risk.risk_level
    return value if value in {"low", "medium", "high", "unknown"} else "unknown"


def _authorization(
    status: str,
    profile: ProfileName,
    diagnostics: list[Diagnostic],
    approval_hash: str | None,
) -> TokenAuthorizationResult:
    return TokenAuthorizationResult(
        ok=all(diagnostic.severity != "error" for diagnostic in diagnostics) and status == "allowed",
        status=status,  # type: ignore[arg-type]
        profile=profile,
        diagnostics=diagnostics,
        approval_record_hash=approval_hash,
    )


def _approval_key(record: ApprovalRecord) -> tuple[tuple[str, str, int, int], str]:
    return (_ref_key(record.token_ref), record.profile)


def _approval_sort_key(record: ApprovalRecord) -> tuple[str, str, int, int, str]:
    namespace, name, version, behavior_version = _ref_key(record.token_ref)
    return (namespace, name, version, behavior_version, record.profile)


def _ref_key(ref: TokenRefV04) -> tuple[str, str, int, int]:
    return (ref.namespace, ref.name, ref.version, ref.behavior_version)


def _audit(
    action: str,
    token_ref: TokenRefV04,
    profile: str,
    outcome: str,
    diagnostics: list[Diagnostic],
    integrity: TokenIntegrityResult,
) -> AuditRecord:
    return AuditRecord(
        action=action,  # type: ignore[arg-type]
        token_ref=token_ref.model_dump(mode="json"),
        profile=profile,
        outcome=outcome,
        diagnostics=[diagnostic.model_dump(mode="json", exclude_none=True) for diagnostic in diagnostics],
        hashes={
            "token_spec_hash": integrity.token_spec_hash,
            "token_pack_hash": integrity.token_pack_hash,
            "implementation_ref_hash": integrity.implementation_ref_hash,
            "runtime_environment_hash": integrity.runtime_environment_hash,
        },
    )


def _diagnostic(
    code: str,
    phase: str,
    message: str,
    *,
    severity: str = "error",
) -> Diagnostic:
    return Diagnostic(
        code=code,
        severity=severity,  # type: ignore[arg-type]
        phase=phase,  # type: ignore[arg-type]
        message=message,
    )
