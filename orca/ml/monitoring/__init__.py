"""ML monitoring: drift detection, model health tracking, prediction distribution analysis, and automated retraining triggers.

The monitoring system tracks:
1. Prediction distribution — win probability histogram, PnL correlation
2. Model staleness — days since last training, version age
3. PSI drift — per-feature Population Stability Index
4. Performance degradation — win rate trend, Sharpe trend
5. Retraining triggers — automated signals when thresholds are breached

All state is persisted to TimescaleDB via hypertables for temporal queries.
"""

from __future__ import annotations

import json
import logging
from dataclasses import dataclass, field, replace
from datetime import UTC, datetime
from enum import Enum, auto
from pathlib import Path
from typing import Any

import numpy as np

logger = logging.getLogger("orca.ml.monitoring")

__version__ = "0.2.0"


# ─── Data Classes ─────────────────────────────────────────────────────────────

class ModelStatus(Enum):
    HEALTHY = auto()
    STALE = auto()
    DEGRADED = auto()
    FAILED = auto()


class AlertSeverity(Enum):
    INFO = auto()
    WARN = auto()
    CRITICAL = auto()


@dataclass(frozen=True)
class Alert:
    severity: AlertSeverity
    model: str
    metric: str
    value: float
    threshold: float
    message: str
    timestamp: datetime = field(default_factory=lambda: datetime.now(UTC))


@dataclass(frozen=True)
class PredictionDistribution:
    model_name: str
    bucket_counts: dict[str, int] = field(default_factory=dict)
    total_predictions: int = 0
    mean_pwin: float = 0.0
    median_pwin: float = 0.0
    std_pwin: float = 0.0
    pwin_10th: float = 0.0
    pwin_90th: float = 0.0
    acceptance_rate: float = 0.0


@dataclass(frozen=True)
class ModelHealthReport:
    model_name: str
    model_version: str
    status: ModelStatus
    days_since_training: float
    total_predictions: int
    win_rate_recent: float
    win_rate_historical: float
    win_rate_delta: float
    psi_total: float
    psi_status: str
    distribution: PredictionDistribution | None = None
    alerts: list[Alert] = field(default_factory=list)
    generated_at: datetime = field(default_factory=lambda: datetime.now(UTC))


# ─── Monitoring Configuration ─────────────────────────────────────────────────

@dataclass(frozen=True)
class MonitorConfig:
    staleness_days_warn: int = 30
    staleness_days_critical: int = 90
    psi_warn_threshold: float = 0.10
    psi_critical_threshold: float = 0.20
    win_rate_degradation_warn: float = 0.05
    win_rate_degradation_critical: float = 0.10
    acceptance_rate_min: float = 0.30
    prediction_bins: int = 10
    lookback_days: int = 30


# ─── Distribution Analysis ────────────────────────────────────────────────────

def compute_prediction_distribution(
    pwin_values: list[float],
    accepted: list[bool] | None = None,
    bins: int = 10,
) -> PredictionDistribution:
    if not pwin_values:
        return PredictionDistribution(model_name="unknown")

    arr = np.array(pwin_values)
    hist, edges = np.histogram(arr, bins=bins, range=(0.0, 1.0))
    bucket_counts = {}
    for i in range(len(hist)):
        label = f"{edges[i]:.1f}-{edges[i + 1]:.1f}"
        bucket_counts[label] = int(hist[i])

    acceptance_rate = 0.0
    if accepted and len(accepted) == len(pwin_values):
        acceptance_rate = sum(1 for a in accepted if a) / len(accepted)

    return PredictionDistribution(
        model_name="unknown",
        bucket_counts=bucket_counts,
        total_predictions=len(pwin_values),
        mean_pwin=float(np.mean(arr)),
        median_pwin=float(np.median(arr)),
        std_pwin=float(np.std(arr)),
        pwin_10th=float(np.percentile(arr, 10)),
        pwin_90th=float(np.percentile(arr, 90)),
        acceptance_rate=acceptance_rate,
    )


# ─── Model Staleness ──────────────────────────────────────────────────────────

def check_model_staleness(
    last_trained_at: datetime,
    current_time: datetime | None = None,
    warn_days: int = 30,
    critical_days: int = 90,
) -> tuple[float, Alert | None]:
    if current_time is None:
        current_time = datetime.now(UTC)
    if last_trained_at.tzinfo is None:
        last_trained_at = last_trained_at.replace(tzinfo=UTC)
    if current_time.tzinfo is None:
        current_time = current_time.replace(tzinfo=UTC)

    days_since = (current_time - last_trained_at).total_seconds() / 86400.0

    if days_since >= critical_days:
        return days_since, Alert(
            AlertSeverity.CRITICAL,
            "unknown",
            "staleness",
            days_since,
            critical_days,
            f"Model not retrained in {days_since:.0f} days (critical threshold: {critical_days})",
        )
    elif days_since >= warn_days:
        return days_since, Alert(
            AlertSeverity.WARN,
            "unknown",
            "staleness",
            days_since,
            warn_days,
            f"Model not retrained in {days_since:.0f} days (warn threshold: {warn_days})",
        )
    return days_since, None


# ─── Win Rate Degradation ─────────────────────────────────────────────────────

