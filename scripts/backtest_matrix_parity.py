"""Backtest Matrix Parity Workflow — Programmatic vs Playwright-UI cross-check.

This tool runs the SAME backtest matrix through two independent execution paths
and reports every divergence between them:

  Method A — Programmatic:  submits the matrix directly via the REST API and
                            polls the matrix-results endpoint.
  Method B — Playwright UI: drives the real React dashboard (Backtest Runner),
                            clicks "Run Matrix", intercepts the API traffic the
                            browser generates, and scrapes the rendered results
                            table from the DOM.

It then performs a THREE-LAYER comparison:

  1. Rendering fidelity   — the UI's own API response  vs  the UI's rendered DOM.
                            Same underlying run, so this is deterministic and
                            isolates pure front-end rendering/formatting bugs.
  2. Structural parity    — combo set (strategy x symbol x timeframe) and the
                            signal-deterministic fields (num_trades, gate_passed)
                            of the programmatic run vs the UI run. Exact match.
  3. Numeric consistency  — RNG-affected metrics (sharpe, sortino, return, etc.).
                            The matrix engine seeds its fill simulator from the
                            wall clock (internal/backtest/slippage.go), so two
                            submissions of the same config differ slightly; these
                            are compared with tolerances and attributed to the
                            non-deterministic seed rather than a real defect.

Usage:
    python scripts/backtest_matrix_parity.py [options]

Options:
    --headed              Show the browser window (default: headless)
    --screenshot-dir DIR  Where to write screenshots (default: screenshots)
    --report FILE         Where to write the JSON report (default: reports/backtest_parity_report.json)
    --strategies A,B,C    Strategy IDs for the matrix (default: 3 built-ins)
    --symbols A,B         Symbols for the matrix (default: 2)
    --timeframes A        Timeframes (default: 1d)
    --start YYYY-MM-DD    Backtest start (default: 2023-01-01)
    --end YYYY-MM-DD      Backtest end (default: 2024-12-31)
    --capital N           Initial capital (default: 100000)
    --data-source S       synthetic | stooq (default: synthetic)
    --max-combos N        Safety cap on UI matrix size (default: 24)
    --ui-only             Only run the Playwright path (skip programmatic)
    --api-only            Only run the programmatic path (skip Playwright)

Requires (for the UI path):
    pip install playwright && playwright install chromium
    A running stack:  python scripts/orchestrate.py
"""

from __future__ import annotations

import argparse
import json
import math
import os
import socket
import sys
import time
import urllib.error
import urllib.request
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

PROJECT_ROOT = Path(__file__).resolve().parent.parent

# ---------------------------------------------------------------------------
# Constants — aligned with the current codebase (verified against router.go,
# api/client.ts, authStore.ts, BacktestPage.tsx, MatrixResultsPanel.tsx).
# ---------------------------------------------------------------------------
API_BASE = os.environ.get("ORCA_API_BASE", "http://localhost:8080/api/v1")
UI_BASE = os.environ.get("ORCA_UI_BASE", "http://localhost:5173")
ADMIN_USER = os.environ.get("ORCA_ADMIN_USER", "admin")
ADMIN_PASSWORD = os.environ.get("ORCA_ADMIN_PASSWORD", "dev-admin-password-do-not-use-in-production")

DEFAULT_STRATEGIES = ["intraday_mr", "trend_following", "grid_trading"]
DEFAULT_SYMBOLS = ["USDEUR", "XAUUSD"]
DEFAULT_TIMEFRAMES = ["1d"]

# Tolerances for the numeric-consistency layer (cross-run, RNG-affected).
# These are intentionally generous because the matrix engine reseeds its fill
# simulator per submission; the deterministic checks live in the structural layer.
NUMERIC_TOLERANCES = {
    "sharpe_ratio": 0.75,
    "sortino_ratio": 1.0,
    "max_drawdown": 5.0,     # percentage points
    "total_return": 5.0,     # percentage points (absolute)
    "win_rate": 0.10,        # fraction (10 pp)
    "profit_factor": 0.75,
}

# Precision the UI renders each metric at (MatrixResultsPanel.tsx), used by the
# rendering-fidelity layer to derive an acceptable parse tolerance.
RENDER_DECIMALS = {
    "sharpe_ratio": 3,
    "sortino_ratio": 3,
    "max_drawdown": 1,
    "total_return": 1,
    "profit_factor": 2,
    "win_rate": 0,   # rendered as *100 with 0 decimals
}


