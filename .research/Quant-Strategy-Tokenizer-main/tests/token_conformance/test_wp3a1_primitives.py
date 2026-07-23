from __future__ import annotations

import math

import pytest

from qst.tokens import (
    TokenReferenceError,
    TokenSpecV2,
    builtin_token_packs,
    evaluate_bool_token,
    evaluate_cmp_token,
    evaluate_math_token,
)

WP3A1_MATH = {
    "math.add",
    "math.sub",
    "math.mul",
    "math.div",
    "math.neg",
    "math.abs",
    "math.pow",
    "math.sqrt",
    "math.log",
    "math.exp",
    "math.min",
    "math.max",
    "math.clip",
    "math.floor",
    "math.ceil",
    "math.round",
    "math.sign",
    "math.isnan",
    "math.isfinite",
    "math.where",
    "math.fill_nan",
}

WP3A1_BOOL = {
    "bool.and",
    "bool.or",
    "bool.not",
    "bool.xor",
    "bool.any",
    "bool.all",
    "bool.count_true",
}

WP3A1_CMP = {
    "cmp.eq",
    "cmp.ne",
    "cmp.gt",
    "cmp.gte",
    "cmp.lt",
    "cmp.lte",
    "cmp.between",
    "cmp.outside",
}


def _all_specs() -> list[TokenSpecV2]:
    return [spec for pack in builtin_token_packs() for spec in pack.tokens]


def _spec_by_name() -> dict[str, TokenSpecV2]:
    return {spec.token_ref.name: spec for spec in _all_specs()}


def _assert_code(exc: pytest.ExceptionInfo[TokenReferenceError], code: str) -> None:
    assert exc.value.code == code


def test_wp3a1_tokens_are_in_builtin_surface_pack() -> None:
    specs = _spec_by_name()
    expected = WP3A1_MATH | WP3A1_BOOL | WP3A1_CMP

    assert expected <= set(specs)
    assert "cmp.crosses" not in specs
    assert {"logic.and", "logic.or", "logic.not", "compare.gt", "compare.eq"} <= set(specs)

    for name in sorted(expected):
        spec = specs[name]
        assert spec.surface.maturity == "accepted"
        assert spec.surface.execution_support == "reference_helper"
        assert spec.surface.contract.numeric
        assert spec.surface.contract.failure_mode

    assert specs["bool.and"].surface.family == "bool"
    assert specs["cmp.between"].surface.category == "range_comparison"
    assert specs["math.fill_nan"].surface.category == "missing_transform"


def test_math_reference_helpers_cover_arithmetic_and_transforms() -> None:
    assert evaluate_math_token("math.add", 1, 2.5) == 3.5
    assert evaluate_math_token("core.math.sub", 5, 2) == 3
    assert evaluate_math_token("math.mul", 3, 4) == 12
    assert evaluate_math_token("math.div", 9, 2) == 4.5
    assert evaluate_math_token("math.neg", 4) == -4
    assert evaluate_math_token("math.abs", -4) == 4
    assert evaluate_math_token("math.pow", 2, 3) == 8
    assert evaluate_math_token("math.sqrt", 9) == 3
    assert evaluate_math_token("math.log", math.e) == pytest.approx(1)
    assert evaluate_math_token("math.floor", 1.9) == 1
    assert evaluate_math_token("math.ceil", 1.1) == 2
    assert evaluate_math_token("math.round", 1.234, ndigits=2) == 1.23
    assert evaluate_math_token("math.sign", -0.5) == -1
    assert evaluate_math_token("math.sign", 0) == 0
    assert evaluate_math_token("math.clip", 5, lower=1, upper=3) == 3


def test_math_reference_helpers_reject_numeric_edge_cases() -> None:
    with pytest.raises(TokenReferenceError) as div_zero:
        evaluate_math_token("math.div", 1, 0)
    _assert_code(div_zero, "QST_TOKEN_MATH_DIVIDE_BY_ZERO")

    with pytest.raises(TokenReferenceError) as sqrt_negative:
        evaluate_math_token("math.sqrt", -1)
    _assert_code(sqrt_negative, "QST_TOKEN_MATH_DOMAIN_ERROR")

    with pytest.raises(TokenReferenceError) as log_non_positive:
        evaluate_math_token("math.log", 0)
    _assert_code(log_non_positive, "QST_TOKEN_MATH_DOMAIN_ERROR")

    with pytest.raises(TokenReferenceError) as bool_numeric:
        evaluate_math_token("math.add", True, 1)
    _assert_code(bool_numeric, "QST_TOKEN_NUMERIC_TYPE_INVALID")

    with pytest.raises(TokenReferenceError) as nonfinite:
        evaluate_math_token("math.add", float("inf"), 1)
    _assert_code(nonfinite, "QST_TOKEN_NUMERIC_NONFINITE")


