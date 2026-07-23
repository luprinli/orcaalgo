"""Temporal validation for qst-ir/0.4."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Literal

from qst.hash import expected_artifact_hash_v2
from qst.ir.schema import NodeV04, StrategyIRV04
from qst.ports import (
    InputSpec,
    PortTemporalSpec,
    TemporalRuleResolutionError,
    resolve_temporal_rule,
    temporal_is_later,
)
from qst.profiles import ProfileName
from qst.validation import Diagnostic, ValidationResult

TemporalProfile = ProfileName
_STRICT_PROFILES: set[TemporalProfile] = {"pretrade", "production_guarded"}


@dataclass(frozen=True)
class TemporalValidationTrace:
    """Deterministic trace of static temporal validation."""

    profile: TemporalProfile
    strategy_id: str
    diagnostics: list[Diagnostic]
    resolved_outputs: dict[str, PortTemporalSpec]

    def to_artifact(self) -> dict[str, object]:
        material: dict[str, object] = {
            "artifact_version": "qst-v04-temporal-validation-trace/0.1",
            "profile": self.profile,
            "strategy": self.strategy_id,
            "diagnostics": [item.model_dump(mode="json", exclude_none=True) for item in self.diagnostics],
            "resolved_outputs": {
                key: value.model_dump(mode="json") for key, value in sorted(self.resolved_outputs.items())
            },
        }
        return {
            **material,
            "expected_artifact_hash": expected_artifact_hash_v2(material),
        }


def validate_temporal_v04(ir: StrategyIRV04, *, profile: TemporalProfile = "research") -> ValidationResult:
    """Validate qst-ir/0.4 temporal declarations."""

    trace = trace_temporal_validation_v04(ir, profile=profile)
    return ValidationResult(diagnostics=trace.diagnostics)


def trace_temporal_validation_v04(
    ir: StrategyIRV04,
    *,
    profile: TemporalProfile = "research",
) -> TemporalValidationTrace:
    """Resolve temporal declarations and retain deterministic evidence."""

    diagnostics: list[Diagnostic] = []
    resolved_outputs: dict[str, PortTemporalSpec] = {}
    nodes = {node.id: node for node in ir.strategy.nodes}

    for node in ir.strategy.nodes:
        input_temporals = _resolve_input_temporals(node, nodes=nodes, resolved_outputs=resolved_outputs)
        diagnostics.extend(_validate_input_requirements(node, input_temporals=input_temporals))

        for port_name, output in sorted(node.signature.outputs.items()):
            resolved = output.port_temporal
            if output.temporal_rule is not None:
                try:
                    rule_temporal = resolve_temporal_rule(
                        output.temporal_rule,
                        inputs=input_temporals,
                        params=node.params,
                    )
                except TemporalRuleResolutionError as exc:
                    diagnostics.append(
                        _diagnostic(
                            code="QST_V2_TEMPORAL_RULE_UNRESOLVED",
                            severity="error",
                            message=str(exc),
                            node=node,
                            port=port_name,
                            profile=profile,
                        )
                    )
                    continue

                if resolved is not None and resolved != rule_temporal:
                    diagnostics.append(
                        _diagnostic(
                            code="QST_V2_TEMPORAL_CONFLICT",
                            severity="error",
                            message="Output port_temporal conflicts with resolved temporal_rule.",
                            node=node,
                            port=port_name,
                            profile=profile,
                        )
                    )
                resolved = resolved or rule_temporal

            if resolved is None:
                resolved = PortTemporalSpec()

            resolved_outputs[_output_ref(node.id, port_name)] = resolved
            if resolved.unsafe_future:
                diagnostics.append(
                    _diagnostic(
                        code="QST_V2_TEMPORAL_UNSAFE_FUTURE",
                        severity=_unsafe_future_severity(profile),
                        message="Output declares unsafe future data.",
                        node=node,
                        port=port_name,
                        profile=profile,
                    )
                )

    return TemporalValidationTrace(
        profile=profile,
        strategy_id=ir.strategy.id,
        diagnostics=diagnostics,
        resolved_outputs=resolved_outputs,
    )


def _resolve_input_temporals(
    node: NodeV04,
    *,
    nodes: dict[str, NodeV04],
    resolved_outputs: dict[str, PortTemporalSpec],
) -> dict[str, PortTemporalSpec]:
    input_temporals: dict[str, PortTemporalSpec] = {}
    for input_name, spec in sorted(node.signature.inputs.items()):
        value = node.inputs.get(input_name)
        resolved = _resolve_ref_temporal(value, resolved_outputs)
        if resolved is None:
            resolved = _temporal_from_input_spec(spec)
        input_temporals[input_name] = resolved

    _ = nodes
    return input_temporals


def _validate_input_requirements(
    node: NodeV04,
    *,
    input_temporals: dict[str, PortTemporalSpec],
) -> list[Diagnostic]:
    diagnostics: list[Diagnostic] = []
    for input_name, input_spec in sorted(node.signature.inputs.items()):
        requirement = input_spec.temporal_requirement
        if requirement is None:
            continue
        resolved = input_temporals[input_name]
        if resolved.unsafe_future and not requirement.allow_unsafe_future:
            diagnostics.append(
                _diagnostic(
                    code="QST_V2_TEMPORAL_REQUIREMENT_UNSATISFIED",
                    severity="error",
                    message="Input requires safe temporal data but upstream is unsafe future.",
                    node=node,
                    port=input_name,
                    profile=None,
                )
            )
        if temporal_is_later(resolved, requirement.max_available_at):
            diagnostics.append(
                _diagnostic(
                    code="QST_V2_TEMPORAL_REQUIREMENT_UNSATISFIED",
                    severity="error",
                    message=(
                        f"Input available_at={resolved.available_at} exceeds "
                        f"max_available_at={requirement.max_available_at}."
                    ),
                    node=node,
                    port=input_name,
                    profile=None,
                )
            )
    return diagnostics


def _resolve_ref_temporal(
    value: object,
    resolved_outputs: dict[str, PortTemporalSpec],
) -> PortTemporalSpec | None:
    if not isinstance(value, str):
        return None
    return resolved_outputs.get(value)


def _temporal_from_input_spec(input_spec: InputSpec) -> PortTemporalSpec:
    intrinsic = input_spec.type.intrinsic_temporal
    if intrinsic is None:
        return PortTemporalSpec()
    return PortTemporalSpec(available_at=intrinsic.default_available_at)


def _unsafe_future_severity(profile: TemporalProfile) -> Literal["warning", "error"]:
    return "error" if profile in _STRICT_PROFILES else "warning"


def _output_ref(node_id: str, port_name: str) -> str:
    return f"{node_id}.{port_name}"


def _diagnostic(
    *,
    code: str,
    severity: Literal["warning", "error"],
    message: str,
    node: NodeV04,
    port: str,
    profile: TemporalProfile | None,
) -> Diagnostic:
    return Diagnostic(
        code=code,
        severity=severity,
        phase="temporal",
        message=message,
        profile=profile,
        node_id=node.id,
        port=port,
        remediation="Adjust port_temporal, temporal_rule, or profile.",
    )
