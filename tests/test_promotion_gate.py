"""Tests for the promotion gate (R15 / Fix 4)."""

from __future__ import annotations

import csv
import io

from orca.sizing.promotion_gate import apply_promotion_gate


def _write_csv(rows: list[dict]) -> str:
    buf = io.StringIO()
    w = csv.DictWriter(buf, fieldnames=list(rows[0].keys()))
    w.writeheader()
    for r in rows:
        w.writerow(r)
    return buf.getvalue()


def _run(rows: list[dict]):
    import os
    import tempfile

    fd, path = tempfile.mkstemp(suffix=".csv")
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            f.write(_write_csv(rows))
        return apply_promotion_gate(path)
    finally:
        os.unlink(path)


def test_gate_rejects_weak_sharpe():
    rows = [
        {
            "Strategy": "s1",
            "Symbol": "X",
            "Tf": "1d",
            "Trades": "30",
            "Sharpe": "0.01",
            "WfISSharpe": "",
            "WfOOSSharpe": "",
        },
        {
            "Strategy": "s2",
            "Symbol": "Y",
            "Tf": "1d",
            "Trades": "30",
            "Sharpe": "-0.5",
            "WfISSharpe": "",
            "WfOOSSharpe": "",
        },
        {
            "Strategy": "s3",
            "Symbol": "Z",
            "Tf": "1d",
            "Trades": "5",
            "Sharpe": "3.0",
            "WfISSharpe": "",
            "WfOOSSharpe": "",
        },
    ]
    result = _run(rows)
    assert result.n_tests == 3
    assert result.n_candidates == 1  # only s1 (Sharpe>0, trades>=20)
    assert result.bh_significant == 0  # Sharpe 0.01 is not significant


def test_gate_passes_strong_sharpe():
    rows = [
        {
            "Strategy": "s1",
            "Symbol": "X",
            "Tf": "1d",
            "Trades": "100",
            "Sharpe": "2.0",
            "WfISSharpe": "2.0",
            "WfOOSSharpe": "1.8",
        },
        {
            "Strategy": "s2",
            "Symbol": "Y",
            "Tf": "1d",
            "Trades": "30",
            "Sharpe": "-0.2",
            "WfISSharpe": "",
            "WfOOSSharpe": "",
        },
    ]
    result = _run(rows)
    assert result.n_candidates == 1
    assert result.bh_significant == 1
    assert result.bonferroni_significant == 1
    assert len(result.survivors) == 1
    assert result.passed is True
    assert result.survivors[0]["strategy"] == "s1"
    assert result.survivors[0]["walk_forward"] == "pass"


def test_gate_walk_forward_degradation():
    # Strong IS Sharpe but OOS degraded >20% -> excluded from survivors.
    rows = [
        {
            "Strategy": "s1",
            "Symbol": "X",
            "Tf": "1d",
            "Trades": "100",
            "Sharpe": "2.0",
            "WfISSharpe": "2.0",
            "WfOOSSharpe": "0.5",
        },
    ]
    result = _run(rows)
    assert result.bh_significant == 1
    assert len(result.survivors) == 0  # OOS 0.5 < 0.8 * 2.0 (degradation >20%)
    assert result.passed is False


def test_gate_trade_distribution_negative_median_rejected():
    # Strong Sharpe + significant, but median trade PnL is negative -> excluded.
    rows = [
        {
            "Strategy": "s1",
            "Symbol": "X",
            "Tf": "1d",
            "Trades": "100",
            "Sharpe": "2.0",
            "WfISSharpe": "",
            "WfOOSSharpe": "",
            "MedianTradePnL": "-5.0",
            "UniqueTickers": "5",
        },
    ]
    result = _run(rows)
    assert result.bh_significant == 1
    assert len(result.survivors) == 0
    assert result.passed is False


def test_gate_trade_distribution_single_ticker_rejected():
    rows = [
        {
            "Strategy": "s1",
            "Symbol": "X",
            "Tf": "1d",
            "Trades": "100",
            "Sharpe": "2.0",
            "WfISSharpe": "",
            "WfOOSSharpe": "",
            "MedianTradePnL": "5.0",
            "UniqueTickers": "1",
        },
    ]
    result = _run(rows)
    assert len(result.survivors) == 0


def test_gate_trade_distribution_absent_is_na():
    # No distribution columns -> gate is n/a and passes.
    rows = [
        {
            "Strategy": "s1",
            "Symbol": "X",
            "Tf": "1d",
            "Trades": "100",
            "Sharpe": "2.0",
            "WfISSharpe": "",
            "WfOOSSharpe": "",
        },
    ]
    result = _run(rows)
    assert len(result.survivors) == 1
    assert result.survivors[0]["trade_distribution"] == "n/a"


def test_gate_trade_distribution_passes():
    rows = [
        {
            "Strategy": "s1",
            "Symbol": "X",
            "Tf": "1d",
            "Trades": "100",
            "Sharpe": "2.0",
            "WfISSharpe": "",
            "WfOOSSharpe": "",
            "MedianTradePnL": "5.0",
            "UniqueTickers": "5",
        },
    ]
    result = _run(rows)
    assert len(result.survivors) == 1
    assert result.survivors[0]["trade_distribution"] == "pass"
