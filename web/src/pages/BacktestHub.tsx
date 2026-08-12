import { useState, useEffect, useMemo, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import toast from 'react-hot-toast'
import { backtests, strategies as strategiesApi, symbols as symbolsApi } from '../api/client'
import { useMatrixStore } from '../stores/matrixStore'
import { useMatrixStream } from '../hooks/useMatrixStream'
import { useCacheStore } from '../stores/cacheStore'
import MatrixProgressBar from '../components/backtest/MatrixProgressBar'
import CancelButton from '../components/backtest/CancelButton'
import MatrixResultsPanel from '../components/backtest/MatrixResultsPanel'
import OrchestrationRunner from '../components/backtest/OrchestrationRunner'
import type { StrategyRow } from '../components/backtest/OrchestrationRunner'
import OrchestrationProgressBar from '../components/backtest/OrchestrationProgressBar'
import OrchestrationDetail from '../components/backtest/OrchestrationDetail'
import OrchestrationHistoryTab from '../components/backtest/OrchestrationHistoryTab'
import { useOrchestrationPoll } from '../hooks/useOrchestrationPoll'
import PromoteToLiveWizard from '../components/deploy/PromoteToLiveWizard'
import ErrorCard from '../components/ErrorCard'
import ErrorBoundary from '../components/ErrorBoundary'
import ConfirmDialog from '../components/ConfirmDialog'
import OverviewTab from '../components/backtest/OverviewTab'
import TradesTab from '../components/backtest/TradesTab'
import OptimizationTab from '../components/backtest/OptimizationTab'
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
import { Popover, PopoverTrigger, PopoverContent } from '../components/ui/popover'
import { ScrollArea } from '../components/ui/scroll-area'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../components/ui/tabs'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/table'
import type {
  MatrixResultsResponse, ComboResult, BacktestHistoryEntry, BacktestMetrics,
  EquityPoint, DailyReturn, TradeSummary, RegimeStat, OptimizationFootprint,
  MonthlyReturn, Strategy,
} from '../types/api'
import { GATE_PROFILES, DATA_SOURCES, ALL_STRATEGIES as FALLBACK_STRATEGIES, type SortField } from '../data/constants'

type HubView = 'runner' | 'history' | 'detail'

const ALL_TIMEFRAMES = ['1d', '4h', '1h', '30m', '15m', '5m']

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
  const detailType = (searchParams.get('type') as 'backtest' | 'orchestration') || 'backtest'

  const setView = useCallback((v: HubView, id?: string, opts?: { type?: 'backtest' | 'orchestration' }) => {
    const params = new URLSearchParams()
    params.set('view', v)
    if (id) params.set('id', id)
    if (opts?.type) params.set('type', opts.type)
    setSearchParams(params, { replace: true })
  }, [setSearchParams])

  if (view === 'history') return <HistoryView setView={setView} t={t} />
  if (view === 'detail' && detailId) {
    if (detailType === 'orchestration') return <div><OrchestrationDetail runId={detailId} /></div>
    return <DetailView id={detailId} setView={setView} t={t} />
  }
  return <RunnerView setView={setView} t={t} searchParams={searchParams} />
}

