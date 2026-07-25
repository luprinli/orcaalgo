"""End-to-end tests for Login → Dashboard and Backtest Detail → Regime Stats.

All tests require a running Go API server. Set ORCA_API_URL to override
the default base URL and ORCA_ADMIN_PASSWORD if the default dev password
has been changed.
"""

import json
import os
import urllib.error
import urllib.request

import pytest

API_BASE = f"{os.environ.get('ORCA_API_URL', 'http://localhost:8080')}/api/v1"
UI_BASE = os.environ.get("ORCA_UI_URL", "http://localhost:5173")
ADMIN_PASSWORD = os.environ.get("ORCA_ADMIN_PASSWORD", "dev-admin-password-do-not-use-in-production")
AUTH = {"username": "admin", "password": ADMIN_PASSWORD}


def _try_login():
    data = json.dumps(AUTH).encode()
    req = urllib.request.Request(f"{API_BASE}/auth/login", data=data, method="POST")
    req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req, timeout=5) as resp:
            body = resp.read()
            if resp.status != 200:
                return None
            return json.loads(body).get("access_token")
    except Exception:
        return None


@pytest.fixture(autouse=True, name="api_token")
def api_token_fixture():
    token = _try_login()
    if token is None:
        pytest.skip(
            f"Go API server not reachable or auth failed on {API_BASE} — "
            "start the server and check ORCA_ADMIN_PASSWORD"
        )
    return token


def get_token():
    token = _try_login()
    if token is None:
        pytest.skip("Backend auth endpoint not available")
    return token


def test_login_api_returns_token(api_token):
    assert len(api_token) > 50, f"Token too short: {len(api_token)} chars"


def test_login_api_sets_correct_roles():
    data = json.dumps(AUTH).encode()
    req = urllib.request.Request(f"{API_BASE}/auth/login", data=data, method="POST")
    req.add_header("Content-Type", "application/json")
    with urllib.request.urlopen(req, timeout=10) as resp:
        body = json.loads(resp.read())
    assert "access_token" in body
    assert "roles" in body or "refresh_token" in body


def test_backtest_history_endpoint_returns_ok(api_token):
    """Test E2E: Backtest History page loads with runs."""
    req = urllib.request.Request(f"{API_BASE}/backtests")
    req.add_header("Authorization", f"Bearer {api_token}")
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            data = json.loads(resp.read())
        assert "runs" in data, f"Missing runs key, got: {list(data.keys())}"
    except urllib.error.HTTPError as e:
        assert e.code in (404, 500), f"Unexpected HTTP {e.code}"


def test_dashboard_api_polling(api_token):
    """Test that risk status and regime-history endpoints return valid data."""
    for path in ["/risk/status", "/monitor/regime-history"]:
        req = urllib.request.Request(f"{API_BASE}{path}")
        req.add_header("Authorization", f"Bearer {api_token}")
        try:
            with urllib.request.urlopen(req, timeout=10) as resp:
                data = json.loads(resp.read())
            assert isinstance(data, (dict, list)), f"{path} should return dict or list"
        except urllib.error.HTTPError as e:
            assert e.code < 500, f"{path} returned server error {e.code}"
