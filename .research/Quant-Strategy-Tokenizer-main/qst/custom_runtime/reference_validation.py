"""custom-token-reference custom-token reference case for WP9."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict

from qst.custom_runtime import (
    ApprovalRequest,
    ApprovalStore,
    TokenRuntimeContext,
    TokenRuntimeService,
    load_token_pack,
)
from qst.hash import expected_artifact_hash_v2
from qst.ir import StrategyIRV04, TokenRefV04, validate_ir_v04
from qst.validation import Diagnostic, ValidationResult

CUSTOM_PV_D_FIXTURE_VERSION: Literal["qst-v04-custom-token-fixture/0.1"] = (
    "qst-v04-custom-token-fixture/0.1"
)
CUSTOM_PV_D_TRACE_ARTIFACT_VERSION: Literal["qst-v04-custom-token-validation-trace/0.1"] = (
    "qst-v04-custom-token-validation-trace/0.1"
)
CUSTOM_PV_D_DIAGNOSTICS_ARTIFACT_VERSION: Literal[
    "qst-v04-custom-token-expected-diagnostics/0.1"
] = "qst-v04-custom-token-expected-diagnostics/0.1"


class CustomTokenPVDFixture(BaseModel):
    """Fixture payload for the custom-token custom-token-reference case."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    artifact_version: Literal["qst-v04-custom-token-fixture/0.1"] = CUSTOM_PV_D_FIXTURE_VERSION
    case: Literal["custom_token_kalman"] = "custom_token_kalman"
    token_pack_path: str
    token_ref: TokenRefV04
    inputs: dict[str, Any]
    profile: Literal["research", "pretrade"] = "research"
    approve: bool = False


