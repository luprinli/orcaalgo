"""Comprehensive frontend audit: navigate every page, check console errors, verify content.

Usage:
    python scripts/frontend_audit.py [--headless] [--pages dashboard,backtest,...]

Requires:
    pip install playwright
    playwright install chromium
    (Go API server running on :8080)
"""

import argparse
import json
import os
import socket
import sys
import time
import urllib.error
import urllib.request
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

API_BASE = "http://localhost:8080/api/v1"
AUTH = {"username": "admin", "password": "dev-admin-password-do-not-use-in-production"}


def _server_reachable(host: str = "localhost", port: int = 8080) -> bool:
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(2)
    try:
        s.connect((host, port))
        s.close()
        return True
    except (socket.timeout, ConnectionRefusedError, OSError):
        return False


# All frontend routes from App.tsx — updated for July 2026 UI state
ALL_PAGES: list[dict[str, Any]] = [
    {"path": "/", "name": "Dashboard", "key_elements": ["Dashboard", "Equity", "Live"]},
    {"path": "/live", "name": "LiveTrading", "key_elements": ["Live", "Symbol", "Timeframe", "Indicators"]},
    {"path": "/execution", "name": "ExecutionPage", "key_elements": ["Execution", "Orders", "Position"]},
    {"path": "/risk", "name": "RiskPage", "key_elements": ["Risk", "Status", "Kill"]},
    {"path": "/market-data", "name": "MarketDataPage", "key_elements": ["Market", "Candles", "OHLC"]},
    {"path": "/calibrate", "name": "CalibratePage", "key_elements": ["Calibrat"]},
    {"path": "/attribute", "name": "AttributionPage", "key_elements": ["Attribut", "PnL"]},
    {"path": "/backtest", "name": "BacktestPage", "key_elements": ["Backtest", "Runner", "Strategy"]},
    {"path": "/backtest/history", "name": "BacktestHistory", "key_elements": ["Backtest History", "Compare"]},
    {"path": "/strategies", "name": "StrategiesPage", "key_elements": ["Strateg"]},
    {"path": "/indicators", "name": "IndicatorsPage", "key_elements": ["Indicator", "SMA", "RSI"]},
    {"path": "/simulate", "name": "SimulatePage", "key_elements": ["Simulat", "Generate"]},
    {"path": "/optimization", "name": "OptimizationPage", "key_elements": ["Optimiz", "Walk-Forward"]},
    {"path": "/accounts", "name": "AccountsPage", "key_elements": ["Account"]},
    {"path": "/propfirm", "name": "PropFirmPage", "key_elements": ["Prop", "FTMO"]},
    {"path": "/settings", "name": "SettingsPage", "key_elements": ["Setting"]},
    {"path": "/settings/2fa", "name": "TwoFAPage", "key_elements": ["2FA", "Two-Factor"]},
    {"path": "/admin", "name": "AdminPage", "key_elements": ["Admin", "System"]},
    {"path": "/admin/symbols", "name": "SymbolAdminPage", "key_elements": ["Symbol", "Provider"]},
    {"path": "/admin/universe", "name": "UniversePage", "key_elements": ["Universe"]},
    {"path": "/nonexistent", "name": "NotFoundPage", "key_elements": ["404", "not found", "Page not found"]},
]

REPORT: dict[str, Any] = {
    "audit_time": datetime.now(UTC).isoformat(),
    "results": [],
    "summary": {"total": 0, "passed": 0, "errors": 0, "warnings": 0},
}


def get_token() -> str:
    if not _server_reachable():
        raise ConnectionError("Go API server not running on :8080. Start with: python scripts/orchestrate.py")
    data = json.dumps(AUTH).encode()
    req = urllib.request.Request(f"{API_BASE}/auth/login", data=data, method="POST")
    req.add_header("Content-Type", "application/json")
    with urllib.request.urlopen(req, timeout=10) as resp:
        return json.loads(resp.read())["access_token"]


