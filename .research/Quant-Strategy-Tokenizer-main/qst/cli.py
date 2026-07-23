"""Cleanline QST command line interface."""

from __future__ import annotations

import json
from dataclasses import asdict
from pathlib import Path
from typing import Annotated, Any

import typer

from qst.adapters.qlib.cli import app as qlib_adapter_app
from qst.canonical_json import stable_json_bytes
from qst.custom_runtime import (
    ApprovalRequest,
    ApprovalStore,
    TokenRuntimeContext,
    TokenRuntimeService,
    load_token_pack,
)
from qst.hash import compute_hashes_v2
from qst.ir import (
    TokenRefV04,
    canonical_bytes_v04,
    is_gkr_source,
    load_ir_v04_file,
    validate_ir_v04,
)
from qst.tokens import TokenRegistryV2, builtin_token_packs
from qst.validation import Diagnostic

app = typer.Typer(no_args_is_help=True)
token_app = typer.Typer(no_args_is_help=True)
approval_app = typer.Typer(no_args_is_help=True)
adapter_app = typer.Typer(no_args_is_help=True)
app.add_typer(token_app, name="token")
token_app.add_typer(approval_app, name="approvals")
app.add_typer(adapter_app, name="adapter")
adapter_app.add_typer(qlib_adapter_app, name="qlib")


def _jsonable(value: Any) -> Any:
    if hasattr(value, "model_dump"):
        return value.model_dump(mode="json")
    if hasattr(value, "__dataclass_fields__"):
        return asdict(value)
    if isinstance(value, dict):
        return {str(key): _jsonable(item) for key, item in value.items()}
    if isinstance(value, (list, tuple)):
        return [_jsonable(item) for item in value]
    return value


def _echo_json(value: Any) -> None:
    typer.echo(json.dumps(_jsonable(value), ensure_ascii=False, indent=2, sort_keys=True))


def _parse_token_ref(value: str) -> TokenRefV04:
    try:
        token_part, version_part, behavior_part = value.split("/")
        namespace, name = token_part.split(".", 1)
        version = int(version_part.removeprefix("v"))
        behavior_version = int(behavior_part.removeprefix("bv"))
    except Exception as exc:
        raise typer.BadParameter("token ref must be formatted as namespace.name/v1/bv1") from exc
    return TokenRefV04(
        namespace=namespace,
        name=name,
        version=version,
        behavior_version=behavior_version,
    )


def _approval_store_path(path: Path | None) -> Path:
    return path or Path(".qst") / "approvals.json"


def _load_approval_store(path: Path | None) -> tuple[Path, ApprovalStore]:
    resolved = _approval_store_path(path)
    return resolved, ApprovalStore.load(resolved)


def _load_inputs(inputs_json: str | None, inputs_file: Path | None) -> dict[str, Any]:
    if inputs_file is not None:
        raw = json.loads(inputs_file.read_text(encoding="utf-8-sig"))
    elif inputs_json is not None:
        raw = json.loads(inputs_json)
    else:
        raw = {}
    if not isinstance(raw, dict):
        raise typer.BadParameter("custom token inputs must be a JSON object")
    return raw


def _load_gkr_source(path: Path) -> Any:
    if not is_gkr_source(path):
        suffix = "".join(path.suffixes) or path.suffix
        pre_gkr_source = ".qst" + ".yaml"
        pre_gkr_package = ".qst" + "pkg"
        pre_gkr_single_file = ".q" + "sp"
        if suffix in {pre_gkr_source, pre_gkr_package, pre_gkr_single_file}:
            raise typer.BadParameter(
                "Pre-GKR source/package extensions are not supported. "
                "Use .gkr.yaml for editable GKR source documents."
            )
        raise typer.BadParameter("strategy source must be an editable .gkr.yaml document")
    return load_ir_v04_file(path)


def _core_packs() -> tuple[Any, ...]:
    return builtin_token_packs()


