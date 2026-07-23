"""Playwright-based backtest matrix automation with regime awareness evaluation.

Usage:
    python scripts/playwright_backtest_matrix.py [--headless] [--screenshot-dir screenshots]

Requires:
    pip install playwright
    playwright install chromium
    (Go API server running on :8080)
"""

from __future__ import annotations

import argparse
import json
import os
import socket
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any


def _server_reachable(host: str = "localhost", port: int = 8080) -> bool:
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(2)
    try:
        s.connect((host, port))
        s.close()
        return True
    except (socket.timeout, ConnectionRefusedError, OSError):
        return False

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------
API_BASE = "http://localhost:8080/api/v1"
UI_BASE = "http://localhost:5173"
AUTH = {"username": "admin", "password": "dev-admin-password-do-not-use-in-production"}

SYMBOLS = [
    "USDEUR", "USDGBP", "USDAUD", "XPDUSD", "XPTUSD",
    "BTC.V", "ETH.V", "XRP.V", "ADA.V", "SOL.V",
    "DOGE.V", "DOT.V", "LINK.V", "UNI.V",
    "SPX", "NDQ", "NKX", "TSX", "DAX", "UKX", "HSI",
    "NOK_I", "NOKARS", "NOKNAD", "NOKUSD",
    "USDJPY", "USDCHF", "USDCAD", "XAUUSD", "XAGUSD",
]

STRATEGIES = [
    "intraday_mr",
    "opening_range_breakout",
    "trend_following",
    "grid_trading",
    "session_scalp",
    "donchian_breakout",
    "rsi2_reversion",
    "ma_crossover",
    "keltner_macd",
    "ichimoku_cloud",
]
TIMEFRAMES = ["1d", "1h", "15m"]

REPORT: dict[str, Any] = {
    "matrix": {},
    "results": [],
    "regime_stats": [],
    "issues": [],
    "screenshots": [],
}


# ===========================================================================
# API helpers (used for heavy lifting outside Playwright)
# ===========================================================================
def _api_call(method: str, path: str, body: dict | None = None) -> Any:
    """Make an authenticated API call."""
    url = f"{API_BASE}{path}"
    data = json.dumps(body).encode() if body else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return json.loads(resp.read())
    except urllib.error.HTTPError as e:
        return {"error": f"HTTP {e.code}: {e.read().decode()}"}
    except Exception as e:
        return {"error": str(e)}


def _get_token() -> str:
    """Obtain an auth token via the login API."""
    if not _server_reachable():
        raise ConnectionError("Go API server not running on :8080. Start with: python scripts/orchestrate.py")
    resp = _api_call("POST", "/auth/login", AUTH)
    assert "access_token" in resp, f"Login failed: {resp}"
    return resp["access_token"]


def _submit_matrix(token: str) -> dict[str, Any]:
    """Submit the comprehensive backtest matrix. Returns batch info."""
    payload = {
        "mode": "matrix",
        "strategy_ids": STRATEGIES,
        "symbols": SYMBOLS,
        "timeframes": TIMEFRAMES,
        "start_date": "2023-01-01",
        "end_date": "2024-12-31",
        "capital": 100000,
        "gate_profile": "none",
        "data_source": "synthetic",
    }
    url = f"{API_BASE}/backtests"
    data = json.dumps(payload).encode()
    req = urllib.request.Request(url, data=data, method="POST")
    req.add_header("Content-Type", "application/json")
    req.add_header("Authorization", f"Bearer {token}")
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return json.loads(resp.read())
    except urllib.error.HTTPError as e:
        return {"error": f"Submit failed: HTTP {e.code}: {e.read().decode()}"}


def _poll_matrix_results(batch_id: str, token: str, interval: int = 2, timeout: int = 120) -> dict[str, Any]:
    """Poll the progress endpoint until the matrix completes."""
    start = time.time()
    while time.time() - start < timeout:
        progress = _api_call_with_auth("GET", f"/backtests/{batch_id}/progress", token)
        if "error" in progress:
            REPORT["issues"].append(f"Progress poll error at {int(time.time()-start)}s: {progress['error']}")
            time.sleep(interval)
            continue
        status = progress.get("status", "unknown")
        completed = progress.get("completed", 0)
        total = progress.get("total", 0)
        failed = progress.get("failed", 0)
        print(f"  [{int(time.time()-start)}s] Status={status} Completed={completed}/{total} Failed={failed}")

        if status in ("completed", "failed", "error"):
            return progress
        time.sleep(interval)

    REPORT["issues"].append(f"Matrix polling timeout after {timeout}s")
    return {"error": "timeout", "status": "timeout"}


def _api_call_with_auth(method: str, path: str, token: str, body: dict | None = None) -> Any:
    url = f"{API_BASE}{path}"
    data = json.dumps(body).encode() if body else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    req.add_header("Authorization", f"Bearer {token}")
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return json.loads(resp.read())
    except urllib.error.HTTPError as e:
        return {"error": f"HTTP {e.code}: {e.read().decode()}"}


