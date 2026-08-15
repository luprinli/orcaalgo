from __future__ import annotations

from dataclasses import dataclass, field
from datetime import UTC

from orca.math.brier import BinStats, murphy_decomposition


@dataclass(frozen=True)
class SegmentReport:
    name: str
    n: int
    brier: float
    reliability: float
    resolution: float
    uncertainty: float
    bin_stats: list[BinStats] = field(default_factory=list)
    needs_calibration: bool = False

    @property
    def sufficient_data(self) -> bool:
        return self.n >= 30


@dataclass(frozen=True)
class CalibrationReport:
    overall: SegmentReport
    segments: dict[str, SegmentReport] = field(default_factory=dict)
    generated_at: str = ""


def run_calibration_audit(trades: list[dict]) -> CalibrationReport:
    """Run full calibration audit on trade records.

    Each trade dict must have: confidence (float), outcome (str: 'win'/'loss').
    """
    from datetime import datetime

    predictions = [float(t["confidence"]) for t in trades]
    outcomes = [1 if t.get("outcome") == "win" else 0 for t in trades]

    if len(trades) == 0:
        return CalibrationReport(
            overall=SegmentReport(
                name="overall",
                n=0,
                brier=0.0,
                reliability=0.0,
                resolution=0.0,
                uncertainty=0.0,
                bin_stats=[],
                needs_calibration=False,
            ),
            generated_at=datetime.now(UTC).isoformat(),
        )

    murphy = murphy_decomposition(predictions, outcomes)
    overall = SegmentReport(
        name="overall",
        n=len(trades),
        brier=murphy.brier,
        reliability=murphy.reliability,
        resolution=murphy.resolution,
        uncertainty=murphy.uncertainty,
        bin_stats=murphy.bin_stats,
        needs_calibration=murphy.reliability > 0.01,
    )

    segments: dict[str, SegmentReport] = {}
    side_groups: dict[str, list[int]] = {}
    for i, t in enumerate(trades):
        side = t.get("side", "unknown")
        side_groups.setdefault(side, []).append(i)

    for side, idxs in side_groups.items():
        side_preds = [predictions[i] for i in idxs]
        side_outs = [outcomes[i] for i in idxs]
        if len(side_preds) >= 10:
            sm = murphy_decomposition(side_preds, side_outs)
            segments[f"side:{side}"] = SegmentReport(
                name=f"Side: {side}",
                n=len(side_preds),
                brier=sm.brier,
                reliability=sm.reliability,
                resolution=sm.resolution,
                uncertainty=sm.uncertainty,
                bin_stats=sm.bin_stats,
                needs_calibration=sm.reliability > 0.01 and len(side_preds) >= 200,
            )

    return CalibrationReport(
        overall=overall,
        segments=segments,
        generated_at=datetime.now(UTC).isoformat(),
    )