def _vocabulary_surface_diagnostics(packs: tuple[Any, ...]) -> list[Diagnostic]:
    diagnostics: list[Diagnostic] = []
    for pack in packs:
        for spec in pack.tokens:
            surface = spec.surface
            if surface.maturity == "reserved_design":
                if surface.execution_support != "metadata_only":
                    diagnostics.append(
                        Diagnostic(
                            code="QST_TOKEN_RESERVED_DESIGN_EXECUTION_SUPPORT_INVALID",
                            severity="error",
                            phase="token_registry",
                            message=f"{spec.token_id} reserved_design must be metadata_only.",
                        )
                    )
                if not surface.capabilities.reserved_only:
                    diagnostics.append(
                        Diagnostic(
                            code="QST_TOKEN_RESERVED_DESIGN_FLAG_MISSING",
                            severity="error",
                            phase="token_registry",
                            message=f"{spec.token_id} reserved_design must set reserved_only=true.",
                        )
                    )
                if surface.capabilities.deterministic_level != "reserved":
                    diagnostics.append(
                        Diagnostic(
                            code="QST_TOKEN_RESERVED_DESIGN_DETERMINISM_INVALID",
                            severity="error",
                            phase="token_registry",
                            message=f"{spec.token_id} reserved_design must use deterministic_level=reserved.",
                        )
                    )
                if surface.contract.scope != "validation_only":
                    diagnostics.append(
                        Diagnostic(
                            code="QST_TOKEN_RESERVED_DESIGN_SCOPE_INVALID",
                            severity="error",
                            phase="token_registry",
                            message=f"{spec.token_id} reserved_design must use validation_only contract scope.",
                        )
                    )
            elif surface.capabilities.reserved_only:
                diagnostics.append(
                    Diagnostic(
                        code="QST_TOKEN_RESERVED_ONLY_MATURITY_INVALID",
                        severity="error",
                        phase="token_registry",
                        message=f"{spec.token_id} sets reserved_only without reserved_design maturity.",
                    )
                )
    return diagnostics


@app.command("vocabulary")
def vocabulary(check: Annotated[bool, typer.Option("--check")] = False) -> None:
    """Print or validate the built-in TokenPack vocabulary."""

    packs = _core_packs()
    registry = TokenRegistryV2.from_packs(packs)
    surface_diagnostics = _vocabulary_surface_diagnostics(packs)
    diagnostics = [*registry.result.diagnostics, *surface_diagnostics]
    ok = registry.result.ok and not any(diagnostic.severity == "error" for diagnostic in diagnostics)
    if check:
        _echo_json(
            {
                "ok": ok,
                "packs": [pack.pack_id for pack in packs],
                "token_count": len(registry.records),
                "diagnostics": diagnostics,
            }
        )
        if not ok:
            raise typer.Exit(1)
        return
    _echo_json(
        [
            {
                "pack_id": pack.pack_id,
                "version": pack.version,
                "tokens": [token.token_id for token in pack.tokens],
            }
            for pack in packs
        ]
    )


@app.command("validate")
def validate(strategy: Path) -> None:
    """Validate a current QST strategy document."""

    ir = _load_gkr_source(strategy)
    result = validate_ir_v04(ir)
    _echo_json(result)
    if not result.ok:
        raise typer.Exit(1)


@app.command("hash")
def hash_strategy(strategy: Path) -> None:
    """Compute current QST strategy graph/param/instance hashes."""

    ir = _load_gkr_source(strategy)
    _echo_json(compute_hashes_v2(ir))


@app.command("canonicalize")
def canonicalize(
    strategy: Path,
    output: Annotated[Path | None, typer.Option("--output")] = None,
) -> None:
    """Emit canonical JSON for a current QST strategy."""

    payload = canonical_bytes_v04(_load_gkr_source(strategy))
    if output is None:
        typer.echo(payload.decode("utf-8"))
    else:
        output.parent.mkdir(parents=True, exist_ok=True)
        output.write_bytes(payload)


@token_app.command("verify")
def token_verify(
    pack: Path,
    token_ref: Annotated[str, typer.Option("--token-ref")],
    profile: Annotated[str, typer.Option("--profile")] = "research",
    approvals: Annotated[Path | None, typer.Option("--approvals")] = None,
    base_path: Annotated[Path | None, typer.Option("--base-path")] = None,
    current_time_utc: Annotated[str | None, typer.Option("--current-time-utc")] = None,
) -> None:
    """Verify custom-token integrity and local authorization separately."""

    token_pack = load_token_pack(pack)
    ref = _parse_token_ref(token_ref)
    _, store = _load_approval_store(approvals)
    service = TokenRuntimeService()
    context = TokenRuntimeContext(
        base_path=base_path or pack.parent,
        profile=profile,  # type: ignore[arg-type]
        current_time_utc=current_time_utc,
    )
    integrity = service.verify_integrity(token_pack, ref, context=context)
    authorization = service.check_authorization(
        integrity,
        profile=profile,  # type: ignore[arg-type]
        approval_store=store,
    )
    _echo_json({"integrity": integrity, "authorization": authorization})
    if not integrity.ok or not authorization.ok:
        raise typer.Exit(1)


