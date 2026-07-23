from __future__ import annotations

import random

import pytest

from orca.calibration.audit import (
    CalibrationReport,
    SegmentReport,
    run_calibration_audit,
)


def _make_trades(n: int, seed: int = 42) -> list[dict]:
    random.seed(seed)
    trades = []
    for _ in range(n):
        p = random.random()
        outcome = "win" if random.random() < p else "loss"
        trades.append({
            "confidence": p,
            "outcome": outcome,
            "side": random.choice(["BUY", "SELL"]),
        })
    return trades


class TestCalibrationAudit:
    def test_empty_trades_returns_report(self):
        report = run_calibration_audit([])
        assert isinstance(report, CalibrationReport)
        assert report.overall.n == 0
        assert report.overall.brier == 0.0

    def test_small_sample_produces_valid_report(self):
        trades = _make_trades(5)
        report = run_calibration_audit(trades)
        assert report.overall.n == 5
        assert 0.0 <= report.overall.brier <= 1.0
        assert 0.0 <= report.overall.reliability <= 1.0
        assert 0.0 <= report.overall.resolution <= 1.0
        assert 0.0 <= report.overall.uncertainty <= 0.25
        assert len(report.overall.bin_stats) == 10

    def test_medium_sample_brier_decomposition(self):
        trades = _make_trades(200)
        report = run_calibration_audit(trades)
        brier = report.overall.brier
        assert 0.0 <= brier <= 1.0

        rel = report.overall.reliability
        res = report.overall.resolution
        unc = report.overall.uncertainty
        assert rel >= 0.0
        assert 0.0 <= unc <= 0.25
        remainder = abs(brier - (rel - res + unc))
        assert remainder < 0.05  # within-bin forecast variance prevents perfect equality

    def test_segment_reports_by_side(self):
        trades = _make_trades(200)
        report = run_calibration_audit(trades)
        assert len(report.segments) >= 1
        for key in report.segments:
            assert key.startswith("side:")
            seg = report.segments[key]
            assert isinstance(seg, SegmentReport)
            assert seg.n > 0
            assert 0.0 <= seg.brier <= 1.0

    def test_segment_needs_calibration_flag(self):
        trades = _make_trades(200)
        report = run_calibration_audit(trades)
        assert isinstance(report.overall.needs_calibration, bool)

    def test_sufficient_data_property(self):
        sr_small = SegmentReport(
            name="test", n=10, brier=0.2, reliability=0.1, resolution=0.05, uncertainty=0.2,
        )
        assert sr_small.sufficient_data is False
        sr_large = SegmentReport(
            name="test", n=50, brier=0.2, reliability=0.1, resolution=0.05, uncertainty=0.2,
        )
        assert sr_large.sufficient_data is True

    def test_report_has_generated_at(self):
        trades = _make_trades(50)
        report = run_calibration_audit(trades)
        assert report.generated_at != ""
        assert "T" in report.generated_at


class TestPlattCalibration:
    def test_too_few_observations_returns_none(self):
        from orca.calibration.platt import platt_calibrate_segment

        result, msg = platt_calibrate_segment(
            [0.6, 0.7, 0.8],
            [1, 0, 1],
            min_cohort_n=50,
        )
        assert result is None
        assert "Insufficient data" in msg

    def test_sufficient_data_produces_result(self):
        from orca.calibration.platt import platt_calibrate_segment
        import numpy as np

        np.random.seed(42)
        n = 300
        raw_p = np.random.uniform(0.3, 0.9, n).tolist()
        outcomes = [1 if random.random() < p else 0 for p in raw_p]

        result, msg = platt_calibrate_segment(
            raw_p, outcomes, min_cohort_n=200, min_improvement_pct=-100.0,
        )
        assert result is not None
        assert result.a is not None
        assert result.b is not None
