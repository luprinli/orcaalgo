"""Validation shim for the qst-ir/0.4 shell."""

from __future__ import annotations

from functools import lru_cache
from typing import Any

from qst.ir.panel_validation import validate_panel_type_layer_v04
from qst.ir.schema import StrategyIRV04
from qst.ir.temporal_validation import validate_temporal_v04
from qst.profiles import ProfileName
from qst.validation import Diagnostic, ValidationResult


def validate_ir_v04(ir: StrategyIRV04, *, profile: ProfileName = "research") -> ValidationResult:
    """Return structured validation diagnostics for a qst-ir/0.4 shell.

    WP9 accepts ``core``, ``panel_type``, ``panel_ops``, ``panel_weights``, and
    ``custom_token_runtime``. Panel operator and weight capabilities must be
    explicitly paired with ``panel_type``; recipe declarations remain gated
    until their owning work package.
    """

    diagnostics: list[Diagnostic] = []
    for capability in ir.capabilities:
        if capability in {
            "core",
            "panel_type",
            "panel_ops",
            "panel_weights",
            "custom_token_runtime",
        }:
            continue
        diagnostics.append(
            Diagnostic(
                code="capability_not_accepted",
                severity="error",
                phase="profile",
                message=f"Capability {capability!r} is not accepted in the WP9 shell.",
                remediation="Remove the capability or wait for its owning work package.",
            )
        )
    if "panel_weights" in ir.capabilities and "panel_type" not in ir.capabilities:
        diagnostics.append(
            Diagnostic(
                code="QST_V2_CAPABILITY_PANEL_WEIGHTS_REQUIRES_PANEL_TYPE",
                severity="error",
                phase="profile",
                message="panel_weights must be explicitly declared together with panel_type.",
                remediation="Add panel_type or remove panel_weights.",
            )
        )
    if "panel_ops" in ir.capabilities and "panel_type" not in ir.capabilities:
        diagnostics.append(
            Diagnostic(
                code="QST_V2_CAPABILITY_PANEL_OPS_REQUIRES_PANEL_TYPE",
                severity="error",
                phase="profile",
                message="panel_ops must be explicitly declared together with panel_type.",
                remediation="Add panel_type or remove panel_ops.",
            )
        )
    diagnostics.extend(_validate_builtin_token_surface(ir, profile=profile))
    diagnostics.extend(validate_panel_type_layer_v04(ir))
    diagnostics.extend(validate_temporal_v04(ir, profile=profile).diagnostics)
    return ValidationResult(diagnostics=diagnostics)


def _validate_builtin_token_surface(
    ir: StrategyIRV04,
    *,
    profile: ProfileName,
) -> list[Diagnostic]:
    from qst.tokens import validate_token_maturity_for_profile

    diagnostics: list[Diagnostic] = []
    by_key = _builtin_token_specs_by_key()
    for node in ir.strategy.nodes:
        if node.token_ref is None:
            continue
        key = (
            node.token_ref.namespace,
            node.token_ref.name,
            node.token_ref.version,
            node.token_ref.behavior_version,
        )
        spec = by_key.get(key)
        if spec is None:
            continue
        diagnostics.extend(
            validate_token_maturity_for_profile(spec, profile=profile, node_id=node.id)
        )
    return diagnostics


@lru_cache(maxsize=1)
def _builtin_token_specs_by_key() -> dict[tuple[str, str, int, int], Any]:
    from qst.tokens import TokenRegistryV2, builtin_token_packs

    registry = TokenRegistryV2.from_packs(builtin_token_packs())
    if not registry.result.ok:
        return {}
    return {record.spec.ref_key: record.spec for record in registry.records}
