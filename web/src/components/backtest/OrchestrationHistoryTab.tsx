"use client"

import { useState, useEffect, useCallback } from "react"
import { RefreshCw, History, BarChart3, AlertCircle } from "lucide-react"
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "../ui/card"
import { Button } from "../ui/button"
import { Badge } from "../ui/badge"
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "../ui/table"
import { orchestrator } from "../../api/client"
import { formatNumber, formatPctRaw } from "../../lib/format"
import type { OrchestrationRun } from "../../types/api"

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
function formatDateRange(start: string, end: string) { return `${start.slice(0, 10)} \u2013 ${end.slice(0, 10)}` }

export default function OrchestrationHistoryTab({ onSelectRun }: { onSelectRun: (id: string) => void }) {
  const [runs, setRuns] = useState<OrchestrationRun[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchRuns = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const result = await orchestrator.list(50, 0)
      setRuns(result.runs ?? [])
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to load history")
    } finally { setLoading(false) }
  }, [])

  useEffect(() => { fetchRuns() }, [fetchRuns])

  if (loading && runs.length === 0) {
    return <Card>
      <CardContent className="flex items-center justify-center py-12">
        <RefreshCw className="h-5 w-5 animate-spin text-muted-foreground" />
        <span className="ml-2 text-sm text-muted-foreground">Loading history...</span>
      </CardContent>
    </Card>
  }

  if (error) {
    return <Card>
      <CardContent className="flex items-center justify-center py-12 text-destructive gap-2">
        <AlertCircle className="h-4 w-4" /> {error}
      </CardContent>
    </Card>
  }

  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between space-y-0 pb-2">
        <div>
          <CardTitle className="flex items-center gap-2 text-sm"><History className="h-4 w-4" /> Run History</CardTitle>
          <CardDescription className="text-xs">{runs.length} orchestration runs</CardDescription>
        </div>
        <Button variant="outline" size="sm" className="h-7 text-xs gap-1" onClick={fetchRuns} disabled={loading}>
          <RefreshCw className={`h-3 w-3 ${loading ? "animate-spin" : ""}`} /> Refresh
        </Button>
      </CardHeader>
      <CardContent>
        {runs.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-muted-foreground gap-2">
            <BarChart3 className="h-8 w-8 opacity-30" />
            <span className="text-sm">No orchestration runs yet</span>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="h-8 text-xs">Run ID</TableHead>
                <TableHead className="h-8 text-xs">Date Range</TableHead>
                <TableHead className="h-8 text-xs">Strategies</TableHead>
                <TableHead className="h-8 text-xs">Status</TableHead>
                <TableHead className="h-8 text-xs text-right">Pool Sharpe</TableHead>
                <TableHead className="h-8 text-xs text-right">Pool MaxDD</TableHead>
                <TableHead className="h-8 text-xs text-right">Pool Return</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {runs.map((run) => (
                <TableRow key={run.id} className="cursor-pointer hover:bg-muted/50 h-8"
                  onClick={() => onSelectRun(run.id)}>
                  <TableCell className="py-1 font-mono text-xs">{formatShortId(run.id)}</TableCell>
                  <TableCell className="py-1 text-xs text-muted-foreground">{formatDateRange(run.start_date, run.end_date)}</TableCell>
                  <TableCell className="py-1">
                    <div className="flex flex-wrap gap-1">
                      {run.strategy_ids.slice(0, 3).map((sid) => (
                        <Badge key={sid} variant="secondary" className="text-[10px] h-4">{sid}</Badge>
                      ))}
                      {run.strategy_ids.length > 3 && (
                        <Badge variant="secondary" className="text-[10px] h-4">+{run.strategy_ids.length - 3}</Badge>
                      )}
                    </div>
                  </TableCell>
                  <TableCell className="py-1">{statusBadge(run.status)}</TableCell>
                  <TableCell className="py-1 text-right font-mono text-xs tabular-nums">
                    {run.pool_sharpe != null ? formatNumber(run.pool_sharpe) : "--"}
                  </TableCell>
                  <TableCell className="py-1 text-right font-mono text-xs tabular-nums">
                    {run.pool_maxdd != null ? formatPctRaw(run.pool_maxdd, 1) : "--"}
                  </TableCell>
                  <TableCell className="py-1 text-right font-mono text-xs tabular-nums">
                    {run.pool_return_pct != null ? formatPctRaw(run.pool_return_pct, 1) : "--"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}
