from __future__ import annotations

import pytest

from orca.sizing.kelly import (
    KellyResult,
    kelly_fraction_binary,
    kelly_fraction_continuous,
    kelly_with_attenuators,
)


class TestKellyFractionBinary:
    def test_yes_side_positive_edge(self):
        result = kelly_fraction_binary(p=0.60, price=0.50, side="yes")
        assert result == pytest.approx(0.20)

    def test_yes_side_no_edge(self):
        result = kelly_fraction_binary(p=0.50, price=0.50, side="yes")
        assert result == pytest.approx(0.0)

    def test_yes_side_negative_edge(self):
        result = kelly_fraction_binary(p=0.40, price=0.50, side="yes")
        assert result == pytest.approx(-0.20)

    def test_no_side_positive_edge(self):
        result = kelly_fraction_binary(p=0.40, price=0.50, side="no")
        assert result == pytest.approx(0.20)

    def test_no_side_no_edge(self):
        result = kelly_fraction_binary(p=0.50, price=0.50, side="no")
        assert result == pytest.approx(0.0)

    def test_known_values(self):
        assert kelly_fraction_binary(p=0.75, price=0.60, side="yes") == pytest.approx(0.375)
        result = kelly_fraction_binary(p=0.30, price=0.70, side="no")
        expected = ((1 - 0.30) - (1 - 0.70)) / (1 - (1 - 0.70))
        assert result == pytest.approx(expected)

    def test_price_out_of_range_raises(self):
        with pytest.raises(ValueError, match="Price must be in"):
            kelly_fraction_binary(p=0.60, price=0.0, side="yes")
        with pytest.raises(ValueError, match="Price must be in"):
            kelly_fraction_binary(p=0.60, price=1.0, side="yes")
        with pytest.raises(ValueError, match="Price must be in"):
            kelly_fraction_binary(p=0.60, price=1.5, side="yes")

    def test_probability_out_of_range_raises(self):
        with pytest.raises(ValueError, match="Probability must be in"):
            kelly_fraction_binary(p=-0.1, price=0.50, side="yes")
        with pytest.raises(ValueError, match="Probability must be in"):
            kelly_fraction_binary(p=1.5, price=0.50, side="yes")

    def test_invalid_side_raises(self):
        with pytest.raises(ValueError, match="Side must be"):
            kelly_fraction_binary(p=0.60, price=0.50, side="maybe")

    def test_case_insensitive_side(self):
        result_upper = kelly_fraction_binary(p=0.60, price=0.50, side="YES")
        result_lower = kelly_fraction_binary(p=0.60, price=0.50, side="yes")
        assert result_upper == pytest.approx(result_lower)


class TestKellyFractionContinuous:
    def test_positive_edge(self):
        result = kelly_fraction_continuous(p_win=0.60, win_loss_ratio=1.0)
        assert result == pytest.approx(0.20)

    def test_no_edge(self):
        result = kelly_fraction_continuous(p_win=0.50, win_loss_ratio=1.0)
        assert result == pytest.approx(0.0)

    def test_high_win_loss_ratio(self):
        result = kelly_fraction_continuous(p_win=0.55, win_loss_ratio=2.0)
        assert result == pytest.approx(0.325)

    def test_p_win_out_of_range(self):
        with pytest.raises(ValueError):
            kelly_fraction_continuous(p_win=-0.1, win_loss_ratio=1.0)
        with pytest.raises(ValueError):
            kelly_fraction_continuous(p_win=1.5, win_loss_ratio=1.0)

    def test_win_loss_ratio_non_positive(self):
        with pytest.raises(ValueError):
            kelly_fraction_continuous(p_win=0.60, win_loss_ratio=0.0)
        with pytest.raises(ValueError):
            kelly_fraction_continuous(p_win=0.60, win_loss_ratio=-1.0)

    def test_zero_p_win(self):
        result = kelly_fraction_continuous(p_win=0.0, win_loss_ratio=2.0)
        assert result == pytest.approx(-0.5)

    def test_sure_win(self):
        result = kelly_fraction_continuous(p_win=1.0, win_loss_ratio=2.0)
        assert result == pytest.approx(1.0)


