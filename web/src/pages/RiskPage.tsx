import { useState, useEffect, useCallback } from 'react'
import { risk, monitor } from '../api/client'
import { useWebSocket } from '../hooks/useWebSocket'
import type { WSRiskData } from '../types/ws'
import type { RiskStatus } from '../types/api'

export default function RiskPage() {
  const [wsRisk, setWsRisk] = useState<WSRiskData | null>(null)
  const [restStatus, setRestStatus] = useState<RiskStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [twoFACode, setTwoFACode] = useState('')
  const [show2FA, setShow2FA] = useState<'stop' | 'resume' | null>(null)
  const [msg, setMsg] = useState('')
  const [regimeHistory, setRegimeHistory] = useState<Array<{ timestamp: string; regime: number }>>([])

  useWebSocket('risk', {
    onMessage: (data) => setWsRisk(data as WSRiskData),
    maxReconnects: 30,
    reconnectInterval: 2000,
  })

  const fetchStatus = useCallback(async () => {
    try {
      const [s, r] = await Promise.all([
        risk.status(),
        monitor.regimeHistory(),
      ])
      setRestStatus(s)
      setRegimeHistory(r?.history ?? [])
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchStatus()
    const interval = setInterval(fetchStatus, 10000)
    return () => clearInterval(interval)
  }, [fetchStatus])

  const halted = wsRisk?.halted ?? restStatus?.halted ?? false
  const equity = wsRisk?.equity ?? restStatus?.equity ?? 0
  const balance = wsRisk?.balance ?? restStatus?.balance ?? 0
  const dailyPnlPct = wsRisk?.daily_pnl_pct ?? restStatus?.daily_pnl_pct ?? 0
  const drawdownUsed = wsRisk?.drawdown_used ?? restStatus?.drawdown_used ?? 0
  const dailyLossUsed = wsRisk?.daily_loss_used ?? restStatus?.daily_loss_used ?? 0
  const dailyLimitPct = wsRisk?.daily_limit_pct ?? restStatus?.daily_limit_pct ?? 5
  const maxDdPct = wsRisk?.max_dd_pct ?? restStatus?.max_dd_pct ?? 10
  const regime = wsRisk?.regime ?? -1
  const regimeLabel = ['Calm', 'Trending', 'HighVol', 'Crisis']
  const regimeColors = ['var(--success)', 'var(--warn)', 'var(--danger-text)', 'var(--danger)']

  const handleEmergency = async (action: 'stop' | 'resume') => {
    setShow2FA(action)
    setTwoFACode('')
    setMsg('')
  }

  const confirm2FA = async () => {
    if (!show2FA || twoFACode.length !== 6) return
    try {
      if (show2FA === 'stop') {
        await risk.emergencyStop(twoFACode)
        setMsg('Emergency stop triggered — trading halted')
      } else {
        await risk.emergencyResume(twoFACode)
        setMsg('Trading resumed')
      }
      setShow2FA(null)
      setTwoFACode('')
      fetchStatus()
    } catch (err) {
      setMsg(err instanceof Error ? err.message : '2FA action failed')
    }
  }

  const format = (v: number | null | undefined, d = 2) =>
    v != null ? Number(v).toFixed(d) : '--'

  if (loading) {
    return (
      <div className="card">
        <p className="text-muted">Loading risk status...</p>
      </div>
    )
  }

  return (
    <div>
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>Risk Dashboard</h1>
        <span className={`badge ${halted ? 'badge-err' : 'badge-ok'}`}>
          {halted ? 'HALTED' : 'ACTIVE'}
        </span>
      </div>

      <div className="grid-3 mb-4">
        <div className="metric-card">
          <div className="metric-label">Balance</div>
          <div className="metric-value">${format(balance)}</div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Equity</div>
          <div className="metric-value">${format(equity)}</div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Daily P&L</div>
          <div className="metric-value" style={{ color: dailyPnlPct >= 0 ? 'var(--success)' : 'var(--danger)' }}>
            {format(dailyPnlPct, 2)}%
          </div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Daily Loss Used</div>
          <div className="metric-value" style={{ color: dailyLossUsed > 80 ? 'var(--danger-text)' : dailyLossUsed > 50 ? 'var(--warn)' : 'var(--success)' }}>
            {format(dailyLossUsed, 1)}%
          </div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Drawdown Used</div>
          <div className="metric-value" style={{ color: drawdownUsed > 80 ? 'var(--danger-text)' : drawdownUsed > 50 ? 'var(--warn)' : 'var(--success)' }}>
            {format(drawdownUsed, 1)}%
          </div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Regime</div>
          <div className="metric-value" style={{ color: regimeColors[regime] ?? 'var(--text-secondary)' }}>
            {regimeLabel[regime] ?? '--'}
          </div>
        </div>
      </div>

      <div className="card mb-4">
        <h2>Risk Limits</h2>
        <div className="grid-2">
          {[
            { label: 'Drawdown Used', value: drawdownUsed, max: maxDdPct },
            { label: 'Daily Loss Used', value: dailyLossUsed, max: dailyLimitPct },
          ].map((g) => {
            const pct = g.max > 0 ? Math.min(100, (g.value / g.max) * 100) : 0
            return (
              <div key={g.label}>
                <div className="flex-between mb-2">
                  <span className="text-muted">{g.label}</span>
                  <span style={{ fontWeight: 600 }}>{format(g.value, 1)}% / {format(g.max, 1)}%</span>
                </div>
                <div style={{ height: 8, background: 'var(--bg-input)', borderRadius: 4, overflow: 'hidden' }}>
                  <div
                    style={{
                      width: `${pct}%`,
                      height: '100%',
                      background: pct > 80 ? 'var(--danger)' : pct > 50 ? 'var(--warn)' : 'var(--success)',
                      borderRadius: 4,
                      transition: 'width .3s',
                    }}
                  />
                </div>
              </div>
            )
          })}
        </div>
      </div>

      <div className="card mb-4">
        <h2>Emergency Controls</h2>
        <p className="text-muted mb-4">
          {halted
            ? 'Trading is currently halted. Use emergency resume to re-enable.'
            : 'Use emergency stop to immediately halt all trading activity.'}
        </p>
        <div className="flex gap-2">
          <button
            className="btn btn-danger"
            onClick={() => handleEmergency('stop')}
            disabled={halted}
          >
            Emergency Stop
          </button>
          <button
            className="btn btn-primary"
            onClick={() => handleEmergency('resume')}
            disabled={!halted}
          >
            Resume Trading
          </button>
        </div>

        {show2FA && (
          <div className="mt-3" style={{ maxWidth: 300 }}>
            <label className="text-muted">2FA Code (required)</label>
            <div className="flex gap-2 mt-2">
              <input
                className="input"
                placeholder="000000"
                maxLength={6}
                value={twoFACode}
                onChange={(e) => setTwoFACode(e.target.value.replace(/\D/g, ''))}
                onKeyDown={(e) => e.key === 'Enter' && confirm2FA()}
              />
              <button className="btn btn-primary" onClick={confirm2FA} disabled={twoFACode.length !== 6}>
                Confirm
              </button>
              <button className="btn btn-outline" onClick={() => setShow2FA(null)}>
                Cancel
              </button>
            </div>
          </div>
        )}

        {msg && (
          <p className="text-muted mt-2" style={{ color: msg.includes('HALTED') || msg.includes('fail') ? 'var(--danger)' : 'var(--success)' }}>
            {msg}
          </p>
        )}
      </div>

      {regimeHistory.length > 0 && (
        <div className="card">
          <h2>Regime History (last {regimeHistory.length})</h2>
          <div style={{ overflowX: 'auto' }}>
            <table className="data-table">
              <thead>
                <tr>
                  <th>Time</th>
                  <th>Regime</th>
                </tr>
              </thead>
              <tbody>
                {regimeHistory.map((r, i) => (
                  <tr key={i}>
                    <td>{new Date(r.timestamp).toLocaleString()}</td>
                    <td style={{ color: regimeColors[r.regime] ?? 'var(--text-secondary)' }}>
                      {regimeLabel[r.regime] ?? r.regime}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}
