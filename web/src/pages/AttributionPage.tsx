import { useState, useCallback } from 'react'
import { attribution } from '../api/client'
import ErrorCard from '../components/ErrorCard'
import type { AttributionReportResponse, AttributionSliceStats } from '../types/api'

export default function AttributionPage() {
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
      setError(err instanceof Error ? err.message : 'Attribution failed')
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
        <h1 style={{ margin: 0 }}>PnL Attribution</h1>
        <button className="btn btn-primary" onClick={runAttribution} disabled={loading}>
          {loading ? 'Running Attribution...' : 'Run Attribution'}
        </button>
      </div>

      {error && <ErrorCard message={error} />}

      {!report && !loading && (
        <div className="card">
          <p className="text-muted">
            Run multi-dimensional PnL attribution to analyze trading performance sliced by side,
            price bucket, and edge bucket with Wilson confidence intervals.
          </p>
        </div>
      )}

      {loading && (
        <div className="card">
          <p className="text-muted">Running PnL attribution against trade ledger...</p>
        </div>
      )}

      {report && !loading && (
        <div>
          <div className="card mb-4">
            <h2>Overall</h2>
            <div className="metric-grid">
              <div className="metric-card">
                <div className="metric-label">Trades</div>
                <div className="metric-value">{report.overall?.n ?? 0}</div>
              </div>
              <div className="metric-card">
                <div className="metric-label">Wins</div>
                <div className="metric-value">{report.overall?.wins ?? 0}</div>
              </div>
              <div className="metric-card">
                <div className="metric-label">Hit Rate</div>
                <div className="metric-value">{formatPct(report.overall?.hit_rate)}</div>
              </div>
              <div className="metric-card">
                <div className="metric-label">Hit Rate CI</div>
                <div className="metric-value" style={{ fontSize: 16 }}>
                  {formatPct(report.overall?.hit_rate_ci_low)} – {formatPct(report.overall?.hit_rate_ci_high)}
                </div>
              </div>
              <div className="metric-card">
                <div className="metric-label">Total PnL</div>
                <div className="metric-value" style={{ color: (report.overall?.total_pnl ?? 0) >= 0 ? 'var(--success)' : 'var(--danger)' }}>
                  {formatUSD(report.overall?.total_pnl)}
                </div>
              </div>
              <div className="metric-card">
                <div className="metric-label">Total Cost</div>
                <div className="metric-value">{formatUSD(report.overall?.total_cost)}</div>
              </div>
              <div className="metric-card">
                <div className="metric-label">ROI</div>
                <div className="metric-value">{formatPct(report.overall?.roi)}</div>
              </div>
              <div className="metric-card">
                <div className="metric-label">Avg Win</div>
                <div className="metric-value">
                  {report.overall ? formatUSD(report.overall.total_pnl / report.overall.n) : '--'}
                </div>
              </div>
            </div>
          </div>

          <div className="card">
            <h2>Slice Dimensions</h2>
            <div className="flex gap-2 mb-3">
              {(['side', 'price', 'edge'] as const).map((d) => (
                <button
                  key={d}
                  className={`btn ${dimension === d ? 'btn-primary' : 'btn-outline'}`}
                  onClick={() => setDimension(d)}
                >
                  By {d.charAt(0).toUpperCase() + d.slice(1)}
                </button>
              ))}
            </div>

            {currentSlices.length === 0 ? (
              <p className="text-muted">No {dimension} slice data available.</p>
            ) : (
              <div style={{ overflowX: 'auto' }}>
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>Slice</th>
                      <th>Trades</th>
                      <th>Wins</th>
                      <th>Hit Rate</th>
                      <th>CI Low</th>
                      <th>CI High</th>
                      <th>Total PnL</th>
                      <th>Cost</th>
                      <th>ROI</th>
                      <th>Avg Win</th>
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
            <p className="text-muted mt-4">Generated: {new Date(report.generated_at).toLocaleString()}</p>
          )}
        </div>
      )}
    </div>
  )
}
