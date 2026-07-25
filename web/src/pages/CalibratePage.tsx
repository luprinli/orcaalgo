import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { calibrate } from '../api/client'
import { ErrorBanner } from '../components/layout'
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/table'
import type { CalibrationReportResponse, CalibrationSegmentReport } from '../types/api'
import MetricCard from '../components/MetricCard'

export default function CalibratePage() {
  const { t } = useTranslation()
  const [report, setReport] = useState<CalibrationReportResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [activeSegment, setActiveSegment] = useState<string>('overall')

  const runCalibration = useCallback(async () => {
    setLoading(true)
    setError(null)
    setReport(null)
    try {
      const res = await calibrate.run()
      if (res.error) {
        setError(res.error)
      } else {
        setReport(res)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : t('calibrate:calibrationFailed', 'Calibration failed'))
    } finally {
      setLoading(false)
    }
  }, [])

  const segment: CalibrationSegmentReport | null = (() => {
    if (!report) return null
    if (activeSegment === 'overall') return report.overall
    return report.segments?.[activeSegment] ?? null
  })()

  const formatPct = (v: number | undefined | null) =>
    v != null ? `${(v * 100).toFixed(2)}%` : '--'

  const formatNum = (v: number | undefined | null, d = 4) =>
    v != null ? v.toFixed(d) : '--'

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="m-0">{t('calibrate:title', 'Calibration Audit')}</h1>
        <Button onClick={runCalibration} disabled={loading}>
          {loading ? t('calibrate:runningAudit', 'Running Audit...') : t('calibrate:runCalibration', 'Run Calibration')}
        </Button>
      </div>

      {error && <ErrorBanner error={error} onDismiss={() => setError(null)} />}

      {!report && !loading && (
        <Card>
          <CardContent className="p-6">
            <CardDescription>
              {t('calibrate:emptyDescription', 'Run a calibration audit to evaluate probability forecast quality using Brier score decomposition, Platt scaling recommendations, and per-segment reliability analysis.')}
            </CardDescription>
          </CardContent>
        </Card>
      )}

      {loading && (
        <Card>
          <CardContent className="p-6">
            <CardDescription>{t('calibrate:runningMessage', 'Running calibration audit against trade ledger...')}</CardDescription>
          </CardContent>
        </Card>
      )}

      {report && !loading && (
        <div>
          <div className="grid grid-cols-3 gap-3 mb-4">
            <MetricCard label={t('calibrate:overallBrier', 'Overall Brier')} value={report.overall?.brier ?? 0} format="decimal" color="auto" />
            <MetricCard label={t('calibrate:reliability', 'Reliability')} value={report.overall?.reliability ?? 0} format="decimal" color="auto" />
            <MetricCard label={t('calibrate:resolution', 'Resolution')} value={report.overall?.resolution ?? 0} format="decimal" />
            <MetricCard label={t('calibrate:uncertainty', 'Uncertainty')} value={report.overall?.uncertainty ?? 0} format="decimal" />
            <MetricCard label={t('calibrate:sampleSize', 'Sample Size')} value={report.overall?.n ?? 0} format="number" />
            <MetricCard label={t('calibrate:needsCalibration', 'Needs Calibration')} value={report.overall?.needs_calibration ? t('common:yes', 'Yes') : t('common:no', 'No')} color={report.overall?.needs_calibration ? 'negative' : 'positive'} />
          </div>

          {report.segments && Object.keys(report.segments).length > 0 && (
            <Card className="mb-4">
              <CardHeader>
                <CardTitle>{t('calibrate:segments', 'Segments')}</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="flex gap-2 flex-wrap mb-3">
                  <Button
                    variant={activeSegment === 'overall' ? 'default' : 'outline'}
                    onClick={() => setActiveSegment('overall')}
                  >
                    {t('calibrate:overallTab', 'Overall')}
                  </Button>
                  {Object.keys(report.segments).map((key) => (
                    <Button
                      key={key}
                      variant={activeSegment === key ? 'default' : 'outline'}
                      onClick={() => setActiveSegment(key)}
                    >
                      {report.segments[key]?.name ?? key}
                    </Button>
                  ))}
                </div>

                {segment && (
                  <div>
                    <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
                      <MetricCard label={t('calibrate:brier', 'Brier')} value={segment.brier ?? 0} format="decimal" color="auto" />
                      <MetricCard label={t('calibrate:reliability', 'Reliability')} value={segment.reliability ?? 0} format="decimal" />
                      <MetricCard label={t('calibrate:resolution', 'Resolution')} value={segment.resolution ?? 0} format="decimal" />
                      <MetricCard label={t('calibrate:uncertainty', 'Uncertainty')} value={segment.uncertainty ?? 0} format="decimal" />
                      <MetricCard label={t('calibrate:sampleSize', 'Sample Size')} value={segment.n ?? 0} format="number" />
                      <MetricCard label={t('calibrate:needsCal', 'Needs Cal.')} value={segment.needs_calibration ? t('common:yes', 'Yes') : t('common:no', 'No')} color={segment.needs_calibration ? 'negative' : 'positive'} />
                    </div>

                    {segment.bin_stats && segment.bin_stats.length > 0 && (
                      <div>
                        <span className="text-muted-foreground text-sm">{t('calibrate:calibrationBins', 'Calibration bins (decile):')}</span>
                        <div className="mt-2">
                          <Table>
                            <TableHeader>
                              <TableRow>
                                <TableHead>{t('calibrate:table.bin', 'Bin')}</TableHead>
                                <TableHead>{t('calibrate:table.count', 'Count')}</TableHead>
                                <TableHead>{t('calibrate:table.meanPred', 'Mean Pred.')}</TableHead>
                                <TableHead>{t('calibrate:table.hitRate', 'Hit Rate')}</TableHead>
                                <TableHead>{t('common:error', 'Error')}</TableHead>
                              </TableRow>
                            </TableHeader>
                            <TableBody>
                              {segment.bin_stats.map((bin, i) => {
                                const error = bin.hit_rate - bin.mean_prediction
                                return (
                                  <TableRow key={i}>
                                    <TableCell>{formatPct(bin.bin_start)} — {formatPct(bin.bin_end)}</TableCell>
                                    <TableCell>{bin.count}</TableCell>
                                    <TableCell>{formatPct(bin.mean_prediction)}</TableCell>
                                    <TableCell>{formatPct(bin.hit_rate)}</TableCell>
                                    <TableCell style={{ color: Math.abs(error) > 0.05 ? 'var(--trading-warning)' : 'var(--trading-success)' }}>
                                      {error >= 0 ? '+' : ''}{formatPct(error)}
                                    </TableCell>
                                  </TableRow>
                                )
                              })}
                            </TableBody>
                          </Table>
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {report.generated_at && (
            <p className="text-muted-foreground text-sm">{t('calibrate:generated', 'Generated: {{date}}', { date: new Date(report.generated_at).toLocaleString() })}</p>
          )}
        </div>
      )}
    </div>
  )
}