class CustomTokenPVDTraceArtifact(BaseModel):
    """Serializable custom-token-reference trace artifact."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    artifact_version: Literal["qst-v04-custom-token-validation-trace/0.1"] = (
        CUSTOM_PV_D_TRACE_ARTIFACT_VERSION
    )
    strategy: str
    case: Literal["custom_token_kalman"]
    profile: str
    integrity: dict[str, Any]
    authorization: dict[str, Any]
    output: Any | None
    audit_chain_hash: str
    audit_records: list[dict[str, Any]]
    expected_artifact_hash: str


def load_custom_pv_d_fixture(path: str | Path) -> CustomTokenPVDFixture:
    """Load a custom-token-reference fixture from JSON."""

    with Path(path).open(encoding="utf-8") as handle:
        loaded = json.load(handle)
    return CustomTokenPVDFixture.model_validate(loaded)


def trace_custom_pv_d_v04(
    ir: StrategyIRV04,
    fixture: CustomTokenPVDFixture,
    *,
    base_path: str | Path = ".",
) -> CustomTokenPVDTraceArtifact:
    """Run the deterministic custom-token-reference reference path."""

    result = run_custom_pv_d_v04(ir, fixture, base_path=base_path)
    artifact = CustomTokenPVDTraceArtifact(
        strategy=ir.strategy.id,
        case=fixture.case,
        profile=fixture.profile,
        integrity=result["integrity"],
        authorization=result["authorization"],
        output=result["output"],
        audit_chain_hash=result["audit_chain_hash"],
        audit_records=result["audit_records"],
        expected_artifact_hash="sha256:" + "0" * 64,
    )
    payload = artifact.model_dump(mode="json")
    payload_without_hash = {key: value for key, value in payload.items() if key != "expected_artifact_hash"}
    return artifact.model_copy(
        update={"expected_artifact_hash": expected_artifact_hash_v2(payload_without_hash)}
    )


def diagnostics_custom_pv_d_v04(
    ir: StrategyIRV04,
    fixture: CustomTokenPVDFixture,
    *,
    base_path: str | Path = ".",
) -> dict[str, Any]:
    """Return deterministic expected diagnostics artifact, including empty diagnostics."""

    result = run_custom_pv_d_v04(ir, fixture, base_path=base_path)
    diagnostics = result["diagnostics"]
    payload: dict[str, Any] = {
        "artifact_version": CUSTOM_PV_D_DIAGNOSTICS_ARTIFACT_VERSION,
        "strategy": ir.strategy.id,
        "case": fixture.case,
        "profile": fixture.profile,
        "diagnostics": diagnostics,
    }
    payload["expected_artifact_hash"] = expected_artifact_hash_v2(payload)
    return payload


def run_custom_pv_d_v04(
    ir: StrategyIRV04,
    fixture: CustomTokenPVDFixture,
    *,
    base_path: str | Path = ".",
) -> dict[str, Any]:
    """Run custom-token-reference with validate_ir_v04 before custom helper execution."""

    validation = validate_ir_v04(ir)
    if not validation.ok:
        return _error_result(ir, fixture, validation.errors)
    if ir.metadata.get("reference_case") != fixture.case:
        return _error_result(
            ir,
            fixture,
            [
                Diagnostic(
                    code="QST_V2_CUSTOM_PVD_CASE_MISMATCH",
                    severity="error",
                    phase="runtime",
                    message="Strategy and custom-token fixture case mismatch.",
                )
            ],
        )
    expected_refs = {
        (
            node.token_ref.namespace,
            node.token_ref.name,
            node.token_ref.version,
            node.token_ref.behavior_version,
        )
        for node in ir.strategy.nodes
        if node.token_ref is not None
    }
    fixture_ref = (
        fixture.token_ref.namespace,
        fixture.token_ref.name,
        fixture.token_ref.version,
        fixture.token_ref.behavior_version,
    )
    if fixture_ref not in expected_refs:
        return _error_result(
            ir,
            fixture,
            [
                Diagnostic(
                    code="QST_V2_CUSTOM_PVD_TOKEN_REF_MISMATCH",
                    severity="error",
                    phase="runtime",
                    message="Strategy does not reference the fixture custom token.",
                )
            ],
        )

    root = Path(base_path)
    pack_path = root / fixture.token_pack_path
    pack = load_token_pack(pack_path)
    service = TokenRuntimeService()
    current_time_utc = "2026-05-15T00:00:00Z"
    context = TokenRuntimeContext(
        base_path=pack_path,
        profile=fixture.profile,
        run_id="pv-d",
        current_time_utc=current_time_utc,
    )
    integrity = service.verify_integrity(pack, fixture.token_ref, context=context)
    store = ApprovalStore()
    if fixture.approve:
        request = ApprovalRequest(
            token_ref=fixture.token_ref,
            profile=fixture.profile,
            approved_by="pv-d",
            allow_token=True,
            ack_risk=True,
            approved_risk_level=integrity.risk_level,
            token_spec_hash=integrity.token_spec_hash,
            token_pack_hash=integrity.token_pack_hash,
            implementation_ref_hash=integrity.implementation_ref_hash,
            runtime_environment_hash=integrity.runtime_environment_hash,
        )
        _, store = service.approve_token_pack(request, approval_store=store)
    authorization = service.check_authorization(
        integrity,
        profile=fixture.profile,
        approval_store=store,
        allow_token=fixture.approve,
        ack_risk=fixture.approve,
    )
    execution = None
    if authorization.ok:
        grant = service.issue_execution_grant(
            integrity,
            authorization,
            run_id="pv-d",
            issued_at_utc=current_time_utc,
        )
        execution = service.execute_custom_token(
            pack,
            fixture.token_ref,
            inputs=fixture.inputs,
            grant=grant,
            context=context,
            approval_store=store,
        )
    diagnostics = [
        *[diagnostic.model_dump(mode="json") for diagnostic in integrity.diagnostics],
        *[diagnostic.model_dump(mode="json") for diagnostic in authorization.diagnostics],
    ]
    if execution is not None:
        diagnostics.extend(diagnostic.model_dump(mode="json") for diagnostic in execution.diagnostics)
    return {
        "integrity": integrity.model_dump(mode="json"),
        "authorization": authorization.model_dump(mode="json"),
        "output": execution.output if execution is not None else None,
        "audit_chain_hash": execution.audit_chain_hash if execution is not None else integrity.audit_chain_hash,
        "audit_records": execution.audit_records if execution is not None else [],
        "diagnostics": diagnostics,
    }


def _error_result(
    ir: StrategyIRV04,
    fixture: CustomTokenPVDFixture,
    diagnostics: list[Diagnostic],
) -> dict[str, Any]:
    validation = ValidationResult(diagnostics=diagnostics)
    return {
        "integrity": {},
        "authorization": {"ok": False, "status": "denied_by_profile", "profile": fixture.profile},
        "output": None,
        "audit_chain_hash": expected_artifact_hash_v2({"strategy": ir.strategy.id, "diagnostics": []}),
        "audit_records": [],
        "diagnostics": [diagnostic.model_dump(mode="json") for diagnostic in validation.diagnostics],
    }
