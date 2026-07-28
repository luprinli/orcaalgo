import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useWebSocket } from '../hooks/useWebSocket'
import { live, orders, positions, risk, monitor, signals, system } from '../api/client'
import { Card, CardContent } from '../components/ui/card'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../components/ui/tabs'
import { Badge } from '../components/ui/badge'
import OverviewTab from './monitor/OverviewTab'
import PositionsTab from './monitor/PositionsTab'
import RiskTab from './monitor/RiskTab'
import SignalsTab from './monitor/SignalsTab'
import SystemHealthTab from './monitor/SystemHealthTab'
import type { LiveMetrics, EquityPoint, Position, Order, TradeSummary, RiskStatus, SystemHealth } from '../types/api'
import type { WSRiskData } from '../types/ws'
import type { OverviewComputed } from './monitor/OverviewTab'
import type { SignalEntry } from './monitor/SignalsTab'

type MonitorTab = 'overview' | 'positions' | 'risk' | 'signals' | 'systemHealth'

export default function MonitorPage() {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<MonitorTab>('overview')

  const [metrics, setMetrics] = useState<LiveMetrics | null>(null)
  const [equity, setEquity] = useState<EquityPoint[]>([])
  const [positionsList, setPositionsList] = useState<Position[]>([])
  const [ordersList, setOrdersList] = useState<Order[]>([])
  const [liveTrades, setLiveTrades] = useState<TradeSummary[]>([])
  const [riskStatus, setRiskStatus] = useState<RiskStatus | null>(null)
  const [wsRisk, setWsRisk] = useState<WSRiskData | null>(null)
  const [regimeHistory, setRegimeHistory] = useState<{ timestamp: string; regime: number }[]>([])
  const [signalEntries, setSignalEntries] = useState<SignalEntry[]>([])
  const [systemHealth, setSystemHealth] = useState<SystemHealth | null>(null)
  const [error, setError] = useState<string | null>(null)

  const { connected: wsConnected } = useWebSocket('risk', {
    onMessage: (data) => setWsRisk(data as WSRiskData),
  })

  useWebSocket('ticks', {
    onMessage: () => { /* ticks feed — data consumed by chart hooks */ },
  })

  const fetchAll = useCallback(async () => {
    try {
      const [m, e, p, o, t, r, rh, sig, sh] = await Promise.all([
        live.metrics(),
        live.equity('90d'),
        positions.list().catch(() => ({ positions: [] as Position[] })),
        orders.list().catch(() => ({ orders: [] as Order[] })),
        live.trades().catch(() => ({ trades: [] as TradeSummary[] })),
        risk.status().catch(() => null as RiskStatus | null),
        monitor.regimeHistory().catch(() => ({ history: [] as { timestamp: string; regime: number }[] })),
        signals.list().catch(() => ({ signals: [] as SignalEntry[] })),
        system.health().catch(() => null as SystemHealth | null),
      ])
      setMetrics(m)
      setEquity(Array.isArray(e) ? e : [])
      setPositionsList(Array.isArray(p?.positions) ? p.positions : [])
      setOrdersList(Array.isArray(o?.orders) ? o.orders : [])
      setLiveTrades(Array.isArray(t?.trades) ? t.trades : [])
      setRiskStatus(r)
      setRegimeHistory(rh?.history ?? [])
      setSignalEntries(sig?.signals ?? [])
      setSystemHealth(sh)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('common:failedToLoad', 'Failed to load'))
    }
  }, [t])

  useEffect(() => {
    fetchAll()
    const interval = setInterval(fetchAll, 10000)
    return () => clearInterval(interval)
  }, [fetchAll])

  const halted = wsRisk?.halted ?? riskStatus?.halted ?? false
  const equityVal = wsRisk?.equity ?? riskStatus?.equity ?? metrics?.equity ?? 0
  const balanceVal = wsRisk?.balance ?? riskStatus?.balance ?? metrics?.balance ?? 0
  const dailyPnl = wsRisk?.daily_pnl_pct ?? riskStatus?.daily_pnl_pct ?? metrics?.daily_pnl_pct ?? 0
  const regime = wsRisk?.regime ?? -1
  const drawdownUsed = wsRisk?.drawdown_used ?? riskStatus?.drawdown_used ?? 0
  const dailyLossUsed = wsRisk?.daily_loss_used ?? riskStatus?.daily_loss_used ?? 0
  const dailyLimitPct = wsRisk?.daily_limit_pct ?? riskStatus?.daily_limit_pct ?? 5
  const maxDdPct = wsRisk?.max_dd_pct ?? riskStatus?.max_dd_pct ?? 10
  const winRate = metrics?.win_rate ?? 0
  const sharpe = metrics?.sharpe ?? 0
  const profitFactor = metrics?.profit_factor ?? 0
  const totalTrades = metrics?.num_trades ?? 0

  const computed: OverviewComputed = {
    halted,
    equityVal,
    balanceVal,
    dailyPnl,
    regime,
    drawdownUsed,
    dailyLossUsed,
    dailyLimitPct,
    maxDdPct,
    winRate,
    sharpe,
    profitFactor,
    totalTrades,
  }

  const fmt = (v: number) => v.toFixed(2)
  const fmtPct = (v: number) => v.toFixed(2) + '%'

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold mb-0">{t('monitor:title', 'Monitor')}</h1>
        <Badge variant={halted ? 'destructive' : 'success'}>
          {halted ? t('common:halted', 'HALTED') : t('common:active', 'ACTIVE')}
        </Badge>
      </div>

      {/* Screen-reader live region for real-time data */}
      <div role="status" aria-live="polite" className="sr-only">
        {halted ? 'Trading halted' : 'Trading active'}. Equity {fmt(computed.equityVal)}. Daily PnL {fmtPct(computed.dailyPnl)}. {positionsList.length} open positions. {ordersList.length} active orders.
      </div>

      {/* Emergency assertive region */}
      <div role="alert" aria-live="assertive" className="sr-only">
        {halted ? 'WARNING: Trading has been halted by kill-switch' : ''}
      </div>

      {/* Error Banner */}
      {error && (
        <Card className="mb-4 border-l-4 border-l-destructive">
          <CardContent className="py-3 text-sm text-destructive">{error}</CardContent>
        </Card>
      )}

      {/* Tabs */}
      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as MonitorTab)}>
        <TabsList className="mb-4">
          <TabsTrigger value="overview">{t('monitor:tab.overview', 'Overview')}</TabsTrigger>
          <TabsTrigger value="positions">{t('monitor:tab.positions', 'Positions & Orders')}</TabsTrigger>
          <TabsTrigger value="risk">{t('monitor:tab.risk', 'Risk')}</TabsTrigger>
          <TabsTrigger value="signals">{t('monitor:tab.signals', 'Signals')}</TabsTrigger>
          <TabsTrigger value="systemHealth">Health</TabsTrigger>
        </TabsList>

        <TabsContent value="overview">
          <OverviewTab computed={computed} equity={equity} metrics={metrics} systemHealth={systemHealth} wsConnected={wsConnected} riskStatus={riskStatus} />
        </TabsContent>

        <TabsContent value="positions">
          <PositionsTab
            positions={positionsList}
            orders={ordersList}
            liveTrades={liveTrades}
            equity={equity}
            computed={computed}
          />
        </TabsContent>

        <TabsContent value="risk">
          <RiskTab computed={computed} regimeHistory={regimeHistory} onRefresh={fetchAll} />
        </TabsContent>

        <TabsContent value="signals">
          <SignalsTab signals={signalEntries} />
        </TabsContent>

        <TabsContent value="systemHealth">
          <SystemHealthTab />
        </TabsContent>
      </Tabs>
    </div>
  )
}
