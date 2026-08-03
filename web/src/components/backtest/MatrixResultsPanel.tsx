import { useState } from 'react'
import { useParameterSensitivity } from '../../hooks/useParameterSensitivity'
import { useWindowedRows } from '../../hooks/useWindowedRows'
import type { MatrixResultsResponse, ComboResult } from '../../types/api'
import { exportMatrixResultsCSV } from '../../lib/export'
import toast from 'react-hot-toast'
import { Card, CardContent } from '../ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../ui/table'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '../ui/select'
import { Button } from '../ui/button'

const STRATEGY_DISPLAY: Record<string, string> = {
  grid: 'Grid Trading',
  grid_trading: 'Grid Trading',
  mean_reversion: 'Mean Reversion',
  intraday_mr: 'Mean Reversion (Intraday)',
  trend: 'Trend Following',
  trend_following: 'Trend Following',
  breakout: 'ORB Breakout',
  opening_range_breakout: 'ORB Breakout',
  scalp: 'Session Scalp',
  session_scalp: 'Session Scalp',
  vol_arb: 'Vol Harvesting',
  volatility_harvesting: 'Vol Harvesting',
  stat_arb: 'Stat Arb',
  pairs_trading: 'Stat Arb',
  ma_crossover: 'MA Crossover',
  macd_rsi: 'MACD RSI',
  rsi2: 'RSI2 Reversion',
  rsi2_reversion: 'RSI2 Reversion',
  donchian: 'Donchian Breakout',
  donchian_breakout: 'Donchian Breakout',
  keltner: 'Keltner MACD',
  keltner_macd: 'Keltner MACD',
  ichimoku: 'Ichimoku Cloud',
  ichimoku_cloud: 'Ichimoku Cloud',
}

function strategyLabel(id: string) { return STRATEGY_DISPLAY[id] ?? id }
import { Badge } from '../ui/badge'

const ROW_HEIGHT = 26
const TABLE_VIEWPORT = 420

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
  onViewDetail?: (runId: string) => void
}

