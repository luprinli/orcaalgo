import { useEffect, useState } from 'react'
import { backtests } from '../../api/client'
import type { RegimeStat, TradeDistribution } from '../../types/api'
import MetricCard from '../MetricCard'

interface Props {
  backtestId: string
  regimeStats: RegimeStat[]
}

interface RobustnessStats {
  n_returns?: number
  sharpe?: number
  sharpe_se?: number
  sharpe_ci_low?: number
  sharpe_ci_high?: number
  deflated_sharpe_ratio?: number
  expected_max_sharpe?: number
  min_trl?: number | null
  n_trials?: number
  error?: string
}

interface BenchmarkGateStats {
  passed: boolean
  kind: string
  metrics?: {
    n_periods?: number
    beta?: number
    alpha_annualized?: number
    information_ratio?: number
    tracking_error?: number
  }
  deflated_active_sharpe?: number
  n_trials?: number
  error?: string
}

export default function OverviewTab({ backtestId, regimeStats }: Props) {
  const [dist, setDist] = useState<TradeDistribution | null>(null)
  const [robustness, setRobustness] = useState<RobustnessStats | null>(null)
  const [benchmarkGate, setBenchmarkGate] = useState<BenchmarkGateStats | null>(null)

  useEffect(() => {
    let active = true
    backtests.tradeDistribution(backtestId).then(d => {
      if (active) setDist(d)
    }).catch(() => {
      if (active) setDist(null)
    })
    backtests.robustness(backtestId).then(r => {
      if (active) setRobustness(r)
    }).catch(() => {
      if (active) setRobustness(null)
    })
    backtests.benchmarkEval(backtestId).then(b => {
      if (active) setBenchmarkGate(b)
    }).catch(() => {
      if (active) setBenchmarkGate(null)
    })
    return () => { active = false }
  }, [backtestId])

  return (
    <div>
      {benchmarkGate && !benchmarkGate.error && benchmarkGate.metrics && (
        <div className="mb-4">
          <h2>Benchmark Gate</h2>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 12 }}>
            <MetricCard label="Passed" value={benchmarkGate.passed ? 'PASS' : 'FAIL'} format="decimal" color={benchmarkGate.passed ? 'positive' : 'negative'} />
            <MetricCard label="Information Ratio" value={benchmarkGate.metrics.information_ratio ?? 0} format="decimal" />
            <MetricCard label="Alpha (ann.)" value={benchmarkGate.metrics.alpha_annualized ?? 0} format="decimal" />
            <MetricCard label="Beta" value={benchmarkGate.metrics.beta ?? 0} format="decimal" />
            <MetricCard label="Tracking Error" value={benchmarkGate.metrics.tracking_error ?? 0} format="decimal" />
            <MetricCard label="Deflated Active Sharpe" value={benchmarkGate.deflated_active_sharpe ?? 0} format="decimal" />
            <MetricCard label="Kind" value={benchmarkGate.kind} format="decimal" />
          </div>
        </div>
      )}

      {robustness && !robustness.error && robustness.sharpe !== undefined && (
        <div className="mb-4">
          <h2>Statistical Robustness</h2>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 12 }}>
            <MetricCard label="Sharpe (ann.)" value={robustness.sharpe} format="decimal" />
            <MetricCard label="Sharpe SE" value={robustness.sharpe_se ?? 0} format="decimal" />
            <MetricCard label="Sharpe 95% CI Low" value={robustness.sharpe_ci_low ?? 0} format="decimal" />
            <MetricCard label="Sharpe 95% CI High" value={robustness.sharpe_ci_high ?? 0} format="decimal" />
            <MetricCard label="Deflated Sharpe" value={robustness.deflated_sharpe_ratio ?? 0} format="decimal" color={robustness.deflated_sharpe_ratio && robustness.deflated_sharpe_ratio >= 0.95 ? 'positive' : 'negative'} />
            <MetricCard label="Min Track Record" value={robustness.min_trl ?? 0} format="number" />
            <MetricCard label="Observations" value={robustness.n_returns ?? 0} format="number" />
          </div>
        </div>
      )}

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
