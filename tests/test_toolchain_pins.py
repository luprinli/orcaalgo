"""Toolchain pin guard (R4) — fail when dev tooling drifts.

Mirrors ML4T's ``tests/test_toolchain_pins.py``: the CI/dev toolchain must stay
compatible across environments. This test asserts the installed lint/type/test
tool versions satisfy the floors declared in ``pyproject.toml``'s ``dev`` extra,
so a silent upgrade that breaks the gates is caught at test time.
"""

from __future__ import annotations

import importlib.metadata

import pytest

FLOORS = {
    "ruff": (0, 1),
    "mypy": (1, 8),
    "pytest": (7, 0),
    "pytest-rerunfailures": (15, 0),
    "pytest-timeout": (2, 3),
}


def _version_tuple(distribution: str) -> tuple[int, ...]:
    raw = importlib.metadata.version(distribution)
    parts = []
    for token in raw.replace("-", ".").split("."):
        digits = "".join(ch for ch in token if ch.isdigit())
        if digits:
            parts.append(int(digits))
        else:
            break
    return tuple(parts)


@pytest.mark.parametrize("distribution", sorted(FLOORS))
def test_toolchain_within_floor(distribution: str) -> None:
    try:
        installed = _version_tuple(distribution)
    except importlib.metadata.PackageNotFoundError:
        pytest.skip(f"{distribution} not installed (dev extra not active)")
    floor = FLOORS[distribution]
    assert installed >= floor, f"{distribution} {installed} is below floor {floor}"