def audit_page(page, page_info: dict, base_url: str, ss_dir: Path) -> dict:
    """Navigate to a page, collect console errors, verify key elements."""
    result: dict[str, Any] = {
        "name": page_info["name"],
        "path": page_info["path"],
        "url": f"{base_url}{page_info['path']}",
        "status": "unknown",
        "console_errors": [],
        "console_warnings": [],
        "content_checks": {},
        "load_time_ms": 0,
    }

    console_messages: list[dict] = []

    def on_console(msg):
        entry = {"type": msg.type, "text": msg.text}
        console_messages.append(entry)
        if msg.type == "error":
            result["console_errors"].append(msg.text[:200])
        elif msg.type == "warning":
            result["console_warnings"].append(msg.text[:200])

    page.on("console", on_console)

    try:
        t0 = time.time()
        response = page.goto(result["url"], wait_until="domcontentloaded", timeout=30000)
        result["http_status"] = response.status if response else "unknown"
        page.wait_for_timeout(2000)  # Let React hydrate
        result["load_time_ms"] = round((time.time() - t0) * 1000)
    except Exception as e:
        result["status"] = "navigation_error"
        result["error"] = str(e)[:200]
        page.remove_listener("console", on_console)
        return result

    body_text = (page.locator("body").text_content() or "")[:500].lower()
    body_text = body_text.encode('ascii', errors='replace').decode()
    result["body_preview"] = body_text[:150]

    for elem in page_info.get("key_elements", []):
        found = elem.lower() in body_text
        result["content_checks"][elem] = found

    not_found_patterns = ["page not found", "not found", "404"]
    for p in not_found_patterns:
        if p in body_text and page_info["name"] != "NotFoundPage":
            result["content_checks"]["shows_404"] = True

    error_patterns = ["an error occurred", "failed to load", "unexpected error", "something went wrong"]
    for p in error_patterns:
        if p in body_text:
            result["content_checks"][f"error_text:{p}"] = True

    if any("cannot read properties" in e.lower() or "is not a function" in e.lower() for e in result["console_errors"]):
        result["status"] = "js_error"
    elif result["console_errors"]:
        result["status"] = "console_errors"
    elif result["http_status"] and result["http_status"] >= 400:
        result["status"] = "http_error"
    elif any(v for k, v in result["content_checks"].items() if k.startswith("error_text:")):
        result["status"] = "page_error"
    elif page_info["name"] == "NotFoundPage":
        result["status"] = "ok" if any(p in body_text for p in not_found_patterns) else "missing_404"
    elif any(result["content_checks"].get(e, False) for e in page_info.get("key_elements", [])):
        result["status"] = "ok"
    else:
        result["status"] = "missing_content"

    try:
        ss_name = page_info["name"].lower().replace(" ", "_")
        page.screenshot(path=str(ss_dir / f"{ss_name}.png"), full_page=False)
        result["screenshot"] = f"{ss_name}.png"
    except Exception as e:
        result["screenshot"] = f"error: {e}"

    page.remove_listener("console", on_console)
    return result


def run_api_audit(token: str) -> dict[str, Any]:
    """Quick audit of key API endpoints."""
    endpoints = [
        ("GET", "/risk/status"),
        ("GET", "/strategies"),
        ("GET", "/backtest-history"),
        ("GET", "/brokers"),
        ("GET", "/accounts"),
        ("GET", "/symbols"),
        ("GET", "/universe/current"),
        ("GET", "/settings"),
        ("GET", "/propfirm/profiles"),
        ("GET", "/indicators"),
        ("GET", "/providers"),
    ]
    results = {}
    for method, path in endpoints:
        try:
            url = f"{API_BASE}{path}"
            req = urllib.request.Request(url, method=method)
            req.add_header("Authorization", f"Bearer {token}")
            with urllib.request.urlopen(req, timeout=10) as resp:
                data = json.loads(resp.read())
            results[path] = {"status": resp.status, "type": type(data).__name__, "size": len(str(data))}
        except urllib.error.HTTPError as e:
            results[path] = {"status": e.code, "error": str(e)[:100]}
        except Exception as e:
            results[path] = {"status": "error", "error": str(e)[:100]}
    return results


