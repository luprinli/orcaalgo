from __future__ import annotations

# ruff: noqa: I001

import pytest

from qst.tokens import (
    TokenReferenceError,
    TokenSpecV2,
    builtin_token_packs,
    evaluate_factor_token,
    evaluate_panel_token,
    evaluate_weight_token,
)
from qst.panel import PanelValue, WeightPanelValue

PR8_TOKENS = {
    "panel.cross_sectional_rank",
    "panel.zscore_by_universe",
    "panel.neutralize_group",
    "selection.top_k",
    "selection.bottom_k",
    "factor.sector_neutral_rank",
    "factor.residualize",
    "factor.beta_neutral_signal",
    "weight.equal_weight",
    "weight.rank_weight",
    "weight.inverse_vol_weight",
    "weight.vol_target_weight",
    "weight.market_neutral_weight",
    "weight.group_neutral_weight",
    "weight.max_weight_clip",
    "weight.normalize_net",
}


def _all_specs() -> list[TokenSpecV2]:
    return [spec for pack in builtin_token_packs() for spec in pack.tokens]


def _spec_by_name() -> dict[str, TokenSpecV2]:
    return {spec.token_ref.name: spec for spec in _all_specs()}


def _panel() -> PanelValue:
    return PanelValue.model_validate(
        {
            "rows": [
                {"timestamp": "t1", "symbol": "AAA", "value": "3"},
                {"timestamp": "t1", "symbol": "BBB", "value": "1"},
                {"timestamp": "t1", "symbol": "CCC", "value": "2"},
            ]
        }
    )


def _weights() -> WeightPanelValue:
    return WeightPanelValue.model_validate(
        {
            "rows": [
                {"timestamp": "t1", "symbol": "AAA", "weight": "0.7"},
                {"timestamp": "t1", "symbol": "BBB", "weight": "0.3"},
                {"timestamp": "t1", "symbol": "CCC", "weight": "-0.2"},
            ]
        }
    )


def test_pr8_tokens_have_accepted_reference_surface() -> None:
    specs = _spec_by_name()

    assert PR8_TOKENS <= set(specs)
    for name in sorted(PR8_TOKENS):
        spec = specs[name]
        assert spec.surface.maturity == "accepted"
        assert spec.surface.execution_support == "reference_helper"
        assert spec.surface.contract.scope == "reference_semantics"
        assert spec.surface.contract.failure_mode
        assert spec.surface.contract.numeric
        assert spec.surface.contract.temporal

    assert specs["factor.sector_neutral_rank"].surface.family == "factor"
    assert specs["weight.inverse_vol_weight"].surface.capabilities.panel_aware is True
    assert specs["weight.inverse_vol_weight"].surface.capabilities.solver_backed is False


def test_panel_aliases_delegate_to_existing_panel_helpers() -> None:
    ranked = evaluate_panel_token("panel.cross_sectional_rank", _panel()).panel
    zscored = evaluate_panel_token("panel.zscore_by_universe", _panel()).panel
    selected = evaluate_panel_token("selection.bottom_k", _panel(), k=1).selection

    assert [row.value for row in ranked.rows] == ["1", "3", "2"]
    assert [row.symbol for row in selected.rows if row.selected] == ["BBB"]
    assert zscored.rows[0].value == "1.224744871391589"


def test_factor_helpers_require_explicit_group_metadata_and_keep_record_boundary() -> None:
    groups = {"AAA": "major", "BBB": "major", "CCC": "alt"}
    ranked = evaluate_factor_token("factor.sector_neutral_rank", _panel(), groups=groups).panel
    residual = evaluate_factor_token(
        "factor.beta_neutral_signal",
        [("t1", 3), ("t2", 5), ("t3", 7)],
        [("t1", 1), ("t2", 2), ("t3", 3)],
        window=3,
    )

    assert [row.value for row in ranked.rows] == ["1", "3", "2"]
    assert residual == [("t3", 0.0)]
    with pytest.raises(TokenReferenceError) as missing_groups:
        evaluate_factor_token("factor.sector_neutral_rank", _panel())
    assert missing_groups.value.code == "QST_TOKEN_FACTOR_GROUPS_REQUIRED"


def test_weight_helpers_cover_inverse_vol_rank_group_and_net_records() -> None:
    selection = evaluate_panel_token("selection.top_k", _panel(), k=2).selection
    equal = evaluate_weight_token("weight.equal_weight", selection).weights
    ranked = evaluate_panel_token("panel.cross_sectional_rank", _panel()).panel
    rank_weight = evaluate_weight_token("weight.rank_weight", ranked).weights
    vol_panel = PanelValue.model_validate(
        {
            "rows": [
                {"timestamp": "t1", "symbol": "AAA", "value": "2"},
                {"timestamp": "t1", "symbol": "BBB", "value": "4"},
            ]
        }
    )
    inverse_vol = evaluate_weight_token("weight.inverse_vol_weight", vol_panel).weights
    grouped = evaluate_weight_token(
        "weight.group_neutral_weight",
        _weights(),
        groups={"AAA": "major", "BBB": "major", "CCC": "alt"},
    ).weights
    normalized = evaluate_weight_token("weight.normalize_net", _weights(), target_net="0").weights

    assert [row.weight for row in equal.rows] == ["0.5", "0.5"]
    assert [row.weight for row in rank_weight.rows] == [
        "0.5",
        "0.1666666666666666666666666667",
        "0.3333333333333333333333333333",
    ]
    assert [row.weight for row in inverse_vol.rows] == [
        "0.6666666666666666666666666667",
        "0.3333333333333333333333333333",
    ]
    assert [row.weight for row in grouped.rows] == ["0.2", "-0.2", "0"]
    assert [row.weight for row in normalized.rows] == [
        "0.4333333333333333333333333333",
        "0.0333333333333333333333333333",
        "-0.4666666666666666666666666667",
    ]


def test_weight_helpers_reject_nonpositive_volatility_and_missing_groups() -> None:
    bad_vol = PanelValue.model_validate({"rows": [{"timestamp": "t1", "symbol": "AAA", "value": "0"}]})
    with pytest.raises(TokenReferenceError) as nonpositive:
        evaluate_weight_token("weight.inverse_vol_weight", bad_vol)
    assert nonpositive.value.code == "QST_TOKEN_WEIGHT_VOLATILITY_NONPOSITIVE"

    with pytest.raises(TokenReferenceError) as missing_groups:
        evaluate_weight_token("weight.group_neutral_weight", _weights())
    assert missing_groups.value.code == "QST_TOKEN_FACTOR_GROUPS_REQUIRED"