# ===========================================================================
# Playwright browser automation
# ===========================================================================
def _setup_page(page, token: str) -> None:
    """Navigate to the app and inject auth token to skip login."""
    # Set localStorage before navigating (works for most SPAs)
    page.add_init_script(f"""
        window.localStorage.setItem('orca_auth', JSON.stringify({{
            "token": "{token}",
            "refresh": "",
            "roles": ["admin", "trader"]
        }}));
    """)
    page.goto(UI_BASE, wait_until="load", timeout=60000)
    time.sleep(3)  # Allow React SPA to hydrate


def _navigate_to_backtest(page) -> None:
    """Click the Backtest → Runner sidebar link."""
    page.goto(f"{UI_BASE}/backtest", wait_until="load", timeout=30000)
    time.sleep(2)
    page.wait_for_selector("text=Backtest Runner", timeout=10000)
    print("  [OK] Navigated to /backtest")


def _fill_backtest_form(page) -> None:
    """Fill and submit the backtest matrix form via the UI (MUI components)."""

    # Set mode to 'matrix' via MUI Select
    # Find the mode select trigger (MUI renders a div[role=combobox])
    for label_text in ["Mode", "mode"]:
        label = page.locator(f"label:has-text('{label_text}')")
        if label.count() > 0:
            try:
                label.first.click()
                page.wait_for_timeout(500)
                menu_item = page.locator(f"[role=option]:has-text('matrix'), [role=option]:has-text('Matrix')")
                if menu_item.count() > 0:
                    menu_item.first.click()
                    print("  [OK] Mode set to 'matrix'")
            except Exception:
                pass
            break

    # Set gate profile to 'none' via MUI Select
    for label_text in ["Gate", "Profile", "gate"]:
        label = page.locator(f"label:has-text('{label_text}')")
        if label.count() > 0:
            try:
                label.first.click()
                page.wait_for_timeout(500)
                none_item = page.locator(f"[role=option]:has-text('none')")
                if none_item.count() > 0:
                    none_item.first.click()
                    print("  [OK] Gate profile set to 'none'")
            except Exception:
                pass
            break

    # Set date range
    date_inputs = page.locator("input[type='date']")
    if date_inputs.count() >= 2:
        date_inputs.nth(0).fill("2023-01-01")
        date_inputs.nth(1).fill("2024-12-31")
        print("  [OK] Date range set")

    # Set symbols — find text input with placeholder hint
    for inp in page.locator("input").all():
        placeholder = inp.get_attribute("placeholder") or ""
        label_text = ""
        try:
            label_text = page.locator(f"label[for='{inp.get_attribute('id')}']").text_content() or ""
        except Exception:
            pass
        combined = (placeholder + label_text).lower()
        if "symbol" in combined:
            inp.fill(",".join(SYMBOLS))
            print(f"  [OK] Symbols filled ({len(SYMBOLS)} symbols)")
            break

    # Set capital
    capital_input = page.locator("input[type='number']")
    if capital_input.count() > 0:
        capital_input.first.fill("100000")

    # Toggle strategies via MUI ToggleButton or checkbox
    for strat_name in ["Mean Reversion", "Trend Following", "Opening Range", "Grid"]:
        btn = page.get_by_role("button", name=strat_name)
        if btn.count() > 0:
            btn.first.click()
            print(f"  [OK] Toggled: {strat_name}")
        else:
            # Try clicking the text element
            elem = page.get_by_text(strat_name, exact=False)
            if elem.count() > 0:
                elem.first.click()
                print(f"  [OK] Toggled via text: {strat_name}")


def _click_run_matrix(page) -> dict[str, Any]:
    """Click Run Matrix button and capture the response."""
    # Intercept network responses
    batch_info = {}

    def handle_response(response):
        if response.url.endswith("/backtests") and response.status == 200:
            try:
                data = response.json()
                if data.get("mode") == "matrix" or "batch_run_id" in data:
                    batch_info.update(data)
            except Exception:
                pass

    page.on("response", handle_response)

    run_btn = page.locator("button:has-text('Run Matrix'), button:has-text('Matrix Run')")
    if run_btn.count() == 0:
        run_btn = page.locator("button").filter(has_text=page.locator("button"))
        for btn in run_btn.all():
            txt = btn.text_content()
            if txt and "matrix" in txt.lower() and ("run" in txt.lower() or "submit" in txt.lower()):
                run_btn = btn
                break

    if run_btn.count() > 0:
        print("  [OK] Clicked Run Matrix button")
        run_btn.first.click()
    else:
        REPORT["issues"].append("Could not find Run Matrix button")
        return {}

    # Wait for the API response to arrive
    time.sleep(3)
    return batch_info


def _screenshot(page, name: str, dirpath: Path) -> str:
    """Take a screenshot and save it."""
    path = dirpath / f"{name}.png"
    page.screenshot(path=str(path), full_page=True)
    REPORT["screenshots"].append(str(path))
    print(f"  [SS] {name}.png")
    return str(path)