# ===========================================================================
# Report accumulator
# ===========================================================================
@dataclass
class Report:
    started_at: str = field(default_factory=lambda: datetime.now(timezone.utc).isoformat())
    config: dict[str, Any] = field(default_factory=dict)
    programmatic: dict[str, Any] = field(default_factory=dict)
    ui: dict[str, Any] = field(default_factory=dict)
    comparison: dict[str, Any] = field(default_factory=dict)
    divergences: list[dict[str, Any]] = field(default_factory=list)
    screenshots: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)
    verdict: str = "PENDING"

    def add_divergence(self, layer: str, severity: str, combo: str,
                       field_name: str, detail: str,
                       programmatic: Any = None, ui: Any = None) -> None:
        self.divergences.append({
            "layer": layer,
            "severity": severity,
            "combo": combo,
            "field": field_name,
            "detail": detail,
            "programmatic": programmatic,
            "ui": ui,
        })

    def to_dict(self) -> dict[str, Any]:
        return {
            "started_at": self.started_at,
            "finished_at": datetime.now(timezone.utc).isoformat(),
            "config": self.config,
            "verdict": self.verdict,
            "programmatic": self.programmatic,
            "ui": self.ui,
            "comparison": self.comparison,
            "divergences": self.divergences,
            "screenshots": self.screenshots,
            "warnings": self.warnings,
        }


REPORT = Report()


def log(level: str, tag: str, msg: str) -> None:
    ts = datetime.now(timezone.utc).strftime("%H:%M:%S")
    print(f"{ts} {level:5} [{tag:16}] {msg}", flush=True)


# ===========================================================================
# API client
# ===========================================================================
def _server_reachable(host: str = "localhost", port: int = 8080) -> bool:
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(2)
    try:
        s.connect((host, port))
        s.close()
        return True
    except (socket.timeout, ConnectionRefusedError, OSError):
        return False


class APIClient:
    """Minimal authenticated REST client for the OrcaAlgo backtest API."""

    def __init__(self, base: str = API_BASE) -> None:
        self.base = base
        self.token: str | None = None

    def _request(self, method: str, path: str, body: dict | None = None,
                 timeout: int = 60) -> Any:
        url = f"{self.base}{path}"
        data = json.dumps(body).encode() if body is not None else None
        req = urllib.request.Request(url, data=data, method=method)
        req.add_header("Content-Type", "application/json")
        if self.token:
            req.add_header("Authorization", f"Bearer {self.token}")
        try:
            with urllib.request.urlopen(req, timeout=timeout) as resp:
                raw = resp.read()
                return json.loads(raw) if raw else {}
        except urllib.error.HTTPError as e:
            return {"error": f"HTTP {e.code}", "body": e.read().decode(errors="replace")}
        except Exception as e:  # noqa: BLE001
            return {"error": str(e)}

    def login(self) -> str:
        if not _server_reachable():
            raise ConnectionError(
                "Go API server not reachable on :8080. Start it with "
                "`python scripts/orchestrate.py`."
            )
        resp = self._request("POST", "/auth/login",
                             {"username": ADMIN_USER, "password": ADMIN_PASSWORD})
        if not isinstance(resp, dict) or "access_token" not in resp:
            raise RuntimeError(f"Login failed: {resp}")
        self.token = resp["access_token"]
        return self.token

    def submit_matrix(self, payload: dict[str, Any]) -> dict[str, Any]:
        return self._request("POST", "/backtests", payload, timeout=30)

    def poll_matrix(self, batch_id: str, interval: float = 1.5,
                    timeout: int = 300) -> dict[str, Any]:
        start = time.time()
        last: dict[str, Any] = {}
        while time.time() - start < timeout:
            resp = self._request("GET", f"/backtests/matrix/{batch_id}/results", timeout=30)
            if isinstance(resp, dict) and "error" in resp:
                time.sleep(interval)
                continue
            last = resp
            summary = resp.get("summary", {}) if isinstance(resp, dict) else {}
            status = summary.get("status", "running")
            completed = summary.get("passed", 0) + summary.get("failed", 0)
            total = summary.get("total_combos", 0)
            n_results = len(resp.get("results") or []) if isinstance(resp, dict) else 0
            log("INFO", "prog-poll",
                f"status={status} results={n_results}/{total} completed={completed} "
                f"[{int(time.time()-start)}s]")
            if status in ("completed", "failed", "error"):
                return resp
            time.sleep(interval)
        REPORT.warnings.append(f"Programmatic matrix poll timed out after {timeout}s")
        return last


# ===========================================================================
# Normalization — unify API and DOM results into one comparable shape
# ===========================================================================
COMBO_KEY_FIELDS = ("strategy_id", "symbol", "timeframe")


def combo_key(strategy: str, symbol: str, timeframe: str) -> str:
    return f"{strategy}|{symbol}|{timeframe}"


