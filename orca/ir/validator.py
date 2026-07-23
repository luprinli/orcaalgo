from __future__ import annotations

from orca.ir.diagnostics import Diagnostic
from orca.models.strategy import StrategyIRV04

VALID_PROFILES = {"research", "paper", "pretrade", "production_guarded"}

PROFILE_RULES: dict[str, dict[str, bool]] = {
    "research": {"unsafe_future": True, "external_read": True, "external_write": True, "unknown_numeric": True},
    "paper": {"unsafe_future": True, "external_read": True, "external_write": False, "unknown_numeric": True},
    "pretrade": {"unsafe_future": False, "external_read": False, "external_write": False, "unknown_numeric": False},
    "production_guarded": {"unsafe_future": False, "external_read": False, "external_write": False, "unknown_numeric": False},
}


def validate_ir(ir: StrategyIRV04, profile: str = "research") -> list[Diagnostic]:
    diagnostics: list[Diagnostic] = []

    if profile not in VALID_PROFILES:
        return [Diagnostic(code="INVALID_PROFILE", severity="error", phase="validation", message=f"Unknown profile: {profile}")]

    if ir.ir_version != "qst-ir/0.4":
        diagnostics.append(
            Diagnostic(code="IR_VERSION_MISMATCH", severity="error", phase="validation", message=f"Expected qst-ir/0.4, got {ir.ir_version}")
        )

    has_core = any(c.name == "core" for c in ir.capabilities)
    if not has_core:
        diagnostics.append(
            Diagnostic(code="MISSING_CORE_CAPABILITY", severity="error", phase="capability_gate", message="Strategy must declare 'core' capability")
        )

    node_ids = {n.id for n in ir.strategy.nodes}
    if len(node_ids) < len(ir.strategy.nodes):
        diagnostics.append(
            Diagnostic(code="DUPLICATE_NODE_ID", severity="error", phase="validation", message="Duplicate node IDs found in strategy"),
        )

    for node in ir.strategy.nodes:
        for inp_port, inp_ref in node.inputs.items():
            parts = inp_ref.split(".")
            if len(parts) == 2:
                src_node, src_port = parts
                if src_node not in node_ids:
                    diagnostics.append(
                        Diagnostic(
                            code="MISSING_INPUT_NODE",
                            severity="error",
                            phase="validation",
                            message=f"Node '{node.id}' references missing upstream node '{src_node}'",
                            node_id=node.id,
                            port=inp_port,
                        ),
                    )

    output_refs = set(ir.strategy.outputs.values())
    for ref in output_refs:
        parts = ref.split(".")
        src_node = parts[0] if len(parts) >= 1 else ref
        if src_node not in node_ids:
            diagnostics.append(
                Diagnostic(code="INVALID_OUTPUT_REF", severity="error", phase="validation", message=f"Output reference '{ref}' does not match any node"),
            )

    rules = PROFILE_RULES.get(profile, PROFILE_RULES["research"])
    if not rules["unsafe_future"]:
        for node in ir.strategy.nodes:
            if node.signature:
                for out_spec in node.signature.outputs.values():
                    if out_spec.port_temporal and out_spec.port_temporal.unsafe_future:
                        diagnostics.append(
                            Diagnostic(
                                code="UNSAFE_FUTURE",
                                severity="error",
                                phase="temporal",
                                message=f"Node '{node.id}' uses unsafe future data, prohibited in profile '{profile}'",
                                node_id=node.id,
                                profile=profile,
                            ),
                        )

    if not rules["external_read"]:
        for node in ir.strategy.nodes:
            if node.signature and getattr(node.signature, "external_read", False):
                diagnostics.append(
                    Diagnostic(
                        code="EXTERNAL_READ",
                        severity="error",
                        phase="validation",
                        message=f"Node '{node.id}' performs external read, prohibited in profile '{profile}'",
                        node_id=node.id,
                        profile=profile,
                    ),
                )

    if not rules["external_write"]:
        for node in ir.strategy.nodes:
            if node.signature and getattr(node.signature, "external_write", False):
                diagnostics.append(
                    Diagnostic(
                        code="EXTERNAL_WRITE",
                        severity="error",
                        phase="validation",
                        message=f"Node '{node.id}' performs external write, prohibited in profile '{profile}'",
                        node_id=node.id,
                        profile=profile,
                    ),
                )

    if not rules["unknown_numeric"]:
        import math
        for node in ir.strategy.nodes:
            for key, value in node.params.items():
                if isinstance(value, float) and (math.isnan(value) or math.isinf(value)):
                    diagnostics.append(
                        Diagnostic(
                            code="UNKNOWN_NUMERIC",
                            severity="error",
                            phase="validation",
                            message=f"Node '{node.id}' param '{key}' has unrecognized numeric value: {value}",
                            node_id=node.id,
                            profile=profile,
                        ),
                    )

    return diagnostics
