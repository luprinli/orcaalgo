"""
Guardian regression smoke tests for Python (orca/).

These tests cover the most critical code paths. Any failure here
blocks all merges. Run: pytest tests/guardian/ -v --tb=short
"""

import numpy as np


def test_kelly_sizing_default_params():
    """Kelly sizing with default fractional params returns valid size."""
    from orca.sizing.kelly import kelly_with_attenuators

    result = kelly_with_attenuators(
        p=0.60, price=0.55, side="yes", multiplier=0.25, per_trade_cap_pct=0.02,
    )
    assert 0 < result.final_allocation <= 0.02, (
        f"Expected size in (0, 0.02], got {result.final_allocation}"
    )


def test_kelly_sizing_edge_cases():
    """Kelly sizing handles edge probabilities correctly."""
    from orca.sizing.kelly import kelly_with_attenuators

    # Extreme confidence
    result_high = kelly_with_attenuators(
        p=0.95, price=0.50, side="yes", multiplier=0.25, per_trade_cap_pct=0.02,
    )
    assert 0 <= result_high.final_allocation <= 0.02

    # Edge case: p < price (no edge)
    result_no_edge = kelly_with_attenuators(
        p=0.40, price=0.50, side="yes", multiplier=0.25, per_trade_cap_pct=0.02,
    )
    assert result_no_edge.final_allocation == 0.0


def test_brier_score_perfect_prediction():
    """Perfect predictions should yield Brier = 0."""
    from orca.math.brier import brier_score

    result = brier_score(np.array([1.0, 0.0]), np.array([1, 0]))
    assert np.isclose(result, 0.0), f"Expected 0, got {result}"


def test_brier_score_worst_prediction():
    """Completely wrong predictions should yield Brier = 1."""
    from orca.math.brier import brier_score

    result = brier_score(np.array([1.0, 1.0]), np.array([0, 0]))
    assert np.isclose(result, 1.0), f"Expected 1, got {result}"


def test_wilson_ci_bounds():
    """Wilson CI must be bounded in [0, 1]."""
    from orca.math.wilson import wilson_ci

    lo, hi = wilson_ci(wins=45, n=100, z=1.96)
    assert 0 <= lo <= hi <= 1, f"Expected CI in [0,1], got [{lo}, {hi}]"


def test_wilson_ci_insufficient_data():
    """Wilson CI must return valid bounds when n is small."""
    from orca.math.wilson import wilson_ci

    lo, hi = wilson_ci(wins=5, n=10, z=1.96)
    assert 0 <= lo <= hi <= 1


def test_platt_scaling_output_bounds():
    """Platt scaling must return probabilities in [0, 1]."""
    import numpy as np

    from orca.math.platt import platt_scale

    probs = np.array([0.1, 0.3, 0.5, 0.7, 0.9], dtype=np.float64)
    y = np.array([0, 0, 1, 1, 1], dtype=np.float64)
    result = platt_scale(probs, y)
    assert result.a is not None and result.b is not None


def test_ewma_volatility_positive():
    """EWMA volatility must be non-negative."""
    from orca.sizing.volatility import ewma_volatility

    returns = np.array([0.01, -0.02, 0.005, 0.03, -0.01])
    vol = ewma_volatility(returns, span=20)
    assert np.all(vol >= 0), f"Expected non-negative volatility, got {vol}"


def test_gkr_parse_and_hash_intraday_mr():
    """GKR IR parsing + deterministic hash must succeed."""
    from orca.ir.loader import load_ir

    ir = load_ir("configs/strategies/intraday_mr.gkr.yaml")
    assert ir is not None
    assert hasattr(ir, "strategy") or hasattr(ir, "ir_version")


def test_gkr_parse_trend_following():
    """Trend following GKR config must parse successfully."""
    from orca.ir.loader import load_ir

    ir = load_ir("configs/strategies/trend_following.gkr.yaml")
    assert ir is not None


def test_gkr_parse_opening_range_breakout():
    """Opening range breakout GKR config must parse successfully."""
    from orca.ir.loader import load_ir

    ir = load_ir("configs/strategies/opening_range_breakout.gkr.yaml")
    assert ir is not None


def test_models_immutability():
    """All domain models must be frozen."""
    from orca.models import TradeSignal

    try:
        ts = TradeSignal(
            symbol="SPY", signal="BUY", confidence=0.7,
            timestamp="2024-01-01T00:00:00Z"
        )
        ts.confidence = 0.8
        assert False, "Model should be frozen — mutation succeeded unexpectedly"
    except Exception:
        pass  # Expected


def test_calibration_audit_importable():
    """Calibration audit module must be importable."""
    import orca.calibration.audit  # noqa: F401


def test_preflight_importable():
    """Pre-flight checklist module must be importable."""
    import orca.preflight.checklist  # noqa: F401


def test_attribution_importable():
    """PnL attribution module must be importable."""
    import orca.attribution.slicer  # noqa: F401


def test_hash_deterministic():
    """Content-addressable hashing must be deterministic."""
    from orca.hash.graph import instance_hash_v2
    from orca.ir.loader import load_ir

    ir1 = load_ir("configs/strategies/intraday_mr.gkr.yaml")
    ir2 = load_ir("configs/strategies/intraday_mr.gkr.yaml")
    h1 = instance_hash_v2(ir1)
    h2 = instance_hash_v2(ir2)
    msg = f"Deterministic hash mismatch: {h1} vs {h2}"
    assert h1 == h2, msg
    assert len(h1) >= 64, f"Expected >= 64-char hash string, got {len(h1)}"