function RunnerView({ setView, t: tFn, searchParams }: { setView: (v: HubView, id?: string, opts?: { type?: 'backtest' | 'orchestration' }) => void; t: ReturnType<typeof useTranslation>['t']; searchParams: URLSearchParams }) {
  const preselectedStrategy = new URLSearchParams(window.location.search).get('strategy')
  const cacheStore = useCacheStore()
  const [availableStrategies, setAvailableStrategies] = useState<Strategy[]>([])
  const [availableSymbols, setAvailableSymbols] = useState<string[]>([])
  const [mode, setMode] = useState<'matrix' | 'single' | 'orchestrated'>(() => {
    const urlMode = searchParams.get('mode')
    if (urlMode === 'orchestrated' || urlMode === 'orch') return 'orchestrated'
    if (urlMode === 'single') return 'single'
    if (searchParams.has('orch_strategy') || searchParams.get('batch_promote') === 'true') return 'orchestrated'
    return 'matrix'
  })
  const [strategies, setStrategies] = useState<string[]>(() => {
    if (preselectedStrategy && FALLBACK_STRATEGIES.includes(preselectedStrategy)) return [preselectedStrategy]
    return FALLBACK_STRATEGIES.slice(0, 1)
  })
  const [symbols, setSymbols] = useState('')
  const [start, setStart] = useState('2023-01-01')
  const [end, setEnd] = useState('2025-12-31')
  const [capital, setCapital] = useState('100000')
  const [dataSource, setDataSource] = useState('synthetic')
  const [gateProfile, setGateProfile] = useState('none')
  const [timeframes, setTimeframes] = useState<string[]>(['1d'])
  const [lightOptimize, setLightOptimize] = useState(true)
  const [favoriteSymbols, setFavoriteSymbols] = useState<string[]>(() => {
    try { const raw = localStorage.getItem('orca_fav_symbols'); return raw ? JSON.parse(raw) : [] } catch { return [] }
  })
  const [result, setResult] = useState<Record<string, unknown> | null>(null)
  const [loading, setLoading] = useState(false)
  const [orchRunId, setOrchRunId] = useState<string | null>(null)
  const [orchStartTime, setOrchStartTime] = useState(0)
  const orchPollState = useOrchestrationPoll(orchRunId, (id) => { setView('detail', id, { type: 'orchestration' }); setOrchRunId(null) })
  const [symbolsOpen, setSymbolsOpen] = useState(false)
  const matrixBatchId = useMatrixStore((s) => s.batchId)

  const toggleFavoriteSymbol = (sym: string) => {
    setFavoriteSymbols(prev => {
      const next = prev.includes(sym) ? prev.filter(s => s !== sym) : [...prev, sym]
      localStorage.setItem('orca_fav_symbols', JSON.stringify(next))
      return next
    })
  }

  useEffect(() => {
    cacheStore.fetchStrategies(() => strategiesApi.list().then(r => r.strategies))
      .then(strats => { if (strats.length > 0) setAvailableStrategies(strats) })
      .catch(() => {})
  }, [])

  useEffect(() => {
    cacheStore.fetchSymbols(() =>
      (symbolsApi.list() as Promise<unknown>).then((r: unknown) => {
        const data = r as { symbols?: { ticker: string }[] }
        return (data.symbols || []).map((s: { ticker: string }) => s.ticker)
      })
    )
      .then(syms => { if (syms.length > 0) setAvailableSymbols(syms) })
      .catch(() => {})
  }, [])

  const displayStrategies = useMemo(() => {
    if (availableStrategies.length > 0) return availableStrategies.map(s => ({ key: s.type, label: s.name }))
    return FALLBACK_STRATEGIES.map(s => ({ key: s, label: s.replace(/_/g, ' ') }))
  }, [availableStrategies])

  const symbolList = useMemo(() => symbols.split(',').map(s => s.trim()).filter(Boolean), [symbols])

  const sortedAvailableSymbols = useMemo(() => {
    const favs: string[] = []
    const rest: string[] = []
    for (const s of availableSymbols) {
      if (favoriteSymbols.includes(s)) favs.push(s)
      else rest.push(s)
    }
    return [...favs.sort(), ...rest.sort()]
  }, [availableSymbols, favoriteSymbols])

  useEffect(() => {
    if (displayStrategies.length > 0 && strategies.length === 0) {
      const initStrats = mode === 'matrix'
        ? displayStrategies.map(s => s.key)
        : [displayStrategies[0].key]
      setStrategies(initStrats)
    }
  }, [displayStrategies])

  useEffect(() => {
    if (mode === 'matrix') {
      if (displayStrategies.length > 0) setStrategies(displayStrategies.map(s => s.key))
      if (availableSymbols.length > 0) setSymbols(availableSymbols.join(','))
      setTimeframes([...ALL_TIMEFRAMES])
    } else {
      if (displayStrategies.length > 0) setStrategies([displayStrategies[0].key])
      if (availableSymbols.length > 0) setSymbols(availableSymbols[0])
      setTimeframes(['1d'])
    }
  }, [mode, displayStrategies, availableSymbols])

  const selectAllStrategies = () => setStrategies(displayStrategies.map(s => s.key))
  const selectAllSymbols = () => setSymbols(availableSymbols.length > 0 ? availableSymbols.join(',') : '')
  const selectAllTimeframes = () => setTimeframes([...ALL_TIMEFRAMES])

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
    const eqKey = sortField === 'return' ? 'total_return' : sortField === 'max_dd' ? 'max_drawdown' : sortField === 'trades' ? 'num_trades' : sortField === 'total_fees' ? 'total_fees' : sortField === 'slippage' ? 'avg_slippage_bps' : sortField === 'candles' ? 'candle_count' : `${sortField}_ratio` as keyof ComboResult
    list.sort((a, b) => { const va = (a[eqKey] as number) ?? 0; const vb = (b[eqKey] as number) ?? 0; return sortAsc ? va - vb : vb - va })
    return list
  }, [filtered, sortField, sortAsc])

  const progressPct = matrixTelemetry.total > 0 ? Math.round((matrixTelemetry.completed / matrixTelemetry.total) * 100) : 0

  const run = async () => {
    setLoading(true)
    setResult(null)
    try {
      const body: Record<string, unknown> = {
        strategy_ids: strategies, symbols: symbolList, start_date: start, end_date: end,
        capital: parseFloat(capital) || 100000, data_source: dataSource, gate_profile: gateProfile,
      }
      if (mode === 'matrix') { body.mode = 'matrix'; body.timeframes = timeframes; body.light_optimize = lightOptimize }
      if (mode === 'single' && lightOptimize) { body.light_optimize = true }
      const data = await backtests.run(body as any) as unknown as Record<string, unknown>
      if (data && typeof data === 'object' && 'batch_run_id' in data) {
        const batchId = data.batch_run_id as string
        const total = (data.total_combos as number) ?? 0
        matrixBegin(batchId, total)
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
  const isAllStrategies = strategies.length === displayStrategies.length && displayStrategies.length > 0
  const isAllSymbols = symbolList.length === availableSymbols.length && availableSymbols.length > 0
  const isAllTimeframes = timeframes.length === ALL_TIMEFRAMES.length

  if (mode === 'orchestrated') return <div>
    <div className="flex items-center justify-between mb-3">
      <h1 className="m-0 text-lg">{tFn('backtest:title', 'Backtest Runner')}</h1>
      <Button variant="outline" size="sm" onClick={() => setView('history')}>{tFn('backtest:historyLink', 'History')}</Button>
    </div>
    <div className="flex items-center gap-3 mb-3">
      <label className="flex items-center gap-1.5 cursor-pointer text-xs">
        <input type="radio" name="run_mode" checked={false} onChange={() => setMode('single')} /> {tFn('backtest:single', 'Single')}
      </label>
      <label className="flex items-center gap-1.5 cursor-pointer text-xs">
        <input type="radio" name="run_mode" checked={false} onChange={() => setMode('matrix')} /> {tFn('backtest:matrix', 'Matrix')}
      </label>
      <label className="flex items-center gap-1.5 cursor-pointer text-xs">
        <input type="radio" name="run_mode" checked={true} onChange={() => setMode('orchestrated')} /> Orch
      </label>
      {orchPollState.status !== 'idle' && (
        <OrchestrationProgressBar status={orchPollState.status} startTime={orchStartTime} />
      )}
    </div>
    <OrchestrationRunner onSubmit={(id) => { setOrchRunId(id); setOrchStartTime(Date.now()) }}
      initialRows={(() => {
        if (searchParams.get('batch_promote') === 'true') {
          try {
            const stored = sessionStorage.getItem('orch_batch_promote')
            if (stored) {
              const parsed = JSON.parse(stored)
              sessionStorage.removeItem('orch_batch_promote')
              if (Array.isArray(parsed) && parsed.length > 0) return parsed as StrategyRow[]
            }
          } catch { /* ignore parse errors */ }
        }
        const s = searchParams.get('orch_strategy')
        const sym = searchParams.get('orch_symbol')
        const tf = searchParams.get('orch_tf')
        if (s) return [{ strategy_id: s, symbol: sym || "SPX500", timeframe: tf || "4h" }]
        return undefined
      })()}
    />
  </div>

  return <div>
    <div className="flex items-center justify-between mb-3">
      <h1 className="m-0 text-lg">{tFn('backtest:title', 'Backtest Runner')}</h1>
      <Button variant="outline" size="sm" onClick={() => setView('history')}>{tFn('backtest:historyLink', 'History')}</Button>
    </div>

    <Card>
      <CardContent className="p-4 space-y-3">
        <div className="flex items-center gap-3">
          <label className="flex items-center gap-1.5 cursor-pointer text-xs">
            <input type="radio" name="run_mode" checked={mode === 'single'} onChange={() => setMode('single')} /> {tFn('backtest:single', 'Single')}
          </label>
          <label className="flex items-center gap-1.5 cursor-pointer text-xs">
            <input type="radio" name="run_mode" checked={mode === 'matrix'} onChange={() => setMode('matrix')} /> {tFn('backtest:matrix', 'Matrix')}
          </label>
          <label className="flex items-center gap-1.5 cursor-pointer text-xs">
            <input type="radio" name="run_mode" checked={(mode as string) === 'orchestrated'} onChange={() => setMode('orchestrated')} /> Orch
          </label>
          <div className="flex items-center gap-2 ml-2">
            <Label className="text-xs whitespace-nowrap">{tFn('backtest:startDate', 'Start')}</Label>
            <Input className="h-7 w-28 text-xs" type="date" value={start} onChange={e => setStart(e.target.value)} />
            <Label className="text-xs whitespace-nowrap">{tFn('backtest:endDate', 'End')}</Label>
            <Input className="h-7 w-28 text-xs" type="date" value={end} onChange={e => setEnd(e.target.value)} />
            <Label className="text-xs whitespace-nowrap">{tFn('backtest:capital', 'Capital')}</Label>
            <Input className="h-7 w-20 text-xs" type="number" value={capital} onChange={e => setCapital(e.target.value)} />
          </div>
        </div>

        <div className="flex gap-3">
          <div className="flex-[2_2_0%] min-w-0">
            <Label className="text-xs">{tFn('backtest:strategies', 'Strategies')}</Label>
            <Select value={isAllStrategies ? '__all__' : strategies[0] || ''} onValueChange={v => { if (v === '__all__') selectAllStrategies(); else setStrategies([v]) }}>
              <SelectTrigger className="h-8 text-xs"><SelectValue>{isAllStrategies ? `All (${strategies.length})` : (displayStrategies.find(s => s.key === strategies[0])?.label || strategies[0])}</SelectValue></SelectTrigger>
              <SelectContent>
                <SelectItem value="__all__">All ({displayStrategies.length})</SelectItem>
                {displayStrategies.map(s => <SelectItem key={s.key} value={s.key}>{s.label}</SelectItem>)}
              </SelectContent>
            </Select>
          </div>
          <div className="flex-[2_2_0%] min-w-0">
            <Label className="text-xs">{tFn('backtest:symbols', 'Symbols')}</Label>
            <Popover open={symbolsOpen} onOpenChange={setSymbolsOpen}>
              <PopoverTrigger asChild>
                <button className="flex h-8 w-full items-center justify-between rounded-md border border-input bg-transparent px-3 text-xs ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50">
                  <span className={isAllSymbols ? '' : 'text-foreground'}>
                    {isAllSymbols ? `All (${symbolList.length})` : symbolList[0] || <span className="text-muted-foreground">{'\u2014'}</span>}
                  </span>
                  <svg className="h-3.5 w-3.5 shrink-0 opacity-50" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" viewBox="0 0 24 24"><path d="m7 15 5 5 5-5"/><path d="m7 9 5-5 5 5"/></svg>
                </button>
              </PopoverTrigger>
              <PopoverContent className="w-56 p-0" align="start">
                <ScrollArea className="h-64">
                  <button
                    className="flex w-full items-center gap-2 px-3 py-1.5 text-xs hover:bg-accent cursor-pointer"
                    onClick={() => { selectAllSymbols(); setSymbolsOpen(false) }}
                  >
                    <span className="font-medium">All ({availableSymbols.length})</span>
                  </button>
                  {favoriteSymbols.length > 0 && (
                    <>
                      <div className="mx-3 my-0.5 border-t border-border" />
                      {sortedAvailableSymbols.filter(s => favoriteSymbols.includes(s)).map(s => (
                        <div key={s} className="flex items-center hover:bg-accent">
                          <button
                            className="flex-1 text-left px-3 py-1.5 text-xs cursor-pointer"
                            onClick={() => { setSymbols(s); setSymbolsOpen(false) }}
                          >★ {s}</button>
                          <button
                            className="shrink-0 px-2 py-1.5 text-xs cursor-pointer hover:text-yellow-500"
                            onClick={(e) => { e.stopPropagation(); toggleFavoriteSymbol(s) }}
                          >★</button>
                        </div>
                      ))}
                      {sortedAvailableSymbols.some(s => !favoriteSymbols.includes(s)) && (
                        <div className="mx-3 my-0.5 border-t border-border" />
                      )}
                    </>
                  )}
                  {sortedAvailableSymbols.filter(s => !favoriteSymbols.includes(s)).map(s => (
                    <div key={s} className="flex items-center hover:bg-accent">
                      <button
                        className="flex-1 text-left px-3 py-1.5 text-xs cursor-pointer"
                        onClick={() => { setSymbols(s); setSymbolsOpen(false) }}
                      >{s}</button>
                      <button
                        className="shrink-0 px-2 py-1.5 text-xs cursor-pointer text-muted-foreground hover:text-yellow-500"
                        onClick={(e) => { e.stopPropagation(); toggleFavoriteSymbol(s) }}
                      >☆</button>
                    </div>
                  ))}
                </ScrollArea>
              </PopoverContent>
            </Popover>
          </div>
          <div className="flex-[1_1_0%] min-w-0">
            <Label className="text-xs">{tFn('backtest:timeframes', 'Timeframes')}</Label>
            <Select value={isAllTimeframes ? '__all__' : timeframes[0] || ''} onValueChange={v => { if (v === '__all__') selectAllTimeframes(); else setTimeframes([v]) }}>
              <SelectTrigger className="h-8 text-xs"><SelectValue>{isAllTimeframes ? `All (${timeframes.length})` : timeframes[0]}</SelectValue></SelectTrigger>
              <SelectContent>
                <SelectItem value="__all__">All ({ALL_TIMEFRAMES.length})</SelectItem>
                {ALL_TIMEFRAMES.map(tf => <SelectItem key={tf} value={tf}>{tf}</SelectItem>)}
              </SelectContent>
            </Select>
          </div>
          <div className="flex-[1.5_1.5_0%] min-w-0">
            <Label className="text-xs">{tFn('backtest:dataSource', 'Data')}</Label>
            <Select value={dataSource} onValueChange={setDataSource}>
              <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
              <SelectContent>{DATA_SOURCES.map(ds => <SelectItem key={ds} value={ds}>{ds}</SelectItem>)}</SelectContent>
            </Select>
          </div>
          <div className="flex-[1.5_1.5_0%] min-w-0">
            <Label className="text-xs">{tFn('backtest:gateProfile', 'Gate')}</Label>
            <Select value={gateProfile} onValueChange={setGateProfile}>
              <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
              <SelectContent>{GATE_PROFILES.map(gp => <SelectItem key={gp} value={gp}>{gp}</SelectItem>)}</SelectContent>
            </Select>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <label className="flex items-center gap-1.5 cursor-pointer text-xs">
            <input type="checkbox" checked={lightOptimize} onChange={() => setLightOptimize(p => !p)} />
            <span>{tFn('backtest:lightOptimize', 'Auto-Optimize')}</span>
          </label>
          <span className="text-xs text-muted-foreground ml-auto whitespace-nowrap">
            {mode === 'single' ? '1 backtest' : `${comboCount} combos: ${strategies.length}s \u00d7 ${symbolList.length}sym \u00d7 ${timeframes.length}tf`}
          </span>
          <Button size="sm" className="h-8 text-xs px-3" onClick={run} disabled={loading || strategies.length === 0 || symbolList.length === 0}>
            {loading ? tFn('backtest:running', 'Running...') : mode === 'matrix' ? tFn('backtest:runMatrix', 'Run Matrix') : tFn('backtest:runBacktest', 'Run')}
          </Button>
        </div>
      </CardContent>
    </Card>

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
        onViewDetail={(runId) => setView('detail', runId)}
        onPromoteToOrch={(sid, sym, tf) => {
          sessionStorage.setItem('orch_batch_promote', JSON.stringify([{ strategy_id: sid, symbol: sym, timeframe: tf }]))
          setMode('orchestrated')
          window.scrollTo({ top: 0, behavior: 'smooth' })
        }}
        onBatchPromoteToOrch={(combos) => {
          sessionStorage.setItem('orch_batch_promote', JSON.stringify(combos))
          setMode('orchestrated')
          window.scrollTo({ top: 0, behavior: 'smooth' })
        }}
      />
    )}

    {matrixBatchId && matrixStatus === 'running' && !matrixResults.length && <Card>
      <CardContent className="p-3"><p className="text-xs text-muted-foreground">Matrix running \u2014 {matrixTelemetry.completed}/{matrixTelemetry.total} combos completed. Results will appear as they finish.</p></CardContent>
    </Card>}

    {result && !matrixBatchId && (<Card>
      <CardContent className="p-4">
        {result.error ? <p className="text-sm text-destructive">{String(result.error)}</p> : <div className="grid grid-cols-5 gap-3 mb-3">
          {[
            { label: 'Sharpe', value: Number(result.sharpe_ratio || 0).toFixed(2) },
            { label: 'Max DD', value: `${Number(result.max_drawdown || 0).toFixed(1)}%` },
            { label: 'Win Rate', value: `${Number(result.win_rate || 0).toFixed(1)}%` },
            { label: 'Trades', value: Number(result.num_trades || 0) },
            { label: 'Profit Factor', value: Number(result.profit_factor || 0).toFixed(2) },
          ].map(m => (
            <div key={m.label} className="text-center"><p className="text-xs text-muted-foreground">{m.label}</p><p className="text-sm font-bold">{m.value}</p></div>
          ))}
        </div>}
        {((result as any).run_id) && <Button variant="outline" size="sm" className="w-full text-xs" onClick={() => setView('detail', String((result as any).run_id))}>View Full Report</Button>}
      </CardContent>
    </Card>)}
  </div>
}

