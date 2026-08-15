from __future__ import annotations

import numpy as np

from orca.math.platt import PlattResult, platt_scale


def platt_calibrate_segment(
    raw_p: list[float],
    outcomes: list[int],
    cohort_name: str = "default",
    min_cohort_n: int = 200,
    min_improvement_pct: float = 5.0,
) -> tuple[PlattResult | None, str]:
    """Calibrate a segment using Platt scaling if sufficient data."""
    if len(raw_p) < min_cohort_n:
        return None, f"Insufficient data: {len(raw_p)} observations, need {min_cohort_n}"

    n = len(raw_p)
    split = int(n * 0.7)
    train_p = np.array(raw_p[:split])
    train_y = np.array(outcomes[:split])
    val_p = np.array(raw_p[split:])
    val_y = np.array(outcomes[split:])

    result = platt_scale(train_p, train_y, val_p, val_y)

    if result.improvement_pct < min_improvement_pct:
        return (
            result,
            f"Improvement ({result.improvement_pct:.1f}%) below threshold ({min_improvement_pct}%)",
        )

    return result, f"Calibrated: improvement {result.improvement_pct:.1f}%"