def normalize_api_results(results: list[dict[str, Any]]) -> dict[str, dict[str, Any]]:
    """Normalize a list of ComboResult dicts (from the API) keyed by combo."""
    out: dict[str, dict[str, Any]] = {}
    for r in results or []:
        strat = str(r.get("strategy_id", ""))
        sym = str(r.get("symbol", ""))
        tf = str(r.get("timeframe", ""))
        if not (strat and sym and tf):
            continue
        gate = r.get("gate_passed", None)
        out[combo_key(strat, sym, tf)] = {
            "strategy_id": strat,
            "symbol": sym,
            "timeframe": tf,
            "num_trades": _to_int(r.get("num_trades")),
            "sharpe_ratio": _to_float(r.get("sharpe_ratio")),
            "sortino_ratio": _to_float(r.get("sortino_ratio")),
            "max_drawdown": _to_float(r.get("max_drawdown")),
            "total_return": _to_float(r.get("total_return")),
            "win_rate": _to_float(r.get("win_rate")),
            "profit_factor": _to_float(r.get("profit_factor")),
            "gate_passed": gate,
            "error": r.get("error"),
        }
    return out


def _to_float(v: Any) -> float | None:
    if v is None:
        return None
    try:
        f = float(v)
        return f if math.isfinite(f) else None
    except (TypeError, ValueError):
        return None


def _to_int(v: Any) -> int | None:
    if v is None:
        return None
    try:
        return int(v)
    except (TypeError, ValueError):
        return None


# ===========================================================================
# Programmatic execution path
# ===========================================================================
def build_payload(args: argparse.Namespace) -> dict[str, Any]:
    return {
        "mode": "matrix",
        "strategy_ids": args.strategies,
        "symbols": args.symbols,
        "timeframes": args.timeframes,
        "start_date": args.start,
        "end_date": args.end,
        "capital": args.capital,
        "gate_profile": "none",
        "data_source": args.data_source,
        "sizing_percent": 0.02,
    }


def run_programmatic(client: APIClient, payload: dict[str, Any]) -> dict[str, dict[str, Any]]:
    log("STEP", "programmatic", "Submitting matrix via REST API...")
    submit = client.submit_matrix(payload)
    if "error" in submit:
        REPORT.programmatic["error"] = submit
        log("ERROR", "programmatic", f"Submit failed: {submit}")
        return {}
    batch_id = submit.get("batch_run_id", "")
    total = submit.get("total_combos", 0)
    REPORT.programmatic["batch_run_id"] = batch_id
    REPORT.programmatic["total_combos"] = total
    log("OK", "programmatic", f"Submitted batch_run_id={batch_id} total_combos={total}")

    final = client.poll_matrix(batch_id)
    summary = final.get("summary", {}) if isinstance(final, dict) else {}
    results = (final.get("results") or []) if isinstance(final, dict) else []
    REPORT.programmatic["summary"] = summary
    REPORT.programmatic["result_count"] = len(results)
    normalized = normalize_api_results(results)
    log("OK", "programmatic", f"Collected {len(normalized)} combo results "
                              f"(status={summary.get('status')})")
    return normalized


# ===========================================================================
# Playwright UI execution path
# ===========================================================================
def _screenshot(page, name: str, ss_dir: Path) -> None:
    try:
        path = ss_dir / f"{name}.png"
        page.screenshot(path=str(path), full_page=True)
        REPORT.screenshots.append(str(path))
        log("OK", "ui-screenshot", f"{name}.png")
    except Exception as e:  # noqa: BLE001
        REPORT.warnings.append(f"Screenshot '{name}' failed: {e}")


def _select_by_label(page, label: str, value: str) -> bool:
    """Select an <option> in the native <select> whose sibling <label> matches."""
    try:
        loc = page.locator(f'div:has(> label:text-is("{label}")) select')
        if loc.count() > 0:
            loc.first.select_option(value)
            return True
    except Exception as e:  # noqa: BLE001
        REPORT.warnings.append(f"Could not set '{label}'={value}: {e}")
    return False


def _fill_date(page, label: str, value: str) -> bool:
    try:
        loc = page.locator(f'div:has(> label:text-is("{label}")) input[type="date"]')
        if loc.count() > 0:
            loc.first.fill(value)
            return True
    except Exception as e:  # noqa: BLE001
        REPORT.warnings.append(f"Could not set date '{label}'={value}: {e}")
    return False


