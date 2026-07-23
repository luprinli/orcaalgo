from __future__ import annotations

from dataclasses import dataclass
from typing import Literal

Severity = Literal["info", "warning", "error"]


@dataclass(frozen=True)
class Diagnostic:
    code: str
    severity: Severity
    phase: str
    message: str
    profile: str | None = None
    node_id: str | None = None
    port: str | None = None
    remediation: str | None = None