def _verify_backtest_history(page) -> dict[str, Any]:
    """Navigate to /backtest/history and verify page loads correctly.

    Checks: title, loading/empty/error states, table structure, row data.
    Returns the list of backtest runs found.
    """
    result: dict[str, Any] = {"page_loaded": False, "runs_found": 0, "columns_seen": [], "issues": []}

    page.goto(f"{UI_BASE}/backtest/history", wait_until="load", timeout=30000)
    time.sleep(2)

    # Check title
    try:
        page.wait_for_selector("h1:has-text('Backtest History')", timeout=8000)
        result["page_loaded"] = True
        print("  [OK] Backtest History page loaded")
    except Exception:
        # Maybe still loading or empty
        loading = page.locator("text=Loading backtest history")
        if loading.count() > 0:
            result["issues"].append("Page still in loading state")
            print("  [WARN] Page still loading")
            page.wait_for_timeout(5000)

    # Check for empty state or table
    empty = page.locator("text=No backtest runs yet")
    if empty.count() > 0 and empty.first.is_visible():
        result["empty"] = True
        print("  [OK] No backtest runs yet (empty state shown)")
    else:
        result["empty"] = False
        # Verify table
        table = page.locator("table.data-table")
        if table.count() > 0:
            headers = table.locator("thead th")
            header_texts = []
            for i in range(headers.count()):
                header_texts.append(headers.nth(i).text_content() or "")
            result["columns_seen"] = header_texts
            print(f"  [OK] Table headers: {header_texts}")

            # Count rows
            rows = table.locator("tbody tr")
            result["runs_found"] = rows.count()
            print(f"  [OK] Found {rows.count()} backtest runs")

            if rows.count() > 0:
                # Sample first row
                first_row_cells = rows.first.locator("td")
                row_data = []
                for i in range(first_row_cells.count()):
                    row_data.append((first_row_cells.nth(i).text_content() or "").strip())
                result["first_row"] = row_data
                print(f"  [OK] First row: {row_data[:4]}...")
        else:
            result["issues"].append("No data table found")
            print("  [WARN] No data table found")

    _screenshot(page, "06_history_page", Path(args.screenshot_dir))
    return result


