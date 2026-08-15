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
import { Badge } from '../ui/badge'
import { Checkbox } from '../ui/checkbox'
import { type SortField, STRATEGY_DISPLAY, SYMBOL_DISPLAY } from '../../data/constants'
import { Layers } from 'lucide-react'

function strategyLabel(id: string) { return STRATEGY_DISPLAY[id] ?? id }
function symbolLabel(id: string) { return SYMBOL_DISPLAY[id] ?? id }

const ROW_HEIGHT = 26
const TABLE_VIEWPORT = 420

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
  onPromoteToOrch?: (strategyId: string, symbol: string, timeframe: string) => void
  onBatchPromoteToOrch?: (combos: { strategy_id: string; symbol: string; timeframe: string }[]) => void
}

export default function MatrixResultsPanel(props: MatrixResultsPanelProps) {
  const {
    matrixResult, progressPct,
    filterStrategy, filterSymbol, filterTf,
    sortedMatrixResults, filterStrats, filterSyms, filterTfs,
    onFilterStrategyChange, onFilterSymbolChange, onFilterTfChange,
    onClearFilters, onSortToggle, sortIndicator, onViewDetail,
    onPromoteToOrch, onBatchPromoteToOrch,
  } = props

  const [showSensitivity, setShowSensitivity] = useState(false)
  const [showAdvanced, setShowAdvanced] = useState(true)
  const [selectedRows, setSelectedRows] = useState<Set<string>>(new Set())
  const sensitivity = useParameterSensitivity(matrixResult)
  const { entries, colorFor } = sensitivity

  const rowKey = (r: ComboResult) => `${r.strategy_id}|${r.symbol}|${r.timeframe}`
  const allSelected = sortedMatrixResults.length > 0 && selectedRows.size === sortedMatrixResults.length

  const toggleAll = () => {
    if (allSelected) {
      setSelectedRows(new Set())
    } else {
      setSelectedRows(new Set(sortedMatrixResults.map((r) => rowKey(r))))
    }
  }

  const toggleRow = (key: string) => {
    const next = new Set(selectedRows)
    if (next.has(key)) { next.delete(key) } else { next.add(key) }
    setSelectedRows(next)
  }

  const batchPromoteToOrch = () => {
    const combos = sortedMatrixResults.filter((r) => selectedRows.has(rowKey(r)))
    if (combos.length === 0) return
    if (onBatchPromoteToOrch) {
      onBatchPromoteToOrch(combos.map(r => ({ strategy_id: r.strategy_id, symbol: r.symbol, timeframe: r.timeframe })))
    } else {
      const orchSets = combos.map(r => ({
        strategy_id: r.strategy_id, symbol: r.symbol, timeframe: r.timeframe,
      }))
      sessionStorage.setItem('orch_batch_promote', JSON.stringify(orchSets))
      window.location.href = '/backtest?view=runner&mode=orchestrated&batch_promote=true'
    }
  }
  const win = useWindowedRows(sortedMatrixResults.length, ROW_HEIGHT, TABLE_VIEWPORT)

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
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" className="h-6 text-[10px] px-2" onClick={() => setShowAdvanced(a => !a)}>
                {showAdvanced ? 'Core Columns' : 'All Columns'}
              </Button>
              <Button variant="outline" size="sm" className="h-6 text-[10px] px-2" onClick={() => { exportMatrixResultsCSV(sortedMatrixResults); toast.success(`Exported ${sortedMatrixResults.length} results`) }}>
                CSV
              </Button>
            </div>
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

            {selectedRows.size > 0 && (
              <div className="flex items-center gap-2 py-1.5 px-2 rounded bg-blue-500/10 mb-2">
                <span className="text-[11px] font-medium text-blue-600">{selectedRows.size} selected</span>
                <Button size="sm" className="h-6 text-[10px] gap-1 bg-blue-600 hover:bg-blue-700" onClick={batchPromoteToOrch}>
                  <Layers className="h-3 w-3" /> Promote to Orch
                </Button>
                <Button variant="ghost" size="sm" className="h-6 text-[10px]" onClick={() => setSelectedRows(new Set())}>
                  Clear
                </Button>
              </div>
            )}

            <div className="overflow-x-auto border rounded-md" style={{ maxHeight: TABLE_VIEWPORT, overflowY: 'auto' }} onScroll={win.onScroll}>
              <Table noWrapper className="text-xs">
                <TableHeader className="sticky top-0 z-10 bg-card [&_tr]:!border-b">
                  <TableRow className="!border-b-2 bg-card">
                    <TableHead className="h-7 px-2 w-8">
                      <Checkbox checked={allSelected} onChange={() => toggleAll()} />
                    </TableHead>
                    <TableHead className="h-7 px-2">Strategy</TableHead>
                    <TableHead className="h-7 px-2">Symbol</TableHead>
                    <TableHead className="h-7 px-2">Timeframe</TableHead>
                    <TableHead className="h-7 px-2 cursor-pointer select-none" onClick={() => onSortToggle('trades')}>Trades{sortIndicator('trades')}</TableHead>
                    <TableHead className="h-7 px-2 cursor-pointer select-none" onClick={() => onSortToggle('sharpe')}>Sharpe{sortIndicator('sharpe')}</TableHead>
                    <TableHead className="h-7 px-2 cursor-pointer select-none" onClick={() => onSortToggle('sortino')}>Sortino{sortIndicator('sortino')}</TableHead>
                    <TableHead className="h-7 px-2 cursor-pointer select-none" onClick={() => onSortToggle('max_dd')}>Max DD{sortIndicator('max_dd')}</TableHead>
                    <TableHead className="h-7 px-2 cursor-pointer select-none" onClick={() => onSortToggle('return')}>Return{sortIndicator('return')}</TableHead>
                    <TableHead className="h-7 px-2 cursor-pointer select-none" onClick={() => onSortToggle('win_rate')}>Win Rate{sortIndicator('win_rate')}</TableHead>
                    <TableHead className="h-7 px-2 cursor-pointer select-none" onClick={() => onSortToggle('profit_factor')}>PF{sortIndicator('profit_factor')}</TableHead>
                    <TableHead className="h-7 px-2">Gate</TableHead>
                    {showAdvanced && <TableHead className="h-7 px-2 cursor-pointer select-none" onClick={() => onSortToggle('calmar')} title="Calmar Ratio (Return/MaxDD)">Calmar{sortIndicator('calmar')}</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Mark-to-Market Sharpe">MTM Sharpe</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Mark-to-Market Max Drawdown %">MTM MaxDD</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Long Gross PnL ($)">Long PnL</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Short Gross PnL ($)">Short PnL</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Long Profit Factor">L-PF</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Short Profit Factor">S-PF</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Parameters optimized">Opt</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2 cursor-pointer select-none" onClick={() => onSortToggle('total_fees')} title="Total Broker Fees ($)">Fees{sortIndicator('total_fees')}</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2 cursor-pointer select-none" onClick={() => onSortToggle('slippage')} title="Avg Slippage (bps)">Slip{sortIndicator('slippage')}</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2 cursor-pointer select-none" onClick={() => onSortToggle('candles')} title="Candle Count">Candles{sortIndicator('candles')}</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Gross Return %">Gross Return</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Max Drawdown Duration (bars)">DD Dur</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Train Split %">Train Split</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Walk-Forward IS Sharpe">Wf IS</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Walk-Forward OOS Sharpe">Wf OOS</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Declared Bars Per Day">BPD Decl</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Effective Bars Per Day">BPD Eff</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Adverse Selection Rate %">Adv Sel</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Zero-PnL Trades">ZeroPnL</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Expected Profit Factor">ExpPF</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Reward:Risk Ratio">R:R</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Daily Volatility %">DayVol</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="First Candle Time">First Candle</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Last Candle Time">Last Candle</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="ML Feature Enabled">ML</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Data Generation ID">GenID</TableHead>}
                    {showAdvanced && <TableHead className="h-7 px-2" title="Warnings / Data Quality">Warn</TableHead>}
                    {onViewDetail && <TableHead className="h-7 px-2 w-12" />}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {win.topPad > 0 && <TableRow style={{ height: win.topPad }}><TableCell colSpan={showAdvanced ? (onViewDetail ? 41 : 40) : (onViewDetail ? 13 : 12)} /></TableRow>}
                  {sortedMatrixResults.slice(win.start, win.end).map((r, i) => (
                    <TableRow key={win.start + i} className={`h-[26px] ${r.sharpe_ratio >= 1.0 ? 'bg-emerald-500/5' : ''}`}>
                      <TableCell className="px-2">
                        <Checkbox checked={selectedRows.has(rowKey(r))} onChange={() => toggleRow(rowKey(r))} />
                      </TableCell>
                      <TableCell className="px-2 text-[11px]">{strategyLabel(r.strategy_id)}</TableCell>
                      <TableCell className="px-2 text-[11px]">{symbolLabel(r.symbol)}</TableCell>
                      <TableCell className="px-2 text-[11px]">{r.timeframe}</TableCell>
                      <TableCell className="px-2 text-[11px] tabular-nums">{r.num_trades}</TableCell>
                      <TableCell className="px-2 text-[11px] tabular-nums" style={{ color: r.sharpe_ratio >= 1.0 ? 'var(--trading-success)' : r.sharpe_ratio <= 0 ? 'var(--trading-danger)' : undefined }}>{r.sharpe_ratio?.toFixed(3)}</TableCell>
                      <TableCell className="px-2 text-[11px] tabular-nums">{r.sortino_ratio?.toFixed(3)}</TableCell>
                      <TableCell className="px-2 text-[11px] tabular-nums">{r.max_drawdown?.toFixed(1)}%</TableCell>
                      <TableCell className="px-2 text-[11px] tabular-nums" style={{ color: r.total_return >= 0 ? 'var(--trading-success)' : 'var(--trading-danger)' }}>{r.total_return?.toFixed(1)}%</TableCell>
                      <TableCell className="px-2 text-[11px] tabular-nums">{r.win_rate != null ? `${r.win_rate.toFixed(0)}%` : '\u2014'}</TableCell>
                      <TableCell className="px-2 text-[11px] tabular-nums">{r.profit_factor?.toFixed(2)}</TableCell>
                      <TableCell className="px-2">{r.gate_passed === true ? <Badge variant="default" className="text-[10px] h-4 px-1">PASS</Badge> : r.gate_passed === false ? <Badge variant="destructive" className="text-[10px] h-4 px-1">FAIL</Badge> : '\u2014'}</TableCell>
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.calmar_ratio != null ? r.calmar_ratio.toFixed(2) : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.mtm_sharpe_ratio != null ? r.mtm_sharpe_ratio.toFixed(3) : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.mtm_max_drawdown != null ? `${r.mtm_max_drawdown.toFixed(1)}%` : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums" style={{ color: (r.long_gross_pnl ?? 0) >= 0 ? 'var(--trading-success)' : 'var(--trading-danger)' }}>{r.long_gross_pnl != null ? `$${r.long_gross_pnl.toFixed(0)}` : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums" style={{ color: (r.short_gross_pnl ?? 0) >= 0 ? 'var(--trading-success)' : 'var(--trading-danger)' }}>{r.short_gross_pnl != null ? `$${r.short_gross_pnl.toFixed(0)}` : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.long_profit_factor != null ? r.long_profit_factor.toFixed(2) : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.short_profit_factor != null ? r.short_profit_factor.toFixed(2) : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2">{r.optimized ? <Badge variant="outline" className="text-[10px] h-4 px-1" title={JSON.stringify(r.strategy_params || r.best_params || {})}>Y</Badge> : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.total_fees != null ? `$${r.total_fees.toFixed(0)}` : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.avg_slippage_bps != null ? r.avg_slippage_bps.toFixed(1) : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.candle_count != null ? r.candle_count.toLocaleString() : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.gross_return_pct != null ? `${r.gross_return_pct.toFixed(1)}%` : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.max_drawdown_duration ?? '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.train_pct != null ? `${r.train_pct.toFixed(1)}%` : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.wf_is_sharpe != null ? r.wf_is_sharpe.toFixed(3) : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.wf_oos_sharpe != null ? r.wf_oos_sharpe.toFixed(3) : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.declared_bars_per_day != null ? r.declared_bars_per_day.toFixed(1) : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.effective_bars_per_day != null ? r.effective_bars_per_day.toFixed(1) : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.adverse_selection_rate != null ? `${r.adverse_selection_rate.toFixed(1)}%` : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.zero_pnl_trades ?? '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.expected_pf != null ? r.expected_pf.toFixed(2) : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.reward_risk_ratio != null ? r.reward_risk_ratio.toFixed(2) : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px] tabular-nums">{r.daily_volatility != null ? `${(r.daily_volatility * 100).toFixed(2)}%` : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px]">{r.first_candle_time ? r.first_candle_time.slice(0, 10) : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 text-[11px]">{r.last_candle_time ? r.last_candle_time.slice(0, 10) : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2">{r.ml_feature_enabled ? <Badge variant="outline" className="text-[10px] h-4 px-1">Y</Badge> : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2 font-mono text-[10px] text-muted-foreground">{r.data_generation_id ? r.data_generation_id.slice(0, 8) : '\u2014'}</TableCell>}
                      {showAdvanced && <TableCell className="px-2">{r.implausible ? <Badge variant="destructive" className="text-[10px] h-4 px-1" title="Implausible result pattern detected">!</Badge> : r.warnings && r.warnings.length > 0 ? <Badge variant="secondary" className="text-[10px] h-4 px-1" title={r.warnings.join('; ')}>{r.warnings.length}</Badge> : '\u2014'}</TableCell>}
                      {onViewDetail && (
                        <TableCell className="px-2">
                          <div className="flex gap-1">
                            <Button variant="link" size="sm" className="h-5 text-[10px] p-0" onClick={() => onViewDetail(r.run_id || `${r.strategy_id}|${r.symbol}|${r.timeframe}`)}>View</Button>
                            {onPromoteToOrch ? (
                              <Button variant="link" size="sm" className="h-5 text-[10px] p-0 text-blue-600" onClick={() => onPromoteToOrch(r.strategy_id, r.symbol, r.timeframe)}>Orch</Button>
                            ) : (
                              <a href={`/backtest?view=runner&mode=orchestrated&orch_strategy=${encodeURIComponent(r.strategy_id)}&orch_symbol=${encodeURIComponent(r.symbol)}&orch_tf=${encodeURIComponent(r.timeframe)}`}
                                className="inline-flex items-center h-5 text-[10px] px-1 no-underline text-blue-600 hover:bg-blue-50 rounded">Orch</a>
                            )}
                          </div>
                        </TableCell>
                      )}
                    </TableRow>
                  ))}
                  {win.bottomPad > 0 && <TableRow style={{ height: win.bottomPad }}><TableCell colSpan={showAdvanced ? (onViewDetail ? 41 : 40) : (onViewDetail ? 13 : 12)} /></TableRow>}
                </TableBody>
              </Table>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}
