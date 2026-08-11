"use client"

import { useState, useEffect, useCallback } from "react"
import {
  Play, RefreshCw, AlertCircle, Plus, Trash2, Zap,
  Grid3X3, Activity, DollarSign, Layers,
} from "lucide-react"
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "../ui/card"
import { Badge } from "../ui/badge"
import { Button } from "../ui/button"
import { Input } from "../ui/input"
import { Label } from "../ui/label"
import { Checkbox } from "../ui/checkbox"
import { Slider } from "../ui/slider"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "../ui/select"
import { FrictionToggle } from "../orchestration/FrictionToggle"
import { orchestrator, strategies as strategiesApi } from "../../api/client"
import OrchMatrixResultsPanel from "./OrchMatrixResultsPanel"
import { useCacheStore } from "../../stores/cacheStore"
import type { OrchestrationSubmitRequest, Strategy } from "../../types/api"

const SYMBOL_OPTIONS = ["ES", "NQ", "YM", "RTY", "JPN225", "SPY", "QQQ", "IWM", "AAPL", "MSFT", "SPX500", "ETHUSD", "BTCUSD"]
const TIMEFRAME_OPTIONS = ["1m", "5m", "15m", "30m", "1h", "4h", "1d"]

interface StrategyRow {
  strategy_id: string
  symbol: string
  timeframe: string
}

interface OrchestrationRunnerProps {
  onSubmit: (runId: string) => void
}