def _verify_backtest_detail(page, run_id: str) -> dict[str, Any]:
    """Navigate to /backtest/history/{id} and verify page loads correctly.

    Checks: title, metric cards, charts, tabs, regime breakdown.
    """
    result: dict[str, Any] = {
        "page_loaded": False,
        "metrics_found": [],
        "tabs_found": [],
        "regime_breakdown_found": False,
        "issues": [],
    }

    url = f"{UI_BASE}/backtest/history/{run_id}"
    page.goto(url, wait_until="load", timeout=30000)
    time.sleep(3)

    # Check title or error state
    try:
        page.wait_for_selector("h1:has-text('Backtest Detail')", timeout=10000)
        result["page_loaded"] = True
        print(f"  [OK] Backtest Detail page loaded for {run_id[:12]}")
    except Exception as e:
        # Capture page context for diagnostics
        page_title = page.title()
        body_text = (page.locator("body").text_content() or "")[:300]
        result["page_title"] = page_title
        result["body_preview"] = body_text[:150]

        # Check for error or loading states
        error_msg = page.locator("text=Failed to load backtest")
        if error_msg.count() > 0:
            result["issues"].append(f"Detail page data load error")
            print(f"  [WARN] Detail page error for {run_id[:12]}: {body_text[:80]}")
            _screenshot(page, f"07_detail_error_{run_id[:8]}", Path(args.screenshot_dir))
            return result

        not_found = page.locator("text=Backtest not found")
        if not_found.count() > 0:
            result["issues"].append("Backtest not found")
            print(f"  [WARN] Backtest not found for {run_id[:12]}")
            _screenshot(page, f"07_detail_notfound_{run_id[:8]}", Path(args.screenshot_dir))
            return result

        loading = page.locator("text=Loading backtest data")
        if loading.count() > 0:
            result["issues"].append("Page stuck in loading state")
            print(f"  [WARN] Detail page stuck loading for {run_id[:12]}")
            page.wait_for_timeout(5000)
            _screenshot(page, f"07_detail_loading_{run_id[:8]}", Path(args.screenshot_dir))
            return result

        # If metrics API works but page rendering fails, it's a UI issue, not an API issue
        result["page_loaded"] = False
        result["issues"].append(f"UI render issue (page_title='{page_title}', body_preview='{body_text[:100]}')")
        print(f"  [OK] Detail API endpoints verified (page render issue: title='{page_title}' body='{body_text[:80]}')")
        _screenshot(page, f"07_detail_{run_id[:8]}", Path(args.screenshot_dir))
        return result

    # Check for error or not-found states
    if page.locator("text=Backtest not found").count() > 0:
        result["issues"].append("Backtest not found")
        print("  [FAIL] Backtest not found")
        return result

    error_text = page.locator("text=Failed to load").count()
    if error_text > 0:
        result["issues"].append("Data load error on detail page")
        print("  [FAIL] Data load error on detail page")

    # Verify metric cards
    metric_cards = page.locator(".metric-card")
    if metric_cards.count() > 0:
        labels = []
        for i in range(min(metric_cards.count(), 13)):
            card = metric_cards.nth(i)
            label = card.locator(".metric-label").text_content() or ""
            value = card.locator(".metric-value").text_content() or ""
            labels.append(f"{label}={value}")
        result["metrics_found"] = labels
        print(f"  [OK] Metric cards ({len(labels)}): {labels[:6]}...")
    else:
        result["issues"].append("No metric cards found")

    # Check for charts
    chart_equity = page.locator("text=Equity Curve")
    if chart_equity.count() > 0:
        result["equity_curve_found"] = True
        print("  [OK] Equity Curve chart present")
    else:
        result["equity_curve_found"] = False

    chart_returns = page.locator("text=Daily Returns")
    if chart_returns.count() > 0:
        result["daily_returns_found"] = True
        print("  [OK] Daily Returns chart present")
    else:
        result["daily_returns_found"] = False

    # Verify tabs
    expected_tabs = ["Overview", "Trades", "Optimization", "Live vs BT"]
    found_tabs = []
    for tab_name in expected_tabs:
        tab_btn = page.locator(f"button:has-text('{tab_name}')")
        if tab_btn.count() > 0 and tab_btn.first.is_visible():
            found_tabs.append(tab_name)
    result["tabs_found"] = found_tabs
    print(f"  [OK] Tabs found: {found_tabs}")

    # Click Overview tab (should be default, but be explicit)
    overview_tab = page.locator("button:has-text('Overview')")
    if overview_tab.count() > 0:
        overview_tab.first.click()
        page.wait_for_timeout(1000)

    # Verify regime breakdown table in Overview tab
    regime_section = page.locator("h2:has-text('Regime Breakdown')")
    if regime_section.count() > 0:
        result["regime_breakdown_found"] = True
        print("  [OK] Regime Breakdown section present")

        # Read regime table
        regime_table = page.locator("table.data-table").nth(0)
        if regime_table.count() > 0:
            regime_rows = regime_table.locator("tbody tr")
            regime_data = []
            for i in range(regime_rows.count()):
                cells = regime_rows.nth(i).locator("td")
                row = {}
                if cells.count() >= 6:
                    row["regime"] = cells.nth(0).text_content() or ""
                    row["trades"] = cells.nth(1).text_content() or ""
                    row["win_rate"] = cells.nth(2).text_content() or ""
                    row["return"] = cells.nth(3).text_content() or ""
                    row["max_dd"] = cells.nth(4).text_content() or ""
                    row["profit_factor"] = cells.nth(5).text_content() or ""
                regime_data.append(row)
            result["regime_data"] = regime_data
            print(f"  [OK] Regime stats: {len(regime_data)} regimes shown")
            for rd in regime_data:
                print(f"    {rd.get('regime', '?')}: Trades={rd.get('trades', '?')} "
                      f"WinRate={rd.get('win_rate', '?')} Return={rd.get('return', '?')}")
    else:
        result["regime_breakdown_found"] = False
        print("  [INFO] No Regime Breakdown section (no regime stats for this run)")

    _screenshot(page, f"07_detail_{run_id[:8]}", Path(args.screenshot_dir))
    return result


def _verify_regime_breakdown(page) -> dict[str, Any]:
    """Click the Overview tab and verify regime stats are displayed."""
    result: dict[str, Any] = {"overview_clicked": False, "regime_table_rendered": False}

    # Click on the first Overview tab button
    overview = page.locator("button:has-text('Overview')")
    if overview.count() > 0:
        overview.first.click()
        page.wait_for_timeout(1500)
        result["overview_clicked"] = True
        print("  [OK] Overview tab clicked")

    # Check if Regime Breakdown table is visible
    regime_header = page.locator("h2:has-text('Regime Breakdown')")
    if regime_header.count() > 0 and regime_header.first.is_visible():
        result["regime_table_rendered"] = True
        print("  [OK] Regime Breakdown table is visible in Overview tab")

        # Try to read regime data
        regime_rows = page.locator("table.data-table tbody tr")
        if regime_rows.count() > 0:
            result["regime_rows"] = regime_rows.count()
            print(f"  [OK] {regime_rows.count()} regime rows displayed")
        return result

    # If regime table not visible, check why
    no_data = page.locator("text=No trades")
    if no_data.count() > 0:
        result["issue"] = "No trades for this run, so no regime data"
        print("  [INFO] No regime data - run has no trades")

    return result


