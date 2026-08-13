import { useState } from 'react'
import { backtests } from '../../api/client'
import type { TradeSummary, TradeDetail } from '../../types/api'

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

interface Props {
  backtestId: string
  trades: TradeSummary[]
  filteredTrades: TradeSummary[]
  filteredMonth: { year: number; month: number } | null
  onClearFilter: () => void
}

function fmt(ts?: string): string {
  if (!ts) return '—'
  const d = new Date(ts)
  return isNaN(d.getTime()) ? ts : d.toLocaleString()
}

export default function TradesTab({ backtestId, trades, filteredTrades, filteredMonth, onClearFilter }: Props) {
  const [detail, setDetail] = useState<TradeDetail | null>(null)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [loadingDetail, setLoadingDetail] = useState(false)

  const openDetail = async (t: TradeSummary) => {
    setSelectedId(t.id)
    setLoadingDetail(true)
    try {
      const d = await backtests.tradeDetail(backtestId, t.id)
      setDetail(d)
    } catch {
      setDetail(null)
    } finally {
      setLoadingDetail(false)
    }
  }

  if (filteredTrades.length === 0) {
    return (
      <div>
        <p className="text-muted">No trades{filteredMonth ? ' in selected month' : ' in this backtest'}.</p>
      </div>
    )
  }

  return (
    <div>
      <div>
        {filteredMonth && (
          <div className="flex-between mb-2">
            <span className="text-muted" style={{ fontSize: 12 }}>
              Showing trades for {MONTHS[filteredMonth.month - 1]} {filteredMonth.year}
              ({filteredTrades.length} of {trades.length} trades)
            </span>
            <button className="btn btn-outline" style={{ fontSize: 11, padding: '2px 8px' }} onClick={onClearFilter}>
              Clear filter
            </button>
          </div>
        )}
        <div style={{ overflowX: 'auto', maxHeight: 500, overflowY: 'auto' }}>
          <table className="data-table">
            <thead>
              <tr>
                <th>Symbol</th>
                <th>Side</th>
                <th>Qty</th>
                <th>Entry</th>
                <th>Exit</th>
                <th>PnL</th>
                <th>Return</th>
                <th>MAE</th>
                <th>MFE</th>
                <th>Duration</th>
                <th>Exit Reason</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {filteredTrades.map((t) => (
                <tr key={t.id} style={{ cursor: 'pointer' }} onClick={() => openDetail(t)}>
                  <td>{t.symbol}</td>
                  <td style={{ color: t.side === 'BUY' ? 'var(--trading-success)' : 'var(--trading-danger)' }}>{t.side}</td>
                  <td>{t.quantity}</td>
                  <td>${t.entry_price?.toFixed(2)}</td>
                  <td>${t.exit_price?.toFixed(2)}</td>
                  <td style={{ color: (t.pnl ?? 0) >= 0 ? 'var(--trading-success)' : 'var(--trading-danger)' }}>${t.pnl?.toFixed(2)}</td>
                  <td style={{ color: (t.pnl_pct ?? 0) >= 0 ? 'var(--trading-success)' : 'var(--trading-danger)' }}>{t.pnl_pct?.toFixed(2)}%</td>
                  <td>${t.mae?.toFixed(2)}</td>
                  <td>${t.mfe?.toFixed(2)}</td>
                  <td>{t.hold_duration?.toFixed(1)}h</td>
                  <td>{t.exit_reason}</td>
                  <td className="text-muted" style={{ fontSize: 11 }}>drill&nbsp;▸</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {selectedId && (
        <div className="mt-4 p-4 border rounded" style={{ background: 'var(--background)' }}>
          <div className="flex-between mb-3">
            <h4 className="m-0">Trade #{selectedId} {detail ? `— ${detail.symbol} ${detail.side}` : ''}</h4>
            <button className="btn btn-outline" style={{ fontSize: 11, padding: '2px 8px' }} onClick={() => { setSelectedId(null); setDetail(null) }}>
              Close
            </button>
          </div>
          {loadingDetail ? (
            <p className="text-muted">Loading detail...</p>
          ) : !detail ? (
            <p className="text-muted">Unable to load trade detail.</p>
          ) : (
            <div className="grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))', gap: 16 }}>
              <div>
                <table className="data-table">
                  <thead><tr><th>Level</th><th>Price</th></tr></thead>
                  <tbody>
                    <tr><td>Entry</td><td>${detail.entry_price?.toFixed(2)}</td></tr>
                    {detail.stop_price > 0 && <tr><td>Stop</td><td className="text-trading-danger">${detail.stop_price.toFixed(2)}</td></tr>}
                    {detail.take_price > 0 && <tr><td>Target</td><td className="text-trading-success">${detail.take_price.toFixed(2)}</td></tr>}
                    <tr><td>Exit</td><td>${detail.exit_price?.toFixed(2)}</td></tr>
                    <tr><td>Lowest</td><td className="text-trading-danger">${detail.lowest_price?.toFixed(2)}</td></tr>
                    <tr><td>Highest</td><td className="text-trading-success">${detail.highest_price?.toFixed(2)}</td></tr>
                  </tbody>
                </table>
                <div className="mt-3 space-y-1 text-sm">
                  <div><span className="text-muted">Entry time:</span> {fmt(detail.entry_time)}</div>
                  <div><span className="text-muted">Exit time:</span> {fmt(detail.exit_time)}</div>
                  <div><span className="text-muted">Exit reason:</span> {detail.exit_reason}</div>
                  <div><span className="text-muted">PnL:</span> <span style={{ color: (detail.pnl ?? 0) >= 0 ? 'var(--trading-success)' : 'var(--trading-danger)' }}>${detail.pnl?.toFixed(2)} ({detail.pnl_pct?.toFixed(2)}%)</span></div>
                </div>
              </div>
              <div>
                <p className="text-muted mb-2" style={{ fontSize: 12 }}>Change history (append-only)</p>
                {detail.changes.length === 0 ? (
                  <p className="text-muted text-sm">No recorded changes.</p>
                ) : (
                  <div style={{ overflowX: 'auto', maxHeight: 260, overflowY: 'auto' }}>
                    <table className="data-table">
                      <thead><tr><th>Time</th><th>Field</th><th>From</th><th>To</th><th>Reason</th></tr></thead>
                      <tbody>
                        {detail.changes.map((c, i) => (
                          <tr key={i}>
                            <td className="text-xs">{fmt(c.timestamp)}</td>
                            <td>{c.field}</td>
                            <td>{c.from || '—'}</td>
                            <td>{c.to || '—'}</td>
                            <td>{c.reason || '—'}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
