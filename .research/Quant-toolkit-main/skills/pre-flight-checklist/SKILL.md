---
name: pre-flight-checklist
description: Run the pre-flight safety checklist before flipping a trading bot from dry-run to live. Use when the user says "go live", "flip to live", "DRY_RUN false", "trade for real", "send it", "I'm about to run live", "pre-flight", "is the bot safe", or is about to launch the bot after any code change, model refit, or pause longer than 24 hours. Also use whenever a bot is being launched after a roadmap card closes — the change itself is the reason to re-check.
---

# Pre-flight checklist

Walk through the standard safety checks before a trading bot trades real money. The goal is to catch the things that have killed bots in the past — concurrent processes, stale models, missed config flips, regime mismatches — before they kill this one.

A pre-flight that takes 3 minutes can prevent an incident that takes a day to clean up. Run it every time, even when it feels redundant.

## The checklist

For each item below, verify before launch and report `pass` / `warn` / `fail`. A single `fail` blocks the launch; one or more `warn`s require explicit user acknowledgement.

### 1. Single-instance guard

Verify only one trading process can run at a time. Bots that don't enforce this will routinely run two copies — one from a cron, one started manually — which double-cancels and double-places orders.

Check: is there an active filesystem lock, a PID file, or another single-instance mechanism? Is any other process holding it? If two processes are eligible to trade right now, this is a `fail`.

### 2. Concurrent scheduled jobs

Even with a single-instance guard, look for *other* scheduled tasks that touch the same resources — order placement, the trade ledger, the model state. A cron that snapshots the order book is fine; a cron that runs a separate strategy on the same account is not.

Check: list active scheduled tasks (cron, launchd, systemd, etc.) and look for anything that could interleave with the bot. Conflict = `fail`; benign neighbors = `pass`.

### 3. Live trading flag

Confirm the environment is configured for live trading if and only if the user intends to trade live. Check both the *configured* setting and the *active* setting — config files and environment variables can disagree.

Check: is `DRY_RUN` (or equivalent) set to `false`? Is the override consistent across `.env`, the launch command, and any wrapper scripts? If unclear, this is a `fail`.

### 4. Fresh backtest evidence

Any code change that affects sizing, filters, or model output should have a backtest report dated *after* the change. Without that, the user is flying blind on whether the change improves or breaks the strategy.

Check: is there a backtest result (CSV, log file, or memo) timestamped after the most recent commit that touches `risk.py` / `*_strategy.py` / model weights? If older than the latest change, this is a `warn`. If no backtest exists at all, `fail`.

### 5. Model artifact freshness

Calibration coefficients, ensemble weights, and similar fitted artifacts drift. A bot running on coefficients fit six months ago against a different regime is taking a risk that probably won't show up until it costs money.

Check: how old is the production calibration artifact (e.g., `emos_coefficients.json` or equivalent)? If older than 60 days, `warn`. If older than 6 months, `fail`. The user can override but must explicitly acknowledge.

### 6. Run-time sanity

Most strategies have a preferred decision window — markets that are still actively re-pricing produce noisy fills, and markets near settlement produce thin liquidity. Confirm the current local time at each traded venue/cohort is in the strategy's intended window.

Check: for each cohort the bot is configured to trade, compute the current local time and compare against the strategy's `intended_window`. If any cohort is outside its window, `warn` and list which ones.

### 7. Recent commits visible

The bot should be running the code the user thinks it's running. Local commits that haven't been pushed, or pushed commits that haven't been pulled by the deployed copy, cause "I fixed that" moments that are actually "the fix isn't deployed."

Check: is the working tree clean? Is `HEAD` the same as the last deployed reference? If divergent, `fail`.

### 8. Risk caps consistent

Look for the trade and exposure caps and verify they're set to values the user has approved. Caps that drifted in a recent refactor are how an unintentional 5× sizing increase happens.

Check: read the cap values (`MAX_TRADE_USD`, `MAX_TOTAL_EXPOSURE_USD`, per-cohort caps) and surface them. The user must explicitly confirm before launch if any cap is at its highest historical value.

### 9. Recent attribution review

If the most recent P&L attribution review showed an open leak — a losing slice with n ≥ 30 — the user should have addressed it (filter change, sizing change, etc.) before scaling up. Unaddressed leaks indicate the strategy isn't in a state to grow.

Check: when was the last P&L attribution run? Were any slices flagged as actionable losses? If yes and not addressed, `warn`.

### 10. Telegram / alerts wired

The bot should be able to reach the user if something goes wrong. A bot that can't alert the user when it gets rate-limited or hits a circuit breaker is one Slack outage away from a silent failure.

Check: is the alert channel (Telegram, Slack, email, etc.) configured and verified end-to-end? A test ping should have been sent and received recently. If unverified, `warn`.

## How to respond

When the user asks to go live or asks for a pre-flight:

1. Walk through items 1–10 in order. Run each check (read files, check process state, query the ledger) before declaring `pass`.
2. Summarize with a single verdict line:
   - `green` — all 10 checks pass.
   - `yellow` — one or more `warn`s; user must acknowledge before proceeding.
   - `red` — at least one `fail`; do not launch.
3. List the failing/warning items with the specific reason and the suggested fix.
4. If `green`, give the user the exact launch command (or step-by-step actions) to start the bot live.
5. If `yellow` or `red`, ask explicit confirmation per warning before any launch command is offered.

## Don't

- Skip checks when the user is in a hurry. "I'm in a rush" is the exact context where pre-flight prevents the most damage.
- Mark a check `pass` without actually verifying. If you can't verify, mark it `unknown` and surface that.
- Launch on a `red` regardless of how the user pushes. The right answer is "fix the failing check first."

## Operational notes

The exact check implementations depend on the bot's stack (filesystem layout, env-var names, ledger location). Treat the 10-item list above as the canonical *what*; the *how* is project-specific and should live in the bot's own checks. When operating on an unfamiliar bot, ask the user to point you at the relevant files (lock file, env, scheduled-tasks list) rather than guessing.