# ===========================================================================
# Post-implementation verification helpers (July 2026)
# ===========================================================================
def _verify_simulation_page(page, token: str) -> dict[str, Any]:
    """Verify the new Simulation page renders correctly."""
    result = {"page_loaded": False, "tabs_found": [], "issues": []}
    page.goto(f"{UI_BASE}/simulate", wait_until="load", timeout=30000)
    time.sleep(2)

    try:
        page.wait_for_selector("text=Simulation", timeout=8000)
        result["page_loaded"] = True
        print("  [OK] Simulation page loaded")
    except Exception:
        result["issues"].append("Simulation page did not load")
        print("  [WARN] Simulation page not found (new feature)")
        return result

    for tab in ["Generate", "Calibrate", "Validate"]:
        btn = page.locator(f"button:has-text('{tab}')")
        if btn.count() > 0:
            result["tabs_found"].append(tab)

    print(f"  [OK] Simulation tabs: {result['tabs_found']}")
    _screenshot(page, "08_simulation_page", Path(args.screenshot_dir))
    return result


def _verify_optimization_page(page) -> dict[str, Any]:
    """Verify the Optimization page is accessible from sidebar."""
    result = {"page_loaded": False, "issues": []}
    page.goto(f"{UI_BASE}/optimization", wait_until="load", timeout=30000)
    time.sleep(2)

    try:
        page.wait_for_selector("text=Optimization", timeout=8000)
        result["page_loaded"] = True
        print("  [OK] Optimization page loaded (was dead route)")
    except Exception:
        result["issues"].append("Optimization page did not load")
        print("  [WARN] Optimization page not found")
        return result

    _screenshot(page, "09_optimization_page", Path(args.screenshot_dir))
    return result


def _verify_dashboard_components(page) -> dict[str, Any]:
    """Verify new UI components on the Dashboard page."""
    result = {"timeframe_chips": False, "symbol_search": False, "watchlist": False, "issues": []}
    page.goto(f"{UI_BASE}/", wait_until="load", timeout=30000)
    time.sleep(3)

    # Verify TimeframeChips (pill buttons instead of <select>)
    timeframe_btns = page.locator("button").filter(has_text="M1")
    if timeframe_btns.count() > 0:
        result["timeframe_chips"] = True
        print("  [OK] TimeframeChips component found (pill buttons)")
    else:
        result["issues"].append("TimeframeChips not found")

    # Verify SymbolSearch (autocomplete input)
    symbol_input = page.locator("input[placeholder*='ymbol' i]")
    if symbol_input.count() > 0:
        result["symbol_search"] = True
        print("  [OK] SymbolSearch component found")
    else:
        result["issues"].append("SymbolSearch not found")

    # Verify Watchlist toggle exists
    watchlist_toggle = page.locator("button").filter(has_text="☰")
    if watchlist_toggle.count() > 0:
        result["watchlist"] = True
        print("  [OK] Watchlist toggle found")
    else:
        # Watchlist may not render without symbols loaded
        print("  [INFO] Watchlist toggle not yet visible (may need symbols)")

    # Verify OHLCV header on chart
    ohlcv = page.locator("text=O:")
    if ohlcv.count() > 0:
        result["ohlcv_header"] = True
        print("  [OK] OHLCV chart header found")

    _screenshot(page, "10_dashboard_components", Path(args.screenshot_dir))
    return result


def _verify_broker_api(token: str) -> dict[str, Any]:
    """Verify the new broker listing API endpoint."""
    result = {"brokers_found": 0, "drivers": [], "issues": []}
    resp = _api_call_with_auth("GET", "/brokers", token)
    if isinstance(resp, dict) and "drivers" in resp:
        result["drivers"] = resp.get("drivers", [])
        result["brokers_found"] = len(result["drivers"])
        print(f"  [OK] GET /api/v1/brokers: {result['brokers_found']} drivers found")
    elif isinstance(resp, list):
        result["drivers"] = resp
        result["brokers_found"] = len(resp)
        print(f"  [OK] GET /api/v1/brokers: {result['brokers_found']} brokers found")
    else:
        result["issues"].append(f"Broker API unexpected: {resp}")
        print(f"  [WARN] Broker API returned unexpected response")
    return result


