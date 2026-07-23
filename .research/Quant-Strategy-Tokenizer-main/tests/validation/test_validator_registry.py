from __future__ import annotations

import pytest

from qst.validation import Diagnostic, ValidatorRegistry


def _diagnostic(code: str) -> Diagnostic:
    return Diagnostic(code=code, severity="warning", phase="schema", message=code)


def test_registry_runs_in_insertion_order() -> None:
    registry = ValidatorRegistry()
    registry.register("b", lambda _context: [_diagnostic("b")])
    registry.register("a", lambda _context: [_diagnostic("a")])

    result = registry.run(object())

    assert registry.names() == ["b", "a"]
    assert [diagnostic.code for diagnostic in result.diagnostics] == ["b", "a"]


def test_registry_rejects_duplicate_names() -> None:
    registry = ValidatorRegistry()
    registry.register("schema", lambda _context: [])

    with pytest.raises(ValueError, match="Duplicate"):
        registry.register("schema", lambda _context: [])
