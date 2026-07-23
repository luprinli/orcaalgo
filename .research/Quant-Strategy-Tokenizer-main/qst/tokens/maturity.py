"""Profile gates for token surface maturity."""

from __future__ import annotations

from qst.profiles import ProfileName
from qst.tokens.spec import TokenSpecV2
from qst.validation import Diagnostic, Severity


def validate_token_maturity_for_profile(
    spec: TokenSpecV2,
    *,
    profile: ProfileName,
    node_id: str | None = None,
) -> list[Diagnostic]:
    """Return diagnostics for profile-specific token maturity use."""

    maturity = spec.surface.maturity
    if maturity in {"accepted", "frozen"}:
        return []
    if maturity == "reserved_design":
        return [
            Diagnostic(
                code="QST_TOKEN_RESERVED_DESIGN_NOT_EXECUTABLE",
                severity="error",
                phase="profile",
                profile=profile,
                node_id=node_id,
                message=f"Token {spec.token_id} is reserved design metadata and is not executable.",
                remediation="Use an accepted token or wait for the owning type/runtime layer.",
            )
        ]
    if maturity == "experimental":
        experimental_severity: Severity = "warning" if profile in {"research", "paper"} else "error"
        return [
            Diagnostic(
                code="QST_TOKEN_EXPERIMENTAL_PROFILE_GATE",
                severity=experimental_severity,
                phase="profile",
                profile=profile,
                node_id=node_id,
                message=f"Token {spec.token_id} is experimental for profile {profile}.",
                remediation="Use research/paper or promote the token with conformance evidence.",
            )
        ]
    deprecated_severity: Severity = "error" if profile == "production_guarded" else "warning"
    return [
        Diagnostic(
            code="QST_TOKEN_DEPRECATED_PROFILE_GATE",
            severity=deprecated_severity,
            phase="profile",
            profile=profile,
            node_id=node_id,
            message=f"Token {spec.token_id} is deprecated for profile {profile}.",
            remediation="Use the documented replacement token.",
        )
    ]
