"""Playwright test: BacktestPage Matrix mode — test auto-select + run."""
import json, re, sys, time, urllib.request
from pathlib import Path

API = "http://localhost:8080/api/v1"
UI = "http://localhost:5173"

def get_token():
    data = json.dumps({"username":"admin","password":"dev-admin-password-do-not-use-in-production"}).encode()
    req = urllib.request.Request(f"{API}/auth/login", data=data, headers={"Content-Type":"application/json"})
    return json.loads(urllib.request.urlopen(req, timeout=10).read())["access_token"]

token = get_token()
print(f"[OK] Token: {token[:20]}...")

from playwright.sync_api import sync_playwright

errors = []

with sync_playwright() as pw:
    browser = pw.chromium.launch(headless=True)
    page = browser.new_page(viewport={"width": 1920, "height": 1080})
    page.on("console", lambda msg: errors.append(msg.text[:200]) if msg.type == "error" else None)

    # Login
    page.goto(UI, wait_until="load", timeout=30000)
    page.evaluate(f"""window.localStorage.setItem('orca_auth', JSON.stringify({{"token":"{token}","refresh":"","roles":["admin","trader"]}}))""")
    page.reload(wait_until="load"); time.sleep(2)
    page.goto(f"{UI}/backtest", wait_until="load", timeout=30000)
    page.wait_for_timeout(2000)
    print(f"[OK] Page: {page.locator('h1').text_content()}")

    # ─── Step 1: Switch to Matrix mode via Mode select ───
    found = False
    for sel in page.locator("select").all():
        try:
            if sel.input_value() == "single":
                sel.select_option("matrix"); found = True; break
        except: pass
    assert found, "Could not find Mode select"
    page.wait_for_timeout(2000)  # Wait for auto-select effect
    print("[OK] Switched to Matrix")

    # ─── Step 2: Check run button text ───
    btn = page.locator("button").filter(has_text="Matrix").first
    btxt = btn.text_content() or ""
    print(f"[BTN] {btxt[:150]}")

    m = re.search(r"(\d+)\s*x\s*(\d+)\s*x\s*(\d+)", btxt)
    assert m, f"No counts in button: {btxt}"
    s, m_, t = int(m.group(1)), int(m.group(2)), int(m.group(3))
    print(f"[CHECK] {s} strats x {m_} syms x {t} tfs")
    assert s > 0, f"Strategies=0! Button: {btxt}"
    assert m_ > 0, f"Symbols=0! Button: {btxt}"
    assert t > 0, f"Timeframes=0! Button: {btxt}"
    print("[PASS] Run button shows all counts")

    # ─── Step 3: Set short date range ───
    page.locator("input[type='date']").first.fill("2024-01-02")
    page.locator("input[type='date']").nth(1).fill("2024-01-10")
    print("[OK] Date: 2024-01-02..2024-01-10")

    # ─── Step 4: Click Run ───
    print("[RUN] Clicking...")
    errors_before = len(errors)
    btn.click()
    page.wait_for_timeout(4000)

    # Check for immediate errors
    new_errs = len(errors) - errors_before
    if new_errs > 0:
        print(f"[WARN] {new_errs} new errors after click")
        for e in errors[errors_before:]:
            print(f"  {e[:150]}")

    # Check for "Select at least" error message
    err_msg = page.locator("text=Select at least")
    if err_msg.count() > 0 and err_msg.first.is_visible():
        print(f"[FAIL] Validation triggered: {err_msg.text_content()}")
        page.screenshot(path="test-results/bt_fail.png")
        browser.close()
        sys.exit(1)

    # Check button shows progress
    btxt2 = btn.text_content() or ""
    print(f"[BTN] After click: {btxt2[:120]}")

    # ─── Step 5: Wait for results (streaming) ───
    seen = False
    for i in range(30):
        page.wait_for_timeout(2000)
        # Check for table rows
        rows = page.locator("table.data-table tbody tr")
        n = rows.count()
        # Check for badge
        badge = page.locator(".badge-ok, .badge-warn")
        badge_text = badge.first.text_content() if badge.count() > 0 else ""
        print(f"  [{i*2+4}s] rows={n} badge={badge_text}")
        if n > 0 or badge_text in ("completed", "failed"):
            seen = True
            if n > 0:
                # Verify first row has trades
                cells = rows.first.locator("td")
                row_data = [cells.nth(j).text_content() for j in range(min(4, cells.count()))]
                print(f"  First row: {row_data}")
                trades = int(row_data[3]) if len(row_data) > 3 else 0  # col 3 = Trades
                print(f"  Trades in first row: {trades}")
            break

    # ─── Summary ───
    page.screenshot(path="test-results/bt_test_final.png")
    Path("test-results").mkdir(exist_ok=True)

    all_errs = errors[errors_before:] if errors_before else errors
    print(f"\n{'='*60}")
    print(f"  Results visible: {'PASS' if seen else 'FAIL'}")
    print(f"  New errors: {len(all_errs)}")
    for e in all_errs[:3]:
        print(f"    {e[:150]}")
    browser.close()
    sys.exit(0 if seen else 1)
