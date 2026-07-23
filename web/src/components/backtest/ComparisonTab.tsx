import EquityCurveChart from '../../charts/EquityCurveChart'
import type { LiveComparisonResponse } from '../../types/api'

interface Props {
  liveComparison: LiveComparisonResponse | null
}

export default function ComparisonTab({ liveComparison }: Props) {
  if (!liveComparison) {
    return <p className="text-muted">No live comparison data available. Start trading this strategy to see comparison.</p>
  }

  return (
    <div>
      <h2>Live vs Backtest Comparison</h2>
      <div className="metric-grid mb-3">
        <div className="metric-card">
          <div className="metric-label">Cumul. Slippage</div>
          <div className="metric-value">{liveComparison.metrics.cumulative_slippage_bps?.toFixed(1)} bps</div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Fill Rate Ratio</div>
          <div className="metric-value">{liveComparison.metrics.fill_rate_ratio?.toFixed(3)}</div>
        </div>
        <div className="metric-card">
          <div className="metric-label">Max Equity Divergence</div>
          <div className="metric-value" style={{ color: (liveComparison.metrics.max_equity_divergence_pct ?? 0) > 10 ? 'var(--danger)' : 'var(--success)' }}>
            {liveComparison.metrics.max_equity_divergence_pct?.toFixed(1)}%
          </div>
        </div>
      </div>
      {liveComparison.backtest_equity && liveComparison.live_equity && (
        <div className="grid-2">
          <EquityCurveChart data={liveComparison.backtest_equity} height={250} title="Backtest Equity" color="#2962FF" />
          <EquityCurveChart data={liveComparison.live_equity} height={250} title="Live Equity" color="#3fb950" />
        </div>
      )}
    </div>
  )
}
