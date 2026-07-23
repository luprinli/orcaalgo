from __future__ import annotations

from qst.validation import Diagnostic, ValidationResult


def test_validation_result_ok_ignores_warnings() -> None:
    result = ValidationResult(
        diagnostics=[
            Diagnostic(
                code="warn",
                severity="warning",
                phase="profile",
                message="allowed with warning",
            )
        ]
    )

    assert result.ok
    assert result.warnings[0].code == "warn"
    assert result.errors == []


def test_validation_result_error_controls_ok() -> None:
    result = ValidationResult(
        diagnostics=[
            Diagnostic(
                code="err",
                severity="error",
                phase="schema",
                message="not accepted",
            )
        ]
    )

    assert not result.ok
    assert result.errors[0].code == "err"


def test_diagnostic_shape_is_serializable() -> None:
    diagnostic = Diagnostic(
        code="capability_not_accepted",
        severity="error",
        phase="profile",
        message="panel is not accepted",
        profile="pretrade",
        node_id="n1",
        port="out",
        remediation="remove capability",
    )

    assert diagnostic.model_dump(mode="json") == {
        "code": "capability_not_accepted",
        "severity": "error",
        "phase": "profile",
        "message": "panel is not accepted",
        "profile": "pretrade",
        "node_id": "n1",
        "port": "out",
        "remediation": "remove capability",
    }
