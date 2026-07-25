import { useMemo } from 'react'
import { TradeSummary } from '../../types/api'
import MetricCard from '../../components/MetricCard'

interface AnalyticsTabProps {
  trades: TradeSummary[]
  dailyReturns?: { date: string; return_pct: number }[]
}

export default function AnalyticsTab({ trades, dailyReturns }: AnalyticsTabProps) {
  // Win rate by day of week
  const winRateByDay = useMemo(() => {
    const days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
    const byDay: Record<string, { wins: number; total: number }> = {}
    days.forEach(d => { byDay[d] = { wins: 0, total: 0 } })
    trades.filter(t => t.entry_time).forEach(t => {
      const day = days[new Date(t.entry_time).getUTCDay()]
      byDay[day].total++
      if ((t.pnl ?? 0) > 0) byDay[day].wins++
    })
    return days.map(d => ({
      day: d,
      winRate: byDay[d].total > 0 ? (byDay[d].wins / byDay[d].total * 100) : null,
      trades: byDay[d].total,
    }))
  }, [trades])

  // Win rate by hour of day (UTC)
  const winRateByHour = useMemo(() => {
    const byHour: Record<number, { wins: number; total: number }> = {}
    for (let h = 0; h < 24; h++) byHour[h] = { wins: 0, total: 0 }
    trades.filter(t => t.entry_time).forEach(t => {
      const h = new Date(t.entry_time).getUTCHours()
      byHour[h].total++
      if ((t.pnl ?? 0) > 0) byHour[h].wins++
    })
    return Array.from({ length: 24 }, (_, h) => ({
      hour: `${String(h).padStart(2, '0')}:00`,
      winRate: byHour[h].total > 0 ? (byHour[h].wins / byHour[h].total * 100) : null,
      trades: byHour[h].total,
    }))
  }, [trades])

  // Trade duration distribution (bucketed by hours)
  const durationBuckets = useMemo(() => {
    const buckets = { '<1h': 0, '1-4h': 0, '4-24h': 0, '1-3d': 0, '>3d': 0 }
    trades.filter(t => t.entry_time && t.exit_time).forEach(t => {
      const durMs = new Date(t.exit_time).getTime() - new Date(t.entry_time).getTime()
      const durH = durMs / (1000 * 60 * 60)
      if (durH < 1) buckets['<1h']++
      else if (durH < 4) buckets['1-4h']++
      else if (durH < 24) buckets['4-24h']++
      else if (durH < 72) buckets['1-3d']++
      else buckets['>3d']++
    })
    return Object.entries(buckets).map(([label, count]) => ({ label, count }))
  }, [trades])

  // PnL distribution stats
  const pnlStats = useMemo(() => {
    const pnls = trades.map(t => t.pnl ?? 0)
    const wins = pnls.filter(p => p > 0)
    const losses = pnls.filter(p => p < 0)
    const mean = pnls.length > 0 ? pnls.reduce((s, p) => s + p, 0) / pnls.length : 0
    const sorted = [...pnls].sort((a, b) => a - b)
    const median = sorted.length > 0 ? sorted[Math.floor(sorted.length / 2)] : 0
    return {
      mean: mean.toFixed(2),
      median: median.toFixed(2),
      maxWin: wins.length > 0 ? Math.max(...wins).toFixed(2) : '—',
      maxLoss: losses.length > 0 ? Math.min(...losses).toFixed(2) : '—',
      avgWin: wins.length > 0 ? (wins.reduce((s, w) => s + w, 0) / wins.length).toFixed(2) : '—',
      avgLoss: losses.length > 0 ? (losses.reduce((s, l) => s + l, 0) / losses.length).toFixed(2) : '—',
    }
  }, [trades])

  // MAE/MFE stats
  const maeMfeStats = useMemo(() => {
    const withExcursion = trades.filter(t => t.mae != null || t.mfe != null)
    if (withExcursion.length === 0) return null
    const avgMae = withExcursion.filter(t => t.mae != null).reduce((s, t) => s + (t.mae ?? 0), 0) / withExcursion.length
    const avgMfe = withExcursion.filter(t => t.mfe != null).reduce((s, t) => s + (t.mfe ?? 0), 0) / withExcursion.length
    const maeMfeRatio = avgMae !== 0 ? avgMfe / Math.abs(avgMae) : 0
    return { avgMae: avgMae.toFixed(2), avgMfe: avgMfe.toFixed(2), ratio: maeMfeRatio.toFixed(2) }
  }, [trades])

  // Rolling Sharpe from daily returns
  const rollingSharpe = useMemo(() => {
    if (!dailyReturns || dailyReturns.length < 30) return null
    const rolling: { label: string; sharpe: number }[] = []
    for (let window of [30, 60, 90]) {
      if (dailyReturns.length < window) continue
      const windowReturns = dailyReturns.slice(-window).map(d => d.return_pct)
      const avg = windowReturns.reduce((s, r) => s + r, 0) / window
      const variance = windowReturns.reduce((s, r) => s + (r - avg) ** 2, 0) / window
      const std = Math.sqrt(variance)
      const sharpe = std > 0 ? (avg / std) * Math.sqrt(252) : 0
      rolling.push({ label: `${window}d`, sharpe })
    }
    return rolling
  }, [dailyReturns])

  if (trades.length === 0) {
    return <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4"><h2>Trade Analytics</h2><p className="text-muted">No trades to analyze.</p></div>
  }

  return (
    <div className="space-y-4">
      {/* PnL Distribution Stats */}
      <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4">
        <h2>PnL Distribution</h2>
        <div className="grid grid grid-cols-3 gap-2 mt-3">
          <Metric label="Mean PnL" value={`$${pnlStats.mean}`} />
          <Metric label="Median PnL" value={`$${pnlStats.median}`} />
          <Metric label="Max Win" value={`$${pnlStats.maxWin}`} color="var(--trading-success)" />
          <Metric label="Max Loss" value={`$${pnlStats.maxLoss}`} color="var(--trading-danger)" />
          <Metric label="Avg Win" value={`$${pnlStats.avgWin}`} color="var(--trading-success)" />
          <Metric label="Avg Loss" value={`$${pnlStats.avgLoss}`} color="var(--trading-danger)" />
        </div>
      </div>

      {/* MAE/MFE */}
      {maeMfeStats && (
        <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4">
          <h2>MAE / MFE Analysis</h2>
          <div className="grid grid grid-cols-3 gap-2 mt-3">
            <Metric label="Avg MAE" value={`$${maeMfeStats.avgMae}`} color="var(--trading-danger)" />
            <Metric label="Avg MFE" value={`$${maeMfeStats.avgMfe}`} color="var(--trading-success)" />
            <Metric label="MFE/MAE Ratio" value={maeMfeStats.ratio} />
          </div>
        </div>
      )}

      {/* Win Rate by Day of Week */}
      <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4">
        <h2>Win Rate by Day of Week</h2>
        <div style={{ overflowX: 'auto' }}>
          <table>
            <thead>
              <tr><th>Day</th><th>Trades</th><th>Win Rate</th><th>Distribution</th></tr>
            </thead>
            <tbody>
              {winRateByDay.map(d => (
                <tr key={d.day}>
                  <td className="font-medium text-white">{d.day}</td>
                  <td>{d.trades}</td>
                  <td style={{ color: d.winRate != null && d.winRate >= 50 ? 'var(--trading-success)' : 'var(--trading-danger)' }}>
                    {d.winRate != null ? `${d.winRate.toFixed(1)}%` : '—'}
                  </td>
                  <td>
                    <div className="h-2 bg-slate-700 rounded-full overflow-hidden" style={{ maxWidth: 120 }}>
                      <div className="h-full rounded-full" style={{
                        width: `${d.winRate ?? 0}%`,
                        backgroundColor: (d.winRate ?? 0) >= 50 ? '#22c55e' : '#ef4444',
                      }} />
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Duration Distribution */}
      <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4">
        <h2>Trade Duration Distribution</h2>
        <div style={{ overflowX: 'auto' }}>
          <table>
            <thead>
              <tr><th>Duration</th><th>Count</th><th>% of Total</th></tr>
            </thead>
            <tbody>
              {durationBuckets.map(d => (
                <tr key={d.label}>
                  <td className="font-medium text-white">{d.label}</td>
                  <td>{d.count}</td>
                  <td>{trades.length > 0 ? (d.count / trades.length * 100).toFixed(1) : 0}%</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Rolling Sharpe */}
      {rollingSharpe && (
        <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4">
          <h2>Rolling Sharpe Ratio</h2>
          <div className="grid grid grid-cols-3 gap-2 mt-3">
            {rollingSharpe.map(r => (
              <Metric key={r.label} label={`${r.label} Rolling`} value={r.sharpe.toFixed(2)}
                color={r.sharpe >= 1 ? 'var(--trading-success)' : r.sharpe >= 0 ? 'var(--trading-warning)' : 'var(--trading-danger)'} />
            ))}
          </div>
        </div>
      )}

      {/* Win Rate by Hour */}
      <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4">
        <h2>Win Rate by Hour (UTC)</h2>
        <div style={{ overflowX: 'auto' }}>
          <table>
            <thead>
              <tr><th>Hour</th><th>Trades</th><th>Win Rate</th></tr>
            </thead>
            <tbody>
              {winRateByHour.filter(h => h.trades > 0).map(h => (
                <tr key={h.hour}>
                  <td className="font-medium text-white">{h.hour}</td>
                  <td>{h.trades}</td>
                  <td style={{ color: h.winRate != null && h.winRate >= 50 ? 'var(--trading-success)' : 'var(--trading-danger)' }}>
                    {h.winRate != null ? `${h.winRate.toFixed(1)}%` : '—'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}

function Metric({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <MetricCard label={label} value={value} color={color === 'var(--trading-success)' ? 'positive' : color === 'var(--trading-danger)' ? 'negative' : 'default'} />
  )
}