export default function MatrixResultsPanel(props: MatrixResultsPanelProps) {
  const {
    matrixResult, progressPct,
    filterStrategy, filterSymbol, filterTf,
    sortedMatrixResults, filterStrats, filterSyms, filterTfs,
    onFilterStrategyChange, onFilterSymbolChange, onFilterTfChange,
    onClearFilters, onSortToggle, sortIndicator, onViewDetail,
  } = props

  const [showSensitivity, setShowSensitivity] = useState(false)
  const sensitivity = useParameterSensitivity(matrixResult)
  const { entries, colorFor, minS, maxS } = sensitivity
  const win = useWindowedRows(sortedMatrixResults.length, ROW_HEIGHT, TABLE_VIEWPORT)
  const range = maxS - minS || 1

  const hasFilters = !!(filterStrategy || filterSymbol || filterTf)

  const Stat = ({ label, value, sub }: { label: string; value: string | number; sub?: string }) => (
    <div className="text-center px-2">
      <p className="text-[10px] text-muted-foreground leading-tight">{label}</p>
      <p className="text-sm font-bold tabular-nums">{value}</p>
      {sub && <p className="text-[9px] text-muted-foreground">{sub}</p>}
    </div>
  )

  return (
    <Card className="mt-4">
      <CardContent className="p-3">
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center gap-2">
            <h2 className="text-sm font-semibold m-0">Matrix Results</h2>
            <span className="text-[11px] text-muted-foreground">(streaming {progressPct}%)</span>
            <Badge variant={matrixResult.summary.status === 'completed' ? 'default' : 'secondary'} className="text-[10px] h-4 px-1.5">
              {matrixResult.summary.status ?? 'running'}
            </Badge>
          </div>
          {matrixResult.results.length > 0 && (
            <Button variant="outline" size="sm" className="h-6 text-[10px] px-2" onClick={() => { exportMatrixResultsCSV(sortedMatrixResults); toast.success(`Exported ${sortedMatrixResults.length} results`) }}>
              CSV
            </Button>
          )}
        </div>

        {matrixResult.results.length > 0 && (
          <>
            <div className="flex items-center gap-3 mb-2 pb-2 border-b">
              <Stat label="Combos" value={matrixResult.summary.total_combos} />
              <Stat label="Trades" value={matrixResult.summary.total_trades} />
              <Stat label="Best Sharpe" value={matrixResult.summary.best_sharpe?.toFixed(3) ?? '--'} />
              <Stat label="Best Strat" value={matrixResult.summary.best_strategy || '--'} />
              <Stat label="Best Sym" value={matrixResult.summary.best_symbol || '--'} />
            </div>

            {entries.length > 1 && (
              <div className="mb-2">
                <button
                  className="flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground cursor-pointer"
                  onClick={() => setShowSensitivity(p => !p)}
                >
                  <span className="text-[10px]">{showSensitivity ? '\u25BC' : '\u25B6'}</span>
                  Parameter Sensitivity ({entries.length})
                </button>
                {showSensitivity && (
                  <div className="mt-1 overflow-x-auto">
                    <Table className="text-[10px]">
                      <TableHeader>
                        <TableRow>
                          <TableHead className="h-6 px-2">Strategy / Sym / TF</TableHead>
                          <TableHead className="h-6 px-2">Best Params</TableHead>
                          <TableHead className="h-6 px-2 w-16">Sharpe</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {entries.map(e => (
                          <TableRow key={e.key} className="h-6">
                            <TableCell className="px-2 font-mono text-[10px]">{e.key}</TableCell>
                            <TableCell className="px-2 font-mono text-[9px] max-w-[200px] truncate" title={e.bestParams && Object.keys(e.bestParams).length > 0 ? JSON.stringify(e.bestParams) : undefined}>
                              {e.bestParams && Object.keys(e.bestParams).length > 0 ? JSON.stringify(e.bestParams) : '\u2014'}
                            </TableCell>
                            <TableCell className="px-2">
                              <Badge variant="secondary" className="text-[10px] font-semibold" style={{ background: colorFor(e.bestSharpe) }}>
                                {e.bestSharpe.toFixed(3)}
                              </Badge>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                )}
              </div>
            )}

            <div className="flex items-center gap-2 mb-2 py-1.5 px-2 rounded bg-muted/50">
              <span className="text-[10px] text-muted-foreground shrink-0">Filter:</span>
              <Select value={filterStrategy || '__none__'} onValueChange={v => onFilterStrategyChange(v === '__none__' ? '' : v)}>
                <SelectTrigger className="h-6 text-[11px] w-[150px]">
                  <SelectValue placeholder={`Strategies (${filterStrats.length})`} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__none__">All ({filterStrats.length})</SelectItem>
                  {filterStrats.map(s => <SelectItem key={s} value={s}>{s}</SelectItem>)}
                </SelectContent>
              </Select>
              <Select value={filterSymbol || '__none__'} onValueChange={v => onFilterSymbolChange(v === '__none__' ? '' : v)}>
                <SelectTrigger className="h-6 text-[11px] w-[130px]">
                  <SelectValue placeholder={`Symbols (${filterSyms.length})`} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__none__">All ({filterSyms.length})</SelectItem>
                  {filterSyms.map(s => <SelectItem key={s} value={s}>{s}</SelectItem>)}
                </SelectContent>
              </Select>
              <Select value={filterTf || '__none__'} onValueChange={v => onFilterTfChange(v === '__none__' ? '' : v)}>
                <SelectTrigger className="h-6 text-[11px] w-[110px]">
                  <SelectValue placeholder={`TFs (${filterTfs.length})`} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__none__">All ({filterTfs.length})</SelectItem>
                  {filterTfs.map(t => <SelectItem key={t} value={t}>{t}</SelectItem>)}
                </SelectContent>
              </Select>
              {hasFilters && <Button variant="ghost" size="sm" className="h-6 text-[10px] px-1.5" onClick={onClearFilters}>Clear</Button>}
              <span className="text-[10px] text-muted-foreground ml-auto">
                {sortedMatrixResults.length === matrixResult.results.length
                  ? `${sortedMatrixResults.length} rows`
                  : `${sortedMatrixResults.length} / ${matrixResult.results.length}`}
              </span>
            </div>

            <div className="overflow-x-auto border rounded-md" style={{ maxHeight: TABLE_VIEWPORT, overflowY: 'auto' }} onScroll={win.onScroll}>
              <Table noWrapper className="text-xs">
                <TableHeader className="sticky top-0 z-10 bg-card [&_tr]:!border-b">
                  <TableRow className="!border-b-2 bg-card">
                    <TableHead className="h-7 px-2">Strategy</TableHead>
                    <TableHead className="h-7 px-2">Symbol</TableHead>
                    <TableHead className="h-7 px-2">TF</TableHead>
                    <TableHead className="h-7 px-2 cursor-pointer select-none" onClick={() => onSortToggle('trades')}>Trades{sortIndicator('trades')}</TableHead>
                    <TableHead className="h-7 px-2 cursor-pointer select-none" onClick={() => onSortToggle('sharpe')}>Sharpe{sortIndicator('sharpe')}</TableHead>
                    <TableHead className="h-7 px-2 cursor-pointer select-none" onClick={() => onSortToggle('sortino')}>Sortino{sortIndicator('sortino')}</TableHead>
                    <TableHead className="h-7 px-2 cursor-pointer select-none" onClick={() => onSortToggle('max_dd')}>Max DD{sortIndicator('max_dd')}</TableHead>
                    <TableHead className="h-7 px-2 cursor-pointer select-none" onClick={() => onSortToggle('return')}>Return{sortIndicator('return')}</TableHead>
                    <TableHead className="h-7 px-2 cursor-pointer select-none" onClick={() => onSortToggle('win_rate')}>Win{sortIndicator('win_rate')}</TableHead>
                    <TableHead className="h-7 px-2 cursor-pointer select-none" onClick={() => onSortToggle('profit_factor')}>PF{sortIndicator('profit_factor')}</TableHead>
                    <TableHead className="h-7 px-2">L</TableHead>
                    <TableHead className="h-7 px-2">S</TableHead>
                    <TableHead className="h-7 px-2">L-PF</TableHead>
                    <TableHead className="h-7 px-2">S-PF</TableHead>
                    <TableHead className="h-7 px-2">Gate</TableHead>
                    <TableHead className="h-7 px-2">Opt</TableHead>
                    {onViewDetail && <TableHead className="h-7 px-2 w-12" />}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {win.topPad > 0 && <TableRow style={{ height: win.topPad }}><TableCell colSpan={onViewDetail ? 17 : 16} /></TableRow>}
                  {sortedMatrixResults.slice(win.start, win.end).map((r, i) => (
                    <TableRow key={win.start + i} className="h-[26px]" style={{ background: r.sharpe_ratio >= 1.0 ? 'rgba(63,185,80,.06)' : undefined }}>
                      <TableCell className="px-2 text-[11px]">{strategyLabel(r.strategy_id)}</TableCell>
                      <TableCell className="px-2 text-[11px]">{r.symbol}</TableCell>
                      <TableCell className="px-2 text-[11px]">{r.timeframe}</TableCell>
                      <TableCell className="px-2 text-[11px] tabular-nums">{r.num_trades}</TableCell>
                      <TableCell className="px-2 text-[11px] tabular-nums" style={{ color: r.sharpe_ratio >= 1.0 ? 'var(--trading-success)' : r.sharpe_ratio <= 0 ? 'var(--trading-danger)' : undefined }}>{r.sharpe_ratio?.toFixed(3)}</TableCell>
                      <TableCell className="px-2 text-[11px] tabular-nums">{r.sortino_ratio?.toFixed(3)}</TableCell>
                      <TableCell className="px-2 text-[11px] tabular-nums">{r.max_drawdown?.toFixed(1)}%</TableCell>
                      <TableCell className="px-2 text-[11px] tabular-nums" style={{ color: r.total_return >= 0 ? 'var(--trading-success)' : 'var(--trading-danger)' }}>{r.total_return?.toFixed(1)}%</TableCell>
                      <TableCell className="px-2 text-[11px] tabular-nums">{r.win_rate != null ? `${r.win_rate.toFixed(0)}%` : '\u2014'}</TableCell>
                      <TableCell className="px-2 text-[11px] tabular-nums">{r.profit_factor?.toFixed(2)}</TableCell>
                      <TableCell className="px-2 text-[11px] tabular-nums" style={{ color: (r.long_gross_pnl ?? 0) >= 0 ? 'var(--trading-success)' : 'var(--trading-danger)' }}>{r.long_gross_pnl != null ? r.long_gross_pnl.toFixed(1) : '\u2014'}</TableCell>
                      <TableCell className="px-2 text-[11px] tabular-nums" style={{ color: (r.short_gross_pnl ?? 0) >= 0 ? 'var(--trading-success)' : 'var(--trading-danger)' }}>{r.short_gross_pnl != null ? r.short_gross_pnl.toFixed(1) : '\u2014'}</TableCell>
                      <TableCell className="px-2 text-[11px] tabular-nums">{r.long_profit_factor != null ? r.long_profit_factor.toFixed(2) : '\u2014'}</TableCell>
                      <TableCell className="px-2 text-[11px] tabular-nums">{r.short_profit_factor != null ? r.short_profit_factor.toFixed(2) : '\u2014'}</TableCell>
                      <TableCell className="px-2">{r.gate_passed === true ? <Badge variant="default" className="text-[10px] h-4 px-1">PASS</Badge> : r.gate_passed === false ? <Badge variant="destructive" className="text-[10px] h-4 px-1">FAIL</Badge> : '\u2014'}</TableCell>
                      <TableCell className="px-2">{r.optimized ? <Badge variant="outline" className="text-[10px] h-4 px-1" title={JSON.stringify(r.strategy_params || r.best_params || {})}>Y</Badge> : '\u2014'}</TableCell>
                      {onViewDetail && (
                        <TableCell className="px-2">
                          <Button variant="link" size="sm" className="h-5 text-[10px] p-0" onClick={() => onViewDetail(r.run_id || `${r.strategy_id}|${r.symbol}|${r.timeframe}`)}>View</Button>
                        </TableCell>
                      )}
                    </TableRow>
                  ))}
                  {win.bottomPad > 0 && <TableRow style={{ height: win.bottomPad }}><TableCell colSpan={onViewDetail ? 17 : 16} /></TableRow>}
                </TableBody>
              </Table>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}
