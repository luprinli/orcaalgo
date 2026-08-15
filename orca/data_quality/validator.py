"""Data quality validation for candle data."""

from __future__ import annotations

from dataclasses import dataclass, field


@dataclass(frozen=True)
class DataQualityCheck:
    check_name: str
    status: str
    message: str
    symbol: str = ""
    timeframe: str = ""


@dataclass(frozen=True)
class DataQualityReport:
    checks: list[DataQualityCheck] = field(default_factory=list)
    passed: int = 0
    warned: int = 0
    failed: int = 0


TIMEFRAME_INTERVALS = {
    "1d": 86400,
    "1h": 3600,
    "5m": 300,
    "15m": 900,
    "30m": 1800,
}


def run_data_quality_checks(
    candles_by_symbol: dict | None = None,
) -> DataQualityReport:
    import numpy as np

    checks: list[DataQualityCheck] = []
    warned = 0
    failed = 0

    if candles_by_symbol is None:
        try:
            from orca.data_quality import _load_candles_from_db

            candles_by_symbol = _load_candles_from_db()
        except Exception as e:
            return DataQualityReport(
                checks=[DataQualityCheck("db_connect", "fail", str(e))],
                failed=1,
            )

    if not candles_by_symbol:
        checks.append(DataQualityCheck("data_exists", "warn", "No candle data found"))
        warned += 1
        return DataQualityReport(checks=checks, warned=warned, failed=failed)

    total_symbols = len(candles_by_symbol)
    checks.append(
        DataQualityCheck("data_exists", "pass", f"Found data for {total_symbols} symbols")
    )

    for symbol, candles in candles_by_symbol.items():
        if not candles:
            checks.append(
                DataQualityCheck(
                    "empty_data",
                    "warn",
                    "No candles for symbol",
                    symbol=symbol,
                )
            )
            warned += 1
            continue

        closes = np.array([getattr(c, "close", 0) or 0 for c in candles])
        volumes = np.array([getattr(c, "volume", 0) or 0 for c in candles])

        if len(closes) < 2:
            checks.append(
                DataQualityCheck(
                    "insufficient_data",
                    "warn",
                    f"Only {len(closes)} candles",
                    symbol=symbol,
                )
            )
            warned += 1
            continue

        returns = np.diff(closes) / closes[:-1]

        max_gap = 0
        consecutive_zeros = 0
        for r in returns:
            if abs(r) < 1e-12:
                consecutive_zeros += 1
                if consecutive_zeros > max_gap:
                    max_gap = consecutive_zeros
            else:
                consecutive_zeros = 0

        if max_gap > 5:
            checks.append(
                DataQualityCheck(
                    "gap_detected",
                    "warn",
                    f"Up to {max_gap} consecutive identical closes",
                    symbol=symbol,
                )
            )
            warned += 1

        max_ret = np.max(np.abs(returns)) * 100 if len(returns) > 0 else 0
        if max_ret > 50:
            checks.append(
                DataQualityCheck(
                    "outlier_detected",
                    "warn",
                    f"Max daily return {max_ret:.1f}% exceeds 50% threshold",
                    symbol=symbol,
                )
            )
            warned += 1

        zero_vol = np.sum(volumes <= 0) if len(volumes) > 0 else 0
        if zero_vol > 0:
            checks.append(
                DataQualityCheck(
                    "zero_volume",
                    "warn",
                    f"{zero_vol} candles with zero volume (stale data)",
                    symbol=symbol,
                )
            )
            warned += 1

    passed = total_symbols
    summary_status = "pass" if warned == 0 and failed == 0 else "warn"
    checks.append(
        DataQualityCheck(
            "summary",
            summary_status,
            f"Checked {total_symbols} symbols: {passed} pass, {warned} warn, {failed} fail",
        )
    )

    return DataQualityReport(checks=checks, passed=passed, warned=warned, failed=failed)
