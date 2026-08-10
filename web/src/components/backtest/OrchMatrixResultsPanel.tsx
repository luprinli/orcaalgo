"use client"

import { Card, CardHeader, CardTitle, CardContent } from "../ui/card"
import { Badge } from "../ui/badge"
import { Button } from "../ui/button"
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "../ui/table"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../ui/select"
import { BarChart3, Download, Filter, Grid3X3 } from "lucide-react"
import { formatNumber, formatPctRaw } from "../../lib/format"
import { exportOrchTradesCSV, exportOrchAllocationCSV, exportOrchBreachesCSV } from "../../lib/export"

interface OrchMatrixResult {
  set_index: number
  strategies: Array<{ strategy_id: string; symbol: string; timeframe: string }>
  pool_sharpe: number
  pool_maxdd: number
  pool_return_pct: number
  num_trades: number
  strategy_pnl?: Record<string, number>
  status: string
  error?: string
  run_id?: string
}

interface OrchMatrixTelemetry {
  total: number
  completed: number
  best_sharpe: number
  best_set: number
  status: string
}

interface OrchMatrixResultsPanelProps {
  results: OrchMatrixResult[]
  telemetry?: OrchMatrixTelemetry
  onViewDetail?: (runId: string) => void
}

export default function OrchMatrixResultsPanel({ results, telemetry, onViewDetail }: OrchMatrixResultsPanelProps) {
  if (!results?.length) return null

  const completed = results.filter(r => r.status === "completed")
  const failed = results.filter(r => r.status === "failed")
  const best = completed.length > 0
    ? completed.reduce((a, b) => (a.pool_sharpe ?? 0) > (b.pool_sharpe ?? 0) ? a : b, completed[0])
    : null

  const hasViewDetail = !!onViewDetail && completed.some(r => r.run_id)

  return (
    <Card className="mt-4">
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2 text-sm">
              <Grid3X3 className="h-4 w-4" /> Orchestration Matrix Results
            </CardTitle>
          </div>
          {telemetry && (
            <div className="flex gap-3 text-[10px] text-muted-foreground">
              <span>{telemetry.completed}/{telemetry.total} completed</span>
              {telemetry.best_sharpe > 0 && <span>Best Sharpe: {formatNumber(telemetry.best_sharpe)}</span>}
            </div>
          )}
        </div>
        <div className="flex gap-3 mt-1">
          <div className="flex items-center gap-1 text-[10px]"><Badge variant="outline" className="h-4 text-[9px]">{completed.length}</Badge>Completed</div>
          {failed.length > 0 && <div className="flex items-center gap-1 text-[10px]"><Badge variant="destructive" className="h-4 text-[9px]">{failed.length}</Badge>Failed</div>}
          {best && (
            <div className="flex items-center gap-1 text-[10px] text-muted-foreground">
              Best: Set #{best.set_index + 1} · Sharpe {formatNumber(best.pool_sharpe)} · Return {formatPctRaw(best.pool_return_pct, 1)}
            </div>
          )}
        </div>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="h-7 text-xs w-12">Set</TableHead>
                <TableHead className="h-7 text-xs">Strategies</TableHead>
                <TableHead className="h-7 text-xs">Status</TableHead>
                <TableHead className="h-7 text-xs text-right">Pool Sharpe</TableHead>
                <TableHead className="h-7 text-xs text-right">Pool MaxDD</TableHead>
                <TableHead className="h-7 text-xs text-right">Pool Return</TableHead>
                <TableHead className="h-7 text-xs text-right">Trades</TableHead>
                {hasViewDetail && <TableHead className="h-7 text-xs w-12" />}
              </TableRow>
            </TableHeader>
            <TableBody>
              {results.map((r, i) => (
                <TableRow key={i} className="h-7"
                  style={{ background: i === (telemetry?.best_set ?? -1) ? 'rgba(63,185,80,.05)' : undefined }}>
                  <TableCell className="py-0.5 text-xs font-mono">#{r.set_index + 1}</TableCell>
                  <TableCell className="py-0.5">
                    <div className="flex flex-wrap gap-0.5">
                      {r.strategies.map((s, j) => (
                        <Badge key={j} variant="secondary" className="text-[9px] h-4">{s.strategy_id}:{s.symbol}</Badge>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell className="py-0.5">
                    <Badge variant={r.status === "completed" ? "outline" : "destructive"} className="text-[9px] h-4">
                      {r.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="py-0.5 text-right font-mono text-xs tabular-nums"
                    style={{ color: (r.pool_sharpe ?? 0) >= 1 ? 'var(--trading-success)' : (r.pool_sharpe ?? 0) > 0 ? 'var(--trading-warning)' : 'var(--trading-danger)' }}>
                    {r.status === "completed" ? formatNumber(r.pool_sharpe ?? 0, 2) : "--"}
                  </TableCell>
                  <TableCell className="py-0.5 text-right font-mono text-xs tabular-nums">
                    {r.status === "completed" ? formatPctRaw(r.pool_maxdd ?? 0, 1) : "--"}
                  </TableCell>
                  <TableCell className="py-0.5 text-right font-mono text-xs tabular-nums"
                    style={{ color: (r.pool_return_pct ?? 0) >= 0 ? 'var(--trading-success)' : 'var(--trading-danger)' }}>
                    {r.status === "completed" ? formatPctRaw(r.pool_return_pct ?? 0, 1) : "--"}
                  </TableCell>
                  <TableCell className="py-0.5 text-right font-mono text-xs tabular-nums">
                    {r.status === "completed" ? r.num_trades : "--"}
                  </TableCell>
                  {hasViewDetail && (
                    <TableCell className="py-0.5">
                      {r.run_id && (
                        <Button variant="ghost" size="sm" className="h-5 text-[10px] px-1" onClick={() => onViewDetail?.(r.run_id!)}>
                          View
                        </Button>
                      )}
                    </TableCell>
                  )}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  )
}