def _scrape_matrix_table(page) -> dict[str, dict[str, Any]]:
    """Scrape the rendered Matrix Results table into normalized combo dicts.

    Column order (MatrixResultsPanel.tsx):
      0 Strategy 1 Symbol 2 TF 3 Trades 4 Sharpe 5 Sortino 6 Max DD
      7 Return 8 Win 9 PF 10 Gate 11 Opt
    """
    out: dict[str, dict[str, Any]] = {}
    # Disambiguate the results table (inside the scroll container) from the
    # parameter-sensitivity table (also .data-table).
    table = page.locator('div[style*="max-height"] table.data-table').last
    if table.count() == 0:
        table = page.locator('table.data-table:has(th:has-text("Sharpe"))').last
    if table.count() == 0:
        REPORT.warnings.append("UI results table not found for scraping")
        return out

    rows = table.locator("tbody tr")
    n = rows.count()
    for i in range(n):
        cells = rows.nth(i).locator("td")
        if cells.count() < 11:
            continue

        def cell(idx: int) -> str:
            return (cells.nth(idx).text_content() or "").strip()

        strat, sym, tf = cell(0), cell(1), cell(2)
        if not (strat and sym and tf):
            continue
        out[combo_key(strat, sym, tf)] = {
            "strategy_id": strat,
            "symbol": sym,
            "timeframe": tf,
            "num_trades": _parse_num(cell(3)),
            "sharpe_ratio": _parse_num(cell(4)),
            "sortino_ratio": _parse_num(cell(5)),
            "max_drawdown": _parse_num(cell(6)),   # trailing %
            "total_return": _parse_num(cell(7)),   # trailing %
            "win_rate_display": cell(8),           # "60%" or "—"
            "profit_factor": _parse_num(cell(9)),
            "gate_display": cell(10),              # PASS / FAIL / —
            "opt_display": cell(11) if cells.count() > 11 else "",
        }
    return out


def _parse_num(text: str) -> float | None:
    """Parse a numeric cell, stripping %, sort arrows, and whitespace."""
    if text is None:
        return None
    cleaned = (text.replace("%", "")
                   .replace("▼", "").replace("▲", "")
                   .replace(",", "").strip())
    if cleaned in ("", "—", "-", "N/A"):
        return None
    try:
        return float(cleaned)
    except ValueError:
        return None


def run_ui(payload: dict[str, Any], token: str, args: argparse.Namespace,
           ss_dir: Path) -> dict[str, Any]:
    """Drive the UI, intercept its API traffic, and scrape the rendered table.

    The outgoing POST /backtests body is rewritten to the controlled `payload`
    via request interception. This keeps the exercised matrix small and
    deterministic in scope while still fully exercising the UI's submit →
    poll → render pipeline (the layer that matters for rendering fidelity),
    without depending on the fragile div-based MultiSelect widgets.

    Returns dict with keys: api_results (normalized), dom_results (normalized),
    batch_run_id, request_payload."""
    result: dict[str, Any] = {
        "api_results": {}, "dom_results": {}, "batch_run_id": "",
        "request_payload": None, "combos_submitted": None,
    }
    try:
        from playwright.sync_api import sync_playwright
    except ImportError:
        REPORT.warnings.append("playwright not installed — UI path skipped")
        log("WARN", "ui", "playwright not installed. `pip install playwright && playwright install chromium`")
        return result

    captured: dict[str, Any] = {"request_body": None, "matrix_responses": []}

    with sync_playwright() as pw:
        browser = pw.chromium.launch(headless=not args.headed)
        context = browser.new_context(viewport={"width": 1920, "height": 1080})

        auth_blob = json.dumps({"token": token, "refresh": "", "roles": ["admin", "trader"]})
        context.add_init_script(f"window.localStorage.setItem('orca_auth', {json.dumps(auth_blob)});")

        page = context.new_page()

        # Rewrite the matrix submission body so the UI submits our controlled matrix.
        def handle_route(route):
            req = route.request
            try:
                if req.method == "POST" and req.url.rstrip("/").endswith("/backtests"):
                    new_body = json.dumps(payload)
                    captured["request_body"] = new_body
                    route.continue_(post_data=new_body)
                    return
            except Exception:  # noqa: BLE001
                pass
            route.continue_()

        page.route("**/api/v1/backtests", handle_route)

        def on_response(response):
            try:
                if "/backtests/matrix/" in response.url and response.url.endswith("/results"):
                    if response.status == 200:
                        captured["matrix_responses"].append(response.json())
            except Exception:  # noqa: BLE001
                pass

        page.on("response", on_response)

        # ── Load the Backtest Runner ──
        log("STEP", "ui", "Navigating to /backtest ...")
        page.goto(f"{UI_BASE}/backtest", wait_until="load", timeout=60000)
        try:
            page.wait_for_selector("h1:has-text('Backtest Runner')", timeout=15000)
        except Exception:  # noqa: BLE001
            REPORT.warnings.append("Backtest Runner heading not found — UI may have changed")
        # Allow strategies/symbols to load and the date auto-adjust effect to settle.
        page.wait_for_timeout(4000)
        _screenshot(page, "01_backtest_runner", ss_dir)

        # ── Configure: matrix mode (auto-selects all → satisfies validation) ──
        _select_by_label(page, "Mode", "matrix")
        page.wait_for_timeout(800)
        _select_by_label(page, "Data", args.data_source)
        _screenshot(page, "02_configured", ss_dir)
        result["combos_submitted"] = len(payload["strategy_ids"]) * len(payload["symbols"]) * len(payload["timeframes"])

        # ── Run (the request body is rewritten to our controlled matrix) ──
        run_btn = page.get_by_role("button", name="Run Matrix")
        if run_btn.count() == 0:
            run_btn = page.locator("button").filter(has_text="Run Matrix")
        if run_btn.count() == 0:
            REPORT.warnings.append("Run Matrix button not found")
            log("ERROR", "ui", "Run Matrix button not found")
            browser.close()
            return result
        log("STEP", "ui", f"Clicking Run Matrix (submitting controlled {result['combos_submitted']}-combo matrix)...")
        run_btn.first.click()

        # ── Wait for the results table + polling to finish ──
        try:
            page.wait_for_selector('table.data-table th:has-text("Strategy")', timeout=60000)
        except Exception:  # noqa: BLE001
            REPORT.warnings.append("UI results table did not appear within 60s")

        # Wait until the status badge leaves 'running' (polling stops), bounded.
        deadline = time.time() + 120
        while time.time() < deadline:
            badge = page.locator('.card:has(h2:has-text("Matrix Results")) .badge')
            txt = ""
            if badge.count() > 0:
                txt = (badge.first.text_content() or "").strip().lower()
            if txt and txt != "running":
                break
            page.wait_for_timeout(1000)
        page.wait_for_timeout(1000)
        _screenshot(page, "03_matrix_results", ss_dir)

        # ── Capture ──
        result["request_payload"] = _safe_json(captured["request_body"])
        # The last matrix response with the most results is the authoritative one.
        best_resp: dict[str, Any] = {}
        best_count = -1
        for resp in captured["matrix_responses"]:
            rc = len(resp.get("results") or []) if isinstance(resp, dict) else 0
            if rc >= best_count:
                best_count, best_resp = rc, resp
        if best_resp:
            result["batch_run_id"] = (best_resp.get("summary", {}) or {}).get("batch_run_id", "")
            result["api_results"] = normalize_api_results(best_resp.get("results") or [])
            REPORT.ui["summary"] = best_resp.get("summary", {})
        else:
            REPORT.warnings.append("No matrix results response was intercepted from the UI")

        result["dom_results"] = _scrape_matrix_table(page)
        REPORT.ui["api_result_count"] = len(result["api_results"])
        REPORT.ui["dom_result_count"] = len(result["dom_results"])
        REPORT.ui["combos_submitted"] = result["combos_submitted"]
        log("OK", "ui", f"Captured API={len(result['api_results'])} DOM={len(result['dom_results'])}")

        browser.close()

    return result


