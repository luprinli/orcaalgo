from __future__ import annotations

from orca.ir.diagnostics import Diagnostic
from orca.models.strategy import StrategyIRV04


def trace_temporal_validation(ir: StrategyIRV04, profile: str) -> list[Diagnostic]:
    diagnostics: list[Diagnostic] = []

    for node in ir.strategy.nodes:
        if node.signature:
            for port_name, out_spec in node.signature.outputs.items():
                if out_spec.port_temporal and out_spec.temporal_rule:
                    declared = out_spec.port_temporal
                    rule = out_spec.temporal_rule

                    if rule.kind == "constant" and rule.value:
                        if isinstance(rule.value, dict) and "unsafe_future" in rule.value:
                            if rule.value.get("unsafe_future") != declared.unsafe_future:
                                diagnostics.append(
                                    Diagnostic(
                                        code="TEMPORAL_CONFLICT",
                                        severity="warning",
                                        phase="temporal",
                                        message=f"Declared temporal and rule temporal conflict on node '{node.id}', port '{port_name}'",
                                        node_id=node.id,
                                        port=port_name,
                                    ),
                                )

    return diagnostics