@token_app.command("approve")
def token_approve(
    pack: Path,
    token_ref: Annotated[str, typer.Option("--token-ref")],
    approved_by: Annotated[str, typer.Option("--approved-by")],
    profile: Annotated[str, typer.Option("--profile")] = "research",
    approvals: Annotated[Path | None, typer.Option("--approvals")] = None,
    base_path: Annotated[Path | None, typer.Option("--base-path")] = None,
    allow_token: Annotated[bool, typer.Option("--allow-token")] = False,
    ack_risk: Annotated[bool, typer.Option("--ack-risk")] = False,
) -> None:
    """Write a local approval for a verified custom token."""

    token_pack = load_token_pack(pack)
    ref = _parse_token_ref(token_ref)
    store_path, store = _load_approval_store(approvals)
    service = TokenRuntimeService()
    integrity = service.verify_integrity(
        token_pack,
        ref,
        context=TokenRuntimeContext(base_path=base_path or pack.parent, profile=profile),  # type: ignore[arg-type]
    )
    if not integrity.ok:
        _echo_json({"integrity": integrity})
        raise typer.Exit(1)
    request = ApprovalRequest(
        token_ref=ref,
        profile=profile,  # type: ignore[arg-type]
        approved_by=approved_by,
        allow_token=allow_token,
        ack_risk=ack_risk,
        approved_risk_level=integrity.risk_level,
        token_spec_hash=integrity.token_spec_hash,
        token_pack_hash=integrity.token_pack_hash,
        implementation_ref_hash=integrity.implementation_ref_hash,
        runtime_environment_hash=integrity.runtime_environment_hash,
    )
    record, updated = service.approve_token_pack(request, approval_store=store)
    updated.save(store_path)
    _echo_json({"approval": record, "store": str(store_path)})


@approval_app.command("list")
def approvals_list(approvals: Annotated[Path | None, typer.Option("--approvals")] = None) -> None:
    """List local custom-token approvals."""

    path, store = _load_approval_store(approvals)
    _echo_json({"path": str(path), "records": store.records})


@approval_app.command("revoke")
def approvals_revoke(
    token_ref: Annotated[str, typer.Option("--token-ref")],
    profile: Annotated[str | None, typer.Option("--profile")] = None,
    approvals: Annotated[Path | None, typer.Option("--approvals")] = None,
) -> None:
    """Revoke local approvals for a token ref."""

    path, store = _load_approval_store(approvals)
    updated = store.revoke(_parse_token_ref(token_ref), profile=profile)  # type: ignore[arg-type]
    updated.save(path)
    _echo_json({"path": str(path), "records": updated.records})


@token_app.command("execute")
def token_execute(
    pack: Path,
    token_ref: Annotated[str, typer.Option("--token-ref")],
    current_time_utc: Annotated[str, typer.Option("--current-time-utc")],
    run_id: Annotated[str, typer.Option("--run-id")] = "manual",
    profile: Annotated[str, typer.Option("--profile")] = "research",
    approvals: Annotated[Path | None, typer.Option("--approvals")] = None,
    base_path: Annotated[Path | None, typer.Option("--base-path")] = None,
    inputs_json: Annotated[str | None, typer.Option("--inputs-json")] = None,
    inputs_file: Annotated[Path | None, typer.Option("--inputs-file")] = None,
    ttl_seconds: Annotated[int, typer.Option("--ttl-seconds")] = 900,
) -> None:
    """Execute an approved custom token with a short-lived grant."""

    token_pack = load_token_pack(pack)
    ref = _parse_token_ref(token_ref)
    _, store = _load_approval_store(approvals)
    service = TokenRuntimeService()
    context = TokenRuntimeContext(
        base_path=base_path or pack.parent,
        profile=profile,  # type: ignore[arg-type]
        run_id=run_id,
        current_time_utc=current_time_utc,
    )
    integrity = service.verify_integrity(token_pack, ref, context=context)
    authorization = service.check_authorization(
        integrity,
        profile=profile,  # type: ignore[arg-type]
        approval_store=store,
    )
    if not integrity.ok or not authorization.ok:
        _echo_json({"integrity": integrity, "authorization": authorization})
        raise typer.Exit(1)
    grant = service.issue_execution_grant(
        integrity,
        authorization,
        run_id=run_id,
        issued_at_utc=current_time_utc,
        ttl_seconds=ttl_seconds,
    )
    result = service.execute_custom_token(
        token_pack,
        ref,
        inputs=_load_inputs(inputs_json, inputs_file),
        grant=grant,
        context=context,
        approval_store=store,
    )
    _echo_json(result)
    if not result.ok:
        raise typer.Exit(1)


@app.command("write-json")
def write_json(
    input_file: Path,
    output: Path,
) -> None:
    """Canonicalize an arbitrary JSON file with QST stable JSON rules."""

    output.write_bytes(stable_json_bytes(json.loads(input_file.read_text(encoding="utf-8"))))


if __name__ == "__main__":
    app()
