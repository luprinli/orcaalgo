from __future__ import annotations

from decimal import Decimal

import pytest

from qst.artifacts.decimal_string import (
    normalize_to_canonical,
    validate_decimal_string,
)


@pytest.mark.parametrize("value", ["0", "0.25", "1", "-1.5"])
def test_decimal_string_accepts_strict_canonical_values(value: str) -> None:
    assert validate_decimal_string(value) == value


@pytest.mark.parametrize("value", ["1.0", "1.00", "0.10", "0.50", "1e-3", "+1.0", "001.0", "-0"])
def test_decimal_string_rejects_non_canonical_values(value: str) -> None:
    with pytest.raises(ValueError):
        validate_decimal_string(value)


def test_normalize_to_canonical_for_adapter_inputs() -> None:
    assert normalize_to_canonical("0.10000000") == "0.1"
    assert normalize_to_canonical("1.0") == "1"
    assert normalize_to_canonical(1.5) == "1.5"
    assert normalize_to_canonical(Decimal("100.000")) == "100"
    assert normalize_to_canonical("-0") == "0"
    assert normalize_to_canonical(Decimal("-0.000")) == "0"
