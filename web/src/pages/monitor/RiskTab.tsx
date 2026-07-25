import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { risk, admin } from '../../api/client'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../components/ui/card'
import { Badge } from '../../components/ui/badge'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Progress } from '../../components/ui/progress'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../components/ui/table'
import MetricCard from '../../components/MetricCard'
import type { OverviewComputed } from './OverviewTab'

interface RiskTabProps {
  computed: OverviewComputed
  regimeHistory: { timestamp: string; regime: number }[]
  onRefresh: () => void
}

export default function RiskTab({ computed: c, regimeHistory, onRefresh }: RiskTabProps) {
  const { t } = useTranslation()
  const [twoFACode, setTwoFACode] = useState('')
  const [show2FA, setShow2FA] = useState<'stop' | 'resume' | null>(null)
  const [msg, setMsg] = useState('')
  const [killHistory, setKillHistory] = useState<{ reason: string; source: string; triggered_at: string; resolved_at?: string }[]>([])

  useEffect(() => {
    admin.killHistory().then(r => setKillHistory(r.events ?? [])).catch(() => {})
  }, [])

  const regimeLabels = [t('risk:regime.calm', 'Calm'), t('risk:regime.trending', 'Trending'), t('risk:regime.highVol', 'HighVol'), t('risk:regime.crisis', 'Crisis')]
  const regimeColorClasses = ['text-trading-success', 'text-trading-warning', 'text-trading-danger', 'text-trading-danger']

  const format = (v: number | null | undefined, d = 2) => v != null ? Number(v).toFixed(d) : t('common:noData', '--')

  const handleEmergency = (action: 'stop' | 'resume') => {
    setShow2FA(action)
    setTwoFACode('')
    setMsg('')
  }

  const confirm2FA = async () => {
    if (!show2FA || twoFACode.length !== 6) return
    try {
      if (show2FA === 'stop') {
        await risk.emergencyStop(twoFACode)
        setMsg(t('risk:emergencyStopTriggered', 'Emergency stop triggered \u2014 trading halted'))
      } else {
        await risk.emergencyResume(twoFACode)
        setMsg(t('risk:tradingResumed', 'Trading resumed'))
      }
      setShow2FA(null)
      setTwoFACode('')
      onRefresh()
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t('risk:2faActionFailed', '2FA action failed'))
    }
  }

  return (
    <div>
      {/* Metric Cards */}
      <div className="grid grid-cols-3 gap-2 mb-4">
        <MetricCard label="Balance" value={c.balanceVal} format="currency" />
        <MetricCard label="Equity" value={c.equityVal} format="currency" />
        <MetricCard label="Daily P&L" value={c.dailyPnl} format="percent_raw" color="auto" />
        <MetricCard label="Daily Loss Used" value={c.dailyLossUsed} format="percent_raw" color={c.dailyLossUsed > 0 ? 'negative' : 'default'} />
        <MetricCard label="Drawdown Used" value={c.drawdownUsed} format="percent_raw" color={c.drawdownUsed > 0 ? 'negative' : 'default'} />
        <MetricCard label="Regime" value={c.regime >= 0 ? regimeLabels[c.regime] : '--'} />
      </div>

      {/* Risk Limits */}
      <Card className="mb-4">
        <CardHeader><CardTitle>{t('risk:riskLimits', 'Risk Limits')}</CardTitle></CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-4">
            {[
              { label: t('risk:drawdownUsed', 'Drawdown Used'), value: c.drawdownUsed, max: c.maxDdPct },
              { label: t('risk:dailyLossUsed', 'Daily Loss Used'), value: c.dailyLossUsed, max: c.dailyLimitPct },
            ].map((g) => {
              const pct = g.max > 0 ? Math.min(100, (g.value / g.max) * 100) : 0
              return (
                <div key={g.label}>
                  <div className="flex items-center justify-between mb-2">
                    <CardDescription>{g.label}</CardDescription>
                    <span className="font-semibold">{format(g.value, 1)}% / {format(g.max, 1)}%</span>
                  </div>
                  <div className="h-2 bg-input rounded overflow-hidden">
                    <div
                      className={`h-full rounded transition-[width_.3s] ${pct > 80 ? 'bg-trading-danger' : pct > 50 ? 'bg-trading-warning' : 'bg-trading-success'}`}
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                </div>
              )
            })}
          </div>
        </CardContent>
      </Card>

      {/* Emergency Controls */}
      <Card className="mb-4">
        <CardHeader><CardTitle>{t('risk:emergencyControls', 'Emergency Controls')}</CardTitle></CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground mb-4">
            {c.halted
              ? t('risk:haltedMessage', 'Trading is currently halted. Use emergency resume to re-enable.')
              : t('risk:activeMessage', 'Use emergency stop to immediately halt all trading activity.')}
          </p>
          <div className="flex gap-2">
            <Button
              variant="destructive"
              onClick={() => handleEmergency('stop')}
              disabled={c.halted}
            >
              {t('risk:emergencyStop', 'Emergency Stop')}
            </Button>
            <Button
              onClick={() => handleEmergency('resume')}
              disabled={!c.halted}
            >
              {t('risk:resumeTrading', 'Resume Trading')}
            </Button>
          </div>

          {show2FA && (
            <div className="mt-3 max-w-[300px]">
              <Label>{t('risk:2faCodeRequired', '2FA Code (required)')}</Label>
              <div className="flex gap-2 mt-2">
                <Input
                  placeholder={t('risk:2faPlaceholder', '000000')}
                  maxLength={6}
                  value={twoFACode}
                  onChange={(e) => setTwoFACode(e.target.value.replace(/\D/g, ''))}
                  onKeyDown={(e) => e.key === 'Enter' && confirm2FA()}
                />
                <Button onClick={confirm2FA} disabled={twoFACode.length !== 6}>
                  {t('risk:confirm', 'Confirm')}
                </Button>
                <Button variant="outline" onClick={() => setShow2FA(null)}>
                  {t('risk:cancel', 'Cancel')}
                </Button>
              </div>
            </div>
          )}

          {msg && (
            <p className={`text-sm mt-2 ${msg.includes('HALTED') || msg.includes('fail') ? 'text-trading-danger' : 'text-trading-success'}`}>
              {msg}
            </p>
          )}
        </CardContent>
      </Card>

      {/* Regime History */}
      {regimeHistory.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>{t('risk:regimeHistory', 'Regime History (last {{n}})', { n: regimeHistory.length })}</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('risk:table.time', 'Time')}</TableHead>
                  <TableHead>{t('risk:table.regime', 'Regime')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {regimeHistory.map((r, i) => (
                  <TableRow key={i}>
                    <TableCell>{new Date(r.timestamp).toLocaleString()}</TableCell>
                    <TableCell className={regimeColorClasses[r.regime] ?? 'text-muted-foreground'}>
                      {regimeLabels[r.regime] ?? r.regime}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Kill-Switch History */}
      {killHistory.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>{t('risk:killHistory', 'Kill-Switch History (last {{n}})', { n: killHistory.length })}</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('risk:table.time', 'Time')}</TableHead>
                  <TableHead>{t('risk:table.reason', 'Reason')}</TableHead>
                  <TableHead>{t('risk:table.source', 'Source')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {killHistory.map((k, i) => (
                  <TableRow key={i}>
                    <TableCell className="text-xs">{new Date(k.triggered_at).toLocaleString()}</TableCell>
                    <TableCell className="text-xs text-destructive">{k.reason}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{k.source}</TableCell>
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