class TestKellyWithAttenuators:
    def test_returns_kelly_result(self):
        result = kelly_with_attenuators(p=0.60, price=0.50)
        assert isinstance(result, KellyResult)

    def test_default_params_positive_edge(self):
        result = kelly_with_attenuators(p=0.60, price=0.50, side="yes")
        assert result.discounted_p == pytest.approx(0.58)
        expected_raw = kelly_fraction_binary(0.58, 0.50, "yes")
        assert result.raw_kelly == pytest.approx(expected_raw)
        assert result.fractional_kelly == pytest.approx(expected_raw * 0.25)
        assert result.final_allocation >= 0

    def test_no_side_default_params(self):
        result = kelly_with_attenuators(p=0.40, price=0.50, side="no", edge_discount=0.05)
        assert result.discounted_p == pytest.approx(0.45)

    def test_per_trade_cap_limit(self):
        result = kelly_with_attenuators(
            p=0.90, price=0.30, side="yes",
            multiplier=1.0, edge_discount=0.0,
            per_trade_cap_pct=0.05,
        )
        assert result.per_trade_cap <= 0.05

    def test_exposure_headroom_limit(self):
        result = kelly_with_attenuators(
            p=0.60, price=0.50,
            total_exposure_cap_pct=0.30,
            current_exposure_pct=0.28,
        )
        assert result.exposure_limit <= 0.02

    def test_exposure_headroom_saturated(self):
        result = kelly_with_attenuators(
            p=0.60, price=0.50,
            total_exposure_cap_pct=0.30,
            current_exposure_pct=0.35,
        )
        assert result.exposure_limit == 0.0
        assert result.final_allocation == 0.0

    def test_no_negative_allocation(self):
        result = kelly_with_attenuators(p=0.30, price=0.60, side="yes")
        assert result.final_allocation >= 0.0

    def test_all_attenuators_applied(self):
        result = kelly_with_attenuators(
            p=0.60, price=0.50,
            multiplier=0.25, edge_discount=0.02,
            per_trade_cap_pct=0.02, total_exposure_cap_pct=0.30,
            current_exposure_pct=0.10,
        )
        raw = result.raw_kelly
        frac = result.fractional_kelly
        assert frac == pytest.approx(raw * 0.25)
        assert result.per_trade_cap == pytest.approx(min(frac, 0.02))
        headroom = max(0.30 - 0.10, 0.0)
        assert result.exposure_limit == pytest.approx(min(result.per_trade_cap, headroom))
        assert result.final_allocation == pytest.approx(max(0.0, result.exposure_limit))

    def test_custom_multiplier(self):
        result = kelly_with_attenuators(p=0.60, price=0.50, multiplier=0.50, edge_discount=0.0)
        expected_raw = kelly_fraction_binary(0.60, 0.50, "yes")
        assert result.fractional_kelly == pytest.approx(expected_raw * 0.50)

    def test_never_exceeds_full_kelly(self):
        result = kelly_with_attenuators(p=0.90, price=0.30, side="yes", multiplier=0.25)
        raw = kelly_fraction_binary(result.discounted_p, 0.30, "yes")
        assert result.final_allocation <= raw

    def test_edge_discount_caps_probability(self):
        result = kelly_with_attenuators(p=0.99, price=0.50, side="yes", edge_discount=0.02)
        assert result.discounted_p == pytest.approx(0.97)

    def test_no_side_edge_discount(self):
        result = kelly_with_attenuators(p=0.01, price=0.50, side="no", edge_discount=0.02)
        assert result.discounted_p == pytest.approx(0.03)


class TestKellyRegimeMultipliers:
    """Regime-specific Kelly multipliers per §5.2 of the Senior Quantitative Review."""

    def test_calm_regime_kelly(self):
        """Calm regime uses standard k=0.25."""
        result = kelly_with_attenuators(p=0.55, price=0.50, side="yes", multiplier=0.25)
        assert result.final_allocation > 0
        assert result.fractional_kelly == pytest.approx(result.raw_kelly * 0.25)

    def test_high_vol_regime_kelly(self):
        """High Vol regime uses reduced k=0.15."""
        result = kelly_with_attenuators(p=0.55, price=0.50, side="yes", multiplier=0.15)
        assert result.final_allocation > 0
        assert result.fractional_kelly == pytest.approx(result.raw_kelly * 0.15)
        # HighVol allocation should be smaller than Calm allocation.
        calm = kelly_with_attenuators(p=0.55, price=0.50, side="yes", multiplier=0.25)
        assert result.fractional_kelly < calm.fractional_kelly

    def test_crisis_regime_kelly(self):
        """Crisis regime uses k=0.0 — no allocation."""
        result = kelly_with_attenuators(p=0.55, price=0.50, side="yes", multiplier=0.0)
        assert result.final_allocation == pytest.approx(0.0)
        assert result.fractional_kelly == pytest.approx(0.0)

    def test_kelly_attenuators_still_apply(self):
        """Even with regime-specific Kelly, all three attenuators must apply."""
        result = kelly_with_attenuators(
            p=0.65, price=0.50, side="yes",
            multiplier=0.15,
            edge_discount=0.02,
            per_trade_cap_pct=0.02,
            total_exposure_cap_pct=0.15,
            current_exposure_pct=0.05,
        )
        assert result.final_allocation >= 0
        assert result.discounted_p < 0.65  # edge discount applied
        assert result.fractional_kelly <= result.raw_kelly * 0.15
        assert result.final_allocation <= result.fractional_kelly  # caps applied
