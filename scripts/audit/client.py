"""Robust, short-lived HTTP client for the OrcaAlgo backtest API.

Design goals:
  * No long-running background processes — every call is a bounded request.
  * Never raise from request helpers; return {"error": ...} so a single failing
    call degrades one check instead of crashing the whole suite.
  * Transient connection errors are retried a small, bounded number of times.
  * Matrix polling is capped by a wall-clock budget and handles the API's
    `results: null` (not `[]`) while a batch is still running.
"""

from __future__ import annotations

import json
import socket
import time
import urllib.error
import urllib.request
from typing import Any

from . import config


class PrerequisiteError(RuntimeError):
    """Raised when the server is unreachable or auth fails — the suite cannot run."""


def _host_port(base: str) -> tuple[str, int]:
    # base like http://localhost:8080/api/v1
    hostport = base.split("://", 1)[-1].split("/", 1)[0]
    host, _, port = hostport.partition(":")
    return host or "localhost", int(port or "80")


class APIClient:
    def __init__(self, base: str = config.API_BASE) -> None:
        self.base = base.rstrip("/")
        self.token: str | None = None

    # ── low level ───────────────────────────────────────────────────────────
    def reachable(self) -> bool:
        host, port = _host_port(self.base)
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.settimeout(3)
        try:
            s.connect((host, port))
            return True
        except (socket.timeout, ConnectionRefusedError, OSError):
            return False
        finally:
            s.close()

    def _request(self, method: str, path: str, body: dict | None = None,
                 timeout: int = config.REQUEST_TIMEOUT) -> Any:
        url = f"{self.base}{path}"
        data = json.dumps(body).encode() if body is not None else None
        last_err = ""
        for attempt in range(config.CONNECT_RETRIES + 1):
            req = urllib.request.Request(url, data=data, method=method)
            req.add_header("Content-Type", "application/json")
            if self.token:
                req.add_header("Authorization", f"Bearer {self.token}")
            try:
                with urllib.request.urlopen(req, timeout=timeout) as resp:
                    raw = resp.read()
                    return json.loads(raw) if raw else {}
            except urllib.error.HTTPError as e:
                # HTTP errors are deterministic; don't retry.
                detail = ""
                try:
                    detail = e.read().decode(errors="replace")[:300]
                except Exception:  # noqa: BLE001
                    pass
                return {"error": f"HTTP {e.code}", "detail": detail}
            except (urllib.error.URLError, ConnectionResetError, socket.timeout) as e:
                last_err = str(e)
                if attempt < config.CONNECT_RETRIES:
                    time.sleep(0.5 * (attempt + 1))
                    continue
            except Exception as e:  # noqa: BLE001
                return {"error": str(e)}
        return {"error": f"connection failed: {last_err}"}

    # ── auth ────────────────────────────────────────────────────────────────
    def login(self) -> str:
        if not self.reachable():
            raise PrerequisiteError(
                f"Server not reachable at {self.base}. Start it first "
                f"(e.g. `python scripts/orchestrate.py`), then re-run the audit."
            )
        resp = self._request(
            "POST", "/auth/login",
            {"username": config.ADMIN_USER, "password": config.ADMIN_PASSWORD},
            timeout=15,
        )
        if not isinstance(resp, dict) or "access_token" not in resp:
            raise PrerequisiteError(f"Login failed: {resp}")
        self.token = resp["access_token"]
        return self.token

    # ── discovery ───────────────────────────────────────────────────────────
    def synthetic_symbols(self, limit: int = 3) -> list[str]:
        """Return synthetic-backed symbols (exchange == SYNTHETIC), preferring
        the configured order; falls back to a static list if discovery fails."""
        resp = self._request("GET", "/symbols", timeout=20)
        syn: list[str] = []
        if isinstance(resp, dict) and isinstance(resp.get("symbols"), list):
            syn = [
                s.get("ticker")
                for s in resp["symbols"]
                if s.get("exchange") == "SYNTHETIC" and s.get("ticker")
            ]
        if not syn:
            return list(config.FALLBACK_SYMBOLS)[:limit]
        ordered = [s for s in config.PREFERRED_SYMBOLS if s in syn]
        ordered += [s for s in syn if s not in ordered]
        return ordered[:limit] if ordered else list(config.FALLBACK_SYMBOLS)[:limit]

    # ── backtests ───────────────────────────────────────────────────────────
    def run_single(self, strategy: str, symbol: str, timeframe: str,
                   *, sizing_percent: float = 0.02, propfirm: bool = False,
                   start: str = config.START_DATE, end: str = config.END_DATE,
                   data_source: str = config.DATA_SOURCE) -> dict[str, Any]:
        payload = {
            "mode": "single",
            "strategy_id": strategy,
            "symbols": [symbol],
            "timeframes": [timeframe],
            "start_date": start,
            "end_date": end,
            "capital": config.CAPITAL,
            "gate_profile": "none",
            "data_source": data_source,
            "sizing_percent": sizing_percent,
            "propfirm_enabled": propfirm,
        }
        return self._request("POST", "/backtests", payload, timeout=config.BACKTEST_TIMEOUT)

    def submit_matrix(self, strategies: list[str], symbols: list[str],
                      timeframes: list[str], *, propfirm: bool = False,
                      start: str = config.START_DATE, end: str = config.END_DATE,
                      data_source: str = config.DATA_SOURCE) -> dict[str, Any]:
        payload = {
            "mode": "matrix",
            "strategy_ids": strategies,
            "symbols": symbols,
            "timeframes": timeframes,
            "start_date": start,
            "end_date": end,
            "capital": config.CAPITAL,
            "gate_profile": "none",
            "data_source": data_source,
            "propfirm_enabled": propfirm,
        }
        return self._request("POST", "/backtests", payload, timeout=60)

    def poll_matrix(self, batch_id: str,
                    timeout: int = config.MATRIX_POLL_TIMEOUT) -> dict[str, Any]:
        """Poll until the batch leaves 'running' or the wall-clock budget expires.
        Handles `results: null` while running (a real API behavior)."""
        start = time.time()
        last: dict[str, Any] = {}
        while time.time() - start < timeout:
            resp = self._request(
                "GET", f"/backtests/matrix/{batch_id}/results", timeout=30
            )
            if isinstance(resp, dict) and "error" not in resp:
                last = resp
                status = (resp.get("summary") or {}).get("status", "running")
                if status in ("completed", "failed", "error"):
                    return resp
            time.sleep(config.MATRIX_POLL_INTERVAL)
        last.setdefault("summary", {})["status"] = last.get("summary", {}).get(
            "status", "timeout"
        )
        return last

    def run_optimization(self, strategy: str, *, symbols: list[str],
                         objective: str = "sharpe", max_combinations: int = 12,
                         train_years: int = 1, test_years: int = 1,
                         step_months: int = 12) -> dict[str, Any]:
        payload = {
            "strategy_id": strategy,
            "objective": objective,
            "max_combinations": max_combinations,
            "train_years": train_years,
            "test_years": test_years,
            "step_months": step_months,
            "symbols": symbols,
            "capital": config.CAPITAL,
        }
        return self._request("POST", "/optimize", payload, timeout=config.OPTIMIZE_TIMEOUT)

    def system_health(self) -> dict[str, Any]:
        return self._request("GET", "/system/health", timeout=15)

    def matrix_results_since(self, batch_id: str, since: int) -> dict[str, Any]:
        return self._request("GET", f"/backtests/matrix/{batch_id}/results?since={since}", timeout=30)

    def cancel_matrix(self, batch_id: str) -> dict[str, Any]:
        return self._request("POST", f"/backtests/matrix/{batch_id}/cancel", {}, timeout=15)


# ── result-parsing helpers (shared by checks) ──────────────────────────────
def distinct_equity_count(result: dict[str, Any]) -> int:
    ec = result.get("equity_curve") or []
    return len({round(float(p.get("value", 0)), 6) for p in ec if isinstance(p, dict)})


def signal_diag(result: dict[str, Any]) -> dict[str, Any]:
    return result.get("signal_diag") or {}
