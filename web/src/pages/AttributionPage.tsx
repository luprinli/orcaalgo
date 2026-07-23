import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { attribution } from '../api/client'
import ErrorCard from '../components/ErrorCard'
import type { AttributionReportResponse, AttributionSliceStats } from '../types/api'

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
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>{t('attribution:title', 'PnL Attribution')}</h1>
        <button className="btn btn-primary" onClick={runAttribution} disabled={loading}>
          {loading ? t('attribution:runningAttribution', 'Running Attribution...') : t('attribution:runAttribution', 'Run Attribution')}
        </button>
      </div>

      {error && <ErrorCard message={error} />}

      {!report && !loading && (
        <div className="card">
          <p className="text-muted">
            {t('attribution:emptyDescription', 'Run multi-dimensional PnL attribution to analyze trading performance sliced by side, price bucket, and edge bucket with Wilson confidence intervals.')}
          </p>
        </div>
      )}

      {loading && (
        <div className="card">
          <p className="text-muted">{t('attribution:runningMessage', 'Running PnL attribution against trade ledger...')}</p>
        </div>
      )}

      {report && !loading && (
        <div>
          <div className="card mb-4">
            <h2>{t('attribution:overallTab', 'Overall')}</h2>
            <div className="metric-grid">
              <div className="metric-card">
                <div className="metric-label">{t('attribution:totalTrades', 'Total Trades')}</div>
                <div className="metric-value">{report.overall?.n ?? 0}</div>
              </div>
              <div className="metric-card">
                <div className="metric-label">{t('attribution:wins', 'Wins')}</div>
                <div className="metric-value">{report.overall?.wins ?? 0}</div>
              </div>
              <div className="metric-card">
                <div className="metric-label">{t('attribution:winRate', 'Win Rate')}</div>
                <div className="metric-value">{formatPct(report.overall?.hit_rate)}</div>
              </div>
              <div className="metric-card">
                <div className="metric-label">{t('attribution:hitRateCi', 'Hit Rate CI')}</div>
                <div className="metric-value" style={{ fontSize: 16 }}>
                  {formatPct(report.overall?.hit_rate_ci_low)} – {formatPct(report.overall?.hit_rate_ci_high)}
                </div>
              </div>
              <div className="metric-card">
                <div className="metric-label">{t('attribution:totalPnl', 'Total PnL')}</div>
                <div className="metric-value" style={{ color: (report.overall?.total_pnl ?? 0) >= 0 ? 'var(--success)' : 'var(--danger)' }}>
                  {formatUSD(report.overall?.total_pnl)}
                </div>
              </div>
              <div className="metric-card">
                <div className="metric-label">{t('attribution:totalCost', 'Total Cost')}</div>
                <div className="metric-value">{formatUSD(report.overall?.total_cost)}</div>
              </div>
              <div className="metric-card">
                <div className="metric-label">{t('attribution:roi', 'ROI')}</div>
                <div className="metric-value">{formatPct(report.overall?.roi)}</div>
              </div>
              <div className="metric-card">
                <div className="metric-label">{t('attribution:avgWin', 'Avg Win')}</div>
                <div className="metric-value">
                  {report.overall ? formatUSD(report.overall.total_pnl / report.overall.n) : '--'}
                </div>
              </div>
            </div>
          </div>

          <div className="card">
            <h2>{t('attribution:sliceDimensions', 'Slice Dimensions')}</h2>
            <div className="flex gap-2 mb-3">
              {(['side', 'price', 'edge'] as const).map((d) => {
                const dimLabels: Record<string, string> = {
                  side: t('attribution:bySide', 'By Side'),
                  price: t('attribution:byPrice', 'By Price'),
                  edge: t('attribution:byEdge', 'By Edge'),
                }
                return (
                  <button
                    key={d}
                    className={`btn ${dimension === d ? 'btn-primary' : 'btn-outline'}`}
                    onClick={() => setDimension(d)}
                  >
                    {dimLabels[d]}
                  </button>
                )
              })}
            </div>

            {currentSlices.length === 0 ? (
              <p className="text-muted">{t('attribution:noSliceData', 'No {{dim}} slice data available.', { dim: dimension })}</p>
            ) : (
              <div style={{ overflowX: 'auto' }}>
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>{t('attribution:table.slice', 'Slice')}</th>
                      <th>{t('attribution:table.trades', 'Trades')}</th>
                      <th>{t('attribution:wins', 'Wins')}</th>
                      <th>{t('attribution:table.winRate', 'Win Rate')}</th>
                      <th>{t('attribution:ciLow', 'CI Low')}</th>
                      <th>{t('attribution:ciHigh', 'CI High')}</th>
                      <th>{t('attribution:table.pnl', 'PnL')}</th>
                      <th>{t('attribution:totalCost', 'Cost')}</th>
                      <th>{t('attribution:roi', 'ROI')}</th>
                      <th>{t('attribution:avgWin', 'Avg Win')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {currentSlices.map(([key, s]) => (
                      <tr key={key}>
                        <td><strong>{key}</strong></td>
                        <td>{s.n}</td>
                        <td>{s.wins}</td>
                        <td>{formatPct(s.hit_rate)}</td>
                        <td>{formatPct(s.hit_rate_ci_low)}</td>
                        <td>{formatPct(s.hit_rate_ci_high)}</td>
                        <td style={{ color: s.total_pnl >= 0 ? 'var(--success)' : 'var(--danger)' }}>
                          {formatUSD(s.total_pnl)}
                        </td>
                        <td>{formatUSD(s.total_cost)}</td>
                        <td>{formatPct(s.roi)}</td>
                        <td>{s.n > 0 ? formatUSD(s.total_pnl / s.n) : '--'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {report.generated_at && (
            <p className="text-muted mt-4">{t('attribution:generated', 'Generated: {{date}}', { date: new Date(report.generated_at).toLocaleString() })}</p>
          )}
        </div>
      )}
    </div>
  )
}
