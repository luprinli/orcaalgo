from __future__ import annotations

import pytest

from orca.ml.monitoring import (
    ModelHealthReport,
    ModelStatus,
    check_acceptance_rate,
    check_model_staleness,
    check_win_rate_degradation,
    compute_prediction_distribution,
    generate_health_report,
)
from datetime import datetime, timedelta, timezone


class TestComputePredictionDistribution:
    def test_basic(self):
        pwins = [0.3, 0.7, 0.85, 0.2, 0.55, 0.9, 0.1, 0.75]
        accepted = [True, True, False, True, True, False, True, False]
        dist = compute_prediction_distribution(pwins, accepted, bins=5)
        assert dist.acceptance_rate > 0
        assert dist.acceptance_rate < 1.0
        assert dist.total_predictions == len(pwins)

    def test_all_accepted(self):
        pwins = [0.5] * 10
        accepted = [True] * 10
        dist = compute_prediction_distribution(pwins, accepted)
        assert dist.acceptance_rate == 1.0

    def test_all_rejected(self):
        pwins = [0.5] * 10
        accepted = [False] * 10
        dist = compute_prediction_distribution(pwins, accepted)
        assert dist.acceptance_rate == 0.0

    def test_empty_prediction(self):
        dist = compute_prediction_distribution([], [])
        assert dist.total_predictions == 0
        assert dist.acceptance_rate == 0.0

    def test_no_accepted_list(self):
        pwins = [0.3, 0.7, 0.5]
        dist = compute_prediction_distribution(pwins, None)
        assert dist.total_predictions == 3


class TestCheckModelStaleness:
    def test_stale(self):
        trained = datetime.now(timezone.utc) - timedelta(days=10)
        alert = check_model_staleness(trained, warn_days=7, critical_days=30)
        assert alert is not None

    def test_critical(self):
        trained = datetime.now(timezone.utc) - timedelta(days=45)
        alert = check_model_staleness(trained, warn_days=7, critical_days=30)
        assert alert is not None


class TestCheckWinRateDegradation:
    def test_no_degradation(self):
        alert = check_win_rate_degradation(0.65, 0.60, warn_delta=0.15, critical_delta=0.25)
        assert alert is None

    def test_warning(self):
        alert = check_win_rate_degradation(0.40, 0.60, warn_delta=0.15, critical_delta=0.25)
        assert alert is not None

    def test_critical(self):
        alert = check_win_rate_degradation(0.20, 0.60, warn_delta=0.15, critical_delta=0.25)
        assert alert is not None


class TestCheckAcceptanceRate:
    def test_above_minimum(self):
        alert = check_acceptance_rate(0.50, min_rate=0.30)
        assert alert is None

    def test_below_minimum(self):
        alert = check_acceptance_rate(0.15, min_rate=0.30)
        assert alert is not None
        assert "acceptance rate" in alert.message.lower()

    def test_with_wilson_ci(self):
        alert = check_acceptance_rate(0.10, min_rate=0.30, accepted_wins=5, accepted_total=100)
        assert alert is not None
        assert "95% CI" in alert.message

    def test_wilson_ci_zero_samples(self):
        alert = check_acceptance_rate(0.10, min_rate=0.30, accepted_wins=0, accepted_total=0)
        assert alert is not None
        assert "95% CI" not in alert.message


class TestGenerateHealthReport:
    def test_healthy_model_produces_report(self):
        pwins = [0.35 + (i % 3) * 0.2 for i in range(20)]
        accepted = [True if p > 0.3 else False for p in pwins]
        report = generate_health_report(
            model_name="test_model",
            model_version="1.0.0",
            last_trained_at=datetime.now(timezone.utc),
            recent_pwins=pwins,
            recent_accepted=accepted,
            historical_win_rate=0.60,
            recent_win_rate=0.58,
            psi_total=0.05,
            psi_status="ok",
            total_predictions=20,
        )
        assert isinstance(report, ModelHealthReport)
        assert report.model_name == "test_model"
        assert report.total_predictions == 20

    def test_stale_model_detected(self):
        pwins = [0.1] * 10
        report = generate_health_report(
            model_name="stale_model",
            model_version="1.0.0",
            last_trained_at=datetime.now(timezone.utc) - timedelta(days=60),
            recent_pwins=pwins,
            recent_accepted=[True] * 10,
            historical_win_rate=0.65,
            recent_win_rate=0.25,
            psi_total=0.05,
            psi_status="ok",
            total_predictions=10,
        )
        assert report.status != ModelStatus.HEALTHY
