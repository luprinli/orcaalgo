import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import type { Strategy } from '../../types/api'
import { Button } from '../../components/ui/button'
import { Card, CardContent } from '../../components/ui/card'
import { Badge } from '../../components/ui/badge'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../components/ui/table'

interface InstancesTabProps {
  dbList: Strategy[]
  loading: boolean
  onEdit: (id: string) => void
  onDelete: (id: string) => void
  onClone: (id: string) => void
  onToggle: (id: string, current: boolean) => void
}

const STRATEGY_DISPLAY: Record<string, string> = {
  grid: 'Grid Trading', mean_reversion: 'Mean Reversion', intraday_mr: 'Mean Reversion',
  trend: 'Trend Following', trend_following: 'Trend Following',
  breakout: 'ORB Breakout', opening_range_breakout: 'ORB Breakout',
  scalp: 'Session Scalp', session_scalp: 'Session Scalp',
  vol_arb: 'Vol Harvesting', stat_arb: 'Stat Arb',
  ma_crossover: 'MA Crossover', rsi2: 'RSI2 Reversion',
  donchian: 'Donchian Breakout', keltner: 'Keltner MACD', ichimoku: 'Ichimoku Cloud',
}

type InstanceRow = Strategy & { isLive: boolean }

export default function InstancesTab({ dbList, loading, onEdit, onDelete, onClone, onToggle }: InstancesTabProps) {
  const { t } = useTranslation()

  if (loading) {
    return (<Card><CardContent className="p-6"><p className="text-xs text-muted-foreground">Loading...</p></CardContent></Card>)
  }

  if (dbList.length === 0) {
    return (
      <Card className="border-dashed"><CardContent className="p-8 text-center">
        <p className="text-sm text-muted-foreground mb-2">No strategy instances created yet.</p>
        <p className="text-[11px] text-muted-foreground/60">Switch to the Catalog tab and click "Add Instance" to get started.</p>
      </CardContent></Card>
    )
  }

  const active = dbList.filter((s) => s.enabled).length
  const inactive = dbList.length - active

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-3 flex-wrap">
        {active > 0 && <Badge variant="default" className="text-[10px] bg-green-500/20 text-green-700">{active} Active</Badge>}
        {inactive > 0 && <Badge variant="secondary" className="text-[10px]">{inactive} Inactive</Badge>}
        <span className="text-[10px] text-muted-foreground">{dbList.length} instances in database</span>
      </div>
      <Card>
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="h-7 text-xs">Name</TableHead>
                <TableHead className="h-7 text-xs">Type</TableHead>
                <TableHead className="h-7 text-xs">Parameters</TableHead>
                <TableHead className="h-7 text-xs">Status</TableHead>
                <TableHead className="h-7 text-xs">Created</TableHead>
                <TableHead className="h-7 text-xs">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {dbList.map((s) => (
                <TableRow key={s.id} className="h-8">
                  <TableCell className="py-1 cursor-pointer font-medium text-xs" onClick={() => onEdit(s.id)}>
                    <div className="flex items-center gap-1">
                      <span className="truncate max-w-[140px]">{s.name}</span>
                      {s.enabled && <span className="w-1.5 h-1.5 rounded-full bg-green-500 shrink-0" />}
                    </div>
                  </TableCell>
                  <TableCell className="py-1">
                    <span className="text-[11px]">{STRATEGY_DISPLAY[s.type] ?? s.type}</span>
                  </TableCell>
                  <TableCell className="py-1">
                    <span className="text-[10px] font-mono text-muted-foreground max-w-[220px] block truncate">
                      {s.parameters ? Object.entries(s.parameters).slice(0, 5).map(([k, v]) => `${k}:${v}`).join('  ') : '\u2014'}
                      {s.parameters && Object.keys(s.parameters).length > 5 ? ' ...' : ''}
                    </span>
                  </TableCell>
                  <TableCell className="py-1">
                    <Badge variant={s.enabled ? 'default' : 'secondary'} className={`text-[9px] h-4 ${s.enabled ? 'bg-green-500/20 text-green-700' : ''}`}>
                      {s.enabled ? 'Live' : 'Inactive'}
                    </Badge>
                  </TableCell>
                  <TableCell className="py-1 text-[10px] text-muted-foreground">
                    {s.created_at ? new Date(s.created_at).toLocaleDateString() : '\u2014'}
                  </TableCell>
                  <TableCell className="py-1">
                    <div className="flex gap-1">
                      <Button variant="ghost" size="sm" className="h-5 text-[10px] px-1.5" onClick={() => onEdit(s.id)}>Edit</Button>
                      <Button variant="ghost" size="sm" className="h-5 text-[10px] px-1.5" onClick={() => onToggle(s.id, s.enabled)}>
                        {s.enabled ? 'Disable' : 'Enable'}
                      </Button>
                      <Button variant="ghost" size="sm" className="h-5 text-[10px] px-1.5" onClick={() => onClone(s.id)}>Clone</Button>
                      <Button variant="ghost" size="sm" className="h-5 text-[10px] px-1.5 text-destructive" onClick={() => onDelete(s.id)}>Del</Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </Card>
    </div>
  )
}
