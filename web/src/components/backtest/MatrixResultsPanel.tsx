import { useParameterSensitivity } from '../../hooks/useParameterSensitivity'
import { useWindowedRows } from '../../hooks/useWindowedRows'
import type { MatrixResultsResponse, ComboResult } from '../../types/api'
import MetricCard from '../../components/MetricCard'
import { exportMatrixResultsCSV } from '../../lib/export'
import toast from 'react-hot-toast'

const ROW_HEIGHT = 22
const TABLE_VIEWPORT = 400

type SortField = 'sharpe' | 'sortino' | 'max_dd' | 'return' | 'win_rate' | 'profit_factor' | 'trades'

export interface MatrixResultsPanelProps {
  matrixResult: MatrixResultsResponse
  matrixBatchId: string | null
  progressPct: number
  filterStrategy: string
  filterSymbol: string
  filterTf: string
  sortedMatrixResults: ComboResult[]
  filterStrats: string[]
  filterSyms: string[]
  filterTfs: string[]
  onFilterStrategyChange: (v: string) => void
  onFilterSymbolChange: (v: string) => void
  onFilterTfChange: (v: string) => void
  onClearFilters: () => void
  onSortToggle: (field: SortField) => void
  sortIndicator: (field: SortField) => string
  onViewDetail?: (comboKey: string) => void
}