function HistoryView({ setView, t: tFn }: { setView: (v: HubView, id?: string, opts?: { type?: 'backtest' | 'orchestration' }) => void; t: ReturnType<typeof useTranslation>['t'] }) {
  const [list, setList] = useState<EntryWithMetrics[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [limit] = useState(50)
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)
  const [compareMode, setCompareMode] = useState(false)
  const [historySubTab, setHistorySubTab] = useState('backtests')
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

    <Tabs value={historySubTab} onValueChange={setHistorySubTab} className="mb-4">
      <TabsList className="h-8">
        <TabsTrigger value="backtests" className="text-xs h-7 data-[state=active]:bg-card">Backtests</TabsTrigger>
        <TabsTrigger value="orchestration" className="text-xs h-7 data-[state=active]:bg-card">Orchestration</TabsTrigger>
      </TabsList>
    </Tabs>

    {historySubTab === 'orchestration' ? (
      <OrchestrationHistoryTab onSelectRun={(id) => setView('detail', id, { type: 'orchestration' })} />
    ) : null}

    <div style={{ display: historySubTab === 'backtests' ? undefined : 'none' }}>

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
    </div>

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

function DetailView({ id, setView, t: tFn }: { id: string; setView: (v: HubView, id?: string, opts?: { type?: 'backtest' | 'orchestration' }) => void; t: ReturnType<typeof useTranslation>['t'] }) {
  const [showWizard, setShowWizard] = useState(false)
  const [run, setRun] = useState<Record<string, any> | null>(null)
  const [equity, setEquity] = useState<EquityPoint[]>([])
  const [dailyReturns, setDailyReturns] = useState<DailyReturn[]>([])
  const [trades, setTrades] = useState<TradeSummary[]>([])
  const [regimeStats, setRegimeStats] = useState<RegimeStat[]>([])
  const [optimization, setOptimization] = useState<OptimizationFootprint | null>(null)
  const [monthlyReturns, setMonthlyReturns] = useState<MonthlyReturn[]>([])
  const [monthlyReturnsError, setMonthlyReturnsError] = useState(false)
  const [filteredMonth, setFilteredMonth] = useState<{ year: number; month: number } | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showMonteCarlo, setShowMonteCarlo] = useState(false)
  const [showCosts, setShowCosts] = useState(false)
  const [activeTab, setActiveTab] = useState<'overview' | 'trades' | 'optimization'>('overview')
  const [mcResult, setMCResult] = useState<MCResultData | null>(null)

  const filteredTrades = useMemo(() => {
    if (!filteredMonth) return trades
    return trades.filter(t => { const d = new Date(t.exit_time); return d.getFullYear() === filteredMonth.year && d.getMonth() + 1 === filteredMonth.month })
  }, [trades, filteredMonth])

  const maxDDDuration = useMemo(() => {
    let peak = 0; let maxDuration = 0; let currentDuration = 0
    for (const p of equity) {
      if (p.value > peak) { peak = p.value; currentDuration = 0; continue }
      currentDuration++
      if (currentDuration > maxDuration) maxDuration = currentDuration
    }
    return maxDuration > 0 ? String(maxDuration) : '--'
  }, [equity])

  const avgHoldingTime = useMemo(() => {
    if (trades.length === 0) return '--'
    const total = trades.reduce((s, t) => s + (t.hold_duration || 0), 0)
    const avg = total / trades.length
    return avg >= 60 * 24 ? `${(avg / 60 / 24).toFixed(1)}d` : avg >= 60 ? `${(avg / 60).toFixed(0)}h` : `${avg.toFixed(0)}m`
  }, [trades])

  useEffect(() => {
    let cancelled = false
    const load = async () => {
      setLoading(true); setError(null)
      try {
        const matrixByKey = useMatrixStore.getState().byKey
        const matrixCombo = matrixByKey[id]
        if (matrixCombo) {
          const base: Record<string, any> = {
            id: matrixCombo.run_id,
            strategy_ids: [matrixCombo.strategy_id],
            symbols: [matrixCombo.symbol],
            timeframes: [matrixCombo.timeframe],
            start_date: '',
            end_date: '',
            sharpe_ratio: matrixCombo.sharpe_ratio,
            sortino_ratio: matrixCombo.sortino_ratio,
            max_drawdown: matrixCombo.max_drawdown,
            total_return: matrixCombo.total_return,
            win_rate: matrixCombo.win_rate,
            profit_factor: matrixCombo.profit_factor,
            num_trades: matrixCombo.num_trades,
            avg_trade: matrixCombo.avg_trade,
            avg_win: matrixCombo.avg_win,
            avg_loss: matrixCombo.avg_loss,
            avg_mae: matrixCombo.avg_mae,
            avg_mfe: matrixCombo.avg_mfe,
            num_wins: matrixCombo.num_wins,
            num_losses: matrixCombo.num_losses,
            gate_passed: matrixCombo.gate_passed,
            engine_version: 'matrix',
          }
          if (cancelled) return
          setRun(base)
          setEquity(matrixCombo.equity_curve || [])
          setTrades(matrixCombo.trades || [])
          setDailyReturns([])
          setLoading(false)
          return
        }
        const base = await backtests.get(id) as unknown as Record<string, any>
        if (cancelled) return
        setRun(base)
        const [e, d, tr] = await Promise.all([
          backtests.equity(id).catch(() => []),
          backtests.dailyReturns(id).catch(() => []),
          backtests.trades(id).catch(() => ({ trades: [] as TradeSummary[] })),
        ])
        if (cancelled) return
        setEquity(Array.isArray(e) ? e.map((p: any) => ({
          time: (typeof p.timestamp === 'string' ? p.timestamp : p.time) ?? '',
          value: (typeof p.equity === 'number' ? p.equity : p.value) ?? 0,
          regime: (typeof p.regime_label === 'number' ? p.regime_label : p.regime) ?? 0,
        })) : [])
        setDailyReturns(Array.isArray(d) ? d : [])
        setTrades(Array.isArray(tr?.trades) ? tr.trades : [])
      } catch (err) {
        if (cancelled) return
        setRun(null)
        setError(err instanceof Error ? err.message : 'Failed to load backtest')
      } finally { if (!cancelled) setLoading(false) }
    }
    load()
    return () => { cancelled = true }
  }, [id])

  useEffect(() => {
    backtests.monthlyReturns(id).then(mr => setMonthlyReturns(Array.isArray(mr) ? mr : [])).catch(() => setMonthlyReturnsError(true))
    backtests.regimeStats(id).then(rs => setRegimeStats(Array.isArray(rs) ? rs : [])).catch(() => {})
    backtests.optimization(id).then(o => setOptimization(o)).catch(() => {})
  }, [id])

  if (loading) return <Card><CardContent className="p-8 text-center"><p className="text-sm text-muted-foreground">Loading backtest...</p></CardContent></Card>
  if (error) return <Card><CardContent className="p-6"><ErrorCard message={error} onRetry={() => { setError(null); setLoading(true); window.location.reload() }} /></CardContent></Card>
  if (!run) return <Card><CardContent className="p-8 text-center"><p className="text-sm text-muted-foreground">Backtest not found</p></CardContent></Card>

  const sharpe = run.sharpe_ratio as number ?? 0
  const sortino = run.sortino_ratio as number
  const maxDD = run.max_drawdown as number
  const maxDDPct = maxDD != null ? maxDD * 100 : (run.max_drawdown_pct as number | undefined)
  const totalReturn = run.total_return as number
  const totalReturnPct = totalReturn != null ? totalReturn * 100 : (run.total_return_pct as number | undefined)
  const winRate = run.win_rate as number
  const winRatePct = winRate != null ? (winRate > 1 ? winRate : winRate * 100) : (run.win_rate_pct as number | undefined)
  const profitFactor = (run.profit_factor ?? 0) as number
  const numTrades = (run.num_trades ?? 0) as number
  const numWins = (run.num_wins ?? 0) as number
  const numLosses = (run.num_losses ?? 0) as number
  const avgMAE = run.avg_mae as number | undefined
  const avgMFE = run.avg_mfe as number | undefined
  const avgTrade = run.avg_trade as number | undefined
  const avgWin = run.avg_win as number | undefined
  const avgLoss = run.avg_loss as number | undefined
  const gatePassed = run.gate_passed as boolean | undefined
  const strategyId = (run.strategy_id ?? run.strategy_ids?.[0] ?? '--') as string
  const symbols = (run.symbols as string[] | undefined) ?? []
  const timeframe = (run.timeframe as string) ?? (run.results_json as any)?.timeframe ?? '1d'
  const startDate = run.start_date as string | undefined
  const endDate = run.end_date as string | undefined
  const capital = (run.initial_capital ?? 100000) as number
  const status = (run.status ?? 'unknown') as string
  const warnings = (run.warnings as string[] | undefined) ?? []

  const Stat = ({ label, value, color = '' }: { label: string; value: string | number; color?: string }) => (
    <Card><CardContent className="p-2 text-center">
      <p className="text-[10px] uppercase tracking-wide text-muted-foreground">{label}</p>
      <p className={`text-sm font-bold tabular-nums leading-tight ${color}`}>{value}</p>
    </CardContent></Card>
  )

  return <div>
    <div className="flex items-center justify-between mb-3">
      <div className="flex items-center gap-3">
        <Button variant="outline" size="sm" onClick={() => setView('history')}>&larr; History</Button>
        <div>
          <h1 className="m-0 text-base font-semibold">{strategyId.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())} {symbols.length > 0 ? `on ${symbols.join(', ')}` : ''} ({timeframe})</h1>
          {startDate && endDate && <p className="text-[11px] text-muted-foreground mt-0.5">{new Date(startDate).toLocaleDateString()} → {new Date(endDate).toLocaleDateString()} · ${capital.toLocaleString()} capital · {status !== 'completed' ? <Badge variant="secondary" className="text-[10px] h-4">{status}</Badge> : null}</p>}
        </div>
      </div>
      <div className="flex gap-2">
        {trades.length > 0 && <Button variant="outline" size="sm" className="text-xs" onClick={() => { exportTradesCSV(filteredTrades); toast.success(`Exported ${filteredTrades.length} trades`) }}>Export Trades</Button>}
        {equity.length > 0 && <Button variant="outline" size="sm" className="text-xs" onClick={() => { exportEquityCSV(equity); toast.success('Exported equity') }}>Export Equity</Button>}
        <Button size="sm" className="text-xs" onClick={() => setShowWizard(true)}>Promote</Button>
      </div>
    </div>

    {warnings.length > 0 && (
      <Card className="mb-3 border-amber-500/40 bg-amber-500/[0.04]">
        <CardContent className="p-2.5"><h3 className="text-amber-500 font-semibold text-[11px] mb-1">Data Quality Warnings</h3><ul className="m-0 pl-4 text-[11px] text-muted-foreground space-y-0.5">{warnings.map((w, i) => <li key={i}>{w}</li>)}</ul></CardContent>
      </Card>
    )}

    <div className="mb-3">
      <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">Performance Metrics</h2>
      <div className="grid grid-cols-5 gap-1.5">
        <Stat label="Sharpe" value={sharpe.toFixed(2)} color={sharpe >= 1 ? 'text-green-500' : sharpe >= 0 ? 'text-foreground' : 'text-red-500'} />
        <Stat label="Sortino" value={sortino != null ? sortino.toFixed(2) : '--'} />
        <Stat label="Max Drawdown" value={maxDDPct != null ? `${maxDDPct.toFixed(1)}%` : '--'} color="text-red-500" />
        <Stat label="Win Rate" value={winRatePct != null ? `${winRatePct.toFixed(1)}%` : '--'} color={winRatePct != null && winRatePct >= 50 ? 'text-green-500' : ''} />
        <Stat label="Profit Factor" value={profitFactor.toFixed(2)} color={profitFactor >= 1.5 ? 'text-green-500' : ''} />
        <Stat label="Total Return" value={totalReturnPct != null ? `${totalReturnPct.toFixed(1)}%` : '--'} color={totalReturnPct != null && totalReturnPct > 0 ? 'text-green-500' : 'text-red-500'} />
        <Stat label="Trades (W/L)" value={`${numTrades} (${numWins}/${numLosses})`} />
        <Stat label="Avg Trade" value={avgTrade != null ? `$${avgTrade.toFixed(2)}` : '--'} color={avgTrade != null && avgTrade > 0 ? 'text-green-500' : 'text-red-500'} />
        <Stat label="Avg Win / Loss" value={avgWin != null && avgLoss != null ? `$${avgWin.toFixed(0)} / $${Math.abs(avgLoss).toFixed(0)}` : '--'} />
        <Stat label="Win/Loss Ratio" value={avgWin != null && avgLoss != null && avgLoss !== 0 ? (Math.abs(avgWin / avgLoss)).toFixed(2) : '--'} />
      </div>
    </div>

    <div className="mb-3">
      <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">Risk Profile</h2>
      <div className="grid grid-cols-5 gap-1.5">
        <Stat label="Max DD %" value={maxDDPct != null ? `${maxDDPct.toFixed(1)}%` : '--'} color="text-red-500" />
        <Stat label="DD Duration" value={maxDDDuration ? `${maxDDDuration} bars` : '--'} />
        <Stat label="Avg MAE" value={avgMAE != null ? `$${avgMAE.toFixed(2)}` : '--'} />
        <Stat label="Avg MFE" value={avgMFE != null ? `$${avgMFE.toFixed(2)}` : '--'} />
        <Stat label="Avg Hold" value={avgHoldingTime} />
        <Stat label="Gate" value={gatePassed === true ? 'PASS' : gatePassed === false ? 'FAIL' : '--'} color={gatePassed === true ? 'text-green-500' : gatePassed === false ? 'text-red-500' : ''} />
      </div>
    </div>

    {equity.length === 0 && dailyReturns.length === 0 && trades.length === 0 && (
      <Card className="mb-3 border-border bg-muted/20">
        <CardContent className="p-2.5"><p className="text-[11px] text-muted-foreground">Detailed results (equity curve, trades, daily returns) are not stored for matrix-run backtests. Run a single backtest for full analytics.</p></CardContent>
      </Card>
    )}

    {equity.length > 0 && (
      <>
        <div className="mb-3"><h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">Equity Curve & Returns</h2>
          <ErrorBoundary><div className="mb-3"><EquityCurveChart data={equity} height={300} title="Equity Curve" color="#2962FF" /></div></ErrorBoundary>
        </div>
        {dailyReturns.length > 0 && (
          <ErrorBoundary>
            <div className="mb-3"><DailyReturnsChart data={dailyReturns} height={200} title="Daily Returns Distribution" /></div>
            <div className="mb-3">
              <button className="w-full text-[11px] text-muted-foreground flex items-center gap-1 cursor-pointer border-none bg-transparent py-1" onClick={() => setShowMonteCarlo(s => !s)}>
                <span style={{ transform: showMonteCarlo ? 'rotate(90deg)' : 'rotate(0deg)', transition: 'transform .15s', display: 'inline-block' }}>&#9654;</span>
                {showMonteCarlo ? 'Hide' : 'Show'} Monte Carlo Simulation (500 trials, 252 days forward)
              </button>
              {showMonteCarlo && (
                <div className="mt-2">
                  <MonteCarloChart dailyReturns={dailyReturns} simulations={500} forwardDays={252} height={280} title="Monte Carlo Simulation" seed={42} onMCResult={setMCResult} />
                  {mcResult && (
                    <div className="flex flex-col gap-3 mt-3">
                      <MonteCarloContextCard data={{ strategyId, barCount: dailyReturns.length,
                        dataStart: dailyReturns[0]?.date ? new Date(dailyReturns[0].date).toLocaleDateString() : undefined,
                        dataEnd: dailyReturns[dailyReturns.length - 1]?.date ? new Date(dailyReturns[dailyReturns.length - 1].date).toLocaleDateString() : undefined,
                        commissionBps: run.commission_bps }} stats={mcResult.stats} seed={42} />
                      <MonteCarloSummaryCard stats={mcResult.stats} />
                      <MonteCarloHistograms allPnlPct={mcResult.allPnlPct} allMaxDDPct={mcResult.allMaxDDPct} stats={mcResult.stats} />
                    </div>
                  )}
                </div>
              )}
            </div>
          </ErrorBoundary>
        )}
      </>
    )}

    {monthlyReturns.length > 0 && (
      <div className="mb-3">
        <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">Monthly Returns</h2>
        <ErrorBoundary><div className="mb-3"><CalendarHeatmap data={monthlyReturns} onMonthClick={(year, month) => { setFilteredMonth(prev => prev?.year === year && prev?.month === month ? null : { year, month }); if (filteredMonth?.year !== year || filteredMonth?.month !== month) setActiveTab('trades') }} /></div></ErrorBoundary>
        <YearlySummaryTable data={monthlyReturns} />
      </div>
    )}

    <div className="mb-3">
      <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">Trade Analysis & Details</h2>
      <Card>
        <CardContent className="p-3">
          <Tabs value={activeTab} onValueChange={v => setActiveTab(v as 'overview' | 'trades' | 'optimization')} className="w-full">
            <TabsList className="mb-3">
              <TabsTrigger value="overview" className="text-xs">Regime Analysis</TabsTrigger>
              <TabsTrigger value="trades" className="text-xs">Trades ({filteredTrades.length})</TabsTrigger>
              <TabsTrigger value="optimization" className="text-xs">Optimization</TabsTrigger>
            </TabsList>
            <TabsContent value="overview"><OverviewTab regimeStats={regimeStats} /></TabsContent>
            <TabsContent value="trades"><TradesTab trades={trades} filteredTrades={filteredTrades} filteredMonth={filteredMonth} onClearFilter={() => setFilteredMonth(null)} /></TabsContent>
            <TabsContent value="optimization"><OptimizationTab optimization={optimization} /></TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </div>

    <button className="w-full text-[11px] text-muted-foreground flex items-center gap-1 cursor-pointer border-none bg-transparent py-1" onClick={() => setShowCosts(s => !s)}>
      <span style={{ transform: showCosts ? 'rotate(90deg)' : 'rotate(0deg)', transition: 'transform .15s', display: 'inline-block' }}>&#9654;</span>
      {showCosts ? 'Hide' : 'Show'} Costs & Run Metadata
    </button>
    {showCosts && (
      <div className="grid grid-cols-5 gap-1.5 mt-1.5">
        {[
          { label: 'Volume', value: (run.trading_volume as number)?.toLocaleString() ?? '--' },
          { label: 'Commission', value: (run.commission_bps as number) != null ? `${(run.commission_bps as number).toFixed(1)} bps` : '--' },
          { label: 'Total Fees', value: (run.total_commission as number) != null ? `$${(run.total_commission as number).toFixed(2)}` : '--' },
          { label: 'Run ID', value: id.slice(0, 8) + '...' },
          { label: 'Data Source', value: run.data_source ?? '--' },
        ].map(m => <Card key={m.label}><CardContent className="p-2 text-center"><p className="text-[10px] uppercase tracking-wide text-muted-foreground">{m.label}</p><p className="text-sm font-bold tabular-nums">{m.value}</p></CardContent></Card>)}
      </div>
    )}

    {showWizard && (
      <PromoteToLiveWizard
        strategyName={strategyId || id || ''} backtestId={id}
        sharpe={sharpe} maxDD={maxDDPct ?? 0}
        passProb={0} profitFactor={profitFactor}
        onClose={() => setShowWizard(false)}
        onDeployed={() => { setShowWizard(false); alert('Strategy deployed successfully!') }}
      />
    )}
  </div>
}
