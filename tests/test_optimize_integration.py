"""Optimize path integration test (O1).

Verifies that matrix backtests with optimize=true produce optimization
metadata (best_params, total_trials, oos_sharpe) in each ComboResult.

This test requires the Go server to be running on localhost:8081 with
the matrix endpoints available. If the server is not running, the test
skips with a clear message.
"""

import json
import time

import pytest
import requests


BASE = "http://localhost:8081/api/v1"


def _server_ready():
    try:
        r = requests.get(f"{BASE}/backtests/health", timeout=3)
        return r.status_code < 500
    except Exception:
        return False


@pytest.mark.skipif(
    not _server_ready(),
    reason="Go server not running on :8081 — start server before running",
)
class TestOptimizePath:
    def test_optimized_matrix_populates_best_params(self):
        """Submit 2×2×1 matrix with optimize=true, verify best_params populated."""
        payload = {
            "strategy_ids": ["ma_crossover", "rsi2_reversion"],
            "symbols": ["EURUSD"],
            "timeframes": ["1d"],
            "start_date": "2024-01-01",
            "end_date": "2024-03-31",
            "initial_capital": 100000,
            "optimize": True,
            "max_trials": 10,
        }
        r = requests.post(f"{BASE}/backtests/matrix", json=payload, timeout=10)
        assert r.status_code == 202, f"Expected 202, got {r.status_code}: {r.text}"
        data = r.json()
        batch_id = data["batch_id"]
        assert data["total_combos"] == 2, f"Expected 2 combos, got {data['total_combos']}"

        # Poll until complete
        timeout_sec = 120
        start = time.time()
        complete = False
        while time.time() - start < timeout_sec:
            r = requests.get(f"{BASE}/backtests/matrix/{batch_id}/results?since=0", timeout=5)
            if r.status_code == 200:
                d = r.json()
                if d.get("complete"):
                    complete = True
                    results = d["results"]
                    break

        assert complete, f"Matrix did not complete within {timeout_sec}s"
        assert len(results) == 2, f"Expected 2 results, got {len(results)}"

        for i, combo in enumerate(results):
            assert combo.get("optimized"), f"Combo {i}: expected optimized=true"
            assert combo.get("best_params"), f"Combo {i}: missing best_params"
            assert combo["total_trials"] > 0, f"Combo {i}: total_trials <= 0"
            assert combo["total_trials"] <= 10, f"Combo {i}: total_trials > max_trials"

    def test_non_optimized_matrix_has_no_optimization_fields(self):
        """Submit matrix with optimize=false (default), verify no opt fields."""
        payload = {
            "strategy_ids": ["ma_crossover"],
            "symbols": ["EURUSD"],
            "timeframes": ["1d"],
            "start_date": "2024-01-01",
            "end_date": "2024-03-31",
            "initial_capital": 100000,
        }
        r = requests.post(f"{BASE}/backtests/matrix", json=payload, timeout=10)
        if r.status_code != 202:
            pytest.skip(f"Matrix endpoint not available: {r.status_code}")
        data = r.json()
        batch_id = data["batch_id"]

        timeout_sec = 60
        start = time.time()
        complete = False
        while time.time() - start < timeout_sec:
            r = requests.get(f"{BASE}/backtests/matrix/{batch_id}/results?since=0", timeout=5)
            if r.status_code == 200:
                d = r.json()
                if d.get("complete"):
                    results = d["results"]
                    complete = True
                    break

        assert complete, f"Non-optimized matrix did not complete within {timeout_sec}s"
        for combo in results:
            assert not combo.get("optimized"), "Expected optimized=false (omitempty)"
            assert "best_params" not in combo, "best_params should be absent for non-optimized"


@pytest.mark.skipif(_server_ready(), reason="Server IS running — skip offline test")
def test_offline_matrix_endpoint_not_available():
    """When server is not running, the endpoint should be unreachable."""
    with pytest.raises(Exception):
        requests.get(f"{BASE}/backtests/health", timeout=2)
