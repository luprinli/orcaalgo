import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { backtests } from '../api/client'
import ErrorCard from '../components/ErrorCard'
import ConfirmDialog from '../components/ConfirmDialog'
import EquityCurveChart from '../charts/EquityCurveChart'
import { formatNumber, formatPctRaw } from '../lib/format'
import { TableSkeleton } from '../components/SkeletonLoader'
import type { BacktestHistoryEntry, BacktestMetrics, EquityPoint } from '../types/api'

interface EntryWithMetrics extends BacktestHistoryEntry {
  _metrics?: BacktestMetrics
  _metricsLoading?: boolean
}

export default function BacktestHistory() {
  const nav = useNavigate()
  const [list, setList] = useState<EntryWithMetrics[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [limit] = useState(50)
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)
  const [compareMode, setCompareMode] = useState(false)
  const [selectedForCompare, setSelectedForCompare] = useState<Set<string>>(new Set())
  const [compareEquity, setCompareEquity] = useState<Record<string, EquityPoint[]>>({})
  const [compareLoading, setCompareLoading] = useState(false)

  const toggleCompareSelect = (id: string) => {
    setSelectedForCompare(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const runComparison = useCallback(async () => {
    if (selectedForCompare.size < 2) return
    setCompareLoading(true)
    const equityMap: Record<string, EquityPoint[]> = {}
    for (const id of selectedForCompare) {
      try {
        const e = await backtests.equity(id)
        /* eslint-disable @typescript-eslint/no-explicit-any */
        equityMap[id] = Array.isArray(e) ? e.map(p => ({
          time: (typeof (p as any).timestamp === 'string' ? (p as any).timestamp : p.time) ?? '',
          value: (typeof (p as any).equity === 'number' ? (p as any).equity : p.value) ?? 0,
          regime: (typeof (p as any).regime_label === 'number' ? (p as any).regime_label : p.regime) ?? 0,
        })) : []
        /* eslint-enable @typescript-eslint/no-explicit-any */
      } catch { /* skip */ }
    }
    setCompareEquity(equityMap)
    setCompareLoading(false)
  }, [selectedForCompare])

  const compareColors = ['#2962FF', '#3fb950', '#d29922', '#f85149', '#58a6ff', '#bc8cff']

  const clearComparison = () => {
    setCompareEquity({})
    setCompareMode(false)
    setSelectedForCompare(new Set())
  }

  const fetchList = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await backtests.list({ limit })
      const runs: EntryWithMetrics[] = (res.runs ?? []).map(r => ({
        ...r,
        _metricsLoading: r.status === 'completed',
      }))
      setList(runs)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load history')
    } finally {
      setLoading(false)
    }
  }, [limit])

  useEffect(() => {
    fetchList()
  }, [fetchList])

  useEffect(() => {
    for (const entry of list) {
      if (!entry._metricsLoading || entry._metrics) continue
      backtests.metrics(entry.id)
        .then(m => {
          setList(prev => prev.map(e =>
            e.id === entry.id ? { ...e, _metrics: m, _metricsLoading: false } : e,
          ))
        })
        .catch(() => {
          setList(prev => prev.map(e =>
            e.id === entry.id ? { ...e, _metricsLoading: false } : e,
          ))
        })
    }
  }, [list])

  const handleDelete = async (id: string) => {
    setConfirmDelete(id)
  }

  const confirmDeleteRun = async () => {
    if (!confirmDelete) return
    try {
      await backtests.delete(confirmDelete)
      setConfirmDelete(null)
      fetchList()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Delete failed')
      setConfirmDelete(null)
    }
  }

  const handleRerun = async (id: string) => {
    try {
      const res = await backtests.rerun(id)
      nav(`/backtest/history/${res.run_id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Rerun failed')
    }
  }

  if (loading && list.length === 0) {
    return (
      <div className="card">
        <h2>Backtest History</h2>
        <TableSkeleton rows={6} cols={8} />
      </div>
    )
  }

  return (
    <div>
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>Backtest History</h1>
        <div className="flex gap-2">
          {compareMode ? (
            <>
              <button className="btn btn-outline" onClick={clearComparison}>
                Cancel Compare
              </button>
              <button
                className="btn btn-primary"
                onClick={runComparison}
                disabled={selectedForCompare.size < 2 || compareLoading}
              >
                {compareLoading ? 'Loading...' : `Compare (${selectedForCompare.size})`}
              </button>
            </>
          ) : (
            <button className="btn btn-outline" onClick={() => setCompareMode(true)}>
              Compare
            </button>
          )}
          <button className="btn btn-outline" onClick={fetchList}>
            Refresh
          </button>
        </div>
      </div>

      {error && <ErrorCard message={error} onRetry={fetchList} />}

      {list.length === 0 ? (
        <div className="card">
          <p className="text-muted">No backtest runs yet. Run a backtest to see results here.</p>
        </div>
      ) : (
        <div className="card">
          <div style={{ overflowX: 'auto' }}>
              <table className="data-table">
                <thead>
                  <tr>
                    {compareMode && <th style={{ width: 32 }}></th>}
                    <th>ID</th>
                    <th>Type</th>
                    <th>Strategies</th>
                    <th>Symbols</th>
                    <th>Sharpe</th>
                  <th>Max DD</th>
                  <th>Win Rate</th>
                  <th>Trades</th>
                  <th>Return</th>
                  <th>Started</th>
                  <th>Status</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                  {list.map((bt) => {
                  const m = bt._metrics
                  const isSelected = selectedForCompare.has(bt.id)
                  return (
                    <tr key={bt.id} style={{ cursor: 'pointer', background: isSelected ? 'rgba(63,185,80,.06)' : undefined }} onClick={() => compareMode ? toggleCompareSelect(bt.id) : nav(`/backtest/history/${bt.id}`)}>
                      {compareMode && (
                        <td onClick={e => e.stopPropagation()}>
                          <input type="checkbox" checked={isSelected} onChange={() => toggleCompareSelect(bt.id)} aria-label={`Select ${bt.id} for comparison`} />
                        </td>
                      )}
                      <td style={{ fontFamily: 'monospace', fontSize: 11 }}>{bt.id?.slice(0, 12)}</td>
                      <td>{bt.run_type}</td>
                      <td>{bt.strategy_ids?.join(', ') || '—'}</td>
                      <td>{bt.symbols?.join(', ') || '—'}</td>
                      <td style={{ color: m ? (m.sharpe_ratio >= 1 ? 'var(--success)' : m.sharpe_ratio >= 0 ? 'var(--warn)' : 'var(--danger)') : undefined }}>
                        {m ? formatNumber(m.sharpe_ratio, 2) : bt._metricsLoading ? '...' : bt.status === 'running' ? '—' : 'N/A'}
                      </td>
                      <td>{m ? formatPctRaw(m.max_drawdown_pct, 1) : '—'}</td>
                      <td>{m ? formatPctRaw(m.win_rate_pct, 1) : '—'}</td>
                      <td>{m ? m.num_trades : '—'}</td>
                      <td style={{ color: m ? (m.total_return_pct >= 0 ? 'var(--success)' : 'var(--danger)') : undefined }}>
                        {m ? formatPctRaw(m.total_return_pct, 1) : '—'}
                      </td>
                      <td>{bt.started_at ? new Date(bt.started_at).toLocaleString() : '—'}</td>
                      <td>
                        <span className={`badge ${bt.status === 'completed' ? 'badge-ok' : bt.status === 'running' ? 'badge-warn' : 'badge-err'}`}>
                          {bt.status || '—'}
                        </span>
                      </td>
                      <td>
                        <div className="flex gap-1">
                          <button
                            className="btn btn-outline"
                            style={{ padding: '2px 8px', fontSize: 11 }}
                            onClick={(e) => { e.stopPropagation(); handleRerun(bt.id) }}
                          >
                            Rerun
                          </button>
                          <button
                            className="btn btn-outline"
                            style={{ padding: '2px 8px', fontSize: 11, color: 'var(--danger)' }}
                            onClick={(e) => { e.stopPropagation(); handleDelete(bt.id) }}
                          >
                            Delete
                          </button>
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {compareEquity && Object.keys(compareEquity).length >= 2 && (
        <div className="card mb-4">
          <div className="flex-between mb-2">
            <h2 style={{ margin: 0 }}>Comparison</h2>
            <button className="btn btn-outline" style={{ fontSize: 11, padding: '2px 8px' }} onClick={clearComparison}>
              Close
            </button>
          </div>

          <div style={{ overflowX: 'auto', marginBottom: 16 }}>
            <table className="data-table" style={{ fontSize: 12 }}>
              <thead>
                <tr>
                  <th>Metric</th>
                  {Object.keys(compareEquity).map((id, idx) => {
                    const entry = list.find(l => l.id === id)
                    return (
                      <th key={id} style={{ color: compareColors[idx % compareColors.length] }}>
                        {entry?.strategy_ids?.[0] ?? id?.slice(0, 8)}
                      </th>
                    )
                  })}
                </tr>
              </thead>
              <tbody>
                {( [
                  ['Sharpe', (m: BacktestMetrics | undefined) => m ? formatNumber(m.sharpe_ratio, 2) : '—'],
                  ['Sortino', (m: BacktestMetrics | undefined) => m ? formatNumber(m.sortino_ratio, 2) : '—'],
                  ['Max DD', (m: BacktestMetrics | undefined) => m ? formatPctRaw(m.max_drawdown_pct, 1) : '—'],
                  ['Win Rate', (m: BacktestMetrics | undefined) => m ? formatPctRaw(m.win_rate_pct, 1) : '—'],
                  ['Profit Factor', (m: BacktestMetrics | undefined) => m ? formatNumber(m.profit_factor, 2) : '—'],
                  ['Total Return', (m: BacktestMetrics | undefined) => m ? formatPctRaw(m.total_return_pct, 1) : '—'],
                  ['Trades', (m: BacktestMetrics | undefined) => m ? String(m.num_trades) : '—'],
                  ['CAGR', (m: BacktestMetrics | undefined) => m ? formatPctRaw(m.cagr, 1) : '—'],
                  ['Calmar', (m: BacktestMetrics | undefined) => m ? formatNumber(m.calmar, 2) : '—'],
                  ['VaR 95%', (m: BacktestMetrics | undefined) => m ? formatPctRaw(m.var_95, 1) : '—'],
                  ['Pass Prob', (m: BacktestMetrics | undefined) => m ? `${m.pass_probability?.toFixed(0)}%` : '—'],
                ] as Array<[string, (m: BacktestMetrics | undefined) => string]> ).map(([label, fmt]) => (
                  <tr key={label}>
                    <td style={{ fontWeight: 600 }}>{label}</td>
                    {Object.keys(compareEquity).map(id => {
                      const entry = list.find(l => l.id === id)
                      return <td key={id}>{fmt(entry?._metrics)}</td>
                    })}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {Object.entries(compareEquity).length >= 2 && (() => {
            const entries = Object.entries(compareEquity)
            const primary = entries[0]
            const primaryEntry = list.find(l => l.id === primary[0])
            const overlayData = entries.slice(1).map(([id, points], idx) => {
              const entry = list.find(l => l.id === id)
              return {
                data: points,
                label: entry?.strategy_ids?.[0] ?? id?.slice(0, 8),
                color: compareColors[(idx + 1) % compareColors.length],
              }
            })
            return (
              <EquityCurveChart
                data={primary[1]}
                height={350}
                title={`${primaryEntry?.strategy_ids?.[0] ?? primary[0]?.slice(0, 8)} vs ${entries.length - 1} others`}
                color={compareColors[0]}
                overlays={overlayData}
              />
            )
          })()}
        </div>
      )}

      {confirmDelete && (
        <ConfirmDialog
          title="Delete Backtest"
          message="Delete this backtest run? This action cannot be undone."
          confirmLabel="Delete"
          danger
          onConfirm={confirmDeleteRun}
          onCancel={() => setConfirmDelete(null)}
        />
      )}
    </div>
  )
}