export default function OrchestrationRunner({ onSubmit }: OrchestrationRunnerProps) {
  const cacheStore = useCacheStore()
  const [strategyRows, setStrategyRows] = useState<StrategyRow[]>([
    { strategy_id: "", symbol: "SPX500", timeframe: "4h" },
  ])
  const [availableStrategies, setAvailableStrategies] = useState<Strategy[]>([])
  const [startDate, setStartDate] = useState("2024-01-01")
  const [endDate, setEndDate] = useState("2025-12-31")
  const [capital, setCapital] = useState("100000")
  const [rebalanceBars, setRebalanceBars] = useState("20")
  const [kellyFraction, setKellyFraction] = useState("0.25")
  const [maxPositionPct, setMaxPositionPct] = useState("2")
  const [enableCorrelationBrake, setEnableCorrelationBrake] = useState(false)
  const [correlationThreshold, setCorrelationThreshold] = useState(0.6)
  const [frictionModel, setFrictionModel] = useState<"realistic" | "idealized">("realistic")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [matrixMode, setMatrixMode] = useState(false)
  const [matrixSets, setMatrixSets] = useState<StrategyRow[][]>(() => [
    [{ strategy_id: "", symbol: "SPX500", timeframe: "4h" }],
  ])
  const [matrixResults, setMatrixResults] = useState<any[]>([])
  const [matrixTelemetry, setMatrixTelemetry] = useState<any>(null)
  const [matrixLoading, setMatrixLoading] = useState(false)

  useEffect(() => {
    cacheStore.fetchStrategies(() => strategiesApi.list().then((r: { strategies: Strategy[] }) => r.strategies ?? []))
      .then((list: Strategy[]) => setAvailableStrategies(list))
      .catch(() => {})
  }, [])

  const addRow = useCallback(() => {
    setStrategyRows((prev) => [...prev, { strategy_id: "", symbol: "SPX500", timeframe: "4h" }])
  }, [])

  const removeRow = useCallback((index: number) => {
    setStrategyRows((prev) => prev.filter((_, i) => i !== index))
  }, [])

  const updateRow = useCallback((index: number, field: keyof StrategyRow, value: string) => {
    setStrategyRows((prev) => prev.map((r, i) => (i === index ? { ...r, [field]: value } : r)))
  }, [])

  const fillRecommended = useCallback(() => {
    setStrategyRows([
      { strategy_id: "grid_trading", symbol: "SPX500", timeframe: "4h" },
      { strategy_id: "grid_trading", symbol: "JPN225", timeframe: "1h" },
      { strategy_id: "rsi2_reversion", symbol: "JPN225", timeframe: "1h" },
    ])
  }, [])

  const handleSubmit = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data: OrchestrationSubmitRequest = {
        strategies: strategyRows.filter((r) => r.strategy_id),
        start_date: startDate,
        end_date: endDate,
        initial_capital: parseFloat(capital) || 100000,
        rebalance_bars: parseInt(rebalanceBars, 10) || 20,
        kelly_fraction: parseFloat(kellyFraction) || 0.25,
        max_position_pct: parseFloat(maxPositionPct) / 100 || 0.02,
        enable_correlation_brake: enableCorrelationBrake,
        correlation_threshold: correlationThreshold,
        friction_model: frictionModel,
      }
      const result = await orchestrator.submit(data)
      onSubmit(result.run_id)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Submission failed")
    } finally {
      setLoading(false)
    }
  }, [strategyRows, startDate, endDate, capital, rebalanceBars, kellyFraction, maxPositionPct, enableCorrelationBrake, correlationThreshold, frictionModel, onSubmit])

  const handleMatrixSubmit = useCallback(async () => {
    setMatrixLoading(true)
    setError(null)
    try {
      const sets = matrixSets.map(set => ({
        strategies: set.filter(r => r.strategy_id).map(r => ({ strategy_id: r.strategy_id, symbol: r.symbol, timeframe: r.timeframe })),
      })).filter(s => s.strategies.length > 0)
      if (sets.length === 0) { setError("No valid sets"); setMatrixLoading(false); return }
      const result = await orchestrator.submitMatrix({
        sets, start_date: startDate, end_date: endDate,
        initial_capital: parseFloat(capital) || 100000,
        rebalance_bars: parseInt(rebalanceBars, 10) || 20,
        kelly_fraction: parseFloat(kellyFraction) || 0.25,
        max_position_pct: parseFloat(maxPositionPct) / 100 || 0.02,
        friction_model: frictionModel,
      })
      setMatrixTelemetry({ total: result.total_sets, completed: 0, best_sharpe: 0, best_set: -1, status: "running" })
      setMatrixResults([])
      const batchTime = Date.now()
      let pollCount = 0
      const pollInterval = setInterval(async () => {
        pollCount++
        try {
          const runs = await orchestrator.list(result.total_sets + 5, 0)
          const recent = (runs.runs ?? []).filter((r: any) =>
            r.created_at && new Date(r.created_at).getTime() > batchTime - 5000)
          const completed = recent.filter((r: any) => r.status === "completed" || r.status === "failed")
          if (completed.length > 0 || pollCount >= 60) {
            const assembled = recent.map((r: any, i: number) => ({
              set_index: i,
              strategies: (r.strategy_ids ?? []).map((sid: string, j: number) => ({
                strategy_id: sid,
                symbol: (r.symbol_tf_pairs?.[j] ?? "").split(":")[0] || "?",
                timeframe: (r.symbol_tf_pairs?.[j] ?? "").split(":")[1] || "?",
              })),
              pool_sharpe: r.pool_sharpe ?? 0,
              pool_maxdd: r.pool_maxdd ?? 0,
              pool_return_pct: r.pool_return_pct ?? 0,
              num_trades: 0,
              strategy_pnl: r.result_json?.strategy_pnl ?? {},
              status: r.status ?? "running",
              run_id: r.id,
            }))
            setMatrixResults(assembled)
            const best = assembled.filter((r: any) => r.status === "completed" && r.pool_sharpe > 0)
              .sort((a: any, b: any) => (b.pool_sharpe ?? 0) - (a.pool_sharpe ?? 0))[0]
            setMatrixTelemetry({
              total: result.total_sets, completed: completed.length,
              best_sharpe: best?.pool_sharpe ?? 0, best_set: best?.set_index ?? -1, status: "done",
            })
            clearInterval(pollInterval)
          }
        } catch { /* polling will retry */ }
      }, 3000)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Matrix submission failed")
    } finally { setMatrixLoading(false) }
  }, [matrixSets, startDate, endDate, capital, rebalanceBars, kellyFraction, maxPositionPct, frictionModel])

  const addMatrixSet = useCallback(() => {
    setMatrixSets(prev => [...prev, [{ strategy_id: "", symbol: "SPX500", timeframe: "4h" }]])
  }, [])

  const addMatrixRow = useCallback((setIdx: number) => {
    setMatrixSets(prev => prev.map((set, i) => i === setIdx ? [...set, { strategy_id: "", symbol: "JPN225", timeframe: "1h" }] : set))
  }, [])

  const removeMatrixRow = useCallback((setIdx: number, rowIdx: number) => {
    setMatrixSets(prev => prev.map((set, i) => i === setIdx ? set.filter((_, j) => j !== rowIdx) : set).filter((set, i) => i === setIdx ? set.length > 0 : true))
  }, [])

  const updateMatrixRow = useCallback((setIdx: number, rowIdx: number, field: keyof StrategyRow, value: string) => {
    setMatrixSets(prev => prev.map((set, i) => i === setIdx ? set.map((r, j) => j === rowIdx ? { ...r, [field]: value } : r) : set))
  }, [])

  return (
    <div className="grid gap-4 lg:grid-cols-3">
      <Card className="lg:col-span-2">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-sm">
            <Grid3X3 className="h-4 w-4" /> Strategy Pairs
          </CardTitle>
          <CardDescription className="text-xs">Multi-strategy orchestration with shared capital pool</CardDescription>
        </CardHeader>
        <CardContent className="space-y-2">
          {strategyRows.map((row, index) => (
            <div key={index} className="flex items-center gap-2">
              <Select value={row.strategy_id} onValueChange={(v: string) => updateRow(index, "strategy_id", v)}>
                <SelectTrigger className="flex-1 h-8 text-xs">
                  <SelectValue placeholder="Select strategy" />
                </SelectTrigger>
                <SelectContent>
                  {availableStrategies.map((s) => (
                    <SelectItem key={s.id} value={s.id} className="text-xs">{s.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Select value={row.symbol} onValueChange={(v: string) => updateRow(index, "symbol", v)}>
                <SelectTrigger className="w-24 h-8 text-xs"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {SYMBOL_OPTIONS.map((s) => <SelectItem key={s} value={s} className="text-xs">{s}</SelectItem>)}
                </SelectContent>
              </Select>
              <Select value={row.timeframe} onValueChange={(v: string) => updateRow(index, "timeframe", v)}>
                <SelectTrigger className="w-20 h-8 text-xs"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {TIMEFRAME_OPTIONS.map((tf) => <SelectItem key={tf} value={tf} className="text-xs">{tf}</SelectItem>)}
                </SelectContent>
              </Select>
              {strategyRows.length > 1 && (
                <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => removeRow(index)}>
                  <Trash2 className="h-3.5 w-3.5 text-muted-foreground" />
                </Button>
              )}
            </div>
          ))}
          <div className="flex items-center gap-2 pt-1">
            <Button variant="outline" size="sm" className="h-7 text-xs gap-1" onClick={addRow}>
              <Plus className="h-3 w-3" /> Add
            </Button>
            <Button variant="ghost" size="sm" className="h-7 text-xs gap-1" onClick={fillRecommended}>
              <Zap className="h-3 w-3" /> Top Picks
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-sm">
            <Activity className="h-4 w-4" /> Configuration
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="grid grid-cols-2 gap-2">
            <div className="space-y-1">
              <Label className="text-xs">Start</Label>
              <Input type="date" className="h-8 text-xs" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">End</Label>
              <Input type="date" className="h-8 text-xs" value={endDate} onChange={(e) => setEndDate(e.target.value)} />
            </div>
          </div>
          <div className="flex items-center gap-2">
            <div className="space-y-1 flex-1">
              <Label className="text-xs">Capital</Label>
              <div className="relative">
                <DollarSign className="absolute left-2 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground" />
                <Input type="number" className="h-8 text-xs pl-6" value={capital} onChange={(e) => setCapital(e.target.value)} />
              </div>
            </div>
            <div className="space-y-1 w-24">
              <Label className="text-xs">Rebalance</Label>
              <Input type="number" className="h-8 text-xs" value={rebalanceBars} onChange={(e) => setRebalanceBars(e.target.value)} />
            </div>
          </div>
          <div className="space-y-1">
            <Label className="text-xs">Kelly Fraction</Label>
            <Input type="number" step="0.05" min="0" max="1" className="h-8 text-xs" value={kellyFraction} onChange={(e) => setKellyFraction(e.target.value)} />
          </div>
          <div className="space-y-1">
            <div className="flex justify-between"><Label className="text-xs">Max Position %</Label><span className="text-xs tabular-nums">{maxPositionPct}%</span></div>
            <Slider min={0.5} max={20} step={0.5} value={[parseFloat(maxPositionPct) || 2]} onValueChange={([v]: number[]) => setMaxPositionPct(String(v ?? 2))} />
          </div>
          <div className="flex items-center gap-2">
            <Checkbox id="corr-brake" checked={enableCorrelationBrake} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEnableCorrelationBrake(e.target.checked)} />
            <Label htmlFor="corr-brake" className="text-xs">Correlation Brake</Label>
          </div>
          {enableCorrelationBrake && (
            <div className="space-y-1">
              <div className="flex justify-between">
                <Label className="text-xs">Threshold</Label>
                <span className="text-xs tabular-nums">{correlationThreshold.toFixed(1)}</span>
              </div>
              <Slider min={0.3} max={0.9} step={0.05} value={[correlationThreshold]} onValueChange={([v]: number[]) => setCorrelationThreshold(v ?? 0.6)} />
            </div>
          )}
          <FrictionToggle model={frictionModel} onChange={setFrictionModel} />
          <div className="flex items-center gap-2">
            <Checkbox id="matrix-mode" checked={matrixMode} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setMatrixMode(e.target.checked)} />
            <Label htmlFor="matrix-mode" className="text-xs cursor-pointer" onClick={() => setMatrixMode(!matrixMode)}>Matrix Mode</Label>
          </div>
          {!matrixMode && (
            <Button className="w-full gap-1.5 h-8 text-xs" onClick={handleSubmit} disabled={loading || strategyRows.every((r) => !r.strategy_id)}>
              {loading ? <><RefreshCw className="h-3.5 w-3.5 animate-spin" /> Submitting...</> : <><Play className="h-3.5 w-3.5" /> Run</>}
            </Button>
          )}
          {matrixMode && (
            <Button className="w-full gap-1.5 h-8 text-xs" onClick={handleMatrixSubmit} disabled={matrixLoading || matrixSets.every(set => set.every(r => !r.strategy_id))}>
              {matrixLoading ? <><RefreshCw className="h-3.5 w-3.5 animate-spin" /> Submitting...</> : <><Layers className="h-3.5 w-3.5" /> Run Matrix</>}
            </Button>
          )}
          {error && (
            <div className="flex items-center gap-2 rounded-md bg-destructive/10 px-3 py-2 text-xs text-destructive">
              <AlertCircle className="h-3.5 w-3.5 shrink-0" /> {error}
            </div>
          )}
        </CardContent>
      </Card>

      {matrixMode && (
        <Card className="lg:col-span-2">
          <CardHeader><CardTitle className="flex items-center gap-2 text-sm"><Layers className="h-4 w-4" /> Matrix Sets</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            {matrixSets.map((set, setIdx) => (
              <div key={setIdx} className="border rounded-md p-2 space-y-1">
                <div className="flex items-center justify-between">
                  <Badge variant="outline" className="text-[10px] h-5">Set {setIdx + 1}</Badge>
                  {matrixSets.length > 1 && (
                    <button className="text-[10px] text-destructive" onClick={() => setMatrixSets(prev => prev.filter((_, i) => i !== setIdx))}>Remove</button>
                  )}
                </div>
                {set.map((row, rowIdx) => (
                  <div key={rowIdx} className="flex items-center gap-2">
                    <Select value={row.strategy_id} onValueChange={(v: string) => updateMatrixRow(setIdx, rowIdx, "strategy_id", v)}>
                      <SelectTrigger className="flex-1 h-7 text-xs"><SelectValue placeholder="Strategy" /></SelectTrigger>
                      <SelectContent>
                        {availableStrategies.map(s => <SelectItem key={s.id} value={s.id} className="text-xs">{s.name}</SelectItem>)}
                      </SelectContent>
                    </Select>
                    <Select value={row.symbol} onValueChange={(v: string) => updateMatrixRow(setIdx, rowIdx, "symbol", v)}>
                      <SelectTrigger className="w-20 h-7 text-xs"><SelectValue /></SelectTrigger>
                      <SelectContent>{SYMBOL_OPTIONS.map(s => <SelectItem key={s} value={s} className="text-xs">{s}</SelectItem>)}</SelectContent>
                    </Select>
                    <Select value={row.timeframe} onValueChange={(v: string) => updateMatrixRow(setIdx, rowIdx, "timeframe", v)}>
                      <SelectTrigger className="w-16 h-7 text-xs"><SelectValue /></SelectTrigger>
                      <SelectContent>{TIMEFRAME_OPTIONS.map(tf => <SelectItem key={tf} value={tf} className="text-xs">{tf}</SelectItem>)}</SelectContent>
                    </Select>
                    {set.length > 1 && <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => removeMatrixRow(setIdx, rowIdx)}><Trash2 className="h-3 w-3" /></Button>}
                  </div>
                ))}
                <Button variant="outline" size="sm" className="h-6 text-[10px] gap-1" onClick={() => addMatrixRow(setIdx)}><Plus className="h-3 w-3" />Add Strategy</Button>
              </div>
            ))}
            <Button variant="outline" size="sm" className="h-6 text-[10px] gap-1" onClick={addMatrixSet}><Plus className="h-3 w-3" />Add Set</Button>
            <p className="text-[10px] text-muted-foreground">Each set is an independent orchestration run. Common config applies to all sets.</p>
          </CardContent>
        </Card>
      )}

      {matrixResults.length > 0 && (
        <OrchMatrixResultsPanel results={matrixResults} telemetry={matrixTelemetry} onViewDetail={(id) => onSubmit(id)} />
      )}
    </div>
  )
}
