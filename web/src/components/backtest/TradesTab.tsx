import type { TradeSummary } from '../../types/api'

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

interface Props {
  trades: TradeSummary[]
  filteredTrades: TradeSummary[]
  filteredMonth: { year: number; month: number } | null
  onClearFilter: () => void
}

export default function TradesTab({ trades, filteredTrades, filteredMonth, onClearFilter }: Props) {
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
              </tr>
            </thead>
            <tbody>
              {filteredTrades.map((t) => (
                <tr key={t.id}>
                  <td>{t.symbol}</td>
                  <td style={{ color: t.side === 'BUY' ? 'var(--success)' : 'var(--danger)' }}>{t.side}</td>
                  <td>{t.quantity}</td>
                  <td>${t.entry_price?.toFixed(2)}</td>
                  <td>${t.exit_price?.toFixed(2)}</td>
                  <td style={{ color: (t.pnl ?? 0) >= 0 ? 'var(--success)' : 'var(--danger)' }}>${t.pnl?.toFixed(2)}</td>
                  <td style={{ color: (t.pnl_pct ?? 0) >= 0 ? 'var(--success)' : 'var(--danger)' }}>{t.pnl_pct?.toFixed(2)}%</td>
                  <td>${t.mae?.toFixed(2)}</td>
                  <td>${t.mfe?.toFixed(2)}</td>
                  <td>{t.hold_duration?.toFixed(1)}h</td>
                  <td>{t.exit_reason}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
