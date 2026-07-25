import { useState, useEffect, useMemo, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import toast from 'react-hot-toast'
import { backtests } from '../api/client'
import { submitOptimizationRun, getOptimizationStatus, getOptimizationResults, type OptimizationResult } from '../api/optimize'
import { useMatrixStore } from '../stores/matrixStore'
import { useMatrixStream } from '../hooks/useMatrixStream'
import MatrixProgressBar from '../components/backtest/MatrixProgressBar'
import CancelButton from '../components/backtest/CancelButton'
import MatrixResultsPanel from '../components/backtest/MatrixResultsPanel'
import PromoteToLiveWizard from '../components/deploy/PromoteToLiveWizard'
import ErrorCard from '../components/ErrorCard'
import ErrorBoundary from '../components/ErrorBoundary'
import ConfirmDialog from '../components/ConfirmDialog'
import MetricCard from '../components/MetricCard'
import OverviewTab from '../components/backtest/OverviewTab'
import TradesTab from '../components/backtest/TradesTab'
import OptimizationTab from '../components/backtest/OptimizationTab'
import ComparisonTab from '../components/backtest/ComparisonTab'
import EquityCurveChart from '../charts/EquityCurveChart'
import DailyReturnsChart from '../charts/DailyReturnsChart'
import MonteCarloChart, { type MCResultData } from '../charts/MonteCarloChart'
import MonteCarloSummaryCard from '../components/backtest/MonteCarloSummaryCard'
import MonteCarloHistograms from '../components/backtest/MonteCarloHistograms'
import MonteCarloContextCard from '../components/backtest/MonteCarloContextCard'
import CalendarHeatmap from '../charts/CalendarHeatmap'
import YearlySummaryTable from '../charts/YearlySummaryTable'
import { exportTradesCSV, exportEquityCSV, exportDailyReturnsCSV } from '../lib/export'
import { formatNumber, formatPctRaw } from '../lib/format'
import { TableSkeleton } from '../components/SkeletonLoader'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from '../components/ui/card'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '../components/ui/select'
import { Label } from '../components/ui/label'
import { Badge } from '../components/ui/badge'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../components/ui/tabs'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/table'
import type {
  MatrixResultsResponse, ComboResult, BacktestHistoryEntry, BacktestMetrics,
  EquityPoint, DailyReturn, TradeSummary, RegimeStat, OptimizationFootprint,
  LiveComparisonResponse, MonthlyReturn,
} from '../types/api'
import { ALL_STRATEGIES, GATE_PROFILES, DATA_SOURCES, type SortField } from '../data/constants'

type HubView = 'runner' | 'history' | 'detail'

const ALL_TIMEFRAMES = ['1d', '1h', '5m']

const COMPARE_COLORS = ['#2962FF', '#3fb950', '#d29922', '#f85149', '#58a6ff', '#bc8cff']

interface EntryWithMetrics extends BacktestHistoryEntry {
  _metrics?: BacktestMetrics
  _metricsLoading?: boolean
}

export default function BacktestHub() {
  const { t } = useTranslation()
  const [searchParams, setSearchParams] = useSearchParams()
  const view = (searchParams.get('view') as HubView) || 'runner'
  const detailId = searchParams.get('id')

  const setView = useCallback((v: HubView, id?: string) => {
    const params = new URLSearchParams()
    params.set('view', v)
    if (id) params.set('id', id)
    setSearchParams(params, { replace: true })
  }, [setSearchParams])

  if (view === 'history') return <HistoryView setView={setView} t={t} />
  if (view === 'detail' && detailId) return <DetailView id={detailId} setView={setView} t={t} />
  return <RunnerView setView={setView} t={t} />
}

function RunnerView({ setView, t: tFn }: { setView: (v: HubView, id?: string) => void; t: ReturnType<typeof useTranslation>['t'] }) {
  const preselectedStrategy = new URLSearchParams(window.location.search).get('strategy')
  const runMode = new URLSearchParams(window.location.search).get('view') === 'optimize' ? 'optimize' as const : null
  const [mode, setMode] = useState<'matrix' | 'single' | 'optimize'>(runMode || 'matrix')
  const [strategies, setStrategies] = useState<string[]>(() => {
    if (preselectedStrategy && ALL_STRATEGIES.includes(preselectedStrategy)) return [preselectedStrategy]
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
  const [optObjective, setOptObjective] = useState('sharpe')
  const [optTrainYears, setOptTrainYears] = useState(2)
  const [optTestYears, setOptTestYears] = useState(1)
  const [optStepMonths, setOptStepMonths] = useState(3)
  const [optMaxCombos, setOptMaxCombos] = useState(100)
  const [optRunId, setOptRunId] = useState<string | null>(null)
  const [optStatus, setOptStatus] = useState<{ status: string; progress?: number; elapsed_seconds?: number } | null>(null)
  const [optResult, setOptResult] = useState<OptimizationResult | null>(null)
  const [optInterval, setOptInterval] = useState<ReturnType<typeof setInterval> | null>(null)

  const toggleStrat = (s: string) => { setStrategies(p => p.includes(s) ? p.filter(x => x !== s) : [...p, s]) }
  const symbolList = symbols.split(',').map(s => s.trim()).filter(Boolean)

  const matrixStatus = useMatrixStore((s) => s.status)
  const matrixTelemetry = useMatrixStore((s) => s.telemetry)
  const matrixByKey = useMatrixStore((s) => s.byKey)
  const matrixOrder = useMatrixStore((s) => s.order)
  const matrixBegin = useMatrixStore((s) => s.begin)
  const matrixReset = useMatrixStore((s) => s.reset)
  useMatrixStream(matrixBatchId)

  useEffect(() => {
    if (!optRunId) return
    const interval = setInterval(async () => {
      try {
        const s = await getOptimizationStatus(optRunId)
        setOptStatus(s)
        if (s.status === 'completed') {
          const r = await getOptimizationResults(optRunId)
          setOptResult(r)
          clearInterval(interval)
          setLoading(false)
        } else if (s.status === 'failed') {
          clearInterval(interval)
          setLoading(false)
        }
      } catch { /* ignore poll errors */ }
    }, 2000)
    return () => clearInterval(interval)
  }, [optRunId])

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
        total_combos: matrixTelemetry.total, passed, failed,
        total_trades: matrixResults.reduce((s, r) => s + (r.num_trades || 0), 0),
        best_sharpe: matrixTelemetry.bestSharpe, best_strategy: matrixTelemetry.bestStrategy,
        best_symbol: matrixTelemetry.bestSymbol, status: matrixStatus,
      },
      results: matrixResults, seq: 0,
    } as unknown as MatrixResultsResponse
  }, [matrixResults, matrixTelemetry, matrixStatus])

  const filterStrats = useMemo(() => { const set = new Set<string>(); matrixResults.forEach(r => set.add(r.strategy_id)); return Array.from(set).sort() }, [matrixResults])
  const filterSyms = useMemo(() => { const set = new Set<string>(); matrixResults.forEach(r => set.add(r.symbol)); return Array.from(set).sort() }, [matrixResults])
  const filterTfs = useMemo(() => { const set = new Set<string>(); matrixResults.forEach(r => set.add(r.timeframe)); return Array.from(set).sort() }, [matrixResults])

  const filtered = useMemo(() => {
    let list = matrixResults
    if (filterStrategy) list = list.filter(r => r.strategy_id === filterStrategy)
    if (filterSymbol) list = list.filter(r => r.symbol === filterSymbol)
    if (filterTf) list = list.filter(r => r.timeframe === filterTf)
    return list
  }, [matrixResults, filterStrategy, filterSymbol, filterTf])

  const sortIndicator = useCallback((field: SortField) => {
    if (sortField !== field) return ''
    return sortAsc ? ' \u25B2' : ' \u25BC'
  }, [sortField, sortAsc])

  const onSortToggle = useCallback((field: SortField) => {
    setSortField(prev => { if (prev === field) { setSortAsc(a => !a); return prev }; setSortAsc(false); return field })
  }, [])

  const sortedMatrixResults = useMemo(() => {
    const list = [...filtered]
    const eqKey = sortField === 'return' ? 'total_return' : sortField === 'max_dd' ? 'max_drawdown' : sortField === 'trades' ? 'num_trades' : `${sortField}_ratio` as keyof ComboResult
    list.sort((a, b) => { const va = (a[eqKey] as number) ?? 0; const vb = (b[eqKey] as number) ?? 0; return sortAsc ? va - vb : vb - va })
    return list
  }, [filtered, sortField, sortAsc])

  const progressPct = matrixTelemetry.total > 0 ? Math.round((matrixTelemetry.completed / matrixTelemetry.total) * 100) : 0

  const run = async () => {
    setLoading(true)
    setResult(null)
    setMatrixBatchId(null)
    if (mode === 'optimize') {
      try {
        const config = {
          strategy_id: strategies[0],
          symbols: symbolList,
          objective: optObjective,
          max_combinations: optMaxCombos,
          train_years: optTrainYears,
          test_years: optTestYears,
          step_months: optStepMonths,
          capital: parseFloat(capital) || 100000,
        }
        const res = await submitOptimizationRun(config as any)
        setOptRunId(res.run_id)
        setOptStatus({ status: 'running', progress: 0 })
        setOptResult(null)
        toast.success('Optimization queued')
      } catch (err) {
        setOptStatus({ status: 'failed' })
      } finally { setLoading(false) }
      return
    }
    try {
      const body: Record<string, unknown> = {
        strategy_ids: strategies, symbols: symbolList, start_date: start, end_date: end,
        capital: parseFloat(capital) || 100000, data_source: dataSource, gate_profile: gateProfile,
      }
      if (mode === 'matrix') { body.mode = 'matrix'; body.timeframes = timeframes }
      const data = await backtests.run(body as any) as unknown as Record<string, unknown>
      if (mode === 'matrix' && data && typeof data === 'object' && 'batch_run_id' in data) {
        const batchId = data.batch_run_id as string
        const total = (data.total_combos as number) ?? 0
        matrixBegin(batchId, total)
        setMatrixBatchId(batchId)
        setResult({ status: 'running', total_combos: total })
        toast.success(`Backtest queued \u2014 ID: ${batchId}`)
      } else {
        matrixReset()
        setResult(data as unknown as Record<string, unknown>)
        toast.success(`Backtest queued \u2014 ID: ${(data as Record<string, unknown>)?.run_id || 'completed'}`)
      }
    } catch (err) {
      matrixReset()
      setResult({ error: err instanceof Error ? err.message : String(tFn('backtest:failed', 'Backtest failed')) })
    } finally { setLoading(false) }
  }

  const comboCount = mode === 'single' ? 1 : strategies.length * symbolList.length * timeframes.length

  return <div>
    <div className="flex items-center justify-between mb-4">
      <h1 className="m-0">{tFn('backtest:title', 'Backtest Runner')}</h1>
      <Button variant="outline" onClick={() => setView('history')}>{tFn('backtest:historyLink', 'History')}</Button>
    </div>

    <Card>
      <CardHeader><CardTitle>{tFn('backtest:configTitle', 'Backtest Configuration')}</CardTitle></CardHeader>
      <CardContent className="space-y-4">
        <div className="flex gap-4">
          <label className="flex items-center gap-2 cursor-pointer text-sm">
            <input type="radio" name="run_mode" checked={mode === 'matrix'} onChange={() => setMode('matrix')} /> {tFn('backtest:matrix', 'Matrix')}
          </label>
          <label className="flex items-center gap-2 cursor-pointer text-sm">
            <input type="radio" name="run_mode" checked={mode === 'single'} onChange={() => setMode('single')} /> {tFn('backtest:single', 'Single')}
          </label>
          <label className="flex items-center gap-2 cursor-pointer text-sm">
            <input type="radio" name="run_mode" checked={mode === 'optimize'} onChange={() => setMode('optimize')} /> {tFn('backtest:optimize', 'Optimize')}
          </label>
        </div>
        <div className="grid grid-cols-2 gap-4">
          {mode !== 'optimize' && (
            <>
              <div><Label>{tFn('backtest:startDate', 'Start Date')}</Label><Input type="date" value={start} onChange={e => setStart(e.target.value)} /></div>
              <div><Label>{tFn('backtest:endDate', 'End Date')}</Label><Input type="date" value={end} onChange={e => setEnd(e.target.value)} /></div>
            </>
          )}
          <div><Label>{tFn('backtest:symbols', 'Symbols')}</Label><Input placeholder="EURUSD, BTCUSD, US30" value={symbols} onChange={e => setSymbols(e.target.value)} /></div>
          <div><Label>{tFn('backtest:capital', 'Capital ($)')}</Label><Input type="number" value={capital} onChange={e => setCapital(e.target.value)} /></div>
        </div>
        {mode !== 'optimize' && (
        <div className="grid grid-cols-2 gap-4">
          <div>
            <Label>{tFn('backtest:dataSource', 'Data Source')}</Label>
            <Select value={dataSource} onValueChange={setDataSource}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>{DATA_SOURCES.map(ds => <SelectItem key={ds} value={ds}>{ds}</SelectItem>)}</SelectContent>
            </Select>
          </div>
          <div>
            <Label>{tFn('backtest:gateProfile', 'Gate Profile')}</Label>
            <Select value={gateProfile} onValueChange={setGateProfile}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>{GATE_PROFILES.map(gp => <SelectItem key={gp} value={gp}>{gp}</SelectItem>)}</SelectContent>
            </Select>
          </div>
        </div>
        )}
        {mode === 'matrix' && (
          <div>
            <Label>{tFn('backtest:timeframes', 'Timeframes')}</Label>
            <div className="flex flex-wrap gap-3 mt-1">
              {ALL_TIMEFRAMES.map(tf => (
                <label key={tf} className="flex items-center gap-1 cursor-pointer text-sm">
                  <input type="checkbox" checked={timeframes.includes(tf)} onChange={() => setTimeframes(p => p.includes(tf) ? p.filter(x => x !== tf) : [...p, tf])} />{tf}
                </label>
              ))}
            </div>
          </div>
        )}
        {mode === 'optimize' && (
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label>Strategy</Label>
              <Select value={strategies[0]} onValueChange={(v) => setStrategies([v])}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {ALL_STRATEGIES.map(s => <SelectItem key={s} value={s}>{s.replace(/_/g, ' ')}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>Objective</Label>
              <Select value={optObjective} onValueChange={setOptObjective}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="sharpe">Sharpe Ratio</SelectItem>
                  <SelectItem value="sortino">Sortino Ratio</SelectItem>
                  <SelectItem value="profit_factor">Profit Factor</SelectItem>
                  <SelectItem value="win_rate">Win Rate</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>Max Combinations</Label>
              <Input type="number" value={optMaxCombos} onChange={e => setOptMaxCombos(Number(e.target.value))} min={1} />
            </div>
            <div>
              <Label>Train Years</Label>
              <Input type="number" value={optTrainYears} onChange={e => setOptTrainYears(Number(e.target.value))} min={1} />
            </div>
            <div>
              <Label>Test Years</Label>
              <Input type="number" value={optTestYears} onChange={e => setOptTestYears(Number(e.target.value))} min={1} />
            </div>
            <div>
              <Label>Step Months</Label>
              <Input type="number" value={optStepMonths} onChange={e => setOptStepMonths(Number(e.target.value))} min={1} />
            </div>
          </div>
        )}
        {mode !== 'optimize' && (
        <div>
          <Label>{tFn('backtest:strategies', 'Strategies')}</Label>
          <div className="flex flex-col gap-1 mt-1">
            {ALL_STRATEGIES.map(s => (
              <label key={s} className="flex items-center gap-2 py-1 cursor-pointer text-sm">
                <input type="checkbox" value={s} checked={strategies.includes(s)} onChange={() => toggleStrat(s)} />
                <span>{tFn('backtest:strategy.' + s, s.replace(/_/g, ' '))}</span>
              </label>
            ))}
          </div>
        </div>
        )}
        {mode === 'single' && strategies.length > 1 && (
          <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4" style={{ background: 'rgba(210,153,34,.08)', border: '1px solid var(--trading-warning, #d29922)', padding: 8, marginTop: 8 }}>
            <p className="text-sm" style={{ color: 'var(--trading-warning, #d29922)' }}>Only <strong>{strategies[0].replace(/_/g, ' ')}</strong> will be run. Single mode uses the first selected strategy.</p>
          </div>
        )}
        <p className="text-sm text-muted-foreground">{mode === 'optimize' ? 'Will run walk-forward optimization' : mode === 'single' ? tFn('backtest:willRunSingle', 'Will run 1 backtest') : tFn('backtest:willRunMatrix', 'Will run {{n}} backtests', { n: comboCount })}</p>
        <Button className="w-full justify-center" onClick={run} disabled={loading || (mode !== 'optimize' && strategies.length === 0)}>
          {loading ? tFn('backtest:running', 'Running...') : mode === 'optimize' ? 'Run Optimization' : mode === 'matrix' ? tFn('backtest:runMatrix', 'Run Matrix') : tFn('backtest:runBacktest', 'Run Backtest')}
        </Button>
      </CardContent>
    </Card>

    {mode !== 'optimize' && (
    <>
    {matrixBatchId && matrixStatus !== 'idle' && <MatrixProgressBar />}
    {matrixBatchId && matrixStatus !== 'idle' && <CancelButton batchId={matrixBatchId} />}

    {matrixBatchId && matrixResults.length > 0 && (
      <MatrixResultsPanel
        matrixResult={allResults} matrixBatchId={matrixBatchId} progressPct={progressPct}
        filterStrategy={filterStrategy} filterSymbol={filterSymbol} filterTf={filterTf}
        sortedMatrixResults={sortedMatrixResults} filterStrats={filterStrats} filterSyms={filterSyms} filterTfs={filterTfs}
        onFilterStrategyChange={setFilterStrategy} onFilterSymbolChange={setFilterSymbol} onFilterTfChange={setFilterTf}
        onClearFilters={() => { setFilterStrategy(''); setFilterSymbol(''); setFilterTf('') }}
        onSortToggle={onSortToggle} sortIndicator={sortIndicator}
        onViewDetail={() => setView('history')}
      />
    )}

    {matrixBatchId && matrixStatus === 'running' && !matrixResults.length && <Card>
      <CardHeader><CardTitle>{tFn('backtest:results', 'Matrix Backtest')}</CardTitle></CardHeader>
      <CardContent><p className="text-sm text-muted-foreground">Matrix running \u2014 {matrixTelemetry.completed}/{matrixTelemetry.total} combos completed. Results will appear as they finish.</p></CardContent>
    </Card>}

    {result && !matrixBatchId && <Card>
      <CardHeader><CardTitle>{tFn('backtest:results', 'Backtest Results')}</CardTitle></CardHeader>
      <CardContent>
        {result.error ? <p className="text-sm text-destructive">{String(result.error)}</p> : <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
          {[
            { tKey: 'backtest:sharpe', value: Number(result.sharpe_ratio || 0).toFixed(2) },
            { tKey: 'backtest:maxDd', value: `${Number(result.max_drawdown || 0).toFixed(1)}%` },
            { tKey: 'backtest:winRate', value: `${Number(result.win_rate || 0).toFixed(1)}%` },
            { tKey: 'backtest:trades', value: Number(result.num_trades || 0) },
            { tKey: 'backtest:profitFactor', value: Number(result.profit_factor || 0).toFixed(2) },
          ].map(m => (
            <Card key={m.tKey}><CardContent className="p-4"><p className="text-sm text-muted-foreground">{tFn(m.tKey)}</p><p className="text-xl font-bold mt-1">{m.value}</p></CardContent></Card>
          ))}
        </div>}
        {((result as any).run_id) && (
          <Button variant="outline" className="mt-3 w-full" onClick={() => setView('detail', String((result as any).run_id))}>View Full Report</Button>
        )}
      </CardContent>
    </Card>}
    </> )}

    {mode === 'optimize' && optStatus && optStatus.status === 'running' && (
      <Card className="mt-4">
        <CardHeader><CardTitle>Optimization Progress</CardTitle></CardHeader>
        <CardContent>
          <p className="text-2xl font-bold">{optStatus.progress ?? 0}%</p>
          <div className="w-full h-2 bg-muted rounded mt-2 overflow-hidden">
            <div className="h-full bg-primary rounded transition-all duration-500" style={{ width: `${optStatus.progress ?? 0}%` }} />
          </div>
          {optStatus.elapsed_seconds != null && <p className="text-xs text-muted-foreground mt-1">Elapsed: {optStatus.elapsed_seconds}s</p>}
        </CardContent>
      </Card>
    )}

    {mode === 'optimize' && optResult && (
      <Card className="mt-4">
        <CardHeader><CardTitle>Optimization Results</CardTitle></CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-3">
            <MetricCard label="Best Metric" value={optResult.best_metric?.toFixed(4) ?? '-'} color="positive" />
            <MetricCard label="Total Trials" value={optResult.trials?.length ?? 0} format="number" />
          </div>
          {optResult.best_params && Object.keys(optResult.best_params).length > 0 && (
            <div className="mt-4">
              <h3 className="text-sm font-medium mb-2">Best Parameters</h3>
              <Table>
                <TableHeader><TableRow><TableHead>Parameter</TableHead><TableHead className="text-right">Value</TableHead></TableRow></TableHeader>
                <TableBody>
                  {Object.entries(optResult.best_params).map(([k, v]) => (
                    <TableRow key={k}><TableCell>{k}</TableCell><TableCell className="text-right font-bold text-trading-success">{v.toFixed(4)}</TableCell></TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    )}
  </div>
}

function HistoryView({ setView, t: tFn }: { setView: (v: HubView, id?: string) => void; t: ReturnType<typeof useTranslation>['t'] }) {
  const [list, setList] = useState<EntryWithMetrics[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [limit] = useState(50)
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)
  const [compareMode, setCompareMode] = useState(false)
  const [selectedForCompare, setSelectedForCompare] = useState<Set<string>>(new Set())
  const [compareEquity, setCompareEquity] = useState<Record<string, EquityPoint[]>>({})
  const [compareLoading, setCompareLoading] = useState(false)

  const toggleCompareSelect = (id: string) => {
    setSelectedForCompare(prev => { const next = new Set(prev); if (next.has(id)) next.delete(id); else next.add(id); return next })
  }

  const runComparison = useCallback(async () => {
    if (selectedForCompare.size < 2) return
    setCompareLoading(true)
    const equityMap: Record<string, EquityPoint[]> = {}
    for (const id of selectedForCompare) {
      try {
        const e = await backtests.equity(id)
        equityMap[id] = Array.isArray(e) ? e.map(p => ({
          time: p.time ?? '',
          value: p.value ?? 0,
          regime: p.regime ?? 0,
        })) : []
      } catch { /* skip */ }
    }
    setCompareEquity(equityMap)
    setCompareLoading(false)
  }, [selectedForCompare])

  const clearComparison = () => {
    setCompareEquity({})
    setCompareMode(false)
    setSelectedForCompare(new Set())
  }

  const fetchList = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await backtests.list({ limit })
      const runs: EntryWithMetrics[] = (res.runs ?? []).map(r => ({ ...r, _metricsLoading: r.status === 'completed' }))
      setList(runs)
    } catch (err) {
      setError(err instanceof Error ? err.message : tFn('backtestHistory:failedToLoad', 'Failed to load history'))
    } finally { setLoading(false) }
  }, [limit])

  useEffect(() => { fetchList() }, [fetchList])

  useEffect(() => {
    for (const entry of list) {
      if (!entry._metricsLoading || entry._metrics) continue
      backtests.metrics(entry.id).then(m => {
        setList(prev => prev.map(e => e.id === entry.id ? { ...e, _metrics: m, _metricsLoading: false } : e))
      }).catch(() => {
        setList(prev => prev.map(e => e.id === entry.id ? { ...e, _metricsLoading: false } : e))
      })
    }
  }, [list])

  const handleDelete = (id: string) => { setConfirmDelete(id) }

  const confirmDeleteRun = async () => {
    if (!confirmDelete) return
    try { await backtests.delete(confirmDelete); setConfirmDelete(null); fetchList() }
    catch (err) { setError(err instanceof Error ? err.message : tFn('backtestHistory:deleteFailed', 'Delete failed')); setConfirmDelete(null) }
  }

  const handleRerun = async (id: string) => {
    try {
      const res = await backtests.rerun(id)
      setView('detail', res.run_id)
    } catch (err) {
      setError(err instanceof Error ? err.message : tFn('backtestHistory:rerunFailed', 'Rerun failed'))
    }
  }

  if (loading && list.length === 0) {
    return <Card><CardHeader><CardTitle>{tFn('backtestHistory:title', 'Backtest History')}</CardTitle></CardHeader><CardContent><TableSkeleton rows={6} cols={8} /></CardContent></Card>
  }

  return <div>
    <div className="flex items-center justify-between mb-4">
      <h1 className="m-0">{tFn('backtestHistory:title', 'Backtest History')}</h1>
      <div className="flex gap-2">
        <Button variant="outline" onClick={() => setView('runner')}>{tFn('backtest:title', 'Runner')}</Button>
        {compareMode ? (<>
          <Button variant="outline" onClick={clearComparison}>{tFn('backtestHistory:cancelCompare', 'Cancel Compare')}</Button>
          <Button onClick={runComparison} disabled={selectedForCompare.size < 2 || compareLoading}>
            {compareLoading ? tFn('backtestHistory:loading', 'Loading...') : tFn('backtestHistory:compareWithCount', 'Compare ({{n}})', { n: selectedForCompare.size })}
          </Button>
        </>) : (
          <Button variant="outline" onClick={() => setCompareMode(true)}>{tFn('backtestHistory:compare', 'Compare')}</Button>
        )}
        <Button variant="outline" onClick={fetchList}>{tFn('backtestHistory:refresh', 'Refresh')}</Button>
      </div>
    </div>

    {error && <ErrorCard message={error} onRetry={fetchList} />}

    {list.length === 0 ? (
      <Card><CardContent className="p-6"><CardDescription>{tFn('backtestHistory:noRuns', 'No backtest runs yet. Run a backtest to see results here.')}</CardDescription></CardContent></Card>
    ) : (
      <Card><CardContent className="p-6">
        <Table>
          <TableHeader>
            <TableRow>
              {compareMode && <TableHead className="w-8" />}
              <TableHead>{tFn('backtestHistory:table:id', 'ID')}</TableHead>
              <TableHead>{tFn('backtestHistory:table:type', 'Type')}</TableHead>
              <TableHead>{tFn('backtestHistory:table:strategies', 'Strategies')}</TableHead>
              <TableHead>{tFn('backtestHistory:table:symbols', 'Symbols')}</TableHead>
              <TableHead>{tFn('backtestHistory:table:sharpe', 'Sharpe')}</TableHead>
              <TableHead>{tFn('backtestHistory:table:maxDd', 'Max DD')}</TableHead>
              <TableHead>{tFn('backtestHistory:table:winRate', 'Win Rate')}</TableHead>
              <TableHead>{tFn('backtestHistory:table:trades', 'Trades')}</TableHead>
              <TableHead>{tFn('backtestHistory:table:return', 'Return')}</TableHead>
              <TableHead>{tFn('backtestHistory:table:started', 'Started')}</TableHead>
              <TableHead>{tFn('backtestHistory:table:status', 'Status')}</TableHead>
              <TableHead>{tFn('backtestHistory:table:actions', 'Actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {list.map(bt => {
              const m = bt._metrics
              const isSelected = selectedForCompare.has(bt.id)
              return (
                <TableRow key={bt.id} className="cursor-pointer"
                  style={{ background: isSelected ? 'rgba(63,185,80,.06)' : undefined }}
                  onClick={() => compareMode ? toggleCompareSelect(bt.id) : setView('detail', bt.id)}>
                  {compareMode && (
                    <TableCell onClick={e => e.stopPropagation()}>
                      <input type="checkbox" checked={isSelected} onChange={() => toggleCompareSelect(bt.id)} aria-label={tFn('backtestHistory:selectForComparison', 'Select {{id}} for comparison', { id: bt.id })} />
                    </TableCell>
                  )}
                  <TableCell className="font-mono text-xs">{bt.id?.slice(0, 12)}</TableCell>
                  <TableCell>{bt.run_type}</TableCell>
                  <TableCell>{bt.strategy_ids?.join(', ') || '\u2014'}</TableCell>
                  <TableCell>{bt.symbols?.join(', ') || '\u2014'}</TableCell>
                  <TableCell style={{ color: m ? (m.sharpe_ratio >= 1 ? 'var(--trading-success)' : m.sharpe_ratio >= 0 ? 'var(--trading-warning)' : 'var(--trading-danger)') : undefined }}>
                    {m ? formatNumber(m.sharpe_ratio, 2) : bt._metricsLoading ? '...' : bt.status === 'running' ? '\u2014' : 'N/A'}
                  </TableCell>
                  <TableCell>{m ? formatPctRaw(m.max_drawdown_pct, 1) : '\u2014'}</TableCell>
                  <TableCell>{m ? formatPctRaw(m.win_rate_pct, 1) : '\u2014'}</TableCell>
                  <TableCell>{m ? m.num_trades : '\u2014'}</TableCell>
                  <TableCell style={{ color: m ? (m.total_return_pct >= 0 ? 'var(--trading-success)' : 'var(--trading-danger)') : undefined }}>
                    {m ? formatPctRaw(m.total_return_pct, 1) : '\u2014'}
                  </TableCell>
                  <TableCell>{bt.started_at ? new Date(bt.started_at).toLocaleString() : '\u2014'}</TableCell>
                  <TableCell>
                    <Badge
                      variant={bt.status === 'completed' ? 'outline' : bt.status === 'running' ? 'outline' : 'destructive'}
                      className={bt.status === 'completed' ? 'text-trading-success border-trading-success/30 bg-trading-success/10' : bt.status === 'running' ? 'text-trading-warning border-yellow-400/30 bg-trading-warning/10' : ''}>
                      {bt.status || '\u2014'}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-1">
                      <Button variant="outline" size="sm" className="px-2 py-0.5 text-xs h-6"
                        onClick={e => { e.stopPropagation(); handleRerun(bt.id) }}>{tFn('backtestHistory:rerun', 'Rerun')}</Button>
                      <Button variant="outline" size="sm" className="px-2 py-0.5 text-xs h-6 text-destructive"
                        onClick={e => { e.stopPropagation(); handleDelete(bt.id) }}>{tFn('backtestHistory:delete', 'Delete')}</Button>
                    </div>
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      </CardContent></Card>
    )}

    {compareEquity && Object.keys(compareEquity).length >= 2 && (
      <Card className="mb-4">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>{tFn('backtestHistory:comparison', 'Comparison')}</CardTitle>
            <Button variant="outline" size="sm" className="px-2 py-0.5 text-xs h-6" onClick={clearComparison}>{tFn('backtestHistory:close', 'Close')}</Button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="mb-4">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{tFn('backtestHistory:compareMetric', 'Metric')}</TableHead>
                  {Object.keys(compareEquity).map((id, idx) => {
                    const entry = list.find(l => l.id === id)
                    return <TableHead key={id} style={{ color: COMPARE_COLORS[idx % COMPARE_COLORS.length] }}>{entry?.strategy_ids?.[0] ?? id?.slice(0, 8)}</TableHead>
                  })}
                </TableRow>
              </TableHeader>
              <TableBody>
                {( [
                  ['backtestDetail:metrics:sharpe', (m: BacktestMetrics | undefined) => m ? formatNumber(m.sharpe_ratio, 2) : '\u2014'],
                  ['backtestDetail:metrics:sortino', (m: BacktestMetrics | undefined) => m ? formatNumber(m.sortino_ratio, 2) : '\u2014'],
                  ['backtestDetail:metrics:maxDd', (m: BacktestMetrics | undefined) => m ? formatPctRaw(m.max_drawdown_pct, 1) : '\u2014'],
                  ['backtestDetail:metrics:winRate', (m: BacktestMetrics | undefined) => m ? formatPctRaw(m.win_rate_pct, 1) : '\u2014'],
                  ['backtestDetail:metrics:profitFactor', (m: BacktestMetrics | undefined) => m ? formatNumber(m.profit_factor, 2) : '\u2014'],
                  ['backtestDetail:metrics:totalReturn', (m: BacktestMetrics | undefined) => m ? formatPctRaw(m.total_return_pct, 1) : '\u2014'],
                  ['backtestDetail:metrics:trades', (m: BacktestMetrics | undefined) => m ? String(m.num_trades) : '\u2014'],
                  ['backtestDetail:metrics:cagr', (m: BacktestMetrics | undefined) => m ? formatPctRaw(m.cagr, 1) : '\u2014'],
                  ['backtestDetail:metrics:calmar', (m: BacktestMetrics | undefined) => m ? formatNumber(m.calmar, 2) : '\u2014'],
                  ['backtestDetail:metrics:var95', (m: BacktestMetrics | undefined) => m ? formatPctRaw(m.var_95, 1) : '\u2014'],
                   ['backtestDetail:metrics:passProb', (m: BacktestMetrics | undefined) => m ? `${m.pass_probability?.toFixed(0)}%` : '\u2014'],
                 ] as Array<[string, (m: BacktestMetrics | undefined) => string]> ).map(([labelKey, fmt]) => (
                  <TableRow key={labelKey}>
                    <TableCell className="font-semibold">{tFn(labelKey)}</TableCell>
                    {Object.keys(compareEquity).map(id => { const entry = list.find(l => l.id === id); return <TableCell key={id}>{fmt(entry?._metrics)}</TableCell> })}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          {Object.entries(compareEquity).length >= 2 && (() => {
            const entries = Object.entries(compareEquity)
            const corrMatrix: number[][] = Array.from({ length: entries.length }, () => Array(entries.length).fill(0))
            for (let i = 0; i < entries.length; i++) {
              for (let j = i; j < entries.length; j++) {
                const returnsA: number[] = []
                const returnsB: number[] = []
                const ptsA = entries[i][1]
                const ptsB = entries[j][1]
                for (let k = 1; k < Math.min(ptsA.length, ptsB.length); k++) {
                  if (ptsA[k-1].value > 0) returnsA.push((ptsA[k].value - ptsA[k-1].value) / ptsA[k-1].value)
                  if (ptsB[k-1].value > 0) returnsB.push((ptsB[k].value - ptsB[k-1].value) / ptsB[k-1].value)
                }
                const n = Math.min(returnsA.length, returnsB.length)
                let sumA = 0, sumB = 0, sumAB = 0, sumA2 = 0, sumB2 = 0
                for (let k = 0; k < n; k++) { sumA += returnsA[k]; sumB += returnsB[k]; sumAB += returnsA[k]*returnsB[k]; sumA2 += returnsA[k]*returnsA[k]; sumB2 += returnsB[k]*returnsB[k] }
                const denom = Math.sqrt((n*sumA2 - sumA*sumA) * (n*sumB2 - sumB*sumB))
                const corr = denom === 0 ? 0 : (n*sumAB - sumA*sumB) / denom
                corrMatrix[i][j] = corr
                corrMatrix[j][i] = corr
              }
            }
            return (
              <Card className="mb-4">
                <CardHeader><CardTitle>Strategy Correlation Matrix</CardTitle></CardHeader>
                <CardContent>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead />
                        {entries.map(([id], idx) => {
                          const entry = list.find(l => l.id === id)
                          return <TableHead key={id} style={{ color: COMPARE_COLORS[idx % COMPARE_COLORS.length] }}>{entry?.strategy_ids?.[0] ?? id?.slice(0, 8)}</TableHead>
                        })}
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {entries.map(([id], i) => {
                        const entry = list.find(l => l.id === id)
                        return (
                          <TableRow key={id}>
                            <TableCell className="font-semibold" style={{ color: COMPARE_COLORS[i % COMPARE_COLORS.length] }}>{entry?.strategy_ids?.[0] ?? id?.slice(0, 8)}</TableCell>
                            {corrMatrix[i].map((c, j) => (
                              <TableCell key={j} style={{ color: Math.abs(c) > 0.7 ? (c > 0 ? 'var(--trading-success)' : 'var(--trading-danger)') : Math.abs(c) > 0.3 ? 'var(--trading-warning)' : undefined }}>{c.toFixed(3)}</TableCell>
                            ))}
                          </TableRow>
                        )
                      })}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            )
          })()}
          {Object.entries(compareEquity).length >= 2 && (() => {
            const entries = Object.entries(compareEquity)
            const primary = entries[0]
            const primaryEntry = list.find(l => l.id === primary[0])
            const overlayData = entries.slice(1).map(([eid, points], idx) => {
              const entry = list.find(l => l.id === eid)
              return { data: points, label: entry?.strategy_ids?.[0] ?? eid?.slice(0, 8), color: COMPARE_COLORS[(idx + 1) % COMPARE_COLORS.length] }
            })
            return <EquityCurveChart data={primary[1]} height={350}
              title={tFn('backtestHistory:vsOthers', '{{name}} vs {{n}} others', { name: primaryEntry?.strategy_ids?.[0] ?? primary[0]?.slice(0, 8), n: entries.length - 1 })}
              color={COMPARE_COLORS[0]} overlays={overlayData} />
          })()}
        </CardContent>
      </Card>
    )}

    {confirmDelete && (
      <ConfirmDialog
        title={tFn('backtestHistory:deleteTitle', 'Delete Backtest')}
        message={tFn('backtestHistory:deleteConfirm', 'Delete this backtest run? This action cannot be undone.')}
        confirmLabel={tFn('backtestHistory:delete', 'Delete')} danger
        onConfirm={confirmDeleteRun} onCancel={() => setConfirmDelete(null)}
      />
    )}
  </div>
}

function DetailView({ id, setView, t: tFn }: { id: string; setView: (v: HubView, id?: string) => void; t: ReturnType<typeof useTranslation>['t'] }) {
  const [showWizard, setShowWizard] = useState(false)
  const [metrics, setMetrics] = useState<BacktestMetrics | null>(null)
  const [equity, setEquity] = useState<EquityPoint[]>([])
  const [dailyReturns, setDailyReturns] = useState<DailyReturn[]>([])
  const [trades, setTrades] = useState<TradeSummary[]>([])
  const [regimeStats, setRegimeStats] = useState<RegimeStat[]>([])
  const [optimization, setOptimization] = useState<OptimizationFootprint | null>(null)
  const [liveComparison, setLiveComparison] = useState<LiveComparisonResponse | null>(null)
  const [monthlyReturns, setMonthlyReturns] = useState<MonthlyReturn[]>([])
  const [monthlyReturnsError, setMonthlyReturnsError] = useState(false)
  const [filteredMonth, setFilteredMonth] = useState<{ year: number; month: number } | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [showCosts, setShowCosts] = useState(false)
  const [activeTab, setActiveTab] = useState<'overview' | 'trades' | 'optimization' | 'comparison'>('overview')
  const [mcResult, setMCResult] = useState<MCResultData | null>(null)

  const filteredTrades = useMemo(() => {
    if (!filteredMonth) return trades
    return trades.filter(t => { const d = new Date(t.exit_time); return d.getFullYear() === filteredMonth.year && d.getMonth() + 1 === filteredMonth.month })
  }, [trades, filteredMonth])

  const maxDDDuration = useMemo(() => {
    let peak = 0
    let maxDuration = 0
    let currentDuration = 0
    for (const p of equity) {
      if (p.value > peak) { peak = p.value; currentDuration = 0; continue }
      currentDuration++
      if (currentDuration > maxDuration) maxDuration = currentDuration
    }
    return maxDuration > 0 ? `${maxDuration} bars` : '--'
  }, [equity])

  const fetchAll = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [m, e, d, tr] = await Promise.all([
        backtests.metrics(id), backtests.equity(id), backtests.dailyReturns(id), backtests.trades(id),
      ])
      setMetrics(m)
      setEquity(Array.isArray(e) ? e.map((p: any) => ({
        time: (typeof p.timestamp === 'string' ? p.timestamp : p.time) ?? '',
        value: (typeof p.equity === 'number' ? p.equity : p.value) ?? 0,
        regime: (typeof p.regime_label === 'number' ? p.regime_label : p.regime) ?? 0,
      })) : [])
      setDailyReturns(Array.isArray(d) ? d : [])
      setTrades(tr?.trades ?? [])
    } catch (err) {
      setError(err instanceof Error ? err.message : tFn('backtestDetail:failedToLoad', 'Failed to load backtest'))
      setMetrics(null)
    } finally { setLoading(false) }
  }, [id])

  const fetchMonthlyReturns = useCallback(async () => {
    try { const mr = await backtests.monthlyReturns(id); setMonthlyReturns(Array.isArray(mr) ? mr : []); setMonthlyReturnsError(false) } catch { setMonthlyReturnsError(true) }
  }, [id])

  const fetchRegimeStats = useCallback(async () => {
    try { setRegimeStats(await backtests.regimeStats(id)) } catch { /* optional */ }
  }, [id])

  const fetchOptimization = useCallback(async () => {
    try { setOptimization(await backtests.optimization(id)) } catch { /* optional */ }
  }, [id])

  const fetchLiveComparison = useCallback(async () => {
    try { setLiveComparison(await backtests.liveComparison(id)) } catch { /* optional */ }
  }, [id])

  useEffect(() => {
    fetchAll()
    fetchMonthlyReturns()
    fetchRegimeStats()
    fetchOptimization()
    fetchLiveComparison()
  }, [fetchAll, fetchMonthlyReturns, fetchRegimeStats, fetchOptimization, fetchLiveComparison])

  if (loading) {
    return <Card><CardContent className="p-6"><p className="text-sm text-muted-foreground">{tFn('backtestDetail:loading', 'Loading backtest data...')}</p></CardContent></Card>
  }

  if (error) {
    return <Card><CardContent className="p-6"><ErrorCard message={error} onRetry={fetchAll} /></CardContent></Card>
  }

  if (!metrics) {
    return <Card><CardContent className="p-6"><p className="text-sm text-muted-foreground">{tFn('backtestDetail:notFound', 'Backtest not found')}</p></CardContent></Card>
  }

  return <div>
    <div className="flex items-center justify-between mb-4">
      <div className="flex items-center gap-3">
        <Button variant="outline" size="sm" onClick={() => setView('history')}>&larr; {tFn('backtestHistory:title', 'History')}</Button>
        <h1 className="m-0">{tFn('backtestDetail:title', 'Backtest Detail')}</h1>
      </div>
      <div className="flex gap-2">
        {trades.length > 0 && (
          <Button variant="outline" size="sm" onClick={() => { exportTradesCSV(filteredTrades); toast.success(tFn('backtestDetail:exportedTrades', 'Exported {{n}} trades', { n: filteredTrades.length })) }}>
            {tFn('backtestDetail:exportTrades', 'Export Trades')}
          </Button>
        )}
        {equity.length > 0 && (
          <Button variant="outline" size="sm" onClick={() => { exportEquityCSV(equity); toast.success(tFn('backtestDetail:exportedEquity', 'Exported equity curve')) }}>
            {tFn('backtestDetail:exportEquity', 'Export Equity')}
          </Button>
        )}
        {dailyReturns.length > 0 && (
          <Button variant="outline" size="sm" onClick={() => { exportDailyReturnsCSV(dailyReturns); toast.success(tFn('backtestDetail:exportedReturns', 'Exported daily returns')) }}>
            {tFn('backtestDetail:exportReturns', 'Export Returns')}
          </Button>
        )}
        <Button onClick={() => setShowWizard(true)}>{tFn('backtestDetail:promoteToLive', 'Promote to Live')}</Button>
      </div>
    </div>

    {metrics.warnings && metrics.warnings.length > 0 && (
      <Card className="mb-4 border-amber-500/50 bg-amber-500/10">
        <CardContent className="p-4">
          <h3 className="text-amber-400 font-semibold mb-2">{tFn('backtestDetail:warnings', 'Warnings')}</h3>
          <ul className="m-0 pl-5 text-sm text-muted-foreground space-y-1">{metrics.warnings.map((w, i) => <li key={i}>{w}</li>)}</ul>
        </CardContent>
      </Card>
    )}

    {(() => {
      const primary = [
        { tKey: 'backtestDetail:metrics:sharpe', value: metrics.sharpe_ratio?.toFixed(2) },
        { tKey: 'backtestDetail:metrics:maxDd', value: metrics.max_drawdown_pct != null ? `${metrics.max_drawdown_pct.toFixed(1)}%` : '--' },
        { tKey: 'backtestDetail:metrics:winRate', value: metrics.win_rate_pct != null ? `${metrics.win_rate_pct.toFixed(1)}%` : '--' },
        { tKey: 'backtestDetail:metrics:profitFactor', value: metrics.profit_factor?.toFixed(2) },
        { tKey: 'backtestDetail:metrics:totalReturn', value: metrics.total_return_pct != null ? `${metrics.total_return_pct.toFixed(1)}%` : '--' },
        { tKey: 'backtestDetail:metrics:trades', value: metrics.num_trades },
      ]
      const advanced = [
        { tKey: 'backtestDetail:metrics:sortino', value: metrics.sortino_ratio?.toFixed(2) },
        { tKey: 'backtestDetail:metrics:maxDdDuration', value: maxDDDuration },
        { tKey: 'backtestDetail:metrics:calmar', value: metrics.calmar?.toFixed(2) },
        { tKey: 'backtestDetail:metrics:cagr', value: metrics.cagr != null ? `${(metrics.cagr * 100).toFixed(1)}%` : '--' },
        { tKey: 'backtestDetail:metrics:var95', value: metrics.var_95 != null ? `${(metrics.var_95 * 100).toFixed(1)}%` : '--' },
        { tKey: 'backtestDetail:metrics:cvar95', value: metrics.cvar_95 != null ? `${(metrics.cvar_95 * 100).toFixed(1)}%` : '--' },
        { tKey: 'backtestDetail:metrics:passProb', value: metrics.pass_probability != null ? `${metrics.pass_probability.toFixed(0)}%` : '--' },
      ]
      const costs = [
        { tKey: 'backtestDetail:metrics:volume', value: metrics.trading_volume?.toLocaleString() },
        { tKey: 'backtestDetail:metrics:commission', value: metrics.commission_bps != null ? `${metrics.commission_bps.toFixed(1)} bps` : '--' },
        { tKey: 'backtestDetail:metrics:totalFees', value: metrics.total_commission != null ? `$${metrics.total_commission.toFixed(2)}` : '--' },
      ]
      const p = primary.map(m => <Card key={m.tKey}><CardContent className="p-3"><p className="text-xs text-muted-foreground">{tFn(m.tKey)}</p><p className="text-lg font-bold mt-0.5">{m.value ?? '--'}</p></CardContent></Card>)
      const a = advanced.map(m => <Card key={m.tKey}><CardContent className="p-3"><p className="text-xs text-muted-foreground">{tFn(m.tKey)}</p><p className="text-lg font-bold mt-0.5">{m.value ?? '--'}</p></CardContent></Card>)
      const c = costs.map(m => <Card key={m.tKey}><CardContent className="p-3"><p className="text-xs text-muted-foreground">{tFn(m.tKey)}</p><p className="text-lg font-bold mt-0.5">{m.value ?? '--'}</p></CardContent></Card>)
      return (
        <>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3 mb-3">{p}</div>
          <button className="w-full text-sm mb-3 flex items-center gap-1 cursor-pointer border-none bg-transparent" style={{ color: 'var(--muted-foreground)' }} onClick={() => setShowAdvanced(s => !s)}>
            <span style={{ transform: showAdvanced ? 'rotate(90deg)' : 'rotate(0deg)', transition: 'transform .15s' }}>&#9654;</span>
            {showAdvanced ? 'Hide' : 'Show'} Advanced Metrics ({advanced.length})
          </button>
          {showAdvanced && <div className="grid grid-cols-2 sm:grid-cols-3 gap-3 mb-3">{a}</div>}
          <button className="w-full text-sm mb-3 flex items-center gap-1 cursor-pointer border-none bg-transparent" style={{ color: 'var(--muted-foreground)' }} onClick={() => setShowCosts(s => !s)}>
            <span style={{ transform: showCosts ? 'rotate(90deg)' : 'rotate(0deg)', transition: 'transform .15s' }}>&#9654;</span>
            {showCosts ? 'Hide' : 'Show'} Cost Metrics ({costs.length})
          </button>
          {showCosts && <div className="grid grid-cols-2 sm:grid-cols-3 gap-3 mb-3">{c}</div>}
        </>
      )
    })()}

    {monthlyReturns.length > 0 && <div className="mb-4"><YearlySummaryTable data={monthlyReturns} /></div>}
    {monthlyReturnsError && (
      <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4 mb-4" style={{ background: 'rgba(210,153,34,.06)', border: '1px solid var(--trading-warning, #d29922)' }}>
        <p className="text-sm" style={{ color: 'var(--trading-warning, #d29922)' }}>Monthly returns data is unavailable. Yearly summary and calendar heatmap may be incomplete.</p>
      </div>
    )}

    <ErrorBoundary>
      <div className="grid grid-cols-2 gap-6 mb-4">
        {equity.length > 0 && <EquityCurveChart data={equity} height={300} title={tFn('backtestDetail:equityCurve', 'Equity Curve')} color="#2962FF" />}
        {dailyReturns.length > 0 && <DailyReturnsChart data={dailyReturns} height={200} title={tFn('backtestDetail:dailyReturns', 'Daily Returns')} />}
      </div>
    </ErrorBoundary>

    {dailyReturns.length >= 2 && (
      <ErrorBoundary>
        <div className="mb-4">
          <MonteCarloChart dailyReturns={dailyReturns} simulations={500} forwardDays={252} height={280}
            title={tFn('backtestDetail:monteCarlo', 'Monte Carlo Simulation')} seed={42} onMCResult={setMCResult} />
          {mcResult && (
            <div className="flex flex-col gap-4 mt-4">
              <MonteCarloContextCard data={{
                strategyId: metrics?.strategy_name, barCount: dailyReturns.length,
                dataStart: dailyReturns[0]?.date ? new Date(dailyReturns[0].date).toLocaleDateString() : undefined,
                dataEnd: dailyReturns[dailyReturns.length - 1]?.date ? new Date(dailyReturns[dailyReturns.length - 1].date).toLocaleDateString() : undefined,
                commissionBps: metrics?.commission_bps,
              }} stats={mcResult.stats} seed={42} />
              <MonteCarloSummaryCard stats={mcResult.stats} />
              <MonteCarloHistograms allPnlPct={mcResult.allPnlPct} allMaxDDPct={mcResult.allMaxDDPct} stats={mcResult.stats} />
            </div>
          )}
        </div>
      </ErrorBoundary>
    )}

    {monthlyReturns.length > 0 && (
      <ErrorBoundary>
        <div className="mb-4">
          <CalendarHeatmap data={monthlyReturns} onMonthClick={(year, month) => {
            setFilteredMonth(prev => prev?.year === year && prev?.month === month ? null : { year, month })
            if (filteredMonth?.year !== year || filteredMonth?.month !== month) setActiveTab('trades')
          }} />
        </div>
      </ErrorBoundary>
    )}

    <Card className="mb-4">
      <CardContent className="p-4">
        <Tabs value={activeTab} onValueChange={v => setActiveTab(v as 'overview' | 'trades' | 'optimization' | 'comparison')} className="w-full">
          <TabsList className="mb-4">
            <TabsTrigger value="overview">{tFn('backtestDetail:tab:overview')}</TabsTrigger>
            <TabsTrigger value="trades">{tFn('backtestDetail:tab:trades', { n: filteredTrades.length })}</TabsTrigger>
            <TabsTrigger value="optimization">{tFn('backtestDetail:tab:optimization')}</TabsTrigger>
            <TabsTrigger value="comparison">{tFn('backtestDetail:tab:liveVsBt')}</TabsTrigger>
          </TabsList>
          <TabsContent value="overview"><OverviewTab regimeStats={regimeStats} /></TabsContent>
          <TabsContent value="trades"><TradesTab trades={trades} filteredTrades={filteredTrades} filteredMonth={filteredMonth} onClearFilter={() => setFilteredMonth(null)} /></TabsContent>
          <TabsContent value="optimization"><OptimizationTab optimization={optimization} /></TabsContent>
          <TabsContent value="comparison"><ComparisonTab liveComparison={liveComparison} /></TabsContent>
        </Tabs>
      </CardContent>
    </Card>

    {showWizard && (
      <PromoteToLiveWizard
        strategyName={metrics?.strategy_name || id || ''} backtestId={id}
        sharpe={metrics?.sharpe_ratio || 0} maxDD={metrics?.max_drawdown_pct || 0}
        passProb={metrics?.pass_probability || 0} profitFactor={metrics?.profit_factor || 0}
        onClose={() => setShowWizard(false)}
        onDeployed={() => { setShowWizard(false); alert(tFn('backtestDetail:deployedSuccess', 'Strategy deployed successfully!')) }}
      />
    )}
  </div>
}
