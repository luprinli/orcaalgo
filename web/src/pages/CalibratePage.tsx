import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { calibrate } from '../api/client'
import { ErrorBanner } from '../components/layout'
import type { CalibrationReportResponse, CalibrationSegmentReport } from '../types/api'

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
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>{t('calibrate:title', 'Calibration Audit')}</h1>
        <button className="btn btn-primary" onClick={runCalibration} disabled={loading}>
          {loading ? t('calibrate:runningAudit', 'Running Audit...') : t('calibrate:runCalibration', 'Run Calibration')}
        </button>
      </div>

      {error && <ErrorBanner error={error} onDismiss={() => setError(null)} />}

      {!report && !loading && (
        <div className="card">
          <p className="text-muted">
            {t('calibrate:emptyDescription', 'Run a calibration audit to evaluate probability forecast quality using Brier score decomposition, Platt scaling recommendations, and per-segment reliability analysis.')}
          </p>
        </div>
      )}

      {loading && (
        <div className="card">
          <p className="text-muted">{t('calibrate:runningMessage', 'Running calibration audit against trade ledger...')}</p>
        </div>
      )}

      {report && !loading && (
        <div>
          <div className="grid-3 mb-4">
            <div className="metric-card">
              <div className="metric-label">{t('calibrate:overallBrier', 'Overall Brier')}</div>
              <div className="metric-value" style={{ color: (report.overall?.brier ?? 1) < 0.25 ? 'var(--success)' : (report.overall?.brier ?? 1) < 0.5 ? 'var(--warn)' : 'var(--danger)' }}>
                {formatNum(report.overall?.brier, 4)}
              </div>
            </div>
            <div className="metric-card">
              <div className="metric-label">{t('calibrate:reliability', 'Reliability')}</div>
              <div className="metric-value" style={{ color: (report.overall?.reliability ?? 1) < 0.01 ? 'var(--success)' : 'var(--warn)' }}>
                {formatNum(report.overall?.reliability, 4)}
              </div>
            </div>
            <div className="metric-card">
              <div className="metric-label">{t('calibrate:resolution', 'Resolution')}</div>
              <div className="metric-value">{formatNum(report.overall?.resolution, 4)}</div>
            </div>
            <div className="metric-card">
              <div className="metric-label">{t('calibrate:uncertainty', 'Uncertainty')}</div>
              <div className="metric-value">{formatNum(report.overall?.uncertainty, 4)}</div>
            </div>
            <div className="metric-card">
              <div className="metric-label">{t('calibrate:sampleSize', 'Sample Size')}</div>
              <div className="metric-value">{report.overall?.n ?? 0}</div>
            </div>
            <div className="metric-card">
              <div className="metric-label">{t('calibrate:needsCalibration', 'Needs Calibration')}</div>
              <div className="metric-value" style={{ color: report.overall?.needs_calibration ? 'var(--warn)' : 'var(--success)' }}>
                {report.overall?.needs_calibration ? t('common:yes', 'Yes') : t('common:no', 'No')}
              </div>
            </div>
          </div>

          {report.segments && Object.keys(report.segments).length > 0 && (
            <div className="card mb-4">
              <h2>{t('calibrate:segments', 'Segments')}</h2>
              <div className="flex gap-2 flex-wrap mb-3">
                <button
                  className={`btn ${activeSegment === 'overall' ? 'btn-primary' : 'btn-outline'}`}
                  onClick={() => setActiveSegment('overall')}
                >
                  {t('calibrate:overallTab', 'Overall')}
                </button>
                {Object.keys(report.segments).map((key) => (
                  <button
                    key={key}
                    className={`btn ${activeSegment === key ? 'btn-primary' : 'btn-outline'}`}
                    onClick={() => setActiveSegment(key)}
                  >
                    {report.segments[key]?.name ?? key}
                  </button>
                ))}
              </div>

              {segment && (
                <div>
                  <div className="metric-grid mb-3">
                    <div className="metric-card">
                      <div className="metric-label">{t('calibrate:brier', 'Brier')}</div>
                      <div className="metric-value">{formatNum(segment.brier, 4)}</div>
                    </div>
                    <div className="metric-card">
                      <div className="metric-label">{t('calibrate:reliability', 'Reliability')}</div>
                      <div className="metric-value">{formatNum(segment.reliability, 4)}</div>
                    </div>
                    <div className="metric-card">
                      <div className="metric-label">{t('calibrate:resolution', 'Resolution')}</div>
                      <div className="metric-value">{formatNum(segment.resolution, 4)}</div>
                    </div>
                    <div className="metric-card">
                      <div className="metric-label">{t('calibrate:uncertainty', 'Uncertainty')}</div>
                      <div className="metric-value">{formatNum(segment.uncertainty, 4)}</div>
                    </div>
                    <div className="metric-card">
                      <div className="metric-label">{t('calibrate:sampleSize', 'Sample Size')}</div>
                      <div className="metric-value">{segment.n}</div>
                    </div>
                    <div className="metric-card">
                      <div className="metric-label">{t('calibrate:needsCal', 'Needs Cal.')}</div>
                      <div className="metric-value" style={{ color: segment.needs_calibration ? 'var(--warn)' : 'var(--success)' }}>
                        {segment.needs_calibration ? t('common:yes', 'Yes') : t('common:no', 'No')}
                      </div>
                    </div>
                  </div>

                  {segment.bin_stats && segment.bin_stats.length > 0 && (
                    <div>
                      <span className="text-muted">{t('calibrate:calibrationBins', 'Calibration bins (decile):')}</span>
                      <div style={{ overflowX: 'auto', marginTop: 8 }}>
                        <table className="data-table">
                          <thead>
                            <tr>
                              <th>{t('calibrate:table.bin', 'Bin')}</th>
                              <th>{t('calibrate:table.count', 'Count')}</th>
                              <th>{t('calibrate:table.meanPred', 'Mean Pred.')}</th>
                              <th>{t('calibrate:table.hitRate', 'Hit Rate')}</th>
                              <th>{t('common:error', 'Error')}</th>
                            </tr>
                          </thead>
                          <tbody>
                            {segment.bin_stats.map((bin, i) => {
                              const error = bin.hit_rate - bin.mean_prediction
                              return (
                                <tr key={i}>
                                  <td>{formatPct(bin.bin_start)} — {formatPct(bin.bin_end)}</td>
                                  <td>{bin.count}</td>
                                  <td>{formatPct(bin.mean_prediction)}</td>
                                  <td>{formatPct(bin.hit_rate)}</td>
                                  <td style={{ color: Math.abs(error) > 0.05 ? 'var(--warn)' : 'var(--success)' }}>
                                    {error >= 0 ? '+' : ''}{formatPct(error)}
                                  </td>
                                </tr>
                              )
                            })}
                          </tbody>
                        </table>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          {report.generated_at && (
            <p className="text-muted">{t('calibrate:generated', 'Generated: {{date}}', { date: new Date(report.generated_at).toLocaleString() })}</p>
          )}
        </div>
      )}
    </div>
  )
}
