# Kill-Switch Procedures

## Activation
- POST /api/v1/emergency/stop (requires authentication + 2FA)
- Kill-switch triggers:
  - Closes all broker positions
  - Cancels all pending orders
  - Broadcasts halted status to WebSocket clients
  - Sends Telegram alert

## Verification
- GET /api/v1/risk/status
- Check kill_switch_history table: SELECT * FROM kill_switch_history ORDER BY triggered_at DESC LIMIT 5;

## Deactivation
- POST /api/v1/emergency/resume (requires authentication + 2FA)
- Re-entrancy guard: isLocked + killSwitchReady both checked before any execution

## Pre-flight Verification
- Run: orca preflight --strict (12-point checklist)
