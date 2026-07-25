import EquityCurveChart from '../../charts/EquityCurveChart'
import type { LiveComparisonResponse } from '../../types/api'
import MetricCard from '../../components/MetricCard'

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
        <MetricCard label="Cumul. Slippage" value={`${liveComparison.metrics.cumulative_slippage_bps?.toFixed(1)} bps`} />
        <MetricCard label="Fill Rate Ratio" value={liveComparison.metrics.fill_rate_ratio?.toFixed(3) ?? '--'} format="decimal" />
        <MetricCard label="Max Equity Divergence" value={liveComparison.metrics.max_equity_divergence_pct ?? 0} format="percent_raw" color={liveComparison.metrics.max_equity_divergence_pct != null ? 'auto' : 'default'} />
      </div>
      {liveComparison.backtest_equity && liveComparison.live_equity && (
        <div className="grid grid-cols-2 gap-4">
          <EquityCurveChart data={liveComparison.backtest_equity} height={250} title="Backtest Equity" color="#2962FF" />
          <EquityCurveChart data={liveComparison.live_equity} height={250} title="Live Equity" color="#3fb950" />
        </div>
      )}
    </div>
  )
}