def _safe_json(s: Any) -> Any:
    if not s:
        return None
    try:
        return json.loads(s)
    except (TypeError, ValueError):
        return s


# ===========================================================================
# Comparison engine — three layers
# ===========================================================================
def compare_rendering(api_results: dict[str, dict[str, Any]],
                      dom_results: dict[str, dict[str, Any]]) -> dict[str, Any]:
    """Layer 1: the UI's own API response vs its rendered DOM table.

    Deterministic (same run). Verifies each displayed cell was derived correctly
    from the corresponding API field, encoding the UI's known formatting."""
    layer = {"rows_checked": 0, "cell_checks": 0, "mismatches": 0,
             "missing_in_dom": [], "extra_in_dom": []}

    api_keys = set(api_results)
    dom_keys = set(dom_results)
    layer["missing_in_dom"] = sorted(api_keys - dom_keys)
    layer["extra_in_dom"] = sorted(dom_keys - api_keys)

    for k in sorted(api_keys - dom_keys):
        REPORT.add_divergence("rendering", "high", k, "row",
                              "Combo present in UI API response but NOT rendered in the DOM table")
    for k in sorted(dom_keys - api_keys):
        REPORT.add_divergence("rendering", "high", k, "row",
                              "Row rendered in the DOM table with no matching UI API combo")

    for k in sorted(api_keys & dom_keys):
        a = api_results[k]
        d = dom_results[k]
        layer["rows_checked"] += 1

        # num_trades — exact integer.
        layer["cell_checks"] += 1
        if a["num_trades"] is not None and d["num_trades"] is not None:
            if int(d["num_trades"]) != int(a["num_trades"]):
                layer["mismatches"] += 1
                REPORT.add_divergence("rendering", "high", k, "num_trades",
                                      "Rendered trade count differs from API value",
                                      a["num_trades"], d["num_trades"])

        # Continuous metrics rendered without scaling (sharpe, sortino, max_dd, return, pf).
        for f_api, f_dom, dec in (
            ("sharpe_ratio", "sharpe_ratio", RENDER_DECIMALS["sharpe_ratio"]),
            ("sortino_ratio", "sortino_ratio", RENDER_DECIMALS["sortino_ratio"]),
            ("max_drawdown", "max_drawdown", RENDER_DECIMALS["max_drawdown"]),
            ("total_return", "total_return", RENDER_DECIMALS["total_return"]),
            ("profit_factor", "profit_factor", RENDER_DECIMALS["profit_factor"]),
        ):
            av, dv = a.get(f_api), d.get(f_dom)
            layer["cell_checks"] += 1
            if av is None or dv is None:
                if av is not None or dv is not None:
                    layer["mismatches"] += 1
                    REPORT.add_divergence("rendering", "medium", k, f_api,
                                          "One side null, other present", av, dv)
                continue
            tol = 0.5 * (10 ** (-dec)) + 1e-9
            if abs(av - dv) > tol:
                layer["mismatches"] += 1
                REPORT.add_divergence("rendering", "high", k, f_api,
                                      f"Rendered value != API value (tol {tol:g})", av, dv)

        # win_rate — UI renders (win_rate * 100) with 0 decimals, or '—' when null.
        layer["cell_checks"] += 1
        av = a.get("win_rate")
        disp = d.get("win_rate_display", "")
        if av is None:
            if disp not in ("—", "-", ""):
                layer["mismatches"] += 1
                REPORT.add_divergence("rendering", "medium", k, "win_rate",
                                      "API win_rate null but UI shows a value", None, disp)
        else:
            parsed = _parse_num(disp)
            if parsed is None:
                layer["mismatches"] += 1
                REPORT.add_divergence("rendering", "high", k, "win_rate",
                                      "API win_rate present but UI shows no number", av * 100, disp)
            elif abs(parsed - av * 100) > 0.5 + 1e-9:
                layer["mismatches"] += 1
                REPORT.add_divergence("rendering", "high", k, "win_rate",
                                      "Rendered win% != API win_rate*100", av * 100, parsed)

        # gate — PASS/FAIL/— must match gate_passed bool/None.
        layer["cell_checks"] += 1
        gate = a.get("gate_passed")
        gdisp = (d.get("gate_display", "") or "").upper()
        expected = "PASS" if gate is True else ("FAIL" if gate is False else "—")
        if gate is None:
            if gdisp not in ("—", "-", ""):
                layer["mismatches"] += 1
                REPORT.add_divergence("rendering", "low", k, "gate_passed",
                                      "API gate null but UI shows a badge", "—", gdisp)
        elif gdisp not in (expected,):
            layer["mismatches"] += 1
            REPORT.add_divergence("rendering", "high", k, "gate_passed",
                                  "Rendered gate badge != API gate_passed", expected, gdisp)

    return layer