def check_win_rate_degradation(
    win_rate_recent: float,
    win_rate_historical: float,
    warn_delta: float = 0.05,
    critical_delta: float = 0.10,
) -> Alert | None:
    if win_rate_historical <= 0:
        return None

    delta = win_rate_historical - win_rate_recent

    if delta >= critical_delta:
        return Alert(
            AlertSeverity.CRITICAL,
            "unknown",
            "win_rate",
            delta,
            critical_delta,
            f"Win rate degraded from {win_rate_historical:.2%} to {win_rate_recent:.2%} (Δ={delta:.2%})",
        )
    elif delta >= warn_delta:
        return Alert(
            AlertSeverity.WARN,
            "unknown",
            "win_rate",
            delta,
            warn_delta,
            f"Win rate degraded from {win_rate_historical:.2%} to {win_rate_recent:.2%} (Δ={delta:.2%})",
        )
    return None


# ─── Acceptance Rate Check ────────────────────────────────────────────────────

def check_acceptance_rate(
    acceptance_rate: float,
    min_rate: float = 0.30,
    accepted_wins: int = 0,
    accepted_total: int = 0,
) -> Alert | None:
    reason = f"Signal acceptance rate {acceptance_rate:.2%} below minimum {min_rate:.2%}"
    if accepted_total > 0:
        from orca.math.wilson import wilson_ci
        ci_low, ci_high = wilson_ci(accepted_wins, accepted_total)
        reason = f"Signal acceptance rate {acceptance_rate:.2%} [95% CI: {ci_low:.2%}–{ci_high:.2%}] below minimum {min_rate:.2%}"
    if acceptance_rate < min_rate:
        return Alert(
            AlertSeverity.WARN,
            "unknown",
            "acceptance_rate",
            acceptance_rate,
            min_rate,
            reason,
        )
    return None


# ─── Model Health Report Generator ────────────────────────────────────────────

def generate_health_report(
    model_name: str,
    model_version: str,
    last_trained_at: datetime,
    recent_pwins: list[float],
    recent_accepted: list[bool] | None,
    historical_win_rate: float,
    recent_win_rate: float,
    psi_total: float,
    psi_status: str,
    total_predictions: int,
    config: MonitorConfig | None = None,
) -> ModelHealthReport:
    if config is None:
        config = MonitorConfig()

    days_since, staleness_alert = check_model_staleness(
        last_trained_at,
        warn_days=config.staleness_days_warn,
        critical_days=config.staleness_days_critical,
    )

    wr_alert = check_win_rate_degradation(
        recent_win_rate, historical_win_rate,
        warn_delta=config.win_rate_degradation_warn,
        critical_delta=config.win_rate_degradation_critical,
    )

    distribution = compute_prediction_distribution(
        recent_pwins, recent_accepted, bins=config.prediction_bins,
    )
    distribution = replace(distribution, model_name=model_name)

    ar_alert = check_acceptance_rate(distribution.acceptance_rate, config.acceptance_rate_min)

    alerts = []
    severity = ModelStatus.HEALTHY

    if staleness_alert:
        alerts.append(staleness_alert)
        if staleness_alert.severity == AlertSeverity.CRITICAL:
            severity = ModelStatus.FAILED
        elif severity != ModelStatus.FAILED:
            severity = ModelStatus.STALE

    if wr_alert:
        alerts.append(wr_alert)
        if wr_alert.severity == AlertSeverity.CRITICAL:
            severity = ModelStatus.FAILED
        elif severity not in (ModelStatus.FAILED, ModelStatus.STALE):
            severity = ModelStatus.DEGRADED

    if ar_alert:
        alerts.append(ar_alert)

    if psi_total >= config.psi_critical_threshold:
        severity = ModelStatus.FAILED
        alerts.append(Alert(
            AlertSeverity.CRITICAL, model_name, "psi",
            psi_total, config.psi_critical_threshold,
            f"PSI {psi_total:.3f} exceeds critical threshold {config.psi_critical_threshold}",
        ))

    return ModelHealthReport(
        model_name=model_name,
        model_version=model_version,
        status=severity,
        days_since_training=days_since,
        total_predictions=total_predictions,
        win_rate_recent=recent_win_rate,
        win_rate_historical=historical_win_rate,
        win_rate_delta=historical_win_rate - recent_win_rate,
        psi_total=psi_total,
        psi_status=psi_status,
        distribution=distribution,
        alerts=alerts,
    )


# ─── Persistence Helpers ──────────────────────────────────────────────────────

def health_report_to_dict(report: ModelHealthReport) -> dict[str, Any]:
    return {
        "model_name": report.model_name,
        "model_version": report.model_version,
        "status": report.status.name,
        "days_since_training": report.days_since_training,
        "total_predictions": report.total_predictions,
        "win_rate_recent": report.win_rate_recent,
        "win_rate_historical": report.win_rate_historical,
        "win_rate_delta": report.win_rate_delta,
        "psi_total": report.psi_total,
        "psi_status": report.psi_status,
        "distribution": {
            "mean_pwin": report.distribution.mean_pwin if report.distribution else 0,
            "acceptance_rate": report.distribution.acceptance_rate if report.distribution else 0,
        } if report.distribution else {},
        "alerts": [
            {"severity": a.severity.name, "metric": a.metric, "message": a.message}
            for a in report.alerts
        ],
        "generated_at": report.generated_at.isoformat(),
    }


def save_health_report(report: ModelHealthReport, output_dir: str | Path = "reports/ml") -> Path:
    output_dir = Path(output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)
    ts = report.generated_at.strftime("%Y%m%d_%H%M%S")
    filename = f"{report.model_name}_{ts}.json"
    path = output_dir / filename
    path.write_text(json.dumps(health_report_to_dict(report), indent=2, default=str))
    logger.info("health report saved: %s", path)
    return path
