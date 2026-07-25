import type { OptimizationFootprint } from '../../types/api'
import MetricCard from '../../components/MetricCard'

function paramDiffTable(bestParamsJson: string) {
  let parsed: Record<string, unknown> = {}
  try { parsed = JSON.parse(bestParamsJson) } catch { return null }
  const entries = Object.entries(parsed)
  if (entries.length === 0) return null

  return (
    <div style={{ overflowX: 'auto' }}>
      <table className="data-table" style={{ fontSize: 11 }}>
        <thead>
          <tr>
            <th>Parameter</th>
            <th>Optimized</th>
          </tr>
        </thead>
        <tbody>
          {entries.map(([key, value]) => (
            <tr key={key}>
              <td style={{ fontFamily: 'monospace', textTransform: 'lowercase' }}>{key}</td>
              <td style={{ fontFamily: 'monospace', color: 'var(--accent-text)', fontWeight: 600 }}>
                {typeof value === 'number' ? value.toFixed(4) : String(value)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

interface Props {
  optimization: OptimizationFootprint | null
}

export default function OptimizationTab({ optimization }: Props) {
  if (!optimization) {
    return <p className="text-muted">No optimization data available for this backtest.</p>
  }

  return (
    <div>
      <h2>Walk-Forward Optimization</h2>
      <div className="metric-grid mb-3">
        <MetricCard label="Deflated Sharpe" value={optimization.deflated_sharpe?.toFixed(3) ?? '--'} />
        <MetricCard label="Conventional Sharpe" value={optimization.conventional_sharpe?.toFixed(3) ?? '--'} />
        <MetricCard label="OOS Avg Sharpe" value={optimization.oos_average_sharpe?.toFixed(3) ?? '--'} />
        <MetricCard label="Sharpe Degradation" value={optimization.sharpe_degradation != null ? `${(optimization.sharpe_degradation * 100).toFixed(1)}%` : '--'} color={optimization.sharpe_degradation != null ? 'auto' : 'default'} />
        <MetricCard label="IVS" value={optimization.ivs?.toFixed(3) ?? '--'} />
        <MetricCard label="Walk-Forward Windows" value={optimization.walk_forward_windows ?? 0} format="number" />
        <MetricCard label="Passed Windows" value={`${optimization.passed_windows}/${optimization.walk_forward_windows}`} />
        <MetricCard label="Grid Passes" value={optimization.grid_passes ?? 0} format="number" />
        <MetricCard label="Bayesian Iterations" value={optimization.bayesian_iterations ?? 0} format="number" />
      </div>
      {optimization.best_params_json && (
        <div>
          <h3 style={{ fontSize: 13, margin: '12px 0 8px', color: 'var(--muted-foreground)' }}>Optimized Parameters</h3>
          {paramDiffTable(optimization.best_params_json)}
        </div>
      )}
    </div>
  )
}
