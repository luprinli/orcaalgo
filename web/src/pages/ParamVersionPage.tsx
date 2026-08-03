import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { Badge } from '../components/ui/badge'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '../components/ui/select'
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '../components/ui/table'
import { Skeleton } from '../components/ui/skeleton'
import { paramVersions } from '../api/client'
import type { ParamVersion } from '../types/api'

const STRATEGY_OPTIONS = [
  { value: 'trend_following', label: 'Trend Following' },
  { value: 'session_scalp', label: 'Session Scalp' },
  { value: 'mean_reversion', label: 'Mean Reversion' },
  { value: 'opening_range_breakout', label: 'Opening Range Breakout' },
  { value: 'pairs_trading', label: 'Pairs Trading' },
  { value: 'volatility_harvesting', label: 'Volatility Harvesting' },
]

export default function ParamVersionPage() {
  const { t } = useTranslation()
  const [strategyId, setStrategyId] = useState('trend_following')
  const [versions, setVersions] = useState<ParamVersion[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [actionMsg, setActionMsg] = useState<string | null>(null)

  const fetchVersions = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await paramVersions.list(strategyId)
      setVersions(data ?? [])
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load versions')
    } finally {
      setLoading(false)
    }
  }, [strategyId])

  useEffect(() => {
    fetchVersions()
  }, [fetchVersions])

  async function handleActivate(versionTag: string) {
    try {
      await paramVersions.activate(strategyId, versionTag)
      setActionMsg(`Activated ${versionTag}`)
      fetchVersions()
    } catch (err: unknown) {
      setActionMsg(`Error: ${err instanceof Error ? err.message : 'Failed'}`)
    }
  }

  async function handleDeactivate() {
    try {
      await paramVersions.deactivate(strategyId)
      setActionMsg('Reverted to registry defaults')
      fetchVersions()
    } catch (err: unknown) {
      setActionMsg(`Error: ${err instanceof Error ? err.message : 'Failed'}`)
    }
  }

  function formatDate(iso?: string) {
    if (!iso) return '—'
    return new Date(iso).toLocaleDateString()
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold mb-0">{t('params:title', 'Parameter Versions')}</h1>
      </div>

      <div className="flex items-center gap-4 mb-4">
        <Select value={strategyId} onValueChange={setStrategyId}>
          <SelectTrigger className="w-60">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {STRATEGY_OPTIONS.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>{opt.label}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button variant="outline" size="sm" onClick={fetchVersions} disabled={loading}>
          Refresh
        </Button>
        <Button variant="ghost" size="sm" onClick={handleDeactivate} disabled={loading}>
          Revert to Defaults
        </Button>
      </div>

      {actionMsg && (
        <Card className="mb-4 border-l-4 border-l-primary">
          <CardContent className="py-2 text-sm">{actionMsg}</CardContent>
        </Card>
      )}

      {error && (
        <Card className="mb-4 border-l-4 border-l-destructive">
          <CardContent className="py-3 text-sm text-destructive">{error}</CardContent>
        </Card>
      )}

      {loading ? (
        <div className="space-y-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : versions.length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            No parameter versions found. Run an optimization to create the first version.
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>Versions — {STRATEGY_OPTIONS.find((s) => s.value === strategyId)?.label}</CardTitle>
            <CardDescription>Click Activate to promote a version to live. Active version is highlighted.</CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Version</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>IS Period</TableHead>
                  <TableHead className="text-right">OOS Sharpe</TableHead>
                  <TableHead className="text-right">OOS Max DD</TableHead>
                  <TableHead className="text-right">OOS Return</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Action</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {versions.map((v) => (
                  <TableRow key={v.id} className={v.is_active ? 'bg-primary/5' : undefined}>
                    <TableCell className="font-mono text-xs">{v.version_tag}</TableCell>
                    <TableCell className="text-xs">{formatDate(v.created_at)}</TableCell>
                    <TableCell className="text-xs">
                      {formatDate(v.in_sample_start)} – {formatDate(v.in_sample_end)}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {v.oos_sharpe != null ? v.oos_sharpe.toFixed(3) : '—'}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {v.oos_max_dd != null ? `${v.oos_max_dd.toFixed(1)}%` : '—'}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {v.oos_return_pct != null ? `${v.oos_return_pct.toFixed(1)}%` : '—'}
                    </TableCell>
                    <TableCell>
                      {v.is_active ? (
                        <Badge variant="default">Active</Badge>
                      ) : (
                        <Badge variant="secondary">Inactive</Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      {!v.is_active && (
                        <Button variant="outline" size="xs" onClick={() => handleActivate(v.version_tag)}>
                          Activate
                        </Button>
                      )}
                    </TableCell>
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
