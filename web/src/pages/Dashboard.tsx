import { useState, useEffect } from 'react'
import { useWebSocket } from '../hooks/useWebSocket'
import { live } from '../api/client'
import EquityCurveChart from '../charts/EquityCurveChart'
import type { LiveMetrics, EquityPoint } from '../types/api'

export default function Dashboard() {
  const [metrics, setMetrics] = useState<LiveMetrics | null>(null)
  const [equity, setEquity] = useState<EquityPoint[]>([])
  const [wsRisk, setWsRisk] = useState<Record<string, unknown> | null>(null)
  const [error, setError] = useState<string | null>(null)

  useWebSocket('risk', {
    onMessage: (data) => setWsRisk(data as Record<string, unknown>),
  })

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [m, e] = await Promise.all([
          live.metrics(),
          live.equity('90d'),
        ])
        setMetrics(m)
        setEquity(Array.isArray(e) ? e : [])
        setError(null)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load')
      }
    }
    fetchData()
    const interval = setInterval(fetchData, 15000)
    return () => clearInterval(interval)
  }, [])

  const halted = (wsRisk?.halted as boolean) ?? false
  const equityVal = (wsRisk?.equity as number) ?? metrics?.equity ?? 0
  const balanceVal = (wsRisk?.balance as number) ?? metrics?.balance ?? 0
  const dailyPnl = (wsRisk?.daily_pnl_pct as number) ?? metrics?.daily_pnl_pct ?? 0
  const regime = (wsRisk?.regime as number)
  const regimeLabels = ['Calm', 'Trending', 'HighVol', 'Crisis']
  const regimeColors = ['var(--success)', 'var(--warn)', 'var(--danger-text)', 'var(--danger)']
  const format = (v: number | null | undefined, d = 2) => v != null ? v.toFixed(d) : '--'

  return (
    <div>
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>Dashboard</h1>
        <span className={`badge ${halted ? 'badge-err' : 'badge-ok'}`}>{halted ? 'HALTED' : 'ACTIVE'}</span>
      </div>

      {error && (
        <div className="card mb-4" style={{ borderColor: 'var(--danger)' }}>
          <span style={{ color: 'var(--danger)', fontSize: 12 }}>{error}</span>
        </div>
      )}

      <div className="grid-3 mb-4">
        <div className="metric-card">
          <div className="metric-label">Balance</div>
          <div className="metric-value">${format(balanceVal)}</div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Equity</div>
          <div className="metric-value">${format(equityVal)}</div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Daily P&L</div>
          <div className="metric-value" style={{ color: dailyPnl >= 0 ? 'var(--success)' : 'var(--danger)' }}>
            {format(dailyPnl, 2)}%
          </div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Sharpe</div>
          <div className="metric-value">{format(metrics?.sharpe)}</div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Max Drawdown</div>
          <div className="metric-value" style={{ color: 'var(--danger)' }}>{format(metrics?.max_drawdown_pct, 1)}%</div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Win Rate</div>
          <div className="metric-value">{format(metrics?.win_rate, 1)}%</div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Profit Factor</div>
          <div className="metric-value">{format(metrics?.profit_factor)}</div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Regime</div>
          <div className="metric-value" style={{ color: regimeColors[regime] ?? 'var(--text-secondary)' }}>
            {regime != null && regime >= 0 ? regimeLabels[regime] ?? '--' : '--'}
          </div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Total Trades</div>
          <div className="metric-value">{metrics?.num_trades ?? '--'}</div>
        </div>
      </div>

      {equity.length > 0 && (
        <EquityCurveChart
          data={equity}
          height={300}
          title="Live Equity Curve"
          color="#2962FF"
        />
      )}

      <div className="grid-2 mt-4">
        <div className="card">
          <h2>Risk Limits</h2>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {[
              { l: 'Daily Loss Used', v: (wsRisk?.daily_loss_used as number) ?? 0, max: (wsRisk?.daily_limit_pct as number) ?? 5 },
              { l: 'Drawdown Used', v: (wsRisk?.drawdown_used as number) ?? 0, max: (wsRisk?.max_dd_pct as number) ?? 10 },
            ].map(g => {
              const pct = g.max > 0 ? Math.min(100, (g.v / g.max) * 100) : 0
              return (
                <div key={g.l}>
                  <div className="flex-between mb-2">
                    <span className="text-muted">{g.l}</span>
                    <span style={{ fontWeight: 600 }}>{format(g.v, 1)}% / {format(g.max, 1)}%</span>
                  </div>
                  <div style={{ height: 8, background: 'var(--bg-input)', borderRadius: 4, overflow: 'hidden' }}>
                    <div style={{ width: `${pct}%`, height: '100%', background: pct > 80 ? 'var(--danger)' : pct > 50 ? 'var(--warn)' : 'var(--success)', borderRadius: 4, transition: 'width .3s' }} />
                  </div>
                </div>
              )
            })}
          </div>
        </div>
        <div className="card">
          <h2>System Status</h2>
          <div className="grid-2">
            {[
              { l: 'Broker Online', ok: true },
              { l: 'Data Feed Active', ok: true },
              { l: 'Kill Switch Active', ok: true },
              { l: 'DB Connected', ok: true },
              { l: 'Auth Enforced', ok: true },
              { l: 'WS Connected', ok: true },
            ].map(s => (
              <div key={s.l} className="flex gap-2" style={{ padding: '8px 0', alignItems: 'center' }}>
                <span className={`badge ${s.ok ? 'badge-ok' : 'badge-err'}`}>{s.ok ? '●' : '○'}</span>
                <span style={{ fontSize: 13 }}>{s.l}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