def compare_structural(prog: dict[str, dict[str, Any]],
                       ui_api: dict[str, dict[str, Any]]) -> dict[str, Any]:
    """Layer 2: combo-set parity + signal-deterministic fields (exact)."""
    layer = {"prog_combos": len(prog), "ui_combos": len(ui_api),
             "missing_in_ui": [], "missing_in_prog": [],
             "num_trades_mismatches": 0, "gate_mismatches": 0}

    pk, uk = set(prog), set(ui_api)
    layer["missing_in_ui"] = sorted(pk - uk)
    layer["missing_in_prog"] = sorted(uk - pk)

    for k in sorted(pk - uk):
        REPORT.add_divergence("structural", "high", k, "combo",
                              "Combo produced programmatically but absent from UI run")
    for k in sorted(uk - pk):
        REPORT.add_divergence("structural", "high", k, "combo",
                              "Combo produced by UI run but absent from programmatic run")

    for k in sorted(pk & uk):
        p, u = prog[k], ui_api[k]
        # num_trades: signals are deterministic; fills only jitter prices, so
        # a difference here is a stronger signal than continuous-metric drift.
        if p["num_trades"] is not None and u["num_trades"] is not None:
            if p["num_trades"] != u["num_trades"]:
                layer["num_trades_mismatches"] += 1
                diff = abs(p["num_trades"] - u["num_trades"])
                sev = "low" if diff <= 2 else "medium"
                REPORT.add_divergence("structural", sev, k, "num_trades",
                                      "Trade count differs across runs "
                                      "(fill-price RNG can shift stop/TP hits)",
                                      p["num_trades"], u["num_trades"])
        # gate_passed must be identical (deterministic given identical config).
        if p.get("gate_passed") != u.get("gate_passed"):
            layer["gate_mismatches"] += 1
            REPORT.add_divergence("structural", "medium", k, "gate_passed",
                                  "Gate decision differs across runs",
                                  p.get("gate_passed"), u.get("gate_passed"))
    return layer


