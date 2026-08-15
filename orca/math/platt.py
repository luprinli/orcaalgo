from __future__ import annotations

from dataclasses import dataclass

import numpy as np
from scipy.optimize import minimize

from orca.math.brier import brier_score


@dataclass(frozen=True)
class PlattResult:
    a: float
    b: float
    train_brier: float
    val_brier: float
    improvement_pct: float
    recommended: bool


def _sigmoid(x: np.ndarray) -> np.ndarray:
    x = np.clip(x, -500.0, 500.0)
    return 1.0 / (1.0 + np.exp(-x))


def platt_scale(
    raw_p: np.ndarray,
    y: np.ndarray,
    val_raw_p: np.ndarray | None = None,
    val_y: np.ndarray | None = None,
) -> PlattResult:
    EPS = 1e-12
    raw_p = np.clip(np.asarray(raw_p, dtype=np.float64), EPS, 1 - EPS)
    y = np.asarray(y, dtype=np.float64)

    z = np.log(raw_p / (1 - raw_p))
    z = np.clip(z, -30, 30)

    def neg_loglik(params: np.ndarray) -> float:
        a, b = params
        p_cal = _sigmoid(a * z + b)
        p_cal = np.clip(p_cal, EPS, 1 - EPS)
        return -float(np.mean(y * np.log(p_cal) + (1 - y) * np.log(1 - p_cal)))

    result = minimize(neg_loglik, x0=np.array([1.0, 0.0]), method="Nelder-Mead")
    a, b = result.x

    p_cal_train = _sigmoid(a * z + b)
    p_cal_train = np.clip(p_cal_train, EPS, 1 - EPS)
    train_brier = brier_score(list(p_cal_train), list(y.astype(int)))

    raw_brier = brier_score(list(raw_p), list(y.astype(int)))

    if val_raw_p is not None and val_y is not None:
        val_raw_p = np.clip(np.asarray(val_raw_p, dtype=np.float64), EPS, 1 - EPS)
        val_y = np.asarray(val_y, dtype=np.float64)
        vz = np.log(val_raw_p / (1 - val_raw_p))
        vz = np.clip(vz, -30, 30)
        p_cal_val = _sigmoid(a * vz + b)
        p_cal_val = np.clip(p_cal_val, EPS, 1 - EPS)
        val_brier = brier_score(list(p_cal_val), list(val_y.astype(int)))
        raw_val_brier = brier_score(list(val_raw_p), list(val_y.astype(int)))
        improvement = (
            ((raw_val_brier - val_brier) / raw_val_brier * 100) if raw_val_brier > 0 else 0.0
        )
    else:
        val_brier = train_brier
        improvement = ((raw_brier - train_brier) / raw_brier * 100) if raw_brier > 0 else 0.0

    return PlattResult(
        a=a,
        b=b,
        train_brier=train_brier,
        val_brier=val_brier,
        improvement_pct=improvement,
        recommended=improvement >= 5.0,
    )
