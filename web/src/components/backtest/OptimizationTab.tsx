import type { OptimizationFootprint } from '../../types/api'

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
        <div className="metric-card">
          <div className="metric-label">Deflated Sharpe</div>
          <div className="metric-value">{optimization.deflated_sharpe?.toFixed(3)}</div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Conventional Sharpe</div>
          <div className="metric-value">{optimization.conventional_sharpe?.toFixed(3)}</div>
        </div>
        <div className="metric-card">
          <div className="metric-label">OOS Avg Sharpe</div>
          <div className="metric-value">{optimization.oos_average_sharpe?.toFixed(3)}</div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Sharpe Degradation</div>
          <div className="metric-value" style={{ color: (optimization.sharpe_degradation ?? 0) > 0.3 ? 'var(--danger)' : 'var(--success)' }}>
            {optimization.sharpe_degradation != null ? `${(optimization.sharpe_degradation * 100).toFixed(1)}%` : '--'}
          </div>
        </div>
        <div className="metric-card">
          <div className="metric-label">IVS</div>
          <div className="metric-value">{optimization.ivs?.toFixed(3)}</div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Walk-Forward Windows</div>
          <div className="metric-value">{optimization.walk_forward_windows}</div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Passed Windows</div>
          <div className="metric-value">{optimization.passed_windows}/{optimization.walk_forward_windows}</div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Grid Passes</div>
          <div className="metric-value">{optimization.grid_passes}</div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Bayesian Iterations</div>
          <div className="metric-value">{optimization.bayesian_iterations}</div>
        </div>
      </div>
      {optimization.best_params_json && (
        <div>
          <h3 style={{ fontSize: 13, margin: '12px 0 8px', color: 'var(--text-secondary)' }}>Optimized Parameters</h3>
          {paramDiffTable(optimization.best_params_json)}
        </div>
      )}
    </div>
  )
}
