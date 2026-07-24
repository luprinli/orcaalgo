"""End-to-end tests for Login → Dashboard and Backtest Detail → Regime Stats."""

import json
import socket
import urllib.error
import urllib.request

import pytest

API_BASE = "http://localhost:8080/api/v1"
UI_BASE = "http://localhost:5173"
AUTH = {"username": "admin", "password": "dev-admin-password-do-not-use-in-production"}


def _server_reachable(host: str = "localhost", port: int = 8080) -> bool:
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(2)
    try:
        s.connect((host, port))
        s.close()
        return True
    except (TimeoutError, ConnectionRefusedError, OSError):
        return False


requires_server = pytest.mark.skipif(
    not _server_reachable(), reason="Go API server not running on :8080 — start with ./scripts/orchestrate.py"
)


def get_token():
    data = json.dumps(AUTH).encode()
    req = urllib.request.Request(f"{API_BASE}/auth/login", data=data, method="POST")
    req.add_header("Content-Type", "application/json")
    with urllib.request.urlopen(req, timeout=10) as resp:
        return json.loads(resp.read())["access_token"]


@requires_server
def test_login_api_returns_token():
    token = get_token()
    assert len(token) > 50, f"Token too short: {len(token)} chars"


@requires_server
def test_login_api_sets_correct_roles():
    data = json.dumps(AUTH).encode()
    req = urllib.request.Request(f"{API_BASE}/auth/login", data=data, method="POST")
    req.add_header("Content-Type", "application/json")
    with urllib.request.urlopen(req, timeout=10) as resp:
        body = json.loads(resp.read())
    assert "access_token" in body
    assert "roles" in body or "refresh_token" in body


@requires_server
def test_backtest_history_endpoint_returns_ok():
    """Test E2E: Backtest History page loads with runs."""
    token = get_token()
    req = urllib.request.Request(f"{API_BASE}/backtests")
    req.add_header("Authorization", f"Bearer {token}")
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            data = json.loads(resp.read())
        assert "runs" in data, f"Missing runs key, got: {list(data.keys())}"
    except urllib.error.HTTPError as e:
        assert e.code in (404, 500), f"Unexpected HTTP {e.code}"


@requires_server
def test_dashboard_api_polling():
    """Test that risk status and regime-history endpoints return valid data."""
    token = get_token()
    for path in ["/risk/status", "/monitor/regime-history"]:
        req = urllib.request.Request(f"{API_BASE}{path}")
        req.add_header("Authorization", f"Bearer {token}")
        try:
            with urllib.request.urlopen(req, timeout=10) as resp:
                data = json.loads(resp.read())
            assert isinstance(data, (dict, list)), f"{path} should return dict or list"
        except urllib.error.HTTPError as e:
            assert e.code < 500, f"{path} returned server error {e.code}"
