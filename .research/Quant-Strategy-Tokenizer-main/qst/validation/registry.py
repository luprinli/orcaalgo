"""Deterministic validator registry for Token System v2."""

from __future__ import annotations

from collections.abc import Callable
from typing import Any, TypeAlias

from qst.validation.models import Diagnostic, ValidationResult

Validator: TypeAlias = Callable[[Any], list[Diagnostic]]


class ValidatorRegistry:
    """Run registered validators in deterministic insertion order."""

    def __init__(self) -> None:
        self._validators: list[tuple[str, Validator]] = []

    def register(self, name: str, validator: Validator) -> None:
        """Register a validator under a unique name."""

        if any(existing == name for existing, _validator in self._validators):
            raise ValueError(f"Duplicate v2 validator: {name}")
        self._validators.append((name, validator))

    def names(self) -> list[str]:
        """Registered validator names in run order."""

        return [name for name, _validator in self._validators]

    def run(self, context: Any) -> ValidationResult:
        """Run all validators and aggregate diagnostics."""

        diagnostics: list[Diagnostic] = []
        for _name, validator in self._validators:
            diagnostics.extend(validator(context))
        return ValidationResult(diagnostics=diagnostics)
