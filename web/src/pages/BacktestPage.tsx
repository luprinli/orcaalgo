import { useState, useMemo, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useSearchParams } from 'react-router-dom'
import toast from 'react-hot-toast'
import { backtests } from '../api/client'
import { useMatrixStore } from '../stores/matrixStore'
import { useMatrixStream } from '../hooks/useMatrixStream'
import MatrixProgressBar from '../components/backtest/MatrixProgressBar'
import CancelButton from '../components/backtest/CancelButton'
import MatrixResultsPanel from '../components/backtest/MatrixResultsPanel'
import type { MatrixResultsResponse, ComboResult } from '../types/api'

type SortField = 'sharpe' | 'sortino' | 'max_dd' | 'return' | 'win_rate' | 'profit_factor' | 'trades'

const ALL_STRATEGIES = [
  'intraday_mr', 'opening_range_breakout', 'trend_following', 'grid_trading', 'session_scalp',
  'ma_crossover', 'rsi2_reversion', 'donchian_breakout', 'keltner_macd', 'ichimoku_cloud',
  'pairs_trading', 'volatility_harvesting',
]

const ALL_TIMEFRAMES = ['1d', '1h', '5m']

const STRATEGY_PARAMS: Record<string, Record<string, { min: number; max: number; step: number; default: number }>> = {
  trend_following: {
    fast_period: { min: 10, max: 50, step: 5, default: 20 },
    slow_period: { min: 50, max: 200, step: 10, default: 100 },
    atr_period: { min: 7, max: 30, step: 1, default: 14 },
    atr_multiplier: { min: 1, max: 5, step: 0.5, default: 2 },
  },
  intraday_mr: {
    entry_z: { min: -3, max: 3, step: 0.5, default: 2 },
    exit_z: { min: 0.1, max: 1.5, step: 0.1, default: 0.5 },
    lookback: { min: 10, max: 50, step: 5, default: 20 },
  },
  opening_range_breakout: {
    range_minutes: { min: 5, max: 60, step: 5, default: 15 },
    entry_buffer_pct: { min: 0.1, max: 2, step: 0.1, default: 0.3 },
  },
}

const OBJECTIVES = ['sharpe', 'sortino', 'calmar', 'profit_factor', 'sharpe / drawdown', 'return / drawdown']

const GATE_PROFILES = ['none', 'standard', 'strict', 'propfirm']
const DATA_SOURCES = ['synthetic', 'stooq']

