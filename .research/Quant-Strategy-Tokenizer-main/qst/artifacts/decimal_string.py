"""Strict canonical decimal string support for artifacts."""

from __future__ import annotations

import re
from decimal import Decimal, InvalidOperation
from typing import Annotated

from pydantic import AfterValidator

DECIMAL_STRING_PATTERN = re.compile(r"^-?(0|[1-9][0-9]*)(\.[0-9]*[1-9])?$")


def validate_decimal_string(value: str) -> str:
    """Validate strict canonical DecimalString form."""

    if not isinstance(value, str):
        raise TypeError(f"DecimalString must be str, got {type(value).__name__}")
    if value == "-0" or not DECIMAL_STRING_PATTERN.match(value):
        raise ValueError(
            f"{value!r} is not strict canonical DecimalString. "
            "Forbidden: exponent / leading plus / insignificant leading zeros / "
            "trailing zeros in fraction / negative zero."
        )
    return value


def normalize_to_canonical(raw: str | float | Decimal) -> str:
    """Normalize a raw numeric value to strict canonical DecimalString."""

    try:
        decimal = Decimal(str(raw)) if isinstance(raw, float) else Decimal(raw)
    except (InvalidOperation, ValueError) as exc:
        raise ValueError(f"Cannot normalize {raw!r} to DecimalString") from exc

    if not decimal.is_finite():
        raise ValueError(f"Cannot normalize non-finite decimal {raw!r}")

    normalized = decimal.normalize()
    text = format(normalized, "f")
    if "." in text:
        text = text.rstrip("0").rstrip(".")
    if text == "-0":
        text = "0"
    return validate_decimal_string(text)


DecimalString = Annotated[str, AfterValidator(validate_decimal_string)]
