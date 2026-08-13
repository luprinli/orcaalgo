import { useEffect, useState } from 'react'
import { backtests } from '../../api/client'
import type { RegimeStat, TradeDistribution } from '../../types/api'
import MetricCard from '../MetricCard'

interface Props {
  backtestId: string
  regimeStats: RegimeStat[]
}

export default function OverviewTab({ backtestId, regimeStats }: Props) {
  const [dist, setDist] = useState<TradeDistribution | null>(null)

  useEffect(() => {
    let active = true
    backtests.tradeDistribution(backtestId).then(d => {
      if (active) setDist(d)
    }).catch(() => {
      if (active) setDist(null)
    })
    return () => { active = false }
  }, [backtestId])

  return (
    <div>
      {dist && dist.total_trades > 0 && (
        <div className="mb-4">
          <h2>Trade Distribution</h2>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 12 }}>
            <MetricCard label="Trades" value={dist.total_trades} format="number" />
            <MetricCard label="Win Rate" value={dist.win_rate_pct} format="percent_raw" color="auto" />
            <MetricCard label="Avg Trade P&L" value={dist.avg_trade_pnl} format="currency" color="auto" />
            <MetricCard label="Median Trade P&L" value={dist.median_trade_pnl} format="currency" color="auto" />
            <MetricCard label="Best Trade" value={dist.best_trade} format="currency" color="positive" />
            <MetricCard label="Worst Trade" value={dist.worst_trade} format="currency" color="negative" />
            <MetricCard label="Avg Win" value={dist.avg_winning_pnl} format="currency" color="positive" />
            <MetricCard label="Avg Loss" value={dist.avg_losing_pnl} format="currency" color="negative" />
            <MetricCard label="Avg Hold" value={dist.avg_trade_duration_hours} format="decimal" />
            <MetricCard label="Median Hold" value={dist.median_trade_duration_hours} format="decimal" />
            <MetricCard label="Unique Tickers" value={dist.unique_tickers} format="number" />
          </div>
        </div>
      )}

      {regimeStats && regimeStats.length > 0 && (
        <div className="mb-4">
          <h2>Regime Breakdown</h2>
          <div style={{ overflowX: 'auto' }}>
            <table className="data-table">
              <thead>
                <tr>
                  <th>Regime</th>
                  <th>Trades</th>
                  <th>Win Rate</th>
                  <th>Total Return</th>
                  <th>Max DD</th>
                  <th>Profit Factor</th>
                </tr>
              </thead>
              <tbody>
                {regimeStats.map((r) => (
                  <tr key={r.regime}>
                    <td>{r.label}</td>
                    <td>{r.num_trades}</td>
                    <td>{(r.win_rate * 100).toFixed(1)}%</td>
                    <td style={{ color: r.total_return >= 0 ? 'var(--trading-success)' : 'var(--trading-danger)' }}>
                      {(r.total_return * 100).toFixed(1)}%
                    </td>
                    <td>{(r.max_drawdown * 100).toFixed(1)}%</td>
                    <td>{r.profit_factor?.toFixed(2)}</td>
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