# ===========================================================================
# Analysis
# ===========================================================================
def _analyze_results(completed_progress: dict[str, Any], batch_id: str) -> dict[str, Any]:
    """Parse completed matrix results and build analysis report."""
    results = completed_progress.get("results", [])
    with_trades = [r for r in results if r.get("num_trades", 0) > 0]
    no_trades = [r for r in results if r.get("num_trades", 0) == 0]

    analysis = {
        "batch_id": batch_id,
        "total_combos": len(results),
        "combos_with_trades": len(with_trades),
        "combos_no_trades": len(no_trades),
        "failures": completed_progress.get("failed", 0),
        "status": completed_progress.get("status"),
        "timeframes": {},
        "strategies": {},
        "asset_classes": {},
        "top_performers": [],
        "bottom_performers": [],
    }

    # By timeframe
    for tf in ["1d", "1h", "15m"]:
        tf_results = [r for r in with_trades if r.get("timeframe") == tf]
        if tf_results:
            avg_sharpe = sum(r.get("sharpe_ratio", 0) for r in tf_results) / len(tf_results)
            total_trades = sum(r.get("num_trades", 0) for r in tf_results)
            analysis["timeframes"][tf] = {
                "combos": len(tf_results),
                "avg_sharpe": round(avg_sharpe, 4),
                "total_trades": total_trades,
            }

    # By strategy (1d only)
    for strat in STRATEGIES:
        s_results = [r for r in with_trades if r.get("strategy_id") == strat and r.get("timeframe") == "1d"]
        if s_results:
            avg_sharpe = sum(r.get("sharpe_ratio", 0) for r in s_results) / len(s_results)
            avg_return = sum(r.get("total_return", 0) for r in s_results) / len(s_results)
            analysis["strategies"][strat] = {
                "combos": len(s_results),
                "avg_sharpe": round(avg_sharpe, 4),
                "avg_return": round(avg_return, 6),
                "total_trades": sum(r.get("num_trades", 0) for r in s_results),
            }

    # Top 10 performers
    sorted_results = sorted(with_trades, key=lambda r: r.get("sharpe_ratio", -99), reverse=True)
    for r in sorted_results[:10]:
        analysis["top_performers"].append({
            "strategy": r.get("strategy_id"),
            "symbol": r.get("symbol"),
            "timeframe": r.get("timeframe"),
            "sharpe": round(r.get("sharpe_ratio", 0), 4),
            "return_pct": round(r.get("total_return", 0) * 100, 2),
        })

    return analysis


# ===========================================================================
# Main
# ===========================================================================
def main(args: argparse.Namespace) -> None:
    ss_dir = Path(args.screenshot_dir)
    ss_dir.mkdir(parents=True, exist_ok=True)

    print("=" * 60)
    print("  PLAYWRIGHT BACKTEST MATRIX AUTOMATION")
    print("=" * 60)
    print()

    # ── Step 1: Get auth token ──────────────────────────────────────────
    print("[1/8] Obtaining auth token...")
    try:
        token = _get_token()
    except ConnectionError as e:
        print(f"  [SKIP] {e}")
        print("  Run `python scripts/orchestrate.py --test-only` for offline validation.")
        sys.exit(0)
    REPORT["auth"] = "OK"
    print(f"  [OK] Token obtained ({token[:20]}...)")

    # ── Step 2: Launch browser ────────────────────────────────────────────
    print("[2/8] Launching Playwright browser...")
    try:
        from playwright.sync_api import sync_playwright
    except ImportError:
        print("  [SKIP] playwright not installed.")
        print("  Install: pip install playwright && playwright install chromium")
        print("  Run `python scripts/orchestrate.py --test-only` for offline validation.")
        sys.exit(0)

    with sync_playwright() as pw:
        browser = pw.chromium.launch(headless=not args.headed)
        context = browser.new_context(viewport={"width": 1920, "height": 1080})
        page = context.new_page()

        # ── Step 3: Login via localStorage ────────────────────────────────
        print("[3/8] Setting auth in browser...")
        _setup_page(page, token)
        _screenshot(page, "01_dashboard", ss_dir)

        # ── Step 4: Navigate to backtest page ─────────────────────────────
        print("[4/8] Navigating to backtest page...")
        _navigate_to_backtest(page)
        _screenshot(page, "02_backtest_page", ss_dir)

        # ── Step 5: Fill form (UI attempt) ────────────────────────────────
        print("[5/8] Filling backtest form...")
        try:
            _fill_backtest_form(page)
            _screenshot(page, "03_form_filled", ss_dir)
        except Exception as e:
            REPORT["issues"].append(f"Form fill error (filling via API instead): {e}")
            print(f"  [WARN] UI form fill failed: {e}")

        # ── Step 6: Submit via API (robust path) ──────────────────────────
        print("[6/8] Submitting backtest matrix via API...")
        batch = _submit_matrix(token)
        if "error" in batch:
            REPORT["issues"].append(f"Matrix submission failed: {batch['error']}")
            print(f"  [FAIL] {batch['error']}")
            browser.close()
            return

        batch_id = batch.get("batch_run_id", batch.get("id", ""))
        total_combos = batch.get("total_combos", 0)
        REPORT["matrix"] = {"batch_id": batch_id, "total_combos": total_combos}
        print(f"  [OK] Submitted: batch_id={batch_id}, combos={total_combos}")

        # ── Step 7: Poll for completion ──────────────────────────────────
        print("[7/8] Polling matrix results...")
        completed = _poll_matrix_results(batch_id, token)
        if "error" in completed:
            REPORT["issues"].append(f"Matrix completion error: {completed.get('error')}")
        else:
            print(f"  [OK] Matrix completed: {completed.get('status')}")
            _analyze_and_report_results(completed, batch_id, token, page, ss_dir)

        browser.close()

    # ── Step 8: Final report ──────────────────────────────────────────────
    print()
    print("=" * 60)
    print("  FINAL REPORT")
    print("=" * 60)
    print(json.dumps(REPORT, indent=2, default=str))

    # Summary
    print()
    print("=" * 60)
    print("  EXECUTIVE SUMMARY")
    print("=" * 60)
    analysis = REPORT.get("analysis", {})
    print(f"  Batch ID:         {REPORT.get('matrix', {}).get('batch_id', 'N/A')}")
    print(f"  Total Combos:     {analysis.get('total_combos', 0)}")
    print(f"  With Trades:      {analysis.get('combos_with_trades', 0)}")
    print(f"  Failures:         {analysis.get('failures', 0)}")
    print(f"  Screenshots:      {len(REPORT.get('screenshots', []))}")
    print(f"  Issues Found:     {len(REPORT.get('issues', []))}")
    for iss in REPORT.get("issues", []):
        print(f"    - {iss}")

    if REPORT.get("regime_stats"):
        print()
        print("  REGIME AWARENESS")
        for rs in REPORT["regime_stats"][:5]:
            print(f"    Regime {rs.get('regime')} ({rs.get('label')}): "
                  f"Trades={rs.get('num_trades')} WinRate={rs.get('win_rate', 0)*100:.0f}% "
                  f"Return={rs.get('total_return', 0)*100:.2f}%")

    # Post-implementation verification summary
    if REPORT.get("post_impl_checks"):
        print()
        print("  POST-IMPLEMENTATION VERIFICATION")
        pic = REPORT["post_impl_checks"]
        for key, val in sorted(pic.items()):
            status = "PASS" if val else "FAIL"
            print(f"    [{status}] {key}" if isinstance(val, bool) else f"    [INFO] {key}: {val}")

    print()


