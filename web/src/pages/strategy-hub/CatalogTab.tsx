import { useState, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import type { Strategy } from '../../types/api'
import { strategies, backtests } from '../../api/client'
import { Button } from '../../components/ui/button'
import { Card, CardContent } from '../../components/ui/card'
import { Badge } from '../../components/ui/badge'

interface CatalogTabProps {
  dbList: Strategy[]
  onEdit: (id: string) => void
  onDelete: (id: string) => void
  onClone: (id: string) => void
  onToggle: (id: string, current: boolean) => void
  onNew: () => void
}

const STYLE_COLORS: Record<string, string> = {
  mean_reversion: 'from-blue-500/10 to-blue-950/20 border-blue-500/20',
  trend_following: 'from-amber-500/10 to-amber-950/20 border-amber-500/20',
  breakout: 'from-red-500/10 to-red-950/20 border-red-500/20',
  scalp: 'from-purple-500/10 to-purple-950/20 border-purple-500/20',
  grid: 'from-green-500/10 to-green-950/20 border-green-500/20',
  volatility: 'from-cyan-500/10 to-cyan-950/20 border-cyan-500/20',
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

type TypeGroup = {
  typeKey: string
  displayName: string
  instances: Strategy[]
  activeCount: number
  bestBacktest: { id: string; sharpe: number; trades: number; timeframe?: string } | null
}

export default function CatalogTab({ dbList, onEdit, onDelete, onClone, onToggle, onNew }: CatalogTabProps) {
  const { t } = useTranslation()
  const [btHistory, setBtHistory] = useState<any[]>([])

  useEffect(() => {
    backtests.list({ run_type: 'single', limit: 50 }).catch(() => {}).then((res) => {
      setBtHistory((res as any)?.runs ?? [])
    }).catch(() => {})
  }, [dbList])

  const groups: TypeGroup[] = useMemo(() => {
    const byType = new Map<string, Strategy[]>()
    for (const s of dbList) {
      if (!byType.has(s.type)) byType.set(s.type, [])
      byType.get(s.type)!.push(s)
    }
    return [...byType.entries()].map(([typeKey, instances]) => {
      const activeCount = instances.filter((s) => s.enabled).length
      let bestBacktest: TypeGroup['bestBacktest'] = null
      for (const bt of btHistory) {
        if (bt.strategy_ids?.includes(typeKey)) {
          bestBacktest = { id: bt.id, sharpe: 0, trades: 0 }
          break
        }
      }
      return { typeKey, displayName: STRATEGY_DISPLAY[typeKey] ?? typeKey, instances, activeCount, bestBacktest }
    })
  }, [dbList, btHistory])

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <p className="text-[11px] text-muted-foreground">
          {t('strategies:catalogDesc', 'Strategy types from database instances with linked backtest history')}
        </p>
        <Button size="sm" variant="outline" className="h-6 text-[11px]" onClick={onNew}>
          + New Strategy
        </Button>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3">
        {groups.map((g) => {
          const gradient = STYLE_COLORS[g.instances[0]?.type === 'grid' ? 'grid' : g.instances[0]?.type === 'trend' ? 'trend_following' : g.instances[0]?.type === 'breakout' ? 'breakout' : g.instances[0]?.type === 'scalp' ? 'scalp' : ''] ?? 'from-muted/10 to-transparent border-border'
          return (
            <Card key={g.typeKey} className={`bg-gradient-to-br ${gradient} border`}>
              <CardContent className="p-3 space-y-2.5">
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <h3 className="text-sm font-semibold truncate">{g.displayName}</h3>
                    <p className="text-[10px] text-muted-foreground font-mono">{g.typeKey}</p>
                  </div>
                  <div className="flex gap-1 shrink-0">
                    <Badge variant={g.activeCount > 0 ? 'default' : 'secondary'} className="text-[9px] h-4">
                      {g.activeCount > 0 ? `${g.activeCount} active` : `${g.instances.length} saved`}
                    </Badge>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-1.5">
                  <div className="bg-card/50 rounded p-1.5">
                    <div className="text-[9px] text-muted-foreground">Instances</div>
                    <div className="text-xs font-bold tabular-nums">{g.instances.length}</div>
                  </div>
                  <div className="bg-card/50 rounded p-1.5">
                    <div className="text-[9px] text-muted-foreground">Active</div>
                    <div className="text-xs font-bold tabular-nums" style={{ color: g.activeCount > 0 ? 'var(--trading-success)' : 'var(--trading-danger)' }}>
                      {g.activeCount}
                    </div>
                  </div>
                </div>

                {g.bestBacktest && (
                  <div className="bg-card/30 rounded p-1.5 border border-border/50">
                    <div className="text-[9px] text-muted-foreground mb-1">Latest Backtest</div>
                    <div className="flex gap-2 text-[10px]">
                      <span className="tabular-nums" style={{ color: g.bestBacktest.sharpe >= 1 ? 'var(--trading-success)' : 'var(--trading-danger)' }}>
                        Sharpe {g.bestBacktest.sharpe.toFixed(2)}
                      </span>
                      <span className="tabular-nums text-muted-foreground">{g.bestBacktest.trades} trades</span>
                    </div>
                  </div>
                )}

                <div className="flex gap-1.5 pt-1">
                  {g.instances.some((i) => i.enabled) ? (
                    <>
                      <Badge variant="default" className="text-[9px] h-4 bg-green-500/20 text-green-700">
                        Live
                      </Badge>
                      {g.instances.filter((i) => i.enabled).map((inst) => (
                        <Button key={inst.id} variant="ghost" size="sm" className="h-5 text-[10px] px-1.5" onClick={() => onEdit(inst.id)}>
                          View
                        </Button>
                      ))}
                    </>
                  ) : (
                    <Button variant="outline" size="sm" className="h-5 text-[10px]" onClick={onNew}>
                      Go Live
                    </Button>
                  )}
                  <Button variant="ghost" size="sm" className="h-5 text-[10px] px-1.5" onClick={onNew}>
                    Add Instance
                  </Button>
                </div>
              </CardContent>
            </Card>
          )
        })}
      </div>
    </div>
  )
}