export default function MatrixResultsPanel(props: MatrixResultsPanelProps) {
  const {
    matrixResult, matrixBatchId, progressPct,
    filterStrategy, filterSymbol, filterTf,
    sortedMatrixResults, filterStrats, filterSyms, filterTfs,
    onFilterStrategyChange, onFilterSymbolChange, onFilterTfChange,
    onClearFilters, onSortToggle, sortIndicator, onViewDetail,
  } = props

  const sensitivity = useParameterSensitivity(matrixResult)
  const { entries, colorFor, minS, maxS } = sensitivity
  const win = useWindowedRows(sortedMatrixResults.length, ROW_HEIGHT, TABLE_VIEWPORT)
  const range = maxS - minS || 1

  return (
    <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4 mb-4">
      <div className="flex-between mb-3"><h2 style={{ margin: 0 }}>Matrix Results {matrixBatchId && <span className="text-muted" style={{ fontSize: 13, marginLeft: 8 }}>(streaming {progressPct}%)</span>}</h2>
        <div className="flex gap-2">{matrixResult.results.length > 0 && <button className="btn btn-outline" style={{ fontSize: 11, padding: '2px 8px' }} onClick={() => { exportMatrixResultsCSV(sortedMatrixResults); toast.success(`Exported ${sortedMatrixResults.length} results`) }}>CSV</button>}<span className={`badge ${matrixResult.status === 'completed' ? 'badge-ok' : 'badge-warn'}`}>{matrixResult.status}</span></div>
      </div>
      <div className="metric-grid mb-3">
        <MetricCard label="Combos" value={matrixResult.summary.total_combos} format="number" />
        <MetricCard label="Passed" value={`${matrixResult.summary.passed}/${matrixResult.summary.total_combos}`} color={matrixResult.summary.passed > 0 ? 'positive' : 'negative'} />
        <MetricCard label="Trades" value={matrixResult.summary.total_trades} format="number" />
        <MetricCard label="Best Sharpe" value={matrixResult.summary.best_sharpe?.toFixed(3) ?? '--'} />
        <MetricCard label="Best Strat" value={matrixResult.summary.best_strategy ?? '--'} />
        <MetricCard label="Best Sym" value={matrixResult.summary.best_symbol ?? '--'} />
      </div>
      {matrixResult.results.length > 0 && (<>
        {entries.length > 1 && (
          <div style={{ marginBottom: 12 }}>
            <h3 style={{ fontSize: 13, margin: '0 0 6px', color: 'var(--muted-foreground)' }}>Parameter Sensitivity</h3>
            <div style={{ overflowX: 'auto' }}>
              <table className="data-table" style={{ fontSize: 10 }}>
                <thead>
                  <tr>
                    <th>Strategy / Sym / TF</th>
                    <th>Best Params</th>
                    <th>Sharpe</th>
                  </tr>
                </thead>
                <tbody>
                  {entries.map(e => {
                    const best = e.bestSharpe
                    return (
                      <tr key={e.key}>
                        <td style={{ fontFamily: 'monospace', fontSize: 10 }}>{e.key}</td>
              <td style={{ fontFamily: 'monospace', fontSize: 9, maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={e.bestParams && Object.keys(e.bestParams).length > 0 ? JSON.stringify(e.bestParams) : undefined}>
                {e.bestParams && Object.keys(e.bestParams).length > 0 ? JSON.stringify(e.bestParams) : '\u2014'}
                        </td>
                        <td>
                          <span style={{
                            display: 'inline-block',
                            background: colorFor(best),
                            color: best > (minS + range * 0.5) ? '#fff' : '#111',
                            padding: '2px 8px',
                            borderRadius: 4,
                            fontWeight: 600,
                          }}>
                            {best.toFixed(3)}
                          </span>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </div>
        )}
        <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4" style={{ padding: '6px 10px', marginBottom: 8, background: 'var(--muted)' }}>
          <div className="flex gap-2 flex-wrap" style={{ alignItems: 'center' }}>
            <span className="text-muted" style={{ fontSize: 10 }}>Filter:</span>
            <select className="input" style={{ fontSize: 10, padding: '3px 6px', width: 170, height: 26 }} value={filterStrategy} onChange={e => onFilterStrategyChange(e.target.value)}><option value="">Strategies ({filterStrats.length})</option>{filterStrats.map(s => <option key={s} value={s}>{s}</option>)}</select>
            <select className="input" style={{ fontSize: 10, padding: '3px 6px', width: 140, height: 26 }} value={filterSymbol} onChange={e => onFilterSymbolChange(e.target.value)}><option value="">Symbols ({filterSyms.length})</option>{filterSyms.map(s => <option key={s} value={s}>{s}</option>)}</select>
            <select className="input" style={{ fontSize: 10, padding: '3px 6px', width: 140, height: 26 }} value={filterTf} onChange={e => onFilterTfChange(e.target.value)}><option value="">TFs ({filterTfs.length})</option>{filterTfs.map(t => <option key={t} value={t}>{t}</option>)}</select>
            {(filterStrategy || filterSymbol || filterTf) && <button className="btn btn-outline" style={{ fontSize: 10, padding: '1px 6px', height: 26 }} onClick={onClearFilters}>Clear</button>}
            <span className="text-muted" style={{ fontSize: 10, marginLeft: 'auto' }}>Showing {sortedMatrixResults.length} of {matrixResult.results.length}</span>
          </div>
        </div>
        <div style={{ overflowX: 'auto', maxHeight: TABLE_VIEWPORT, overflowY: 'auto' }} onScroll={win.onScroll}><table className="data-table"><thead><tr>
          <th>Strategy</th><th>Symbol</th><th>TF</th><th style={{ cursor: 'pointer' }} onClick={() => onSortToggle('trades')}>Trades{sortIndicator('trades')}</th>
          <th style={{ cursor: 'pointer' }} onClick={() => onSortToggle('sharpe')}>Sharpe{sortIndicator('sharpe')}</th>
          <th style={{ cursor: 'pointer' }} onClick={() => onSortToggle('sortino')}>Sortino{sortIndicator('sortino')}</th>
          <th style={{ cursor: 'pointer' }} onClick={() => onSortToggle('max_dd')}>Max DD{sortIndicator('max_dd')}</th>
          <th style={{ cursor: 'pointer' }} onClick={() => onSortToggle('return')}>Return{sortIndicator('return')}</th>
          <th style={{ cursor: 'pointer' }} onClick={() => onSortToggle('win_rate')}>Win{sortIndicator('win_rate')}</th>
          <th style={{ cursor: 'pointer' }} onClick={() => onSortToggle('profit_factor')}>PF{sortIndicator('profit_factor')}</th>
           <th>Gate</th><th>Opt</th>{onViewDetail && <th style={{ width: 50 }} />}
        </tr></thead><tbody>
          {win.topPad > 0 && <tr style={{ height: win.topPad }}><td colSpan={12} /></tr>}
          {sortedMatrixResults.slice(win.start, win.end).map((r: ComboResult, i: number) => (
          <tr key={win.start + i} style={{ height: ROW_HEIGHT, background: r.sharpe_ratio >= 1.0 ? 'rgba(63,185,80,.06)' : undefined }}>
            <td>{r.strategy_id}</td><td>{r.symbol}</td><td>{r.timeframe}</td><td>{r.num_trades}</td>
            <td style={{ color: r.sharpe_ratio >= 1.0 ? 'var(--trading-success)' : r.sharpe_ratio <= 0 ? 'var(--trading-danger)' : undefined }}>{r.sharpe_ratio?.toFixed(3)}</td>
            <td>{r.sortino_ratio?.toFixed(3)}</td><td>{r.max_drawdown?.toFixed(1)}%</td>
            <td style={{ color: r.total_return >= 0 ? 'var(--trading-success)' : 'var(--trading-danger)' }}>{r.total_return?.toFixed(1)}%</td>
            <td>{r.win_rate != null ? `${(r.win_rate * 100).toFixed(0)}%` : '\u2014'}</td>
            <td>{r.profit_factor?.toFixed(2)}</td>
             <td>{r.gate_passed === true ? <span className="badge badge-ok">PASS</span> : r.gate_passed === false ? <span className="badge badge-err">FAIL</span> : '\u2014'}</td>
            <td>{r.optimized ? <span className="badge badge-ok" title={JSON.stringify(r.best_params || {})}>Y</span> : '\u2014'}</td>
            {onViewDetail && <td><button className="text-xs" style={{ background: 'none', border: 'none', color: 'var(--accent)', cursor: 'pointer', padding: '1px 4px' }} onClick={() => onViewDetail(`${r.strategy_id}|${r.symbol}|${r.timeframe}`)}>View</button></td>}
          </tr>))}
          {win.bottomPad > 0 && <tr style={{ height: win.bottomPad }}><td colSpan={12} /></tr>}
        </tbody></table></div>
      </>)}
    </div>
  )
}
