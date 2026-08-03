import type { OptimizationFootprint } from '../../types/api'
import { Card, CardContent } from '../ui/card'

interface Props {
  optimization: OptimizationFootprint | null
}

export default function OptimizationTab({ optimization }: Props) {
  if (!optimization) {
    return <p className="text-sm text-muted-foreground p-2">No optimization data available for this backtest.</p>
  }

  const hasData = (optimization.walk_forward_windows ?? 0) > 0 ||
    (optimization.grid_passes ?? 0) > 0 ||
    (optimization.bayesian_iterations ?? 0) > 0

  if (!hasData) {
    return <p className="text-sm text-muted-foreground p-2">This backtest did not run optimization. Run a walk-forward optimization from the Strategy Hub, or enable &quot;Auto-Optimize&quot; when using Matrix mode.</p>
  }

  const stats = [
    { label: 'Deflated Sharpe', value: optimization.deflated_sharpe?.toFixed(3) ?? '--' },
    { label: 'Conventional Sharpe', value: optimization.conventional_sharpe?.toFixed(3) ?? '--' },
    { label: 'OOS Avg Sharpe', value: optimization.oos_average_sharpe?.toFixed(3) ?? '--' },
    { label: 'Sharpe Degradation', value: optimization.sharpe_degradation != null ? `${(optimization.sharpe_degradation * 100).toFixed(1)}%` : '--' },
    { label: 'IVS', value: optimization.ivs?.toFixed(3) ?? '--' },
    { label: 'W-F Windows', value: optimization.walk_forward_windows ?? 0 },
    { label: 'Passed', value: `${optimization.passed_windows}/${optimization.walk_forward_windows}` },
    { label: 'Grid Passes', value: optimization.grid_passes ?? 0 },
    { label: 'Bayesian Iters', value: optimization.bayesian_iterations ?? 0 },
  ]

  return (
    <div>
      <div className="grid grid-cols-3 gap-2 mb-4">
        {stats.map(s => (
          <Card key={s.label}><CardContent className="p-2 text-center">
            <p className="text-[10px] uppercase tracking-wide text-muted-foreground">{s.label}</p>
            <p className="text-sm font-bold tabular-nums leading-tight">{s.value}</p>
          </CardContent></Card>
        ))}
      </div>
      {optimization.best_params_json && (() => {
        let parsed: Record<string, unknown> = {}
        try { parsed = JSON.parse(optimization.best_params_json) } catch { return null }
        const entries = Object.entries(parsed)
        if (entries.length === 0) return null
        return (
          <div className="overflow-x-auto">
            <table className="w-full text-xs border-collapse">
              <thead><tr><th className="text-left p-1 border-b">Parameter</th><th className="text-right p-1 border-b">Optimized</th></tr></thead>
              <tbody>{entries.map(([k, v]) => (
                <tr key={k}><td className="p-1 font-mono text-[11px]">{k}</td><td className="p-1 font-mono text-[11px] text-right font-semibold">{typeof v === 'number' ? v.toFixed(4) : String(v)}</td></tr>
              ))}</tbody>
            </table>
          </div>
        )
      })()}
    </div>
  )
}