def test_math_reference_helpers_missing_and_predicate_behavior() -> None:
    assert evaluate_math_token("math.isnan", float("nan")) is True
    assert evaluate_math_token("math.isnan", 1.0) is False
    assert evaluate_math_token("math.isfinite", float("inf")) is False
    assert evaluate_math_token("math.isfinite", 1.0) is True
    assert evaluate_math_token("math.fill_nan", float("nan"), replacement=7) == 7
    assert evaluate_math_token("math.fill_nan", 3, replacement=7) == 3
    assert evaluate_math_token("math.where", True, 1, 2) == 1
    assert evaluate_math_token("math.where", False, 1, 2) == 2

    with pytest.raises(TokenReferenceError) as fill_inf:
        evaluate_math_token("math.fill_nan", float("inf"), replacement=7)
    _assert_code(fill_inf, "QST_TOKEN_NUMERIC_NONFINITE")

    with pytest.raises(TokenReferenceError) as where_condition:
        evaluate_math_token("math.where", 1, 1, 2)
    _assert_code(where_condition, "QST_TOKEN_BOOL_TYPE_INVALID")


def test_math_reductions_have_explicit_empty_input_policy() -> None:
    assert evaluate_math_token("math.min", [3, 1, 2]) == 1
    assert evaluate_math_token("math.max", [3, 1, 2]) == 3
    assert evaluate_math_token("math.min", [], allow_empty=True) is None

    with pytest.raises(TokenReferenceError) as empty_min:
        evaluate_math_token("math.min", [])
    _assert_code(empty_min, "QST_TOKEN_EMPTY_INPUT")


def test_bool_reference_helpers_truth_tables_and_empty_policy() -> None:
    assert evaluate_bool_token("bool.and", True, True) is True
    assert evaluate_bool_token("bool.and", True, False) is False
    assert evaluate_bool_token("bool.or", False, True) is True
    assert evaluate_bool_token("bool.not", False) is True
    assert evaluate_bool_token("bool.xor", True, False) is True
    assert evaluate_bool_token("bool.xor", True, True) is False
    assert evaluate_bool_token("bool.any", [False, True]) is True
    assert evaluate_bool_token("bool.all", [True, True]) is True
    assert evaluate_bool_token("bool.count_true", [True, False, True]) == 2
    assert evaluate_bool_token("bool.all", [], allow_empty=True) is True

    with pytest.raises(TokenReferenceError) as empty_any:
        evaluate_bool_token("bool.any", [])
    _assert_code(empty_any, "QST_TOKEN_EMPTY_INPUT")

    with pytest.raises(TokenReferenceError) as invalid_bool:
        evaluate_bool_token("bool.and", 1, True)
    _assert_code(invalid_bool, "QST_TOKEN_BOOL_TYPE_INVALID")


def test_cmp_reference_helpers_truth_tables_and_range_policy() -> None:
    assert evaluate_cmp_token("cmp.eq", 1, 1) is True
    assert evaluate_cmp_token("cmp.ne", 1, 2) is True
    assert evaluate_cmp_token("cmp.gt", 2, 1) is True
    assert evaluate_cmp_token("cmp.gte", 2, 2) is True
    assert evaluate_cmp_token("cmp.lt", 1, 2) is True
    assert evaluate_cmp_token("cmp.lte", 2, 2) is True
    assert evaluate_cmp_token("cmp.between", 2, 1, 2) is True
    assert evaluate_cmp_token("cmp.between", 2, 1, 2, inclusive=False) is False
    assert evaluate_cmp_token("cmp.outside", 3, 1, 2) is True

    with pytest.raises(TokenReferenceError) as invalid_range:
        evaluate_cmp_token("cmp.between", 1, 2, 1)
    _assert_code(invalid_range, "QST_TOKEN_CMP_RANGE_INVALID")

    with pytest.raises(TokenReferenceError) as bool_cmp:
        evaluate_cmp_token("cmp.eq", True, 1)
    _assert_code(bool_cmp, "QST_TOKEN_NUMERIC_TYPE_INVALID")
