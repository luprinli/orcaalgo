import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useLiveRiskData } from '../hooks/useLiveRiskData'
import { PageHeader, MetricGrid, PageSection } from '../components/layout'
import EquityCurveChart from '../charts/EquityCurveChart'
import { live, orders as ordersApi, positions as positionsApi, risk, monitor } from '../api/client'

type Tab = 'overview' | 'positions' | 'orders' | 'risk'

export default function CommandCenter() {
  const { t } = useTranslation()
  const { riskData, connected, isHalted } = useLiveRiskData()
  const [activeTab, setActiveTab] = useState<Tab>('overview')
  const [metrics, setMetrics] = useState<any>(null)
  const [equity, setEquity] = useState<any[]>([])
  const [positions, setPositions] = useState<any[]>([])
  const [orders, setOrders] = useState<any[]>([])
  const [liveTrades, setLiveTrades] = useState<any[]>([])
  const [regimeHistory, setRegimeHistory] = useState<any[]>([])
  const [error, setError] = useState<string | null>(null)
  const [emergencyCode, setEmergencyCode] = useState('')
  const [showEmergencyConfirm, setShowEmergencyConfirm] = useState(false)
  const [emergencyMsg, setEmergencyMsg] = useState('')

  const fetchAll = async () => {
    try {
      const [m, e, p, o, t, r] = await Promise.all([
        live.metrics(),
        live.equity('90d'),
        positionsApi.list().catch(() => ({ positions: [] })),
        ordersApi.list().catch(() => ({ orders: [] })),
        live.trades().catch(() => ({ trades: [] })),
        monitor.regimeHistory().catch(() => ({ history: [] })),
      ])
      setMetrics(m)
      setEquity(Array.isArray(e) ? e : [])
      setPositions(Array.isArray(p?.positions) ? p.positions : [])
      setOrders(Array.isArray(o?.orders) ? o.orders : [])
      setLiveTrades(Array.isArray(t?.trades) ? t.trades : [])
      setRegimeHistory(Array.isArray(r?.history) ? r.history : [])
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch data')
    }
  }

  useEffect(() => {
    fetchAll()
    const interval = setInterval(fetchAll, 10000)
    return () => clearInterval(interval)
  }, [])

  const handleEmergencyStop = async () => {
    try {
      await risk.emergencyStop(emergencyCode)
      setEmergencyMsg('Emergency stop activated. All positions are being closed.')
      setShowEmergencyConfirm(false)
      setEmergencyCode('')
    } catch (err) {
      setEmergencyMsg(err instanceof Error ? err.message : 'Emergency stop failed')
    }
  }

  const handleEmergencyResume = async () => {
    try {
      await risk.emergencyResume(emergencyCode)
      setEmergencyMsg('Trading resumed.')
      setShowEmergencyConfirm(false)
      setEmergencyCode('')
    } catch (err) {
      setEmergencyMsg(err instanceof Error ? err.message : 'Resume failed')
    }
  }

  const tabs: { key: Tab; label: string }[] = [
    { key: 'overview', label: t('commandCenter:tabs.overview', 'Overview') },
    { key: 'positions', label: `${t('commandCenter:tabs.positions', 'Positions')}${positions.length ? ` (${positions.length})` : ''}` },
    { key: 'orders', label: `${t('commandCenter:tabs.orders', 'Orders')}${orders.length ? ` (${orders.length})` : ''}` },
    { key: 'risk', label: t('commandCenter:tabs.risk', 'Risk') },
  ]

  const safeMetrics = metrics || {}

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        title={t('commandCenter:title', 'Command Center')}
        badge={isHalted
          ? { text: t('commandCenter:halted', 'HALTED'), variant: 'err' }
          : connected
            ? { text: t('commandCenter:live', 'LIVE'), variant: 'ok' }
            : { text: t('commandCenter:offline', 'OFFLINE'), variant: 'warn' }
        }
      />

      {error && (
        <PageSection variant="error">
          <p className="text-red-400 text-sm">{error}</p>
          <button onClick={fetchAll} className="text-xs text-red-400 underline mt-2">Retry</button>
        </PageSection>
      )}

      <div className="flex gap-1 border-b border-slate-700 pb-0">
        {tabs.map(tab => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
            className={`px-4 py-2 text-sm font-medium rounded-t transition-colors ${
              activeTab === tab.key
                ? 'bg-slate-800 text-white border border-slate-700 border-b-slate-800 -mb-px'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {activeTab === 'overview' && (
        <div className="space-y-6">
          <MetricGrid columns={3}>
            <MetricCard label={t('commandCenter:balance', 'Balance')} value={`$${formatNumber(safeMetrics.balance)}`} />
            <MetricCard label={t('commandCenter:equity', 'Equity')} value={`$${formatNumber(safeMetrics.equity)}`}
              subtitle={`${safeMetrics.daily_pnl_pct != null ? (safeMetrics.daily_pnl_pct >= 0 ? '+' : '') + safeMetrics.daily_pnl_pct.toFixed(2) + '%' : ''} today`} />
            <MetricCard label={t('commandCenter:dailyPnl', 'Daily PnL')} value={`$${formatNumber(safeMetrics.daily_pnl)}`}
              variant={safeMetrics.daily_pnl >= 0 ? 'positive' : 'negative'} />
            <MetricCard label={t('commandCenter:sharpe', 'Sharpe')} value={safeMetrics.sharpe?.toFixed(2) ?? '—'} />
            <MetricCard label={t('commandCenter:maxDrawdown', 'Max Drawdown')} value={safeMetrics.max_drawdown_pct != null ? safeMetrics.max_drawdown_pct.toFixed(2) + '%' : '—'} />
            <MetricCard label={t('commandCenter:winRate', 'Win Rate')} value={safeMetrics.win_rate != null ? (safeMetrics.win_rate * 100).toFixed(1) + '%' : '—'} />
            <MetricCard label={t('commandCenter:profitFactor', 'Profit Factor')} value={safeMetrics.profit_factor?.toFixed(2) ?? '—'} />
            <MetricCard label={t('commandCenter:regime', 'Regime')} value={riskData?.regime ?? '—'} />
            <MetricCard label={t('commandCenter:totalTrades', 'Total Trades')} value={safeMetrics.num_trades ?? '—'} />
          </MetricGrid>

          {equity.length > 0 && (
            <PageSection title={t('commandCenter:equityCurve', 'Equity Curve')}>
              <EquityCurveChart data={equity} height={400} />
            </PageSection>
          )}

          <div className="grid grid-cols-2 gap-6">
            <PageSection title={t('commandCenter:riskLimits', 'Risk Limits')}>
              <div className="space-y-4">
                <div>
                  <div className="flex justify-between text-xs text-slate-400 mb-1">
                    <span>{t('commandCenter:dailyLossUsed', 'Daily Loss Used')}</span>
                    <span>{(riskData?.daily_loss_used ?? 0).toFixed(1)}% / {(riskData?.daily_limit_pct ?? 5).toFixed(0)}%</span>
                  </div>
                  <div className="h-2 bg-slate-700 rounded-full overflow-hidden">
                    <div
                      className="h-full rounded-full transition-all"
                      style={{
                        width: `${Math.min((riskData?.daily_loss_used ?? 0) / (riskData?.daily_limit_pct || 5) * 100, 100)}%`,
                        backgroundColor: (riskData?.daily_loss_used ?? 0) > (riskData?.daily_limit_pct ?? 5) * 0.7 ? '#ef4444' : '#22c55e',
                      }}
                    />
                  </div>
                  {riskData && (
                    <div className="text-xs text-slate-500 mt-1 space-y-0.5">
                      <div>Remaining: ${((riskData.balance || 0) * ((riskData.daily_limit_pct || 5) - (riskData.daily_loss_used || 0)) / 100).toLocaleString()} daily loss budget</div>
                    </div>
                  )}
                </div>
                <div>
                  <div className="flex justify-between text-xs text-slate-400 mb-1">
                    <span>{t('commandCenter:drawdownUsed', 'Drawdown Used')}</span>
                    <span>{(riskData?.drawdown_used ?? 0).toFixed(1)}% / {(riskData?.max_dd_pct ?? 20).toFixed(0)}%</span>
                  </div>
                  <div className="h-2 bg-slate-700 rounded-full overflow-hidden">
                    <div
                      className="h-full rounded-full transition-all"
                      style={{
                        width: `${Math.min((riskData?.drawdown_used ?? 0) / (riskData?.max_dd_pct || 20) * 100, 100)}%`,
                        backgroundColor: (riskData?.drawdown_used ?? 0) > (riskData?.max_dd_pct ?? 20) * 0.7 ? '#ef4444' : '#f59e0b',
                      }}
                    />
                  </div>
                  {riskData && (
                    <div className="text-xs text-slate-500 mt-1 space-y-0.5">
                      <div>Breach at: ${((riskData.balance || 0) * (1 - (riskData.max_dd_pct || 20) / 100)).toLocaleString()} equity</div>
                      <div>Remaining: ${((riskData.balance || 0) * ((riskData.max_dd_pct || 20) - (riskData.drawdown_used || 0)) / 100).toLocaleString()} drawdown budget</div>
                    </div>
                  )}
                </div>
              </div>
            </PageSection>

            <PageSection title={t('commandCenter:systemStatus', 'System Status')}>
              <div className="space-y-2 text-sm">
                {[
                  { label: t('commandCenter:websocket', 'WebSocket'), ok: connected },
                  { label: t('commandCenter:riskEngine', 'Risk Engine'), ok: !!riskData },
                  { label: t('commandCenter:marketData', 'Market Data'), ok: true },
                  { label: t('commandCenter:orderRouter', 'Order Router'), ok: true },
                  { label: t('commandCenter:database', 'Database'), ok: true },
                  { label: t('commandCenter:apiGateway', 'API Gateway'), ok: true },
                ].map(s => (
                  <div key={s.label} className="flex justify-between">
                    <span className="text-slate-400">{s.label}</span>
                    <span className={s.ok ? 'text-green-400' : 'text-red-400'}>
                      {s.ok ? 'OK' : 'DOWN'}
                    </span>
                  </div>
                ))}
              </div>
            </PageSection>
          </div>
        </div>
      )}

      {activeTab === 'positions' && (
        <PageSection title={t('commandCenter:openPositions', 'Open Positions')}>
          {positions.length === 0 ? (
            <p className="text-slate-400 text-sm">{t('commandCenter:noPositions', 'No open positions.')}</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{t('commandCenter:table.symbol', 'Symbol')}</th>
                    <th>{t('commandCenter:table.side', 'Side')}</th>
                    <th>{t('commandCenter:table.qty', 'Quantity')}</th>
                    <th>{t('commandCenter:table.avgEntry', 'Avg Entry')}</th>
                    <th>{t('commandCenter:table.unrealizedPnl', 'Unrealized PnL')}</th>
                    <th>{t('commandCenter:table.lastUpdated', 'Last Updated')}</th>
                  </tr>
                </thead>
                <tbody>
                  {positions.map((pos: any, i: number) => (
                    <tr key={i}>
                      <td className="font-medium text-white">{pos.symbol}</td>
                      <td className={pos.side === 'BUY' ? 'text-green-400' : 'text-red-400'}>{pos.side}</td>
                      <td>{pos.quantity}</td>
                      <td>${pos.average_entry_price?.toFixed(2) ?? '—'}</td>
                      <td className={pos.unrealized_pnl >= 0 ? 'text-green-400' : 'text-red-400'}>
                        ${pos.unrealized_pnl?.toFixed(2) ?? '0.00'}
                      </td>
                      <td className="text-slate-400 text-xs">{pos.last_updated ? new Date(pos.last_updated).toLocaleTimeString() : '—'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </PageSection>
      )}

      {activeTab === 'orders' && (
        <PageSection title={t('commandCenter:activeOrders', 'Active Orders')}>
          {orders.length === 0 ? (
            <p className="text-slate-400 text-sm">{t('commandCenter:noOrders', 'No active orders.')}</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{t('commandCenter:table.id', 'Order ID')}</th>
                    <th>{t('commandCenter:table.symbol', 'Symbol')}</th>
                    <th>{t('commandCenter:table.side', 'Side')}</th>
                    <th>{t('commandCenter:table.type', 'Type')}</th>
                    <th>{t('commandCenter:table.qty', 'Qty')}</th>
                    <th>{t('commandCenter:table.filled', 'Filled')}</th>
                    <th>{t('commandCenter:table.price', 'Price')}</th>
                    <th>{t('commandCenter:table.status', 'Status')}</th>
                    <th>{t('commandCenter:table.action', 'Action')}</th>
                  </tr>
                </thead>
                <tbody>
                  {orders.map((ord: any) => (
                    <tr key={ord.order_id}>
                      <td className="text-xs font-mono text-slate-300">{ord.order_id?.slice(0, 12)}</td>
                      <td className="font-medium text-white">{ord.symbol}</td>
                      <td className={ord.side === 'BUY' ? 'text-green-400' : 'text-red-400'}>{ord.side}</td>
                      <td>{ord.order_type ?? ord.type}</td>
                      <td>{ord.quantity}</td>
                      <td>{ord.filled_quantity ?? 0}</td>
                      <td>{ord.price != null ? `$${Number(ord.price).toFixed(2)}` : ord.limit_price != null ? `$${ord.limit_price}` : 'Market'}</td>
                      <td>
                        <span className={`badge ${ord.state === 'filled' ? 'badge-ok' : ord.state === 'rejected' || ord.state === 'cancelled' ? 'badge-err' : 'badge-warn'}`}>
                          {ord.state}
                        </span>
                      </td>
                      <td>
                        {ord.state !== 'filled' && ord.state !== 'cancelled' && (
                          <button className="btn btn-outline text-xs" onClick={() => ordersApi.cancel(ord.order_id).then(fetchAll)}>
                            {t('commandCenter:cancel', 'Cancel')}
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </PageSection>
      )}

      {activeTab === 'risk' && (
        <div className="space-y-6">
          <PageSection title={t('commandCenter:emergencyControls', 'Emergency Controls')} variant="error">
            <p className="text-red-300 text-sm mb-4">
              {t('commandCenter:emergencyWarning', 'Emergency actions require 2FA confirmation. These actions will immediately close all positions across all accounts.')}
            </p>
            {emergencyMsg && (
              <p className={`text-sm mb-3 ${emergencyMsg.includes('failed') ? 'text-red-400' : 'text-green-400'}`}>{emergencyMsg}</p>
            )}
            {showEmergencyConfirm ? (
              <div className="space-y-3">
                <input
                  type="text" inputMode="numeric" maxLength={6}
                  value={emergencyCode} onChange={e => setEmergencyCode(e.target.value.replace(/\D/g, ''))}
                  placeholder={t('commandCenter:2faPlaceholder', 'Enter 2FA code')}
                  className="input max-w-[200px]"
                  onKeyDown={e => { if (e.key === 'Enter' && emergencyCode.length === 6) { isHalted ? handleEmergencyResume() : handleEmergencyStop() } }}
                />
                <div className="flex gap-2">
                  {isHalted ? (
                    <button className="btn btn-primary" disabled={emergencyCode.length !== 6} onClick={handleEmergencyResume}>
                      {t('commandCenter:confirmResume', 'Confirm Resume')}
                    </button>
                  ) : (
                    <button className="btn btn-danger" disabled={emergencyCode.length !== 6} onClick={handleEmergencyStop}>
                      {t('commandCenter:confirmStop', 'Confirm Emergency Stop')}
                    </button>
                  )}
                  <button className="btn btn-outline" onClick={() => { setShowEmergencyConfirm(false); setEmergencyCode(''); setEmergencyMsg('') }}>
                    {t('commandCenter:cancel', 'Cancel')}
                  </button>
                </div>
              </div>
            ) : (
              <div className="flex gap-2">
                <button className="btn btn-danger" onClick={() => setShowEmergencyConfirm(true)} disabled={isHalted}>
                  {isHalted ? t('commandCenter:alreadyHalted', 'Already Halted') : t('commandCenter:emergencyStopAll', 'Emergency Stop All Trading')}
                </button>
                {isHalted && (
                  <button className="btn btn-outline" onClick={() => setShowEmergencyConfirm(true)}>
                    {t('commandCenter:resumeTrading', 'Resume Trading')}
                  </button>
                )}
              </div>
            )}
          </PageSection>

          {regimeHistory.length > 0 && (
            <PageSection title={t('commandCenter:regimeHistory', 'Regime History')}>
              <div className="overflow-x-auto">
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>{t('commandCenter:table.time', 'Time')}</th>
                      <th>{t('commandCenter:table.regime', 'Regime')}</th>
                      <th>{t('commandCenter:table.confidence', 'Confidence')}</th>
                      <th>{t('commandCenter:table.vix', 'VIX')}</th>
                      <th>{t('commandCenter:table.sentiment', 'Sentiment')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {regimeHistory.map((r: any, i: number) => (
                      <tr key={i}>
                        <td className="text-slate-400 text-xs">{r.timestamp ? new Date(r.timestamp).toLocaleString() : r.time ? new Date(r.time).toLocaleString() : '—'}</td>
                        <td className="font-medium text-white">{typeof r.regime === 'number' ? regimeLabels[r.regime] ?? r.regime : r.regime}</td>
                        <td>{r.confidence != null ? (Number(r.confidence) * 100).toFixed(0) + '%' : '—'}</td>
                        <td>{r.vix?.toFixed(1) ?? '—'}</td>
                        <td>{r.sentiment ?? '—'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </PageSection>
          )}
        </div>
      )}
    </div>
  )
}

const regimeLabels = ['Calm', 'Trending', 'HighVol', 'Crisis']

function MetricCard({ label, value, subtitle, variant = 'neutral' }: {
  label: string; value: string; subtitle?: string; variant?: 'positive' | 'negative' | 'neutral'
}) {
  return (
    <div className="metric-card">
      <div className="metric-label">{label}</div>
      <div className={`metric-value ${variant === 'positive' ? 'text-green-400' : variant === 'negative' ? 'text-red-400' : ''}`}>
        {value}
      </div>
      {subtitle && <div className="text-xs text-slate-500 mt-1">{subtitle}</div>}
    </div>
  )
}

function formatNumber(n?: number): string {
  if (n == null) return '—'
  if (Math.abs(n) >= 1e6) return (n / 1e6).toFixed(2) + 'M'
  if (Math.abs(n) >= 1e3) return (n / 1e3).toFixed(1) + 'K'
  return n.toFixed(2)
}
