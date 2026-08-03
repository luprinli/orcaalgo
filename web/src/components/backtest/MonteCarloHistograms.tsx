import { type MCStats } from './MonteCarloSummaryCard'
import { buildHistogramTraces } from '../../lib/histogram'

interface MonteCarloHistogramsProps {
  allPnlPct: number[]
  allMaxDDPct: number[]
  stats: MCStats
}

export default function MonteCarloHistograms({
  allPnlPct,
  allMaxDDPct,
  stats,
}: MonteCarloHistogramsProps) {
  if (!allPnlPct || !allPnlPct.length || !allMaxDDPct || !allMaxDDPct.length) {
    return (
      <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4" style={{ padding: 16 }}>
        <p className="text-muted">Insufficient data for Monte Carlo histograms.</p>
      </div>
    )
  }

  const { pnlTrace, ddTrace } = buildHistogramTraces(allPnlPct, allMaxDDPct, 20)

  const chartStyle: React.CSSProperties = {
    width: '100%',
    height: 40,
    fontFamily: 'monospace',
    fontSize: 10,
    display: 'flex',
    alignItems: 'flex-end',
    gap: 0,
    padding: '4px 8px',
    boxSizing: 'border-box',
  }

  const pnlMax = Math.max(...pnlTrace.y, 1)
  const ddMax = Math.max(...ddTrace.y, 1)

  return (
    <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4" style={{ padding: '12px 16px', display: 'flex', flexDirection: 'column', gap: 16 }}>
      <div>
        <h4 style={{ margin: '0 0 8px' }}>P/L% Distribution</h4>
        <div style={chartStyle}>
          {pnlTrace.y.map((count, i) => (
            <div
              key={i}
              title={`${pnlTrace.x[i]}: ${count} paths`}
              style={{
                flex: 1,
                height: `${Math.max(4, (count / pnlMax) * 100)}%`,
                minHeight: count > 0 ? 4 : 1,
                background: count > 0 ? 'var(--chart-line, #2962FF)' : 'transparent',
                opacity: 0.7,
                marginRight: 1,
                borderRadius: '2px 2px 0 0',
              }}
            />
          ))}
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 10, color: 'var(--muted-foreground)', padding: '0 8px' }}>
          <span>{pnlTrace.x[0]?.split(' to ')[0] ?? ''}%</span>
          <span>{pnlTrace.x[pnlTrace.x.length - 1]?.split(' to ')[1] ?? ''}%</span>
        </div>
      </div>

      <div>
        <h4 style={{ margin: '0 0 8px' }}>Max Drawdown Distribution</h4>
        <div style={chartStyle}>
          {ddTrace.y.map((count, i) => (
            <div
              key={i}
              title={`${ddTrace.x[i]}: ${count} paths`}
              style={{
                flex: 1,
                height: `${Math.max(4, (count / ddMax) * 100)}%`,
                minHeight: count > 0 ? 4 : 1,
                background: count > 0 ? 'var(--trading-danger, #EF5350)' : 'transparent',
                opacity: 0.7,
                marginRight: 1,
                borderRadius: '2px 2px 0 0',
              }}
            />
          ))}
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 10, color: 'var(--muted-foreground)', padding: '0 8px' }}>
          <span>{ddTrace.x[0]?.split(' to ')[0] ?? ''}%</span>
          <span>{ddTrace.x[ddTrace.x.length - 1]?.split(' to ')[1] ?? ''}%</span>
        </div>
      </div>

      <div style={{ fontSize: 11, color: 'var(--muted-foreground)', textAlign: 'center' }}>
        {stats.numSimulations.toLocaleString()} paths, {stats.numDays} days per path
      </div>
    </div>
  )
}