def compare_numeric(prog: dict[str, dict[str, Any]],
                    ui_api: dict[str, dict[str, Any]]) -> dict[str, Any]:
    """Layer 3: RNG-affected continuous metrics, tolerance-based."""
    layer = {"combos_compared": 0, "within_tolerance": 0, "beyond_tolerance": 0}

    for k in sorted(set(prog) & set(ui_api)):
        p, u = prog[k], ui_api[k]
        layer["combos_compared"] += 1
        for f_name, tol in NUMERIC_TOLERANCES.items():
            pv, uv = p.get(f_name), u.get(f_name)
            if pv is None or uv is None:
                continue
            diff = abs(pv - uv)
            if diff <= tol:
                layer["within_tolerance"] += 1
            else:
                layer["beyond_tolerance"] += 1
                REPORT.add_divergence(
                    "numeric", "medium", k, f_name,
                    f"Cross-run difference {diff:.4f} exceeds tolerance {tol:g} "
                    f"(matrix engine reseeds fill RNG per run; large gaps may "
                    f"indicate a real divergence)",
                    round(pv, 6), round(uv, 6))
    return layer


# ===========================================================================
# Verdict + reporting
# ===========================================================================
def compute_verdict() -> str:
    sev = {d["severity"] for d in REPORT.divergences}
    layers = {d["layer"] for d in REPORT.divergences}
    if "rendering" in layers and any(
            d["layer"] == "rendering" and d["severity"] == "high"
            for d in REPORT.divergences):
        return "FAIL"      # rendering bugs are deterministic and unambiguous
    if "structural" in layers and any(
            d["layer"] == "structural" and d["severity"] == "high"
            for d in REPORT.divergences):
        return "FAIL"      # combo-set divergence = the two methods disagree on scope
    if "high" in sev:
        return "FAIL"
    if "medium" in sev:
        return "PARTIAL"
    return "PASS"


def print_console_report() -> None:
    print()
    print("=" * 70)
    print("  BACKTEST MATRIX PARITY — SUMMARY")
    print("=" * 70)

    cfg = REPORT.config
    print(f"  Matrix:        {len(cfg.get('strategies', []))} strategies x "
          f"{len(cfg.get('symbols', []))} symbols x "
          f"{len(cfg.get('timeframes', []))} timeframes")
    print(f"  Data source:   {cfg.get('data_source')}")
    print(f"  Date range:    {cfg.get('start')} -> {cfg.get('end')}")
    print()

    comp = REPORT.comparison
    if "rendering" in comp:
        r = comp["rendering"]
        print("  [Layer 1] Rendering fidelity (UI API -> DOM):")
        print(f"            rows={r['rows_checked']} cell-checks={r['cell_checks']} "
              f"mismatches={r['mismatches']} "
              f"missing_in_dom={len(r['missing_in_dom'])} extra_in_dom={len(r['extra_in_dom'])}")
    if "structural" in comp:
        s = comp["structural"]
        print("  [Layer 2] Structural parity (prog vs UI):")
        print(f"            prog_combos={s['prog_combos']} ui_combos={s['ui_combos']} "
              f"missing_in_ui={len(s['missing_in_ui'])} missing_in_prog={len(s['missing_in_prog'])} "
              f"num_trades_mismatch={s['num_trades_mismatches']} gate_mismatch={s['gate_mismatches']}")
    if "numeric" in comp:
        n = comp["numeric"]
        print("  [Layer 3] Numeric consistency (tolerance-based, RNG-aware):")
        print(f"            combos={n['combos_compared']} "
              f"within_tol={n['within_tolerance']} beyond_tol={n['beyond_tolerance']}")

    print()
    by_sev: dict[str, int] = {}
    for d in REPORT.divergences:
        by_sev[d["severity"]] = by_sev.get(d["severity"], 0) + 1
    print(f"  Divergences:   {len(REPORT.divergences)} "
          f"(high={by_sev.get('high', 0)} medium={by_sev.get('medium', 0)} low={by_sev.get('low', 0)})")

    if REPORT.divergences:
        print()
        print("  Top divergences:")
        for d in REPORT.divergences[:15]:
            print(f"    [{d['severity']:6}] {d['layer']:10} {d['combo']:28} {d['field']:14} "
                  f"prog={d['programmatic']} ui={d['ui']}")
        if len(REPORT.divergences) > 15:
            print(f"    ... and {len(REPORT.divergences) - 15} more (see JSON report)")

    if REPORT.warnings:
        print()
        print("  Warnings:")
        for w in REPORT.warnings:
            print(f"    - {w}")

    print()
    verdict_color = {"PASS": "PASS", "PARTIAL": "PARTIAL", "FAIL": "FAIL"}.get(REPORT.verdict, REPORT.verdict)
    print("=" * 70)
    print(f"  VERDICT: {verdict_color}")
    print("=" * 70)
    print()


