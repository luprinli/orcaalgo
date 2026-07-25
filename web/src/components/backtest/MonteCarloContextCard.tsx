import type { MCStats } from './MonteCarloSummaryCard'

interface MCContextData {
  strategyId?: string
  symbol?: string
  timeframe?: string
  dataStart?: string
  dataEnd?: string
  barCount?: number
  commissionBps?: number
}

interface MonteCarloContextCardProps {
  data: MCContextData
  stats: MCStats
  seed?: number
}

export default function MonteCarloContextCard({ data, stats, seed }: MonteCarloContextCardProps) {
  return (
    <details open style={{ marginBottom: 16 }}>
      <summary style={{ cursor: 'pointer', fontWeight: 600, fontSize: 14, marginBottom: 8 }}>
        Simulation Context
      </summary>
      <table className="context-table">
        <tbody>
          {data.strategyId && (
            <tr><td>Strategy</td><td>{data.strategyId}</td></tr>
          )}
          {data.symbol && (
            <tr><td>Symbol</td><td>{data.symbol}</td></tr>
          )}
          {data.timeframe && (
            <tr><td>Timeframe</td><td>{data.timeframe}</td></tr>
          )}
          {data.barCount != null && (
            <tr><td>Historical Bars</td><td>{data.barCount.toLocaleString()}</td></tr>
          )}
          {data.dataStart && (
            <tr><td>Data Start</td><td>{data.dataStart}</td></tr>
          )}
          {data.dataEnd && (
            <tr><td>Data End</td><td>{data.dataEnd}</td></tr>
          )}
          {data.commissionBps != null && (
            <tr><td>Commission</td><td>{data.commissionBps.toFixed(1)} bps</td></tr>
          )}
          <tr><td>Iterations</td><td>{stats.numSimulations.toLocaleString()}</td></tr>
          <tr><td>Days per Sim</td><td>{stats.numDays}</td></tr>
          {seed != null && (
            <tr><td>Seed</td><td style={{ fontFamily: 'monospace' }}>{seed}</td></tr>
          )}
        </tbody>
      </table>
      <style>{`
        .context-table {
          width: auto;
          min-width: 300px;
          border-collapse: collapse;
          margin: 0;
        }
        .context-table td {
          border: 1px solid var(--border, #444);
          padding: 4px 10px;
          font-size: 12px;
        }
        .context-table td:first-child {
          font-weight: 600;
          background: var(--surface-secondary, #3a3a3a);
          min-width: 140px;
          color: var(--muted-foreground);
        }
        .context-table td:last-child {
          font-family: monospace;
          color: var(--foreground);
        }
      `}</style>
    </details>
  )
}
