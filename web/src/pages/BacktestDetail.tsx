import { useState, useEffect, useCallback, useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { backtests } from '../api/client'
import PromoteToLiveWizard from '../components/deploy/PromoteToLiveWizard'
import ErrorBoundary from '../components/ErrorBoundary'
import ErrorCard from '../components/ErrorCard'
import EquityCurveChart from '../charts/EquityCurveChart'
import DailyReturnsChart from '../charts/DailyReturnsChart'
import MonteCarloChart, { type MCResultData } from '../charts/MonteCarloChart'
import MonteCarloSummaryCard from '../components/backtest/MonteCarloSummaryCard'
import MonteCarloHistograms from '../components/backtest/MonteCarloHistograms'
import MonteCarloContextCard from '../components/backtest/MonteCarloContextCard'
import CalendarHeatmap from '../charts/CalendarHeatmap'
import YearlySummaryTable from '../charts/YearlySummaryTable'
import { exportTradesCSV, exportEquityCSV, exportDailyReturnsCSV } from '../lib/export'
import { showToast } from '../stores/toastStore'
import OverviewTab from '../components/backtest/OverviewTab'
import TradesTab from '../components/backtest/TradesTab'
import OptimizationTab from '../components/backtest/OptimizationTab'
import ComparisonTab from '../components/backtest/ComparisonTab'
import type { BacktestMetrics, EquityPoint, DailyReturn, TradeSummary, RegimeStat, OptimizationFootprint, LiveComparisonResponse, MonthlyReturn } from '../types/api'

export default function BacktestDetail() {
  const { id } = useParams()
  const [showWizard, setShowWizard] = useState(false)
  const [metrics, setMetrics] = useState<BacktestMetrics | null>(null)
  const [equity, setEquity] = useState<EquityPoint[]>([])
  const [dailyReturns, setDailyReturns] = useState<DailyReturn[]>([])
  const [trades, setTrades] = useState<TradeSummary[]>([])
  const [regimeStats, setRegimeStats] = useState<RegimeStat[]>([])
  const [optimization, setOptimization] = useState<OptimizationFootprint | null>(null)
  const [liveComparison, setLiveComparison] = useState<LiveComparisonResponse | null>(null)
  const [monthlyReturns, setMonthlyReturns] = useState<MonthlyReturn[]>([])
  const [filteredMonth, setFilteredMonth] = useState<{ year: number; month: number } | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<'overview' | 'trades' | 'optimization' | 'comparison'>('overview')
  const [mcResult, setMCResult] = useState<MCResultData | null>(null)

  const filteredTrades = useMemo(() => {
    if (!filteredMonth) return trades
    return trades.filter((t) => {
      const d = new Date(t.exit_time)
      return d.getFullYear() === filteredMonth.year && d.getMonth() + 1 === filteredMonth.month
    })
  }, [trades, filteredMonth])

  const fetchAll = useCallback(async () => {
    if (!id) return
    setLoading(true)
    setError(null)
    try {
      const [m, e, d, t] = await Promise.all([
        backtests.metrics(id),
        backtests.equity(id),
        backtests.dailyReturns(id),
        backtests.trades(id),
      ])
      setMetrics(m)
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      setEquity(Array.isArray(e) ? e.map((p: any) => ({
        time: (typeof p.timestamp === 'string' ? p.timestamp : p.time) ?? '',
        value: (typeof p.equity === 'number' ? p.equity : p.value) ?? 0,
        regime: (typeof p.regime_label === 'number' ? p.regime_label : p.regime) ?? 0,
      })) : [])
      setDailyReturns(Array.isArray(d) ? d : [])
      setTrades(t?.trades ?? [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load backtest')
      setMetrics(null)
    } finally {
      setLoading(false)
    }
  }, [id])

  const fetchMonthlyReturns = useCallback(async () => {
    if (!id) return
    try {
      const mr = await backtests.monthlyReturns(id)
      setMonthlyReturns(Array.isArray(mr) ? mr : [])
    } catch {
      // optional data
    }
  }, [id])

  const fetchRegimeStats = useCallback(async () => {
    if (!id) return
    try {
      const r = await backtests.regimeStats(id)
      setRegimeStats(r)
    } catch {
      // optional data
    }
  }, [id])

  const fetchOptimization = useCallback(async () => {
    if (!id) return
    try {
      const o = await backtests.optimization(id)
      setOptimization(o)
    } catch {
      // optional data
    }
  }, [id])

  const fetchLiveComparison = useCallback(async () => {
    if (!id) return
    try {
      const lc = await backtests.liveComparison(id)
      setLiveComparison(lc)
    } catch {
      // optional data
    }
  }, [id])

  useEffect(() => {
    fetchAll()
    fetchMonthlyReturns()
    fetchRegimeStats()
    fetchOptimization()
    fetchLiveComparison()
  }, [fetchAll, fetchMonthlyReturns, fetchRegimeStats, fetchOptimization, fetchLiveComparison])

  if (loading) {
    return (
      <div className="card">
        <p className="text-muted">Loading backtest data...</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="card">
        <ErrorCard message={error} onRetry={fetchAll} />
      </div>
    )
  }

  if (!metrics) {
    return (
      <div className="card">
        <p className="text-muted">Backtest not found</p>
      </div>
    )
  }

  return (
    <div>
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>Backtest Detail</h1>
        <div className="flex gap-2">
          {trades.length > 0 && (
            <button className="btn btn-outline" style={{ fontSize: 11, padding: '4px 10px' }} onClick={() => {
              exportTradesCSV(filteredTrades)
              showToast('success', `Exported ${filteredTrades.length} trades`)
            }}>
              Export Trades
            </button>
          )}
          {equity.length > 0 && (
            <button className="btn btn-outline" style={{ fontSize: 11, padding: '4px 10px' }} onClick={() => {
              exportEquityCSV(equity)
              showToast('success', `Exported equity curve`)
            }}>
              Export Equity
            </button>
          )}
          {dailyReturns.length > 0 && (
            <button className="btn btn-outline" style={{ fontSize: 11, padding: '4px 10px' }} onClick={() => {
              exportDailyReturnsCSV(dailyReturns)
              showToast('success', `Exported daily returns`)
            }}>
              Export Returns
            </button>
          )}
          <button className="btn btn-primary" onClick={() => setShowWizard(true)}>
            Promote to Live
          </button>
        </div>
      </div>

      {metrics.warnings && metrics.warnings.length > 0 && (
        <div className="card mb-4" style={{ background: 'rgba(210,153,34,.12)', border: '1px solid var(--warn)' }}>
          <h3 style={{ color: 'var(--warn)', margin: '0 0 8px' }}>Warnings</h3>
          <ul style={{ margin: 0, paddingLeft: 18, fontSize: 13, color: 'var(--text-secondary)' }}>
            {metrics.warnings.map((w, i) => <li key={i}>{w}</li>)}
          </ul>
        </div>
      )}

      <div className="metric-grid mb-4">
        {[
          { label: 'Sharpe', value: metrics.sharpe_ratio?.toFixed(2) },
          { label: 'Sortino', value: metrics.sortino_ratio?.toFixed(2) },
          { label: 'Max DD', value: metrics.max_drawdown_pct != null ? `${metrics.max_drawdown_pct.toFixed(1)}%` : '--' },
          { label: 'Win Rate', value: metrics.win_rate_pct != null ? `${metrics.win_rate_pct.toFixed(1)}%` : '--' },
          { label: 'Profit Factor', value: metrics.profit_factor?.toFixed(2) },
          { label: 'Total Return', value: metrics.total_return_pct != null ? `${metrics.total_return_pct.toFixed(1)}%` : '--' },
          { label: 'Trades', value: metrics.num_trades },
          { label: 'Volume', value: metrics.trading_volume?.toLocaleString() },
          { label: 'Calmar', value: metrics.calmar?.toFixed(2) },
          { label: 'VaR 95%', value: metrics.var_95 != null ? `${(metrics.var_95 * 100).toFixed(1)}%` : '--' },
          { label: 'CVaR 95%', value: metrics.cvar_95 != null ? `${(metrics.cvar_95 * 100).toFixed(1)}%` : '--' },
          { label: 'CAGR', value: metrics.cagr != null ? `${(metrics.cagr * 100).toFixed(1)}%` : '--' },
          { label: 'Pass Prob', value: metrics.pass_probability != null ? `${metrics.pass_probability.toFixed(0)}%` : '--' },
          { label: 'Commission', value: metrics.commission_bps != null ? `${metrics.commission_bps.toFixed(1)} bps` : '--' },
          { label: 'Total Fees', value: metrics.total_commission != null ? `$${metrics.total_commission.toFixed(2)}` : '--' },
        ].map((m) => (
          <div key={m.label} className="metric-card">
            <div className="metric-label">{m.label}</div>
            <div className="metric-value">{m.value ?? '--'}</div>
          </div>
        ))}
      </div>

      {monthlyReturns.length > 0 && (
        <div className="mb-4">
          <YearlySummaryTable data={monthlyReturns} />
        </div>
      )}

      <ErrorBoundary>
        <div className="grid-2 mb-4">
          {equity.length > 0 && (
            <EquityCurveChart data={equity} height={300} title="Equity Curve" color="#2962FF" />
          )}
          {dailyReturns.length > 0 && (
            <DailyReturnsChart data={dailyReturns} height={200} title="Daily Returns" />
          )}
        </div>
      </ErrorBoundary>

      {dailyReturns.length >= 2 && (
        <ErrorBoundary>
          <div className="mb-4">
            <MonteCarloChart
              dailyReturns={dailyReturns}
              simulations={500}
              forwardDays={252}
              height={280}
              title="Monte Carlo Simulation"
              seed={42}
              onMCResult={setMCResult}
            />
            {mcResult && (
              <div style={{ marginTop: 16, display: 'flex', flexDirection: 'column', gap: 16 }}>
                <MonteCarloContextCard
                  data={{
                    strategyId: metrics?.strategy_name,
                    barCount: dailyReturns.length,
                    dataStart: dailyReturns[0]?.date ? new Date(dailyReturns[0].date).toLocaleDateString() : undefined,
                    dataEnd: dailyReturns[dailyReturns.length - 1]?.date ? new Date(dailyReturns[dailyReturns.length - 1].date).toLocaleDateString() : undefined,
                    commissionBps: metrics?.commission_bps,
                  }}
                  stats={mcResult.stats}
                  seed={42}
                />
                <MonteCarloSummaryCard stats={mcResult.stats} />
                <MonteCarloHistograms
                  allPnlPct={mcResult.allPnlPct}
                  allMaxDDPct={mcResult.allMaxDDPct}
                  stats={mcResult.stats}
                />
              </div>
            )}
          </div>
        </ErrorBoundary>
      )}

      {monthlyReturns.length > 0 && (
        <ErrorBoundary>
          <div className="mb-4">
            <CalendarHeatmap
              data={monthlyReturns}
              onMonthClick={(year, month) => {
                setFilteredMonth((prev) =>
                  prev?.year === year && prev?.month === month ? null : { year, month }
                )
                if (filteredMonth?.year !== year || filteredMonth?.month !== month) {
                  setActiveTab('trades')
                }
              }}
            />
          </div>
        </ErrorBoundary>
      )}

      <div className="card mb-4">
        <div className="flex gap-2" style={{ borderBottom: '1px solid var(--border)', marginBottom: 12, paddingBottom: 8 }}>
          {(['overview', 'trades', 'optimization', 'comparison'] as const).map((tab) => (
            <button
              key={tab}
              className={`btn ${activeTab === tab ? 'btn-primary' : 'btn-outline'}`}
              onClick={() => setActiveTab(tab)}
            >
              {tab === 'overview' ? 'Overview' : tab === 'trades' ? `Trades (${filteredTrades.length})` : tab === 'optimization' ? 'Optimization' : 'Live vs BT'}
            </button>
          ))}
        </div>

        {activeTab === 'overview' && <OverviewTab regimeStats={regimeStats} />}
        {activeTab === 'trades' && <TradesTab trades={trades} filteredTrades={filteredTrades} filteredMonth={filteredMonth} onClearFilter={() => setFilteredMonth(null)} />}
        {activeTab === 'optimization' && <OptimizationTab optimization={optimization} />}
        {activeTab === 'comparison' && <ComparisonTab liveComparison={liveComparison} />}
      </div>

      {showWizard && (
        <PromoteToLiveWizard
          strategyName={metrics?.strategy_name || id || ''}
          backtestId={id || ''}
          sharpe={metrics?.sharpe_ratio || 0}
          maxDD={metrics?.max_drawdown_pct || 0}
          passProb={metrics?.pass_probability || 0}
          profitFactor={metrics?.profit_factor || 0}
          onClose={() => setShowWizard(false)}
          onDeployed={() => {
            setShowWizard(false)
            alert('Strategy deployed successfully!')
          }}
        />
      )}
    </div>
  )
}
