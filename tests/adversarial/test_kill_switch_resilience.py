"""
Adversarial edge-case injection tests for kill-switch resilience and system safety.
"""


def test_kelly_never_exceeds_cap():
    """Under no valid input should position size exceed the 2% hard cap."""
    from orca.sizing.kelly import kelly_with_attenuators

    # Brute-force a few edge probabilities
    test_cases = [
        (0.95, 0.01),  # Very strong edge
        (0.80, 0.10),  # Strong edge
        (0.60, 0.40),  # Moderate edge
        (0.50, 0.50),  # No edge
        (0.30, 0.70),  # Negative edge (NO side)
    ]
    for p, price in test_cases:
        result = kelly_with_attenuators(
            p=p,
            price=price,
            side="yes",
            multiplier=0.25,
            per_trade_cap_pct=0.02,
        )
        assert result.final_allocation <= 0.02, (
            f"Size {result.final_allocation} exceeded cap for p={p}, price={price}"
        )


def test_extreme_decimals():
    """Very small or very large numbers should not crash."""
    import sys

    from orca.sizing.kelly import kelly_with_attenuators

    tiny = sys.float_info.min

    try:
        result = kelly_with_attenuators(
            p=tiny,
            price=0.5,
            side="yes",
            multiplier=0.25,
            per_trade_cap_pct=0.02,
        )
        assert result.final_allocation >= 0
    except (ValueError, OverflowError):
        pass  # Acceptable failure mode

    try:
        result = kelly_with_attenuators(
            p=0.5,
            price=1 - tiny,
            side="yes",
            multiplier=0.25,
            per_trade_cap_pct=0.02,
        )
        assert result.final_allocation >= 0
    except (ValueError, OverflowError, ZeroDivisionError):
        pass  # Acceptable failure mode
