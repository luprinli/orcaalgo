import { useTranslation } from 'react-i18next'
import { Card, CardContent, CardHeader, CardTitle } from '../../components/ui/card'
import { Progress } from '../../components/ui/progress'
import { Badge } from '../../components/ui/badge'
import MetricCard from '../../components/MetricCard'
import EquityCurveChart from '../../charts/EquityCurveChart'
import { useAuthStore } from '../../stores/authStore'
import type { LiveMetrics, EquityPoint, SystemHealth, RiskStatus } from '../../types/api'

export interface OverviewComputed {
  halted: boolean
  equityVal: number
  balanceVal: number
  dailyPnl: number
  regime: number
  drawdownUsed: number
  dailyLossUsed: number
  dailyLimitPct: number
  maxDdPct: number
  winRate: number
  sharpe: number
  profitFactor: number
  totalTrades: number
}

interface OverviewTabProps {
  computed: OverviewComputed
  equity: EquityPoint[]
  metrics: LiveMetrics | null
  systemHealth: SystemHealth | null
  wsConnected: boolean
  riskStatus: RiskStatus | null
}

export default function OverviewTab({ computed, equity, metrics, systemHealth, wsConnected, riskStatus }: OverviewTabProps) {
  const { t } = useTranslation()
  const isAuth = useAuthStore((s) => s.token) !== null

  const c = computed
  const regimeLabels = [t('risk:regime.calm', 'Calm'), t('risk:regime.trending', 'Trending'), t('risk:regime.highVol', 'HighVol'), t('risk:regime.crisis', 'Crisis')]

  const format = (v: number | null | undefined, d = 2) => v != null ? v.toFixed(d) : t('common:noData', '--')

  return (
    <div>
      {/* Metric Cards */}
      <div className="grid grid-cols-3 gap-2 mb-4">
        <MetricCard label="Balance" value={c.balanceVal} format="currency" />
        <MetricCard label="Equity" value={c.equityVal} format="currency" />
        <MetricCard label="Daily P&L" value={c.dailyPnl} format="percent_raw" color="auto" />
        <MetricCard label="Sharpe" value={c.sharpe} format="decimal" />
        <MetricCard label="Max Drawdown" value={c.maxDdPct} format="percent_raw" color="negative" />
        <MetricCard label="Win Rate" value={c.winRate} format="percent_raw" />
        <MetricCard label="Profit Factor" value={c.profitFactor} format="decimal" />
        <MetricCard label="Regime" value={c.regime >= 0 ? regimeLabels[c.regime] : '--'} />
        <MetricCard label="Total Trades" value={c.totalTrades} format="number" />
      </div>

      {/* Equity Curve */}
      {equity.length > 0 && (
        <EquityCurveChart
          data={equity}
          height={300}
          title={t('dashboard:liveEquityCurve', 'Live Equity Curve')}
          color="#2962FF"
        />
      )}

      {/* Risk Limits + System Status */}
      <div className="grid grid-cols-2 gap-4 mt-4">
        <Card>
          <CardHeader><CardTitle>{t('dashboard:riskLimits', 'Risk Limits')}</CardTitle></CardHeader>
          <CardContent>
            <div className="flex flex-col gap-3">
              <div>
                <div className="flex justify-between mb-1">
                  <span className="text-sm text-muted-foreground">{t('dashboard:dailyLossUsed', 'Daily Loss Used')}</span>
                  <span className="text-sm font-semibold">{format(c.dailyLossUsed, 1)}% / {format(c.dailyLimitPct, 1)}%</span>
                </div>
                <Progress value={c.dailyLossUsed} max={c.dailyLimitPct} />
              </div>
              <div>
                <div className="flex justify-between mb-1">
                  <span className="text-sm text-muted-foreground">{t('dashboard:drawdownUsed', 'Drawdown Used')}</span>
                  <span className="text-sm font-semibold">{format(c.drawdownUsed, 1)}% / {format(c.maxDdPct, 1)}%</span>
                </div>
                <Progress value={c.drawdownUsed} max={c.maxDdPct} />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle>{t('dashboard:systemStatus', 'System Status')}</CardTitle></CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-4">
              {[
                { l: t('dashboard:brokerOnline', 'Broker Online'), ok: !c.halted },
                { l: t('dashboard:dataFeedActive', 'Data Feed Active'), ok: wsConnected },
                { l: t('dashboard:killSwitchActive', 'Kill Switch Active'), ok: riskStatus !== null },
                { l: t('dashboard:dbConnected', 'DB Connected'), ok: (systemHealth?.db_pool_in_use ?? 0) > 0 },
                { l: t('dashboard:authEnforced', 'Auth Enforced'), ok: isAuth },
                { l: t('dashboard:wsConnected', 'WS Connected'), ok: wsConnected },
              ].map(s => (
                <div key={s.l} className="flex items-center gap-2 py-2">
                  <Badge variant={s.ok ? 'success' : 'destructive'} size="sm">{s.ok ? '●' : '○'}</Badge>
                  <span className="text-sm">{s.l}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