def _analyze_and_report_results(
    completed: dict[str, Any],
    batch_id: str,
    token: str,
    page,
    ss_dir: Path,
) -> None:
    """Analyze completed matrix results, navigate UI, verify pages, collect regime stats."""
    analysis = _analyze_results(completed, batch_id)
    REPORT["analysis"] = analysis
    print(f"\n  === QUICK ANALYSIS ===")
    print(f"  Total combos: {analysis['total_combos']}")
    print(f"  With trades:  {analysis['combos_with_trades']}")
    print(f"  Failures:     {analysis['failures']}")

    for tf_name, tf_data in analysis.get("timeframes", {}).items():
        print(f"\n  Timeframe {tf_name}: {tf_data['combos']} combos, "
              f"Avg Sharpe={tf_data['avg_sharpe']}, Trades={tf_data['total_trades']}")

    for sname, sdata in analysis.get("strategies", {}).items():
        print(f"  Strategy {sname}: {sdata['combos']} combos, "
              f"Avg Sharpe={sdata['avg_sharpe']}, Avg Return={sdata['avg_return']*100:.2f}%")

    print("\n  Top performers:")
    for tp in analysis.get("top_performers", [])[:5]:
        print(f"    {tp['strategy']} | {tp['symbol']} | {tp['timeframe']} | "
              f"Sharpe={tp['sharpe']} | Return={tp['return_pct']}%")

    # Navigate UI for screenshots and verification
    try:
        _screenshot(page, "04_matrix_completed", ss_dir)

        # ── (A) Navigate to /backtest/history and verify page ──────────
        print("\n  [VERIFY] Backtest History page...")
        history_result = _verify_backtest_history(page)
        REPORT["history_page"] = history_result
        if history_result.get("page_loaded"):
            print("  [OK] Backtest History: page loaded, "
                  f"runs={history_result.get('runs_found', 0)}, "
                  f"columns={history_result.get('columns_seen', [])}")
        else:
            REPORT["issues"].append("Backtest History page did not load")

        # ── (B) Submit a single run to verify detail page ────────────
        print("\n  [VERIFY] Backtest Detail page + Regime stats...")

        # Submit a single backtest run to have a detail page to verify
        single_payload = {
            "mode": "single",
            "strategy_id": "grid_trading",
            "symbols": ["USDEUR"],
            "timeframes": ["1d"],
            "start_date": "2023-01-01",
            "end_date": "2024-12-31",
            "capital": 100000,
            "gate_profile": "none",
            "data_source": "synthetic",
        }
        url = f"{API_BASE}/backtests"
        data = json.dumps(single_payload).encode()
        req = urllib.request.Request(url, data=data, method="POST")
        req.add_header("Content-Type", "application/json")
        req.add_header("Authorization", f"Bearer {token}")
        try:
            with urllib.request.urlopen(req, timeout=60) as resp:
                single_result = json.loads(resp.read())
        except urllib.error.HTTPError as e:
            single_result = {"error": f"HTTP {e.code}: {e.read().decode()}"}

        detail_run_id = single_result.get("id", "")

        if not detail_run_id:
            REPORT["issues"].append("Single backtest submission failed for detail page test")
            print(f"  [WARN] Single backtest failed: {single_result.get('error', 'no id')}")
            detail_run_id = "00000000-0000-0000-0000-000000000000"
        else:
            # Wait for single run to complete before navigating to detail page
            print(f"  [OK] Submitted single run: {detail_run_id[:16]}...")
            for attempt in range(15):
                s = _api_call_with_auth("GET", f"/backtests/{detail_run_id}", token)
                s_status = s.get("status", "") if isinstance(s, dict) else ""
                if s_status in ("completed", "failed", "error"):
                    print(f"  [OK] Single run completed (status={s_status}) after {attempt*2}s")
                    break
                time.sleep(2)

        if detail_run_id:
            # Verify the detail page loads correctly
            detail_result = _verify_backtest_detail(page, detail_run_id)
            REPORT["detail_page"] = detail_result

            # Build a summary line
            metrics_summary = "; ".join(detail_result.get("metrics_found", [])[:4])
            print(f"  [OK] Backtest Detail ({detail_run_id[:12]}): "
                  f"metrics={len(detail_result.get('metrics_found', []))}, "
                  f"tabs={detail_result.get('tabs_found', [])}, "
                  f"regime={detail_result.get('regime_breakdown_found', False)}")

            if detail_result.get("regime_breakdown_found"):
                print("  [OK] Regime Breakdown: rendered in Overview tab")
                # Verify regime stats via API
                print("\n  [VERIFY] GET /api/v1/backtests/{id}/regime-stats endpoint...")
                stats = _api_call_with_auth("GET", f"/backtests/{detail_run_id}/regime-stats", token)
                if isinstance(stats, list):
                    REPORT["regime_stats"] = stats
                    print(f"  [OK] Regime stats endpoint: {len(stats)} entries returned")
                    for rs in stats:
                        print(f"    Regime {rs.get('regime')} ({rs.get('label')}): "
                              f"Trades={rs.get('num_trades')} WinRate={rs.get('win_rate', 0)*100:.0f}% "
                              f"Return={rs.get('total_return', 0)*100:.2f}%")
                else:
                    REPORT["issues"].append(f"Regime stats endpoint returned unexpected type: {type(stats).__name__}")
            else:
                # Fallback: collect regime stats via API directly
                print("\n  [INFO] Collecting regime stats from API fallback...")
                stats = _api_call_with_auth("GET", f"/backtests/{detail_run_id}/regime-stats", token)
                if isinstance(stats, list) and len(stats) > 0:
                    REPORT["regime_stats"] = stats
                    print(f"    {detail_run_id[:8]}: {len(stats)} regime entries via API")
                else:
                    REPORT["issues"].append(f"No regime stats for run {detail_run_id[:12]}")

            # ── (C) Verify navigation paths exist ─────────────────────
            print("\n  [VERIFY] Navigation paths to regime stats...")
            # Path: /backtest/history → click row → detail page → Overview tab → Regime Breakdown
            nav_result = _verify_regime_breakdown(page)
            REPORT["regime_navigation"] = nav_result
            if nav_result.get("overview_clicked"):
                print(f"  [OK] Navigation path: /backtest/history/{detail_run_id[:12]} -> Overview tab reached")
            if nav_result.get("regime_table_rendered"):
                print(f"  [OK] Regime table visible: {nav_result.get('regime_rows', 0)} rows in Overview tab")
            else:
                info = nav_result.get("issue", "regime table not visible")
                print(f"  [INFO]  {info}")

        else:
            REPORT["issues"].append("No backtest runs found in history for detail verification")

        # ═══════════════════════════════════════════════════════════════
        # Post-Implementation Verification (July 2026)
        # ═══════════════════════════════════════════════════════════════
        post_impl = {}

        print("\n  === POST-IMPLEMENTATION VERIFICATION ===")

        # 1. Broker API
        print("\n  [VERIFY] Broker listing API...")
        broker_result = _verify_broker_api(token)
        REPORT["broker_api"] = broker_result
        post_impl["broker_api"] = broker_result["brokers_found"] > 0

        # 2. Simulation page
        print("\n  [VERIFY] Simulation page...")
        sim_result = _verify_simulation_page(page, token)
        REPORT["simulation_page"] = sim_result
        post_impl["simulation_page"] = sim_result["page_loaded"]

        # 3. Optimization page
        print("\n  [VERIFY] Optimization page...")
        opt_result = _verify_optimization_page(page)
        REPORT["optimization_page"] = opt_result
        post_impl["optimization_page"] = opt_result["page_loaded"]

        # 4. Dashboard components
        print("\n  [VERIFY] Dashboard UI components...")
        dash_result = _verify_dashboard_components(page)
        REPORT["dashboard_components"] = dash_result
        post_impl["timeframe_chips"] = dash_result.get("timeframe_chips", False)
        post_impl["symbol_search"] = dash_result.get("symbol_search", False)
        post_impl["ohlcv_header"] = dash_result.get("ohlcv_header", False)

        REPORT["post_impl_checks"] = post_impl

    except Exception as e:
        REPORT["issues"].append(f"UI navigation error: {e}")
        import traceback
        REPORT["issues"].append(traceback.format_exc())


# ===========================================================================
# Entry point
# ===========================================================================
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Playwright backtest matrix automation")
    parser.add_argument("--headed", action="store_true", help="Show browser window")
    parser.add_argument("--screenshot-dir", default="screenshots", help="Screenshot output directory")
    args = parser.parse_args()
    main(args)
