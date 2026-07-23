export interface MCStats {
  numSimulations: number
  numDays: number
  avgPnlPct: number
  medianPnlPct: number
  p5PnlPct: number
  p10PnlPct: number
  avgMaxDDPct: number
  medianMaxDDPct: number
  p95MaxDDPct: number
  bustProbability: number
}

interface MonteCarloSummaryCardProps {
  stats: MCStats
}

export default function MonteCarloSummaryCard({ stats }: MonteCarloSummaryCardProps) {
  const format = (v: number) => v.toFixed(2)

  return (
    <div className="card" style={{ padding: '12px 16px' }}>
      <h4 style={{ margin: '0 0 10px' }}>Monte Carlo Summary</h4>
      <table className="summary-stats-table">
        <tbody>
          <tr>
            <td>Avg. Return</td>
            <td>{format(stats.avgPnlPct)}%</td>
          </tr>
          <tr>
            <td>Median Return</td>
            <td>{format(stats.medianPnlPct)}%</td>
          </tr>
          <tr>
            <td>5th Percentile (VaR 95%)</td>
            <td>{format(stats.p5PnlPct)}%</td>
          </tr>
          <tr>
            <td>10th Percentile (VaR 90%)</td>
            <td>{format(stats.p10PnlPct)}%</td>
          </tr>
          <tr>
            <td colSpan={2} style={{ height: 8, border: 'none', background: 'none' }} />
          </tr>
          <tr>
            <td>Avg. Drawdown</td>
            <td>{format(stats.avgMaxDDPct)}%</td>
          </tr>
          <tr>
            <td>Median Drawdown</td>
            <td>{format(stats.medianMaxDDPct)}%</td>
          </tr>
          <tr>
            <td>95th Percentile Drawdown</td>
            <td>{format(stats.p95MaxDDPct)}%</td>
          </tr>
          <tr>
            <td colSpan={2} style={{ height: 8, border: 'none', background: 'none' }} />
          </tr>
          <tr>
            <td>Probability of Loss</td>
            <td style={{ color: stats.bustProbability > 0.5 ? 'var(--danger)' : 'var(--text-primary)' }}>
              {format(stats.bustProbability * 100)}%
            </td>
          </tr>
          <tr>
            <td>Simulations</td>
            <td>{stats.numSimulations.toLocaleString()}</td>
          </tr>
        </tbody>
      </table>
      <style>{`
        .summary-stats-table {
          width: auto;
          min-width: 300px;
          border-collapse: collapse;
          margin: 0;
        }
        .summary-stats-table td {
          border: 1px solid var(--border, #444);
          padding: 5px 10px;
          font-size: 13px;
        }
        .summary-stats-table td:first-child {
          font-weight: 600;
          background: var(--surface-secondary, #3a3a3a);
          min-width: 200px;
          color: var(--text-secondary);
        }
        .summary-stats-table td:last-child {
          text-align: right;
          min-width: 80px;
          font-family: monospace;
          color: var(--text-primary);
        }
      `}</style>
    </div>
  )
}
