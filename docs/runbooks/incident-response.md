# Incident Response Playbook

## Alert: KillSwitchActive (critical)
1. Check reason: GET /api/v1/risk/status
2. Review positions: GET /api/v1/positions
3. Check kill_switch_history for trigger details
4. Investigate root cause before resuming
5. Resume only after confirming safe: POST /api/v1/emergency/resume

## Alert: BrokerDisconnected (critical)
1. Check broker status: GET /api/v1/brokers
2. Verify API keys are valid
3. Check network connectivity
4. Paper trading fallback activates automatically

## Alert: HighDrawdown (warning)
1. Check daily PnL: GET /api/v1/risk/status
2. Review open positions
3. Evaluate whether to manually intervene

## Alert: LatencySpike (warning)
1. Check DB pool stats in application logs
2. Check PostgreSQL load
3. Review tick processing backlog
