"use client"

import { useState, useEffect, useCallback } from 'react'
import { strategyStatus } from '../../api/client'
import type { StrategyStatus } from '../../types/api'
import { Card, CardContent } from '../../components/ui/card'
import { Badge } from '../../components/ui/badge'
import { Button } from '../../components/ui/button'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../components/ui/table'
import { Progress } from '../../components/ui/progress'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '../../components/ui/dialog'
import { Textarea } from '../../components/ui/textarea'
import { Label } from '../../components/ui/label'
import { Slider } from '../../components/ui/slider'
import { Activity, TrendingUp, TrendingDown, Clock, ExternalLink } from 'lucide-react'

const STATUS_VARIANT: Record<string, { variant: 'default' | 'secondary' | 'destructive' | 'outline'; label: string; color: string }> = {
  active: { variant: 'default', label: 'Active', color: 'bg-green-500/20 text-green-700' },
  inactive: { variant: 'secondary', label: 'Inactive', color: 'bg-gray-500/20 text-gray-600' },
  standby: { variant: 'outline', label: 'Standby', color: 'bg-yellow-500/20 text-yellow-700' },
  violated: { variant: 'destructive', label: 'Violated', color: 'bg-red-500/20 text-red-700' },
  validated: { variant: 'default', label: 'Validated', color: 'bg-blue-500/20 text-blue-700' },
}

