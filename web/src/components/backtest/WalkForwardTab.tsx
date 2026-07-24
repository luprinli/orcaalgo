import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { WalkForwardResponse } from '../../types/api'

interface WalkForwardTabProps {
  data: WalkForwardResponse | null
}

export function WalkForwardTab({ data }: WalkForwardTabProps) {
  const { t } = useTranslation()

  if (!data) {
    return (
      <div className="card">
        <h2>Walk-Forward Analysis</h2>
        <p className="text-muted">Loading walk-forward data...</p>
      </div>
    )
  }

  if (data.message) {
    return (
      <div className="card">
        <h2>Walk-Forward Analysis</h2>
        <p className="text-muted">{data.message}</p>
      </div>
    )
  }

  if (data.total_windows === 0) {
    return (
      <div className="card">
        <h2>Walk-Forward Analysis</h2>
        <p className="text-muted">No walk-forward windows available for this run. Run an optimized backtest to generate walk-forward metrics.</p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Summary */}
      <div className="card">
        <h2>Walk-Forward Summary</h2>
        <div className="grid grid-3 mt-3">
          <Metric label="Windows Passed" value={`${data.passed_windows} / ${data.total_windows}`}
            color={data.passed_windows === data.total_windows ? 'var(--success)' : data.passed_windows > data.total_windows / 2 ? 'var(--warn)' : 'var(--danger)'} />
          <Metric label="OOS Avg Sharpe" value={data.oos_avg_sharpe?.toFixed(3) ?? '—'} />
          <Metric label="Sharpe Degradation" value={data.sharpe_degradation != null ? (data.sharpe_degradation * 100).toFixed(1) + '%' : '—'}
            color={data.sharpe_degradation != null && data.sharpe_degradation < 0.3 ? 'var(--success)' : 'var(--warn)'} />
          <Metric label="Overall Sharpe" value={data.overall_sharpe?.toFixed(3) ?? '—'} />
          <Metric label="Overall Win Rate" value={data.overall_win_rate != null ? (data.overall_win_rate * 100).toFixed(1) + '%' : '—'} />
          <Metric label="IS→OOS Stability" value={data.oos_avg_sharpe != null && data.sharpe_degradation != null
            ? (data.oos_avg_sharpe > 0 ? 'Stable' : 'Degraded')
            : '—'}
            color={data.oos_avg_sharpe != null && data.oos_avg_sharpe > 0 ? 'var(--success)' : 'var(--danger)'} />
        </div>
      </div>

      {/* Per-Window Table */}
      <div className="card">
        <h2>Walk-Forward Windows</h2>
        <div style={{ overflowX: 'auto' }}>
          <table>
            <thead>
              <tr>
                <th>Window</th>
                <th>Train Start</th>
                <th>Test Start</th>
                <th>Test End</th>
                <th>IS Sharpe</th>
                <th>OOS Sharpe</th>
                <th>OOS Win Rate</th>
                <th>OOS Trades</th>
                <th>Passed</th>
              </tr>
            </thead>
            <tbody>
              {data.windows.map((w, i) => (
                <tr key={i} style={w.passed_compliance === false ? { opacity: 0.5 } : undefined}>
                  <td className="font-medium text-white">{w.window ?? i + 1}</td>
                  <td className="text-muted text-xs">{w.train_start ? new Date(w.train_start).toLocaleDateString() : '—'}</td>
                  <td className="text-muted text-xs">{w.test_start ? new Date(w.test_start).toLocaleDateString() : '—'}</td>
                  <td className="text-muted text-xs">{w.test_end ? new Date(w.test_end).toLocaleDateString() : '—'}</td>
                  <td style={{ color: (w.in_sample_sharpe ?? 0) >= 0 ? 'var(--success)' : 'var(--danger)' }}>
                    {w.in_sample_sharpe?.toFixed(3) ?? '—'}
                  </td>
                  <td style={{ color: (w.out_sample_sharpe ?? 0) >= 0 ? 'var(--success)' : 'var(--danger)' }}>
                    {w.out_sample_sharpe?.toFixed(3) ?? '—'}
                  </td>
                  <td>{w.oos_win_rate != null ? (w.oos_win_rate * 100).toFixed(1) + '%' : '—'}</td>
                  <td>{w.oos_trades ?? '—'}</td>
                  <td>
                    <span className={`badge ${w.passed_compliance ? 'badge-ok' : 'badge-err'}`}>
                      {w.passed_compliance ? 'PASS' : 'FAIL'}
                    </span>
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
    <div className="metric-card">
      <div className="metric-label">{label}</div>
      <div className="metric-value" style={color ? { color } : undefined}>{value}</div>
    </div>
  )
}
