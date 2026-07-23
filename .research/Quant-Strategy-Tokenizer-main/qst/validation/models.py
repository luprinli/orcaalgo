"""Serializable validation diagnostics for Token System v2."""

from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, model_validator

Severity = Literal["info", "warning", "error"]
ValidationPhase = Literal[
    "schema",
    "profile",
    "signature",
    "temporal",
    "token_registry",
    "package",
    "runtime",
]


class Diagnostic(BaseModel):
    """A structured validation diagnostic."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    code: str = Field(min_length=1)
    severity: Severity
    phase: ValidationPhase
    message: str = Field(min_length=1)
    profile: str | None = None
    node_id: str | None = None
    port: str | None = None
    remediation: str | None = None


class ValidationResult(BaseModel):
    """Validation result where ``ok`` only depends on error diagnostics."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    ok: bool = True
    diagnostics: list[Diagnostic] = Field(default_factory=list)

    @model_validator(mode="before")
    @classmethod
    def _derive_ok(cls, value: Any) -> Any:
        if not isinstance(value, dict):
            return value
        diagnostics = value.get("diagnostics", [])
        value = dict(value)
        value["ok"] = all(_severity(diagnostic) != "error" for diagnostic in diagnostics)
        return value


    @property
    def errors(self) -> list[Diagnostic]:
        """Error diagnostics."""

        return [diagnostic for diagnostic in self.diagnostics if diagnostic.severity == "error"]

    @property
    def warnings(self) -> list[Diagnostic]:
        """Warning diagnostics."""

        return [diagnostic for diagnostic in self.diagnostics if diagnostic.severity == "warning"]


def _severity(diagnostic: Diagnostic | dict[str, Any]) -> str | None:
    if isinstance(diagnostic, Diagnostic):
        return diagnostic.severity
    return diagnostic.get("severity")