export default function BacktestPage() {
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()
  const preselectedStrategy = searchParams.get('strategy')
  const [mode, setMode] = useState<'backtest' | 'optimize'>('backtest')
  const [matrixMode, setMatrixMode] = useState<'matrix' | 'single'>('matrix')
  const [strategies, setStrategies] = useState<string[]>(() => {
    if (preselectedStrategy && ALL_STRATEGIES.includes(preselectedStrategy)) {
      return [preselectedStrategy]
    }
    return ['ma_crossover']
  })
  const [symbols, setSymbols] = useState('SPX500,NAS100')
  const [start, setStart] = useState('2024-01-01')
  const [end, setEnd] = useState('2024-06-30')
  const [capital, setCapital] = useState('100000')
  const [dataSource, setDataSource] = useState('synthetic')
  const [gateProfile, setGateProfile] = useState('none')
  const [timeframes, setTimeframes] = useState<string[]>(['1d'])
  const [result, setResult] = useState<Record<string, unknown> | null>(null)
  const [loading, setLoading] = useState(false)
  const [matrixBatchId, setMatrixBatchId] = useState<string | null>(null)

  const [optStrategy, setOptStrategy] = useState('trend_following')
  const [optObjective, setOptObjective] = useState('sharpe / drawdown')

  const toggleStrat = (s: string) => { setStrategies(p => p.includes(s) ? p.filter(x => x !== s) : [...p, s]) }

  const currentParams = STRATEGY_PARAMS[optStrategy] ?? {}
  const symbolList = symbols.split(',').map(s => s.trim()).filter(Boolean)

  const matrixStatus = useMatrixStore((s) => s.status)
  const matrixTelemetry = useMatrixStore((s) => s.telemetry)
  const matrixByKey = useMatrixStore((s) => s.byKey)
  const matrixOrder = useMatrixStore((s) => s.order)
  const matrixBegin = useMatrixStore((s) => s.begin)
  const matrixReset = useMatrixStore((s) => s.reset)
  useMatrixStream(matrixBatchId)

  const [filterStrategy, setFilterStrategy] = useState('')
  const [filterSymbol, setFilterSymbol] = useState('')
  const [filterTf, setFilterTf] = useState('')
  const [sortField, setSortField] = useState<SortField>('sharpe')
  const [sortAsc, setSortAsc] = useState(false)

  const matrixResults = useMemo(() => matrixOrder.map(k => matrixByKey[k]), [matrixByKey, matrixOrder])

  const allResults = useMemo(() => {
    if (!matrixResults.length) return { results: [], seq: 0 } as unknown as MatrixResultsResponse
    const passed = matrixResults.filter(r => r.gate_passed === true).length
    const failed = matrixResults.filter(r => r.gate_passed === false).length
    return {
      summary: {
        total_combos: matrixTelemetry.total,
        passed,
        failed,
        total_trades: matrixResults.reduce((s, r) => s + (r.num_trades || 0), 0),
        best_sharpe: matrixTelemetry.bestSharpe,
        best_strategy: matrixTelemetry.bestStrategy,
        best_symbol: matrixTelemetry.bestSymbol,
        status: matrixStatus,
      },
      results: matrixResults,
      seq: 0,
    } as unknown as MatrixResultsResponse
  }, [matrixResults, matrixTelemetry, matrixStatus])

  const filterStrats = useMemo(() => {
    const set = new Set<string>()
    matrixResults.forEach(r => set.add(r.strategy_id))
    return Array.from(set).sort()
  }, [matrixResults])
  const filterSyms = useMemo(() => {
    const set = new Set<string>()
    matrixResults.forEach(r => set.add(r.symbol))
    return Array.from(set).sort()
  }, [matrixResults])
  const filterTfs = useMemo(() => {
    const set = new Set<string>()
    matrixResults.forEach(r => set.add(r.timeframe))
    return Array.from(set).sort()
  }, [matrixResults])

  const filtered = useMemo(() => {
    let list = matrixResults
    if (filterStrategy) list = list.filter(r => r.strategy_id === filterStrategy)
    if (filterSymbol) list = list.filter(r => r.symbol === filterSymbol)
    if (filterTf) list = list.filter(r => r.timeframe === filterTf)
    return list
  }, [matrixResults, filterStrategy, filterSymbol, filterTf])

  const sortIndicator = useCallback((field: SortField) => {
    if (sortField !== field) return ''
    return sortAsc ? ' ▲' : ' ▼'
  }, [sortField, sortAsc])

  const onSortToggle = useCallback((field: SortField) => {
    setSortField(prev => {
      if (prev === field) { setSortAsc(a => !a); return prev }
      setSortAsc(false)
      return field
    })
  }, [])

  const sortedMatrixResults = useMemo(() => {
    const list = [...filtered]
    const key = sortField === 'return' ? 'total_return' : sortField === 'max_dd' ? 'max_drawdown' : sortField === 'trades' ? 'num_trades' : `${sortField}_ratio` as keyof ComboResult
    const eqKey = sortField === 'return' ? 'total_return' : sortField === 'max_dd' ? 'max_drawdown' : sortField === 'trades' ? 'num_trades' : `${sortField}_ratio` as keyof ComboResult
    list.sort((a, b) => {
      const va = (a[eqKey] as number) ?? 0
      const vb = (b[eqKey] as number) ?? 0
      return sortAsc ? va - vb : vb - va
    })
    return list
  }, [filtered, sortField, sortAsc])

  const progressPct = matrixTelemetry.total > 0 ? Math.round((matrixTelemetry.completed / matrixTelemetry.total) * 100) : 0

  const run = async () => {
    setLoading(true)
    setResult(null)
    setMatrixBatchId(null)
    try {
      const body: Record<string, unknown> = {
        strategy_ids: strategies,
        symbols: symbolList,
        start_date: start,
        end_date: end,
        capital: parseFloat(capital) || 100000,
        data_source: dataSource,
        gate_profile: gateProfile,
        ...(mode === 'optimize' ? { optimize: true } : {}),
      }
      if (matrixMode === 'matrix') {
        body.mode = 'matrix'
        body.timeframes = timeframes
      }
      const data = await backtests.run(body as any) as unknown as Record<string, unknown>

      if (matrixMode === 'matrix' && data && typeof data === 'object' && 'batch_run_id' in data) {
        const batchId = data.batch_run_id as string
        const total = (data.total_combos as number) ?? 0
        matrixBegin(batchId, total)
        setMatrixBatchId(batchId)
        setResult({ status: 'running', total_combos: total })
        toast.success(`Backtest queued — ID: ${batchId}`)
      } else {
        matrixReset()
        setResult(data as unknown as Record<string, unknown>)
        toast.success(`Backtest queued — ID: ${(data as Record<string, unknown>)?.run_id || 'completed'}`)
      }
    } catch (err) {
      matrixReset()
      setResult({ error: err instanceof Error ? err.message : String(t('backtest:failed', 'Backtest failed')) })
    } finally { setLoading(false) }
  }

  const comboCount = matrixMode === 'single' ? 1 : strategies.length * symbolList.length * timeframes.length

  return <div>
    <div className="flex-between mb-4"><h1 style={{ margin: 0 }}>{t('backtest:title', 'Backtest Runner')}</h1><Link to="/backtest/history" className="btn btn-outline">{t('backtest:historyLink', 'History')}</Link></div>

    <div className="flex gap-2 mb-4">
      <button className={`btn ${mode === 'backtest' ? 'btn-primary' : 'btn-outline'}`} onClick={() => setMode('backtest')}>{t('backtest:mode_backtest', 'Backtest')}</button>
      <button className={`btn ${mode === 'optimize' ? 'btn-primary' : 'btn-outline'}`} onClick={() => setMode('optimize')}>{t('backtest:mode_optimize', 'Optimize')}</button>
    </div>

    <div className="card mb-4">
      <h2>{mode === 'backtest' ? t('backtest:configTitle', 'Backtest Configuration') : t('backtest:optimizeConfigTitle', 'Optimization Configuration')}</h2>

      {mode === 'backtest' ? (
        <>
          <div className="flex gap-3 mb-3">
            <label className="flex gap-2" style={{ alignItems: 'center', cursor: 'pointer' }}>
              <input type="radio" name="run_mode" checked={matrixMode === 'matrix'} onChange={() => setMatrixMode('matrix')} /> {t('backtest:matrix', 'Matrix')}
            </label>
            <label className="flex gap-2" style={{ alignItems: 'center', cursor: 'pointer' }}>
              <input type="radio" name="run_mode" checked={matrixMode === 'single'} onChange={() => setMatrixMode('single')} /> {t('backtest:single', 'Single')}
            </label>
          </div>
          <div className="grid-2">
            <div><label className="text-muted">{t('backtest:startDate', 'Start Date')}</label><input className="input" type="date" value={start} onChange={e => setStart(e.target.value)} /></div>
            <div><label className="text-muted">{t('backtest:endDate', 'End Date')}</label><input className="input" type="date" value={end} onChange={e => setEnd(e.target.value)} /></div>
            <div><label className="text-muted">{t('backtest:symbols', 'Symbols')}</label><input className="input" placeholder="EURUSD, BTCUSD, US30" value={symbols} onChange={e => setSymbols(e.target.value)} /></div>
            <div><label className="text-muted">{t('backtest:capital', 'Capital ($)')}</label><input className="input" type="number" value={capital} onChange={e => setCapital(e.target.value)} /></div>
          </div>
          <div className="grid-2 mt-3">
            <div>
              <label className="text-muted">{t('backtest:dataSource', 'Data Source')}</label>
              <select className="input" value={dataSource} onChange={e => setDataSource(e.target.value)}>
                {DATA_SOURCES.map(ds => <option key={ds} value={ds}>{ds}</option>)}
              </select>
            </div>
            <div>
              <label className="text-muted">{t('backtest:gateProfile', 'Gate Profile')}</label>
              <select className="input" value={gateProfile} onChange={e => setGateProfile(e.target.value)}>
                {GATE_PROFILES.map(gp => <option key={gp} value={gp}>{gp}</option>)}
              </select>
            </div>
          </div>
          {matrixMode === 'matrix' && (
            <div className="mt-3">
              <label className="text-muted">{t('backtest:timeframes', 'Timeframes')}</label>
              <div className="flex flex-wrap gap-2 mt-1">
                {ALL_TIMEFRAMES.map(tf => (
                  <label key={tf} className="flex gap-1" style={{ alignItems: 'center', cursor: 'pointer', fontSize: 13 }}>
                    <input type="checkbox" checked={timeframes.includes(tf)} onChange={() => setTimeframes(p => p.includes(tf) ? p.filter(x => x !== tf) : [...p, tf])} />
                    {tf}
                  </label>
                ))}
              </div>
            </div>
          )}
          <div className="mt-3"><label className="text-muted">{t('backtest:strategies', 'Strategies')}</label>
            <div className="flex flex-wrap gap-2 mt-1" style={{ flexDirection: 'column' }}>
              {ALL_STRATEGIES.map(s => (
                <label key={s} className="flex gap-2" style={{ alignItems: 'center', padding: '4px 0', cursor: 'pointer' }}>
                  <input type="checkbox" value={s} checked={strategies.includes(s)} onChange={() => toggleStrat(s)} />
                  <span style={{ fontSize: 13 }}>{t('backtest:strategy.' + s, s.replace(/_/g, ' '))}</span>
                </label>
              ))}
            </div>
          </div>
          <div className="text-muted mt-3">{matrixMode === 'single' ? t('backtest:willRunSingle', 'Will run 1 backtest') : t('backtest:willRunMatrix', 'Will run {{n}} backtests', { n: comboCount })}</div>
          <button className="btn btn-primary mt-2" onClick={run} disabled={loading || strategies.length === 0} style={{ justifyContent: 'center', width: '100%' }}>
            {loading ? t('backtest:running', 'Running...') : matrixMode === 'matrix' ? t('backtest:runMatrix', 'Run Matrix') : t('backtest:runBacktest', 'Run Backtest')}
          </button>
        </>
      ) : (
        <>
          <select className="input mb-3" value={optStrategy} onChange={e => setOptStrategy(e.target.value)}>
            {ALL_STRATEGIES.map(s => <option key={s} value={s}>{t('backtest:strategy.' + s, s.replace(/_/g, ' '))}</option>)}
          </select>
          <select className="input mb-3" value={optObjective} onChange={e => setOptObjective(e.target.value)}>
            {OBJECTIVES.map(o => <option key={o} value={o}>{t('backtest:objectives.' + o.replace(/[ /]/g, '_'), o.replace(/_/g, ' '))}</option>)}
          </select>
          <div className="mb-3"><span className="text-muted" style={{ fontWeight: 600 }}>{t('backtest:searchSpace', 'Search Space')}</span></div>
          {Object.keys(currentParams).length > 0 ? (
            <table className="data-table mb-3">
              <thead><tr><th>{t('backtest:table.parameter', 'Parameter')}</th><th>{t('backtest:table.min', 'Min')}</th><th>{t('backtest:table.max', 'Max')}</th><th>{t('backtest:table.step', 'Step')}</th></tr></thead>
              <tbody>
                {Object.entries(currentParams).map(([name, def]) => (
                  <tr key={name}>
                    <td>{name}</td><td>{def.min}</td><td>{def.max}</td><td>{def.step}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : <p className="text-muted">{t('backtest:noSearchSpace', 'No parameter search space defined for this strategy.')}</p>}
          <button className="btn btn-primary" onClick={run} disabled={loading} style={{ justifyContent: 'center', width: '100%' }}>
            {loading ? t('backtest:running', 'Running...') : t('backtest:runOptimization', 'Run Optimization')}
          </button>
        </>
      )}
    </div>

    {matrixBatchId && matrixStatus !== 'idle' && (
      <MatrixProgressBar />
    )}

    {matrixBatchId && matrixStatus !== 'idle' && (
      <CancelButton batchId={matrixBatchId} />
    )}

    {matrixBatchId && matrixResults.length > 0 && (
      <MatrixResultsPanel
        matrixResult={allResults}
        matrixBatchId={matrixBatchId}
        progressPct={progressPct}
        filterStrategy={filterStrategy}
        filterSymbol={filterSymbol}
        filterTf={filterTf}
        sortedMatrixResults={sortedMatrixResults}
        filterStrats={filterStrats}
        filterSyms={filterSyms}
        filterTfs={filterTfs}
        onFilterStrategyChange={setFilterStrategy}
        onFilterSymbolChange={setFilterSymbol}
        onFilterTfChange={setFilterTf}
        onClearFilters={() => { setFilterStrategy(''); setFilterSymbol(''); setFilterTf('') }}
        onSortToggle={onSortToggle}
        sortIndicator={sortIndicator}
      />
    )}

    {matrixBatchId && matrixStatus === 'running' && !matrixResults.length && <div className="card">
      <h2>{t('backtest:results', 'Matrix Backtest')}</h2>
      <p className="text-muted">Matrix running — {matrixTelemetry.completed}/{matrixTelemetry.total} combos completed. Results will appear as they finish.</p>
    </div>}

    {result && !matrixBatchId && <div className="card">
      <h2>{t('backtest:results', 'Backtest Results')}</h2>
      {result.error ? <p className="text-muted" style={{ color: 'var(--danger)' }}>{String(result.error)}</p> : <div className="metric-grid">
        <div className="metric-card"><div className="metric-label orca-metric-card__label">{t('backtest:sharpe', 'Sharpe')}</div><div className="metric-value orca-metric-card__value">{Number(result.sharpe_ratio || 0).toFixed(2)}</div></div>
        <div className="metric-card"><div className="metric-label orca-metric-card__label">{t('backtest:maxDd', 'Max DD')}</div><div className="metric-value orca-metric-card__value">{Number(result.max_drawdown || 0).toFixed(1)}%</div></div>
        <div className="metric-card"><div className="metric-label orca-metric-card__label">{t('backtest:winRate', 'Win Rate')}</div><div className="metric-value orca-metric-card__value">{Number(result.win_rate || 0).toFixed(1)}%</div></div>
        <div className="metric-card"><div className="metric-label orca-metric-card__label">{t('backtest:trades', 'Trades')}</div><div className="metric-value orca-metric-card__value">{Number(result.num_trades || 0)}</div></div>
        <div className="metric-card"><div className="metric-label orca-metric-card__label">{t('backtest:profitFactor', 'Profit Factor')}</div><div className="metric-value orca-metric-card__value">{Number(result.profit_factor || 0).toFixed(2)}</div></div>
      </div>}
    </div>}
  </div>
}