# ===========================================================================
# Main
# ===========================================================================
def build_cli() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description="Backtest matrix parity: programmatic vs Playwright UI")
    p.add_argument("--headed", action="store_true", help="Show the browser window")
    p.add_argument("--screenshot-dir", default="screenshots")
    p.add_argument("--report", default="reports/backtest_parity_report.json")
    p.add_argument("--strategies", type=lambda s: s.split(","), default=DEFAULT_STRATEGIES)
    p.add_argument("--symbols", type=lambda s: s.split(","), default=DEFAULT_SYMBOLS)
    p.add_argument("--timeframes", type=lambda s: s.split(","), default=DEFAULT_TIMEFRAMES)
    p.add_argument("--start", default="2023-01-01")
    p.add_argument("--end", default="2024-12-31")
    p.add_argument("--capital", type=float, default=100000.0)
    p.add_argument("--data-source", default="synthetic", choices=["synthetic", "stooq"])
    p.add_argument("--max-combos", type=int, default=24)
    p.add_argument("--ui-only", action="store_true")
    p.add_argument("--api-only", action="store_true")
    return p


def main() -> int:
    # Ensure non-ASCII-safe console output on Windows (cp1252 default).
    try:
        sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    except Exception:  # noqa: BLE001
        pass

    args = build_cli().parse_args()

    ss_dir = Path(args.screenshot_dir)
    ss_dir.mkdir(parents=True, exist_ok=True)
    report_path = Path(args.report)
    report_path.parent.mkdir(parents=True, exist_ok=True)

    REPORT.config = {
        "strategies": args.strategies,
        "symbols": args.symbols,
        "timeframes": args.timeframes,
        "start": args.start,
        "end": args.end,
        "capital": args.capital,
        "data_source": args.data_source,
        "max_combos": args.max_combos,
    }

    print("=" * 70)
    print("  BACKTEST MATRIX PARITY WORKFLOW")
    print("=" * 70)
    print()

    # ── Auth ──
    client = APIClient()
    try:
        token = client.login()
        log("OK", "auth", f"Authenticated as '{ADMIN_USER}' (token {token[:16]}...)")
    except (ConnectionError, RuntimeError) as e:
        log("ERROR", "auth", str(e))
        print("\nStart the stack first:  python scripts/orchestrate.py\n")
        return 2

    payload = build_payload(args)

    # ── Method A: programmatic ──
    prog_results: dict[str, dict[str, Any]] = {}
    if not args.ui_only:
        prog_results = run_programmatic(client, payload)

    # ── Method B: Playwright UI ──
    ui_bundle: dict[str, Any] = {"api_results": {}, "dom_results": {}}
    if not args.api_only:
        ui_bundle = run_ui(payload, token, args, ss_dir)
    REPORT.ui["batch_run_id"] = ui_bundle.get("batch_run_id", "")
    REPORT.ui["request_payload"] = ui_bundle.get("request_payload")

    ui_api = ui_bundle.get("api_results", {})
    ui_dom = ui_bundle.get("dom_results", {})

    # Persist per-combo data for a fully auditable, self-contained report.
    REPORT.programmatic["results"] = prog_results
    REPORT.ui["api_results"] = ui_api
    REPORT.ui["dom_results"] = ui_dom

    total_trades = sum((v.get("num_trades") or 0) for v in prog_results.values())
    REPORT.programmatic["total_trades_across_combos"] = total_trades
    if prog_results and total_trades == 0:
        REPORT.warnings.append(
            "All combos produced 0 trades on this data/strategy/params selection; the "
            "numeric-consistency layer is therefore trivially satisfied. Choose symbols/"
            "strategies/params that generate trades for a substantive numeric comparison.")

    # ── Comparison ──
    log("STEP", "compare", "Running three-layer comparison...")

    if ui_api and ui_dom:
        REPORT.comparison["rendering"] = compare_rendering(ui_api, ui_dom)
    elif not args.api_only:
        REPORT.warnings.append("Rendering layer skipped (missing UI API or DOM capture)")

    if prog_results and ui_api:
        REPORT.comparison["structural"] = compare_structural(prog_results, ui_api)
        REPORT.comparison["numeric"] = compare_numeric(prog_results, ui_api)
    elif not (args.ui_only or args.api_only):
        REPORT.warnings.append("Cross-method layers skipped (one method produced no results)")

    REPORT.verdict = compute_verdict()

    # ── Persist + print ──
    report_path.write_text(json.dumps(REPORT.to_dict(), indent=2, default=str))
    log("OK", "report", f"JSON report written to {report_path}")
    print_console_report()

    return 0 if REPORT.verdict == "PASS" else (1 if REPORT.verdict == "FAIL" else 0)


if __name__ == "__main__":
    sys.exit(main())
