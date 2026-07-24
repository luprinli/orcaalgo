from __future__ import annotations

import random

from orca.attribution.slicer import (
    AttributionReport,
    SliceStats,
    _compute_slice,
    attribute_pnl,
)


def _make_trades(n: int, seed: int = 42) -> list[dict]:
    random.seed(seed)
    symbols = ["SPY", "QQQ", "AAPL", "MSFT", "TSLA"]
    trades = []
    for _ in range(n):
        s = random.choice(symbols)
        side = random.choice(["BUY", "SELL"])
        entry = random.uniform(20, 500)
        conf = random.uniform(0.5, 0.9)
        pnl = random.uniform(-100, 200)
        trades.append({
            "symbol": s,
            "side": side,
            "entry_price": round(entry, 2),
            "exit_price": round(entry * (1 + pnl / (entry * 10)), 2),
            "quantity": random.randint(1, 100),
            "pnl": round(pnl, 2),
            "cost": round(entry * random.randint(1, 100), 2),
            "confidence": round(conf, 4),
            "outcome": "win" if pnl > 0 else "loss",
        })
    return trades


class TestComputeSlice:
    def test_empty_slice(self):
        stats = _compute_slice([])
        assert stats.n == 0
        assert stats.wins == 0
        assert stats.hit_rate == 0.0
        assert stats.hit_rate_ci_low == 0.0
        assert stats.hit_rate_ci_high == 1.0
        assert stats.total_pnl == 0.0
        assert stats.roi == 0.0

    def test_slice_with_all_wins(self):
        trades = [
            {"pnl": 50.0, "cost": 1000.0},
            {"pnl": 30.0, "cost": 500.0},
            {"pnl": 10.0, "cost": 200.0},
        ]
        stats = _compute_slice(trades)
        assert stats.n == 3
        assert stats.wins == 3
        assert stats.hit_rate == 1.0
        assert stats.total_pnl == 90.0
        assert stats.total_cost == 1700.0
        assert stats.roi == 90.0 / 1700.0

    def test_slice_with_all_losses(self):
        trades = [
            {"pnl": -20.0, "cost": 1000.0},
            {"pnl": -10.0, "cost": 500.0},
        ]
        stats = _compute_slice(trades)
        assert stats.n == 2
        assert stats.wins == 0
        assert stats.hit_rate == 0.0
        assert stats.total_pnl == -30.0

    def test_sufficient_data_property(self):
        stats_small = SliceStats(n=10, wins=5, hit_rate=0.5, hit_rate_ci_low=0.2, hit_rate_ci_high=0.8, total_pnl=100.0, total_cost=1000.0, roi=0.1)
        assert stats_small.sufficient_data is False
        stats_large = SliceStats(n=50, wins=25, hit_rate=0.5, hit_rate_ci_low=0.3, hit_rate_ci_high=0.7, total_pnl=500.0, total_cost=5000.0, roi=0.1)
        assert stats_large.sufficient_data is True

    def test_avg_win(self):
        stats = SliceStats(n=4, wins=2, hit_rate=0.5, hit_rate_ci_low=0.1, hit_rate_ci_high=0.9, total_pnl=200.0, total_cost=1000.0, roi=0.2)
        assert stats.avg_win == 50.0


class TestAttributePnl:
    def test_empty_trades(self):
        report = attribute_pnl([])
        assert isinstance(report, AttributionReport)
        assert report.overall.n == 0

    def test_full_attribution(self):
        trades = _make_trades(50)
        report = attribute_pnl(trades)
        assert report.overall.n == 50
        assert isinstance(report.overall, SliceStats)
        assert report.generated_at != ""

    def test_by_side_slices(self):
        trades = _make_trades(50)
        report = attribute_pnl(trades)
        assert len(report.by_side) >= 1
        for side_key, stats in report.by_side.items():
            assert isinstance(stats, SliceStats)
            assert stats.n > 0

    def test_by_price_bucket_slices(self):
        trades = _make_trades(100)
        report = attribute_pnl(trades)
        assert len(report.by_price_bucket) >= 1
        for bucket_key, stats in report.by_price_bucket.items():
            assert isinstance(stats, SliceStats)
            assert bucket_key in ("0-30", "30-50", "50-70", "70+")

    def test_by_edge_bucket_slices(self):
        trades = _make_trades(100)
        report = attribute_pnl(trades)
        assert len(report.by_edge_bucket) >= 1
        for bucket_key, stats in report.by_edge_bucket.items():
            assert isinstance(stats, SliceStats)
            assert bucket_key in ("0-5%", "5-10%", "10-20%", "20%+")

    def test_hit_rate_between_zero_and_one(self):
        trades = _make_trades(50)
        report = attribute_pnl(trades)
        assert 0.0 <= report.overall.hit_rate <= 1.0

    def test_wilson_ci_bounds(self):
        trades = _make_trades(50)
        report = attribute_pnl(trades)
        assert 0.0 <= report.overall.hit_rate_ci_low <= report.overall.hit_rate
        assert report.overall.hit_rate <= report.overall.hit_rate_ci_high <= 1.0
