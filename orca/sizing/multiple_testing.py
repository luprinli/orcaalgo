"""Multiple testing correction for backtest matrix results.

Implements Bonferroni correction and Benjamini-Hochberg (BH) false discovery rate
control for matrix sweep results where many strategy/symbol/timeframe combinations
are tested simultaneously.

Required before claiming statistical significance from exploratory matrix sweeps.
"""

from __future__ import annotations

from typing import Any

import numpy as np


def bonferroni_correction(p_values: list[float], alpha: float = 0.05) -> dict[str, Any]:
    """Apply Bonferroni correction to a list of p-values.

    The Bonferroni correction divides alpha by the number of tests,
    controlling the family-wise error rate (FWER). It is conservative
    and works best with small numbers of tests.

    Args:
        p_values: List of p-values from hypothesis tests.
        alpha: Significance level (default 0.05).

    Returns:
        Dict with {
            corrected_alpha: adjusted significance threshold,
            significant: boolean mask of significant tests,
            n_tests: total number of tests,
        }
    """
    n = len(p_values)
    corrected_alpha = alpha / max(n, 1)
    significant = [p <= corrected_alpha for p in p_values]
    return {
        "corrected_alpha": corrected_alpha,
        "significant": significant,
        "n_significant": sum(significant),
        "n_tests": n,
        "method": "bonferroni",
    }


def benjamini_hochberg_correction(
    p_values: list[float],
    alpha: float = 0.05,
) -> dict[str, Any]:
    """Apply Benjamini-Hochberg (BH) false discovery rate control.

    BH is less conservative than Bonferroni and controls the expected
    proportion of false discoveries rather than the probability of any
    false discovery. Better suited for large matrix sweeps.

    Args:
        p_values: List of p-values from hypothesis tests.
        alpha: FDR threshold (default 0.05).

    Returns:
        Dict with {
            significant: boolean mask of BH-significant tests,
            critical_values: BH critical values per rank,
            n_significant: count of significant tests,
            n_tests: total number of tests,
        }
    """
    n = len(p_values)
    if n == 0:
        return {
            "significant": [],
            "critical_values": [],
            "n_significant": 0,
            "n_tests": 0,
            "method": "benjamini_hochberg",
        }

    ranks = np.argsort(p_values)
    sorted_p = np.array(p_values)[ranks]

    i = np.arange(1, n + 1)
    critical_values = i * alpha / n

    max_k = 0
    for k in range(n - 1, -1, -1):
        if sorted_p[k] <= critical_values[k]:
            max_k = k + 1
            break

    significant = np.zeros(n, dtype=bool)
    if max_k > 0:
        significant[ranks[:max_k]] = True

    return {
        "significant": significant.tolist(),
        "critical_values": critical_values.tolist(),
        "n_significant": int(np.sum(significant)),
        "n_tests": n,
        "method": "benjamini_hochberg",
    }


def apply_multiple_testing_correction(
    p_values: list[float],
    alpha: float = 0.05,
    method: str = "bh",
) -> dict[str, Any]:
    """Apply the specified multiple testing correction.

    Args:
        p_values: List of p-values.
        alpha: Significance/FDR level.
        method: 'bonferroni' or 'bh' (Benjamini-Hochberg).

    Returns:
        Correction result dict.
    """
    if method == "bonferroni":
        return bonferroni_correction(p_values, alpha)
    return benjamini_hochberg_correction(p_values, alpha)
