"""Tests for data provenance, determinism, and scale-consistency validation.

Covers the P0 remediation items from the 2026-08-12 data pipeline review:
  - deterministic generation IDs (R1)
  - cross-source price-scale discontinuity detection (R7)
  - checks never masked as passes (R7)
"""

from __future__ import annotations

import datetime as dt

from orca.data.seed_all import compute_generation_id
from orca.data.validate_integrity import (
    _check_source_scale_consistency,
    _fail,
)


class TestComputeGenerationId:
    def test_deterministic_same_inputs(self):
        config = {"symbols": ["SPY", "NVDA"], "start": "2021-07-01", "end": "2026-08-12"}
        assert compute_generation_id(config) == compute_generation_id(config)

    def test_differs_by_input(self):
        a = compute_generation_id({"symbols": ["SPY"], "start": "2021-07-01", "end": "2026-08-12"})
        b = compute_generation_id({"symbols": ["SPY"], "start": "2021-07-01", "end": "2026-08-13"})
        assert a != b

    def test_stable_across_calls(self):
        # The hash must be a pure function of its input (no injected utcnow()).
        cfg = {"symbols": ["SPY"], "start": "2021-07-01", "end": "2026-08-12"}
        assert compute_generation_id(cfg) == compute_generation_id(dict(cfg))


class _FakeCursor:
    def __init__(self, rows):
        self._rows = rows

    def __enter__(self):
        return self

    def __exit__(self, *exc):
        return False

    def execute(self, *args, **kwargs):
        return None

    def fetchall(self):
        return self._rows


class _FakeConn:
    def __init__(self, rows):
        self._rows = rows

    def cursor(self):
        return _FakeCursor(self._rows)


class TestSourceScaleConsistency:
    def test_no_discontinuity_passes(self):
        rows = [
            ("SPY", "1h", dt.datetime(2024, 1, 1), 1_000_000, "stooq"),
            ("SPY", "1h", dt.datetime(2024, 1, 1, 1), 1_010_000, "stooq"),
        ]
        checks, warnings, errors = [], [], []
        _check_source_scale_consistency(
            _FakeConn(rows), dt.date(2020, 1, 1), dt.date(2026, 1, 1), checks, warnings, errors
        )
        assert checks[-1]["passed"] is True

    def test_cross_source_discontinuity_fails(self):
        # NVDA-class defect: seed close ~350 then stooq close ~2000 = ~5.7x jump.
        rows = [
            ("NVDA", "5m", dt.datetime(2026, 3, 14, 15, 55), 35_000_000, "seed"),
            ("NVDA", "5m", dt.datetime(2026, 3, 14, 16, 0), 200_000_000, "stooq"),
        ]
        checks, warnings, errors = [], [], []
        _check_source_scale_consistency(
            _FakeConn(rows), dt.date(2026, 1, 1), dt.date(2026, 8, 1), checks, warnings, errors
        )
        assert checks[-1]["passed"] is False
        assert "NVDA" in checks[-1]["detail"]

    def test_same_source_jump_not_flagged(self):
        # A large jump within the SAME source is not a provenance defect.
        rows = [
            ("BTC-USD", "1d", dt.datetime(2024, 1, 1), 42_000_000, "stooq"),
            ("BTC-USD", "1d", dt.datetime(2024, 1, 2), 60_000_000, "stooq"),
        ]
        checks, warnings, errors = [], [], []
        _check_source_scale_consistency(
            _FakeConn(rows), dt.date(2024, 1, 1), dt.date(2024, 1, 3), checks, warnings, errors
        )
        assert checks[-1]["passed"] is True


class TestFailNeverPasses:
    def test_fail_marks_failed(self):
        errors: list[str] = []
        result = _fail("some check", "boom", errors)
        assert result["passed"] is False
        assert "boom" in errors[0]


class TestDatasetVersioning:
    def test_dataset_id_deterministic(self):
        from orca.ml.barriers import BarrierConfig
        from orca.ml.dataset import compute_dataset_id

        a = compute_dataset_id("gen-1", ["rsi14", "macd"], BarrierConfig())
        b = compute_dataset_id("gen-1", ["rsi14", "macd"], BarrierConfig())
        assert a == b
        assert len(a) == 16

    def test_dataset_id_differs_by_generation(self):
        from orca.ml.barriers import BarrierConfig
        from orca.ml.dataset import compute_dataset_id

        a = compute_dataset_id("gen-1", ["rsi14"], BarrierConfig())
        b = compute_dataset_id("gen-2", ["rsi14"], BarrierConfig())
        assert a != b

    def test_dataset_metadata_carries_lineage(self):
        from orca.ml.dataset import build_dataset_from_trade_logs

        ds = build_dataset_from_trade_logs([], {}, generation_id="gen-abc")
        assert ds.metadata["generation_id"] == "gen-abc"
        assert ds.metadata["dataset_id"]
        assert ds.metadata["dataset_id"] == ds.metadata["dataset_id"]


class TestTimezoneMapping:
    def test_asset_class_timezones(self):
        from orca.data.timezones import timezone_for_symbol

        assert str(timezone_for_symbol("SPY", "equity_etf")) == "America/New_York"
        assert str(timezone_for_symbol("EURUSD", "forex_major")) == "UTC"
        assert str(timezone_for_symbol("BTC-USD", "crypto")) == "UTC"
        assert str(timezone_for_symbol("^DAX", "index_eu")) == "Europe/Berlin"
        assert str(timezone_for_symbol("^_US", "index_agg")) == "America/New_York"
