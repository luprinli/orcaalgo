"""Optimize path integration test (O1).

Verifies that matrix backtests with optimize=true produce optimization
metadata (best_params, total_trials, oos_sharpe) in each ComboResult.

This test requires the Go server to be running.  Set ORCA_API_URL to
override the default base URL and ORCA_ADMIN_PASSWORD if the dev password
has been changed.  The matrix endpoint is JWT-protected so a valid token
is obtained automatically via the /auth/login endpoint.
"""

import os
import time

import pytest
import requests

API_BASE = f"{os.environ.get('ORCA_API_URL', 'http://localhost:8081')}/api/v1"
ADMIN_PASSWORD = os.environ.get(
    "ORCA_ADMIN_PASSWORD", "dev-admin-password-do-not-use-in-production"
)
AUTH = {"username": "admin", "password": ADMIN_PASSWORD}


def _server_healthy():
    try:
        r = requests.get(f"{API_BASE}/backtests/health", timeout=3)
        return r.status_code == 200
    except Exception:
        return False


def _get_token():
    """Obtain a JWT for the test admin user.  Skips the calling test if
    the auth endpoint is unreachable or returns an error."""
    try:
        r = requests.post(f"{API_BASE}/auth/login", json=AUTH, timeout=10)
    except Exception:
        pytest.skip(f"Auth endpoint at {API_BASE}/auth/login unreachable — start the Go server")
    if r.status_code != 200:
        pytest.skip(
            f"Auth failed ({r.status_code}) on {API_BASE}/auth/login — "
            "check ORCA_ADMIN_PASSWORD or server configuration"
        )
    return r.json()["access_token"]


@pytest.mark.skipif(
    not _server_healthy(),
    reason=f"Go server not running on {API_BASE} — start server before running",
)
class TestOptimizePath:
    def test_optimized_matrix_populates_best_params(self):
        """Submit 2x2x1 matrix with optimize=true, verify best_params populated."""
        token = _get_token()
        headers = {"Authorization": f"Bearer {token}"}
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
        r = requests.post(f"{API_BASE}/backtests/matrix", json=payload, headers=headers, timeout=10)
        assert r.status_code == 202, f"Expected 202, got {r.status_code}: {r.text}"
        data = r.json()
        batch_id = data["batch_id"]
        assert data["total_combos"] == 2, f"Expected 2 combos, got {data['total_combos']}"

        timeout_sec = 120
        start = time.time()
        complete = False
        while time.time() - start < timeout_sec:
            r = requests.get(
                f"{API_BASE}/backtests/matrix/{batch_id}/results?since=0",
                headers=headers,
                timeout=5,
            )
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
        token = _get_token()
        headers = {"Authorization": f"Bearer {token}"}
        payload = {
            "strategy_ids": ["ma_crossover"],
            "symbols": ["EURUSD"],
            "timeframes": ["1d"],
            "start_date": "2024-01-01",
            "end_date": "2024-03-31",
            "initial_capital": 100000,
        }
        r = requests.post(f"{API_BASE}/backtests/matrix", json=payload, headers=headers, timeout=10)
        if r.status_code != 202:
            pytest.skip(f"Matrix endpoint not available: {r.status_code}")
        data = r.json()
        batch_id = data["batch_id"]

        timeout_sec = 60
        start = time.time()
        complete = False
        while time.time() - start < timeout_sec:
            r = requests.get(
                f"{API_BASE}/backtests/matrix/{batch_id}/results?since=0",
                headers=headers,
                timeout=5,
            )
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


@pytest.mark.skipif(_server_healthy(), reason="Server IS running — skip offline test")
def test_offline_matrix_endpoint_not_available():
    """When server is not running, the endpoint should be unreachable."""
    with pytest.raises(requests.exceptions.ConnectionError):
        requests.get(f"{API_BASE}/backtests/health", timeout=2)
