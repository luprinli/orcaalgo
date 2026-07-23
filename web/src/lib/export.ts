import type { TradeSummary, EquityPoint, DailyReturn, ComboResult } from '../types/api'

function escapeCSV(value: unknown): string {
  const s = String(value ?? '')
  if (s.includes(',') || s.includes('"') || s.includes('\n')) {
    return `"${s.replace(/"/g, '""')}"`
  }
  return s
}

function downloadBlob(content: string, filename: string, mime = 'text/csv;charset=utf-8') {
  const blob = new Blob([content], { type: mime })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

export function exportTradesCSV(trades: TradeSummary[], filename = 'trades.csv') {
  const headers = ['Symbol', 'Side', 'Qty', 'Entry', 'Exit', 'PnL', 'Return%', 'MAE', 'MFE', 'Duration(h)', 'EntryTime', 'ExitTime', 'ExitReason']
  const rows = trades.map(t => [
    t.symbol, t.side, t.quantity,
    t.entry_price?.toFixed(4), t.exit_price?.toFixed(4),
    t.pnl?.toFixed(2), t.pnl_pct?.toFixed(2),
    t.mae?.toFixed(4), t.mfe?.toFixed(4),
    t.hold_duration?.toFixed(1),
    t.entry_time, t.exit_time, t.exit_reason,
  ].map(escapeCSV).join(','))

  const csv = [headers.join(','), ...rows].join('\n')
  downloadBlob(csv, filename)
}

export function exportEquityCSV(points: EquityPoint[], filename = 'equity.csv') {
  const headers = ['Time', 'Equity', 'Regime']
  const rows = points.map(p => [p.time, p.value, p.regime].map(escapeCSV).join(','))
  const csv = [headers.join(','), ...rows].join('\n')
  downloadBlob(csv, filename)
}

export function exportDailyReturnsCSV(returns: DailyReturn[], filename = 'daily_returns.csv') {
  const headers = ['Date', 'Return%', 'PnL']
  const rows = returns.map(r => [r.date, r.return_pct, r.pnl].map(escapeCSV).join(','))
  const csv = [headers.join(','), ...rows].join('\n')
  downloadBlob(csv, filename)
}

export function exportMatrixResultsCSV(results: ComboResult[], filename = 'matrix_results.csv') {
  const headers = ['Strategy', 'Symbol', 'Timeframe', 'Trades', 'Sharpe', 'Sortino', 'MaxDD%', 'Return%', 'WinRate', 'ProfitFactor', 'GatePassed']
  const rows = results.map(r => [
    r.strategy_id, r.symbol, r.timeframe, r.num_trades,
    r.sharpe_ratio?.toFixed(4), r.sortino_ratio?.toFixed(4),
    r.max_drawdown?.toFixed(2), r.total_return?.toFixed(2),
    r.win_rate != null ? r.win_rate.toFixed(4) : '',
    r.profit_factor?.toFixed(2),
    r.gate_passed,
  ].map(escapeCSV).join(','))
  const csv = [headers.join(','), ...rows].join('\n')
  downloadBlob(csv, filename)
}

export function exportMetricsCSV(metrics: Record<string, unknown>, filename = 'metrics.csv') {
  const rows = Object.entries(metrics).map(([k, v]) => [k, String(v ?? '')].map(escapeCSV).join(','))
  const csv = ['Metric,Value', ...rows].join('\n')
  downloadBlob(csv, filename)
}
