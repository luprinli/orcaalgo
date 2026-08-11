"use client"

import { useState, useEffect, useCallback } from "react"
import {
  Activity, TrendingUp, TrendingDown, Percent, DollarSign, Clock,
  PieChart, Grid3X3, AlertCircle, RefreshCw, Download, BarChart3, ChevronDown, ChevronRight,
} from "lucide-react"
import { Card, CardHeader, CardTitle, CardContent } from "../ui/card"
import { Badge } from "../ui/badge"
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "../ui/table"
import { orchestrator } from "../../api/client"
import { exportOrchTradesCSV, exportOrchAllocationCSV, exportOrchBreachesCSV } from "../../lib/export"
import { CorrelationHeatmap } from "../orchestration/CorrelationHeatmap"
import { AllocationPie } from "../orchestration/AllocationPie"
import EquityCurveChart from "../../charts/EquityCurveChart"
import MonteCarloChart from "../../charts/MonteCarloChart"
import { formatNumber, formatPctRaw } from "../../lib/format"
import type {
  OrchestrationRun, OrchestrationRunResult, AllocationEntry, BreachEvent, EquityPoint,
} from "../../types/api"

function statusBadge(status: string) {
  const variants: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
    running: "default", completed: "outline", failed: "destructive", cancelled: "secondary",
  }
  const labels: Record<string, string> = {
    running: "Running", completed: "Completed", failed: "Failed", cancelled: "Cancelled",
  }
  return <Badge variant={variants[status] || "secondary"}>{(labels[status] || status)}</Badge>
}

function formatShortId(id: string) { return id.length > 8 ? `${id.slice(0, 8)}...` : id }
function formatDateRange(start: string, end: string) { return `${start.slice(0, 10)} - ${end.slice(0, 10)}` }