def main():
    parser = argparse.ArgumentParser(description="OrcaAlgo frontend audit")
    parser.add_argument("--headless", action="store_true", default=True)
    parser.add_argument("--headed", action="store_true")
    parser.add_argument("--pages", type=str, help="Comma-separated page names to audit (default: all)")
    parser.add_argument("--base-url", default="http://localhost:5173")
    parser.add_argument("--screenshot-dir", default="screenshots/audit")
    args = parser.parse_args()

    ss_dir = Path(args.screenshot_dir)
    ss_dir.mkdir(parents=True, exist_ok=True)

    print("=" * 70)
    print("  ORCAALGO FRONTEND AUDIT")
    print(f"  Base URL: {args.base_url}")
    print(f"  Time: {datetime.now(UTC).isoformat()}")
    print("=" * 70)

    # ── Auth token ──
    try:
        token = get_token()
        print("\n[OK] Auth token obtained")
    except ConnectionError as e:
        print(f"\n[SKIP] {e}")
        print("Run `python scripts/orchestrate.py --test-only` for offline validation.")
        sys.exit(0)
    except Exception as e:
        print(f"\n[FAIL] Cannot obtain auth token: {e}")
        print("Ensure the Go API server is running on port 8080")
        sys.exit(1)

    # ── API audit ──
    print("\n[API AUDIT]")
    api_results = run_api_audit(token)
    REPORT["api_audit"] = api_results
    for path, result in sorted(api_results.items()):
        status = result.get("status", "?")
        marker = "OK" if isinstance(status, int) and status < 400 else "FAIL"
        extra = f" ({result.get('type', '')} len={result.get('size', '?')})" if "type" in result else f" ({result.get('error', '')})"
        print(f"  {marker} {path} -> {status}{extra}")

    # ── Frontend audit ──
    pages_to_audit = ALL_PAGES
    if args.pages:
        requested = set(args.pages.lower().split(","))
        pages_to_audit = [p for p in ALL_PAGES if p["name"].lower() in requested or p["path"] in requested]

    print(f"\n[FRONTEND AUDIT] {len(pages_to_audit)} pages")
    print("-" * 70)

    try:
        from playwright.sync_api import sync_playwright
    except ImportError:
        print("Playwright not installed. Skipping frontend audit.")
        print("  Install: pip install playwright && playwright install chromium")
        print("  Run `python scripts/orchestrate.py --test-only` for offline validation.")
        save_report()
        return

    print(f"\n[FRONTEND AUDIT] {len(pages_to_audit)} pages (via Playwright)")
    print("-" * 70)

    with sync_playwright() as pw:
        browser = pw.chromium.launch(headless=not args.headed)
        context = browser.new_context(viewport={"width": 1920, "height": 1080})
        page = context.new_page()
        page.goto(args.base_url, wait_until="load", timeout=30000)

        # Inject auth
        page.evaluate(
            f"""window.localStorage.setItem('orca_auth', JSON.stringify({{
            "token": "{token}",
            "refresh": "",
            "roles": ["admin", "trader"]
        }}));"""
        )
        page.reload(wait_until="load", timeout=30000)
        page.wait_for_timeout(2000)

        for i, page_info in enumerate(pages_to_audit):
            name = page_info["name"]
            path = page_info["path"]
            print(f"\n[{i+1}/{len(pages_to_audit)}] {name} ({path})")
            result = audit_page(page, page_info, args.base_url, ss_dir)
            REPORT["results"].append(result)

            status_icon = {"ok": "[OK]", "missing_content": "[WARN]", "missing_404": "[WARN]", "js_error": "[FAIL]", "console_errors": "[WARN]", "http_error": "[FAIL]", "page_error": "[FAIL]", "navigation_error": "[FAIL]"}.get(result["status"], "[?]")
            print(f"  {status_icon} Status: {result['status']}  ({result['load_time_ms']}ms)")
            if result["console_errors"]:
                for err in result["console_errors"][:3]:
                    safe = err[:120].encode('ascii', errors='replace').decode()
                    print(f"     Console error: {safe}")
            if result["console_warnings"]:
                for warn in result["console_warnings"][:2]:
                    safe = warn[:120].encode('ascii', errors='replace').decode()
                    print(f"     Console warn: {safe}")
            checks = {k: v for k, v in result.get("content_checks", {}).items() if not k.startswith("error_text:") and not k == "shows_404"}
            if checks:
                found = sum(1 for v in checks.values() if v)
                print(f"     Content: {found}/{len(checks)} key elements found  {list(checks.keys())}")

        browser.close()

    # ── Summary ──
    summary = REPORT["summary"]
    for r in REPORT["results"]:
        summary["total"] += 1
        if r["status"] == "ok":
            summary["passed"] += 1
        elif r["status"] in ("js_error", "http_error", "page_error", "navigation_error"):
            summary["errors"] += 1
        else:
            summary["warnings"] += 1

    print(f"\n{'='*70}")
    print(f"  AUDIT SUMMARY")
    print(f"{'='*70}")
    print(f"  Total pages:    {summary['total']}")
    print(f"  Passed:         {summary['passed']}")
    print(f"  Errors:         {summary['errors']}")
    print(f"  Warnings:       {summary['warnings']}")

    failures = [r for r in REPORT["results"] if r["status"] not in ("ok",)]
    if failures:
        print(f"\n  Pages needing attention:")
        for r in failures:
            print(f"    [{r['status']:20}] {r['name']:25} {r['path']}")
            if r.get("console_errors"):
                for e in r["console_errors"][:2]:
                    print(f"                        {e[:130]}")

    save_report()


def save_report():
    report_path = Path("test-results") / "frontend_audit_report.json"
    report_path.parent.mkdir(exist_ok=True)
    with open(report_path, "w") as f:
        json.dump(REPORT, f, indent=2, default=str)
    print(f"\n  Report saved: {report_path}")


if __name__ == "__main__":
    main()
