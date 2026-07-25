import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { attribution } from '../api/client'
import ErrorCard from '../components/ErrorCard'
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/table'
import type { AttributionReportResponse, AttributionSliceStats } from '../types/api'
import MetricCard from '../components/MetricCard'

export default function AttributionPage() {
  const { t } = useTranslation()
  const [report, setReport] = useState<AttributionReportResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [dimension, setDimension] = useState<'side' | 'price' | 'edge'>('side')

  const runAttribution = useCallback(async () => {
    setLoading(true)
    setError(null)
    setReport(null)
    try {
      const res = await attribution.run()
      if (res.error) {
        setError(res.error)
      } else {
        setReport(res)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : t('attribution:attributionFailed', 'Attribution failed'))
    } finally {
      setLoading(false)
    }
  }, [])

  const formatPct = (v: number | undefined | null, d = 2) =>
    v != null ? `${(v * 100).toFixed(d)}%` : '--'

  const formatUSD = (v: number | undefined | null) =>
    v != null ? `$${v.toFixed(2)}` : '--'

  type SliceEntry = [string, AttributionSliceStats]

  const currentSlices: SliceEntry[] = (() => {
    if (!report) return []
    if (dimension === 'side' && report.by_side) return Object.entries(report.by_side)
    if (dimension === 'price' && report.by_price_bucket) return Object.entries(report.by_price_bucket)
    if (dimension === 'edge' && report.by_edge_bucket) return Object.entries(report.by_edge_bucket)
    return []
  })()

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="m-0">{t('attribution:title', 'PnL Attribution')}</h1>
        <Button onClick={runAttribution} disabled={loading}>
          {loading ? t('attribution:runningAttribution', 'Running Attribution...') : t('attribution:runAttribution', 'Run Attribution')}
        </Button>
      </div>

      {error && <ErrorCard message={error} />}

      {!report && !loading && (
        <Card>
          <CardContent className="p-6">
            <CardDescription>
              {t('attribution:emptyDescription', 'Run multi-dimensional PnL attribution to analyze trading performance sliced by side, price bucket, and edge bucket with Wilson confidence intervals.')}
            </CardDescription>
          </CardContent>
        </Card>
      )}

      {loading && (
        <Card>
          <CardContent className="p-6">
            <CardDescription>{t('attribution:runningMessage', 'Running PnL attribution against trade ledger...')}</CardDescription>
          </CardContent>
        </Card>
      )}

      {report && !loading && (
        <div>
          <Card className="mb-4">
            <CardHeader>
              <CardTitle>{t('attribution:overallTab', 'Overall')}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
                <MetricCard label={t('attribution:totalTrades', 'Total Trades')} value={report.overall?.n ?? 0} format="number" />
                <MetricCard label={t('attribution:wins', 'Wins')} value={report.overall?.wins ?? 0} format="number" />
                <MetricCard label={t('attribution:winRate', 'Win Rate')} value={report.overall?.hit_rate ?? 0} format="percent" />
                <MetricCard label={t('attribution:hitRateCi', 'Hit Rate CI')} value={`${formatPct(report.overall?.hit_rate_ci_low)} – ${formatPct(report.overall?.hit_rate_ci_high)}`} />
                <MetricCard label={t('attribution:totalPnl', 'Total PnL')} value={report.overall?.total_pnl ?? 0} format="currency" color="auto" />
                <MetricCard label={t('attribution:totalCost', 'Total Cost')} value={report.overall?.total_cost ?? 0} format="currency" />
                <MetricCard label={t('attribution:roi', 'ROI')} value={report.overall?.roi ?? 0} format="percent" />
                <MetricCard label={t('attribution:avgWin', 'Avg Win')} value={report.overall ? report.overall.total_pnl / report.overall.n : 0} format="currency" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t('attribution:sliceDimensions', 'Slice Dimensions')}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex gap-2 mb-3">
                {(['side', 'price', 'edge'] as const).map((d) => {
                  const dimLabels: Record<string, string> = {
                    side: t('attribution:bySide', 'By Side'),
                    price: t('attribution:byPrice', 'By Price'),
                    edge: t('attribution:byEdge', 'By Edge'),
                  }
                  return (
                    <Button
                      key={d}
                      variant={dimension === d ? 'default' : 'outline'}
                      onClick={() => setDimension(d)}
                    >
                      {dimLabels[d]}
                    </Button>
                  )
                })}
              </div>

              {currentSlices.length === 0 ? (
                <p className="text-muted-foreground text-sm">{t('attribution:noSliceData', 'No {{dim}} slice data available.', { dim: dimension })}</p>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('attribution:table.slice', 'Slice')}</TableHead>
                      <TableHead>{t('attribution:table.trades', 'Trades')}</TableHead>
                      <TableHead>{t('attribution:wins', 'Wins')}</TableHead>
                      <TableHead>{t('attribution:table.winRate', 'Win Rate')}</TableHead>
                      <TableHead>{t('attribution:ciLow', 'CI Low')}</TableHead>
                      <TableHead>{t('attribution:ciHigh', 'CI High')}</TableHead>
                      <TableHead>{t('attribution:table.pnl', 'PnL')}</TableHead>
                      <TableHead>{t('attribution:totalCost', 'Cost')}</TableHead>
                      <TableHead>{t('attribution:roi', 'ROI')}</TableHead>
                      <TableHead>{t('attribution:avgWin', 'Avg Win')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {currentSlices.map(([key, s]) => (
                      <TableRow key={key}>
                        <TableCell className="font-bold">{key}</TableCell>
                        <TableCell>{s.n}</TableCell>
                        <TableCell>{s.wins}</TableCell>
                        <TableCell>{formatPct(s.hit_rate)}</TableCell>
                        <TableCell>{formatPct(s.hit_rate_ci_low)}</TableCell>
                        <TableCell>{formatPct(s.hit_rate_ci_high)}</TableCell>
                        <TableCell style={{ color: s.total_pnl >= 0 ? 'var(--trading-success)' : 'var(--trading-danger)' }}>
                          {formatUSD(s.total_pnl)}
                        </TableCell>
                        <TableCell>{formatUSD(s.total_cost)}</TableCell>
                        <TableCell>{formatPct(s.roi)}</TableCell>
                        <TableCell>{s.n > 0 ? formatUSD(s.total_pnl / s.n) : '--'}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>

          {report.generated_at && (
            <p className="text-muted-foreground text-sm mt-4">{t('attribution:generated', 'Generated: {{date}}', { date: new Date(report.generated_at).toLocaleString() })}</p>
          )}
        </div>
      )}
    </div>
  )
}