export default function StatusTab() {
  const [statuses, setStatuses] = useState<StrategyStatus[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionTarget, setActionTarget] = useState<{ id: string; action: 'promote' | 'demote' } | null>(null)
  const [reason, setReason] = useState('')
  const [allocPct, setAllocPct] = useState(0)
  const [submitting, setSubmitting] = useState(false)

  const fetch = useCallback(() => {
    setLoading(true)
    strategyStatus.list()
      .then(setStatuses)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => { fetch() }, [fetch])

  const handlePromote = useCallback(async () => {
    if (!actionTarget) return
    setSubmitting(true)
    try {
      await strategyStatus.promote(actionTarget.id, reason || 'Manual promotion', allocPct || 0.5)
      setActionTarget(null)
      setReason('')
      fetch()
    } catch (err) { setError((err as Error).message) }
    finally { setSubmitting(false) }
  }, [actionTarget, reason, allocPct, fetch])

  const handleDemote = useCallback(async () => {
    if (!actionTarget) return
    setSubmitting(true)
    try {
      await strategyStatus.demote(actionTarget.id, reason || 'Manual demotion', allocPct)
      setActionTarget(null)
      setReason('')
      setAllocPct(0)
      fetch()
    } catch (err) { setError((err as Error).message) }
    finally { setSubmitting(false) }
  }, [actionTarget, reason, allocPct, fetch])

  const maxDDColor = (dd?: number) => {
    if (!dd || dd < 10) return 'text-green-600'
    if (dd < 20) return 'text-yellow-600'
    return 'text-red-600'
  }

  if (loading) {
    return <Card><CardContent className="p-6"><p className="text-xs text-muted-foreground">Loading strategy statuses...</p></CardContent></Card>
  }

  if (error) {
    return <Card className="border-destructive/50 bg-destructive/5"><CardContent className="p-3 text-xs text-destructive">{error}</CardContent></Card>
  }

  const activeCount = statuses.filter(s => s.status === 'active' || s.status === 'validated').length
  const inactiveCount = statuses.filter(s => s.status === 'inactive').length
  const violatedCount = statuses.filter(s => s.status === 'violated').length

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-3 flex-wrap">
        <Badge variant="default" className="text-[10px] bg-green-500/20 text-green-700">{activeCount} Active</Badge>
        {inactiveCount > 0 && <Badge variant="secondary" className="text-[10px]">{inactiveCount} Inactive</Badge>}
        {violatedCount > 0 && <Badge variant="destructive" className="text-[10px]">{violatedCount} Violated</Badge>}
        <Button variant="ghost" size="sm" className="h-5 text-[10px] px-1.5 ml-auto" onClick={fetch}>
          <Activity className="w-3 h-3 mr-1" /> Refresh
        </Button>
      </div>

      {statuses.length === 0 ? (
        <Card className="border-dashed"><CardContent className="p-8 text-center">
          <p className="text-sm text-muted-foreground">No live strategy statuses available.</p>
          <p className="text-[11px] text-muted-foreground/60">Run an orchestrated backtest and deploy strategies to see live status tracking.</p>
        </CardContent></Card>
      ) : (
        <Card>
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="h-7 text-xs">Strategy</TableHead>
                  <TableHead className="h-7 text-xs">Status</TableHead>
                  <TableHead className="h-7 text-xs">Allocation</TableHead>
                  <TableHead className="h-7 text-xs">Sharpe</TableHead>
                  <TableHead className="h-7 text-xs">MaxDD</TableHead>
                  <TableHead className="h-7 text-xs">Last Signal</TableHead>
                  <TableHead className="h-7 text-xs">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {statuses.map((s) => {
                  const info = STATUS_VARIANT[s.status] ?? STATUS_VARIANT.inactive
                  return (
                    <TableRow key={s.strategy_id} className="h-8">
                      <TableCell className="py-1 font-medium text-xs">{s.strategy_id}</TableCell>
                      <TableCell className="py-1">
                        <Badge variant={info.variant} className={`text-[9px] h-4 ${info.color}`}>{info.label}</Badge>
                      </TableCell>
                      <TableCell className="py-1">
                        <div className="flex items-center gap-1.5">
                          <Progress value={s.allocation_pct * 100} className="h-1.5 w-16" />
                          <span className="text-[10px] text-muted-foreground">{(s.allocation_pct * 100).toFixed(0)}%</span>
                        </div>
                      </TableCell>
                      <TableCell className="py-1 text-[10px]">
                        {s.trailing_sharpe != null ? s.trailing_sharpe.toFixed(2) : '\u2014'}
                      </TableCell>
                      <TableCell className={`py-1 text-[10px] font-medium ${maxDDColor(s.trailing_maxdd)}`}>
                        {s.trailing_maxdd != null ? `${s.trailing_maxdd.toFixed(1)}%` : '\u2014'}
                      </TableCell>
                      <TableCell className="py-1 text-[10px] text-muted-foreground">
                        {s.last_signal_at ? new Date(s.last_signal_at).toLocaleTimeString() : <Clock className="w-3 h-3 inline" />}
                      </TableCell>
                      <TableCell className="py-1">
                        <div className="flex gap-1">
                          <Button
                            variant="ghost" size="sm" className="h-5 text-[10px] px-1.5 text-green-600"
                            disabled={s.status === 'active' || s.status === 'validated'}
                            onClick={() => { setActionTarget({ id: s.strategy_id, action: 'promote' }); setAllocPct(50) }}
                          >Promote</Button>
                          <Button
                            variant="ghost" size="sm" className="h-5 text-[10px] px-1.5 text-destructive"
                            disabled={s.status === 'inactive' || s.status === 'violated'}
                            onClick={() => { setActionTarget({ id: s.strategy_id, action: 'demote' }); setAllocPct(0) }}
                          >Demote</Button>
                          {s.orchestration_run_id && (
                            <a href={`/backtest?view=detail&id=${s.orchestration_run_id}&type=orchestration`}
                              className="inline-flex items-center h-5 text-[10px] px-1.5 rounded-md text-blue-600 hover:bg-blue-50 no-underline"
                            ><ExternalLink className="w-3 h-3 mr-0.5" />Run</a>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </div>
        </Card>
      )}

      <Dialog open={actionTarget !== null} onOpenChange={(v) => { if (!v) setActionTarget(null) }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="capitalize">{actionTarget?.action} Strategy</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <div>
              <Label className="text-xs">Strategy ID</Label>
              <p className="text-sm font-mono">{actionTarget?.id}</p>
            </div>
            <div>
              <Label className="text-xs">Reason</Label>
              <Textarea value={reason} onChange={(e) => setReason(e.target.value)} className="h-16 text-xs" placeholder="Enter reason..." />
            </div>
            {actionTarget?.action === 'demote' && (
              <div>
                <Label className="text-xs">Allocation %: {allocPct}%</Label>
                <Slider value={[allocPct]} onValueChange={([v]) => setAllocPct(v)} min={0} max={100} step={1} className="mt-1" />
              </div>
            )}
            {actionTarget?.action === 'promote' && (
              <div>
                <Label className="text-xs">Allocation %: {allocPct}%</Label>
                <Slider value={[allocPct]} onValueChange={([v]) => setAllocPct(v)} min={0} max={100} step={1} className="mt-1" />
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" size="sm" className="text-xs" onClick={() => setActionTarget(null)}>Cancel</Button>
            <Button
              size="sm" className="text-xs"
              variant={actionTarget?.action === 'demote' ? 'destructive' : 'default'}
              onClick={actionTarget?.action === 'promote' ? handlePromote : handleDemote}
              disabled={submitting || !reason.trim()}
            >{submitting ? '...' : 'Confirm'}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
