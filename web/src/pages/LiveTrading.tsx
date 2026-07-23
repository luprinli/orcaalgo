import { useState, useEffect, useCallback } from 'react'
import { useWebSocket } from '../hooks/useWebSocket'
import { live, orders as ordersApi, positions as positionsApi, risk } from '../api/client'
import EquityCurveChart from '../charts/EquityCurveChart'
import type { LiveMetrics, EquityPoint, Position, Order, TradeSummary, RiskStatus } from '../types/api'

export default function LiveTrading() {
  const [metrics, setMetrics] = useState<LiveMetrics | null>(null)
  const [equity, setEquity] = useState<EquityPoint[]>([])
  const [positions, setPositions] = useState<Position[]>([])
  const [orders, setOrders] = useState<Order[]>([])
  const [liveTrades, setLiveTrades] = useState<TradeSummary[]>([])
  const [riskStatus, setRiskStatus] = useState<RiskStatus | null>(null)
  const [wsTicks, setWsTicks] = useState<unknown[]>([])
  const [error, setError] = useState<string | null>(null)

  useWebSocket('ticks', { onMessage: (data) => setWsTicks(prev => [...prev.slice(-199), data]) })

  const fetchAll = useCallback(async () => {
    try {
      const [m, e, p, o, t, r] = await Promise.all([
        live.metrics(),
        live.equity('90d'),
        positionsApi.list().catch(() => ({ positions: [] })),
        ordersApi.list().catch(() => ({ orders: [] })),
        live.trades().catch(() => ({ trades: [] })),
        risk.status().catch(() => null),
      ])
      setMetrics(m)
      setEquity(Array.isArray(e) ? e : [])
      setPositions(Array.isArray(p?.positions) ? p.positions : [])
      setOrders(Array.isArray(o?.orders) ? o.orders : [])
      setLiveTrades(Array.isArray(t?.trades) ? t.trades : [])
      setRiskStatus(r)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    }
  }, [])

  useEffect(() => { fetchAll(); const i = setInterval(fetchAll, 10000); return () => clearInterval(i) }, [fetchAll])

  const format = (v: number | null | undefined, d = 2) => v != null ? v.toFixed(d) : '--'

  return (
    <div>
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>Live Trading</h1>
        <h2 style={{ margin: 0, fontSize: 13, color: 'var(--text-secondary)' }} className="mt-2">Positions & Orders</h2>
        <span className={`badge ${riskStatus?.halted ? 'badge-err' : 'badge-ok'}`}>
          {riskStatus?.halted ? 'HALTED' : 'ACTIVE'}
        </span>
      </div>

      {error && <div className="card mb-4" style={{ borderColor: 'var(--danger)' }}><span style={{ color: 'var(--danger)', fontSize: 12 }}>{error}</span></div>}

      <div className="grid-3 mb-4">
        {[
          { l: 'Equity', v: `$${format(riskStatus?.equity ?? metrics?.equity)}` },
          { l: 'Daily P&L', v: `${format(riskStatus?.daily_pnl_pct ?? metrics?.daily_pnl_pct)}%`, c: (riskStatus?.daily_pnl_pct ?? 0) >= 0 ? 'var(--success)' : 'var(--danger)' },
          { l: 'Open Positions', v: positions.length },
          { l: 'Active Orders', v: orders.length },
          { l: 'Today Trades', v: liveTrades.length },
          { l: 'Win Rate', v: `${format(metrics?.win_rate)}%` },
        ].map(m => (
          <div key={m.l} className="metric-card">
            <div className="metric-label orca-metric-card__label">{m.l}</div>
            <div className="metric-value" style={m.c ? { color: m.c } : {}}>{String(m.v)}</div>
          </div>
        ))}
      </div>

      {equity.length > 0 && (
        <EquityCurveChart data={equity} height={260} title="Live Equity Curve" color="#3fb950" />
      )}

      <div className="grid-2 mt-4">
        {positions.length > 0 && (
          <div className="card">
            <h2>Positions ({positions.length})</h2>
            <table className="data-table">
              <thead><tr><th>Symbol</th><th>Side</th><th>Qty</th><th>Entry</th><th>P&L</th></tr></thead>
              <tbody>
                {positions.map((p, i) => (
                  <tr key={i}>
                    <td><strong>{p.symbol}</strong></td>
                    <td>{p.side}</td>
                    <td>{p.quantity}</td>
                    <td>${format(p.average_entry_price)}</td>
                    <td style={{ color: (p.unrealized_pnl ?? 0) >= 0 ? 'var(--success)' : 'var(--danger)' }}>
                      ${format(p.unrealized_pnl)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {orders.length > 0 && (
          <div className="card">
            <h2>Active Orders ({orders.length})</h2>
            <table className="data-table">
              <thead><tr><th>Symbol</th><th>Side</th><th>Type</th><th>Qty</th><th>Price</th><th>State</th></tr></thead>
              <tbody>
                {orders.map((o, i) => (
                  <tr key={o.order_id ?? i}>
                    <td><strong>{o.symbol}</strong></td>
                    <td>{o.side}</td>
                    <td>{o.order_type}</td>
                    <td>{o.quantity}</td>
                    <td>{o.price != null ? `$${format(o.price)}` : '—'}</td>
                    <td><span className={`badge ${o.state === 'filled' ? 'badge-ok' : o.state === 'cancelled' ? 'badge-err' : 'badge-warn'}`}>{o.state}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {liveTrades.length > 0 && (
        <div className="card mt-4">
          <h2>Recent Trades ({liveTrades.length})</h2>
          <div style={{ overflowX: 'auto', maxHeight: 400, overflowY: 'auto' }}>
            <table className="data-table">
              <thead><tr><th>Symbol</th><th>Side</th><th>Qty</th><th>Entry</th><th>Exit</th><th>P&L</th><th>Date</th></tr></thead>
              <tbody>
                {liveTrades.map((t, i) => (
                  <tr key={t.id ?? i}>
                    <td><strong>{t.symbol}</strong></td>
                    <td>{t.side}</td>
                    <td>{t.quantity}</td>
                    <td>${format(t.entry_price)}</td>
                    <td>${format(t.exit_price)}</td>
                    <td style={{ color: (t.pnl ?? 0) >= 0 ? 'var(--success)' : 'var(--danger)' }}>${format(t.pnl)}</td>
                    <td style={{ fontSize: 11 }}>{t.entry_time ? new Date(t.entry_time).toLocaleDateString() : '—'}</td>
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