export default function OrchestrationDetail({ runId }: { runId: string }) {
  const [run, setRun] = useState<OrchestrationRun | null>(null)
  const [allocation, setAllocation] = useState<AllocationEntry[]>([])
  const [correlationMatrix, setCorrelationMatrix] = useState<Record<string, Record<string, number>>>({})
  const [correlationBreaches, setCorrelationBreaches] = useState<BreachEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [mcOpen, setMcOpen] = useState(false)

  const fetchDetail = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [runData, allocData, corrData] = await Promise.all([
        orchestrator.get(runId),
        orchestrator.getAllocation(runId),
        orchestrator.getCorrelation(runId),
      ])
      setRun(runData)
      setAllocation(allocData ?? [])

      const resultJson = (corrData as any).result_json ?? (runData as any).result_json
      const breaches: BreachEvent[] = resultJson?.correlation_breaches ?? []

      const strategyIds = (corrData as any).strategy_ids ?? runData.strategy_ids
      const matrix: Record<string, Record<string, number>> = {}
      if (strategyIds && strategyIds.length > 0) {
        for (const a of strategyIds) {
          matrix[a] = {}
          for (const b of strategyIds) {
            if (a === b) matrix[a][b] = 1.0
            else {
              const breach = breaches.find((br) =>
                (br.strategy_a === a && br.strategy_b === b) || (br.strategy_a === b && br.strategy_b === a))
              matrix[a][b] = breach?.correlation ?? 0
            }
          }
        }
      }
      setCorrelationMatrix(matrix)
      setCorrelationBreaches(breaches)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to load run detail")
    } finally { setLoading(false) }
  }, [runId])

  useEffect(() => { fetchDetail() }, [fetchDetail])

  if (loading) {
    return <Card><CardContent className="flex items-center justify-center py-16"><RefreshCw className="h-5 w-5 animate-spin text-muted-foreground" /><span className="ml-2 text-sm text-muted-foreground">Loading run detail...</span></CardContent></Card>
  }
  if (error) {
    return <Card><CardContent className="flex items-center justify-center py-12 text-destructive gap-2"><AlertCircle className="h-4 w-4" /> {error}</CardContent></Card>
  }
  if (!run) {
    return <Card><CardContent className="flex items-center justify-center py-12 text-muted-foreground">Run not found</CardContent></Card>
  }

  const resultJson: OrchestrationRunResult | undefined = (run as any).result_json
  const poolEquity: EquityPoint[] = resultJson?.pool_equity ?? []
  const strategyPnl: Record<string, number> = resultJson?.strategy_pnl ?? {}
  const breaches: BreachEvent[] = resultJson?.correlation_breaches ?? correlationBreaches

  const finalAllocations = allocation.length > 0
    ? (() => {
        const latest = allocation.reduce<Record<string, AllocationEntry>>((acc, entry) => {
          if (!acc[entry.strategy_id] || entry.bar_time > acc[entry.strategy_id].bar_time) {
            acc[entry.strategy_id] = entry
          }
          return acc
        }, {})
        return Object.values(latest).map((e) => ({ strategyId: e.strategy_id, weight: e.weight }))
      })()
    : []

  const poolMetrics = [
    { label: "Sharpe", value: run.pool_sharpe ?? resultJson?.pool_sharpe, format: (v: number) => formatNumber(v), icon: TrendingUp },
    { label: "Sortino", value: run.pool_sortino ?? resultJson?.pool_sortino, format: (v: number) => formatNumber(v), icon: Activity },
    { label: "Max DD", value: run.pool_maxdd ?? resultJson?.pool_maxdd, format: (v: number) => formatPctRaw(v, 1), icon: TrendingDown },
    { label: "Return", value: run.pool_return_pct ?? resultJson?.pool_return_pct, format: (v: number) => formatPctRaw(v, 1), icon: Percent },
    { label: "Rebalance", value: run.rebalance_costs ?? resultJson?.rebalance_costs, format: (v: number) => `$${formatNumber(v)}`, icon: DollarSign },
  ]

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <CardTitle className="font-mono text-sm">{formatShortId(run.id)}</CardTitle>
              {statusBadge(run.status)}
            </div>
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Clock className="h-3.5 w-3.5" />
              {formatDateRange(run.start_date, run.end_date)}
            </div>
          </div>
          {run.status === 'completed' && (
            <div className="flex gap-1 mt-2">
              <button className="text-[10px] text-muted-foreground hover:text-foreground flex items-center gap-1 border rounded px-1.5 py-0.5" onClick={() => orchestrator.getTrades(run.id).then((t) => exportOrchTradesCSV(t, `orch_trades_${run.id.slice(0, 8)}.csv`))}>
                <Download className="h-2.5 w-2.5" />Trades
              </button>
              <button className="text-[10px] text-muted-foreground hover:text-foreground flex items-center gap-1 border rounded px-1.5 py-0.5" onClick={() => exportOrchAllocationCSV(allocation, `orch_allocation_${run.id.slice(0, 8)}.csv`)}>
                <Download className="h-2.5 w-2.5" />Alloc
              </button>
              {breaches.length > 0 && (
                <button className="text-[10px] text-muted-foreground hover:text-foreground flex items-center gap-1 border rounded px-1.5 py-0.5" onClick={() => exportOrchBreachesCSV(breaches, `orch_breaches_${run.id.slice(0, 8)}.csv`)}>
                  <Download className="h-2.5 w-2.5" />Breaches
                </button>
              )}
            </div>
          )}
        </CardHeader>
      </Card>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
        {poolMetrics.map((metric) => (
          <Card key={metric.label}>
            <CardContent className="p-3">
              <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground mb-1">
                <metric.icon className="h-3 w-3" /> {metric.label}
              </div>
              <div className="font-mono text-base font-semibold tabular-nums">
                {metric.value != null ? metric.format(metric.value) : "--"}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader><CardTitle className="flex items-center gap-2 text-sm"><TrendingUp className="h-4 w-4" /> Pool Equity Curve</CardTitle></CardHeader>
        <CardContent>
          {poolEquity.length > 0 ? <EquityCurveChart data={poolEquity} height={280} /> : (
            <div className="flex items-center justify-center py-10 text-sm text-muted-foreground">No equity data available</div>
          )}
        </CardContent>
      </Card>

      {(() => {
        const dailyReturns = (resultJson as any)?.daily_returns as Array<{ return?: number }> | undefined
        const mcResult = (resultJson as any)?.monte_carlo
        const returnVals = dailyReturns?.map(d => d.return ?? 0).filter(r => !isNaN(r)) ?? []
        if (returnVals.length < 5) return null
        return (
          <Card>
            <CardHeader className="cursor-pointer pb-2" onClick={() => setMcOpen(!mcOpen)}>
              <CardTitle className="flex items-center gap-2 text-sm">
                <BarChart3 className="h-4 w-4" /> Monte Carlo Simulation
                {mcOpen ? <ChevronDown className="h-4 w-4 ml-auto" /> : <ChevronRight className="h-4 w-4 ml-auto" />}
              </CardTitle>
              {mcResult?.summary && (
                <div className="flex gap-4 text-[10px] text-muted-foreground mt-1">
                  <span>Avg PnL: {mcResult.summary.avg_pnl_pct?.toFixed(1)}%</span>
                  <span>P10 PnL: {mcResult.summary.p10_pnl_pct?.toFixed(1)}%</span>
                  <span>Avg MaxDD: {mcResult.summary.avg_max_dd_pct?.toFixed(1)}%</span>
                  <span>Pass: {((mcResult.pass_probability ?? 0) * 100).toFixed(0)}%</span>
                </div>
              )}
            </CardHeader>
            {mcOpen && (
              <CardContent>
                {(() => {
                  const dr = (resultJson as any)?.daily_returns as Array<{ return?: number; date?: string }> | undefined
                  const dailyRetObjects = dr?.map((d, i) => ({ date: d.date ?? `day_${i}`, return_pct: (d.return ?? 0) * 100, pnl: 0 })).filter(d => !isNaN(d.return_pct)) ?? []
                  return dailyRetObjects.length >= 5
                    ? <MonteCarloChart dailyReturns={dailyRetObjects} forwardDays={Math.min(252, dailyRetObjects.length)} />
                    : <div className="text-xs text-muted-foreground py-4">Insufficient daily returns for Monte Carlo simulation</div>
                })()}
              </CardContent>
            )}
          </Card>
        )
      })()}

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader><CardTitle className="flex items-center gap-2 text-sm"><PieChart className="h-4 w-4" /> Final Allocation</CardTitle></CardHeader>
          <CardContent><AllocationPie allocations={finalAllocations} history={allocation} title="Strategy Weights" /></CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle className="flex items-center gap-2 text-sm"><Grid3X3 className="h-4 w-4" /> Correlation Matrix</CardTitle></CardHeader>
          <CardContent>
            {run.strategy_ids.length > 0 ? (
              <CorrelationHeatmap correlationMatrix={correlationMatrix} strategyIds={run.strategy_ids} />
            ) : (
              <div className="flex items-center justify-center py-10 text-sm text-muted-foreground">No strategy data available</div>
            )}
          </CardContent>
        </Card>
      </div>

      {Object.keys(strategyPnl).length > 0 && (
        <Card>
          <CardHeader><CardTitle className="flex items-center gap-2 text-sm"><Activity className="h-4 w-4" /> Strategy Performance</CardTitle></CardHeader>
          <CardContent>
            <Table>
              <TableHeader><TableRow><TableHead>Strategy</TableHead><TableHead className="text-right">PnL</TableHead></TableRow></TableHeader>
              <TableBody>
                {Object.entries(strategyPnl).map(([id, pnl]) => (
                  <TableRow key={id}>
                    <TableCell className="font-mono text-xs">{id}</TableCell>
                    <TableCell className={`text-right font-mono text-xs tabular-nums ${pnl >= 0 ? "text-emerald-500" : "text-red-500"}`}>${formatNumber(pnl)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {breaches.length > 0 && (
        <Card>
          <CardHeader><CardTitle className="flex items-center gap-2 text-sm"><AlertCircle className="h-4 w-4" /> Correlation Breaches</CardTitle></CardHeader>
          <CardContent>
            <Table>
              <TableHeader><TableRow><TableHead>Strategy A</TableHead><TableHead>Strategy B</TableHead><TableHead className="text-right">Corr</TableHead><TableHead>Action</TableHead></TableRow></TableHeader>
              <TableBody>
                {breaches.map((breach, i) => (
                  <TableRow key={i}>
                    <TableCell className="font-mono text-xs">{breach.strategy_a}</TableCell>
                    <TableCell className="font-mono text-xs">{breach.strategy_b}</TableCell>
                    <TableCell className="text-right font-mono text-xs tabular-nums">{breach.correlation.toFixed(3)}</TableCell>
                    <TableCell><Badge variant="outline" className="text-[10px]">{breach.action.replace(/_/g, " ")}</Badge></TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
