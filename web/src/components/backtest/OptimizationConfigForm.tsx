import { useState } from 'react'

interface OptimizationConfigFormProps {
  strategyId: string
  objective: string
  symbol: string
  trainYears: number
  testYears: number
  stepMonths: number
  trials: number
  capital: number
  onStrategyChange: (v: string) => void
  onObjectiveChange: (v: string) => void
  onSymbolChange: (v: string) => void
  onTrainYearsChange: (v: number) => void
  onTestYearsChange: (v: number) => void
  onStepMonthsChange: (v: number) => void
  onTrialsChange: (v: number) => void
  onCapitalChange: (v: number) => void
  onSubmit: () => void
  loading?: boolean
  submitLabel?: string
}

export function OptimizationConfigForm({
  strategyId, objective, symbol,
  trainYears, testYears, stepMonths, trials, capital,
  onStrategyChange, onObjectiveChange, onSymbolChange,
  onTrainYearsChange, onTestYearsChange, onStepMonthsChange,
  onTrialsChange, onCapitalChange,
  onSubmit, loading, submitLabel = 'Run Optimization',
}: OptimizationConfigFormProps) {
  return (
    <div className="space-y-3">
      <div className="grid grid-2">
        <div>
          <label className="text-xs text-slate-400 block mb-1">Strategy</label>
          <select className="input" value={strategyId} onChange={e => onStrategyChange(e.target.value)}>
            <option value="intraday_mr">Intraday Mean Reversion</option>
            <option value="trend_following">Trend Following</option>
            <option value="grid">Grid Trading</option>
            <option value="session_scalp">Session Scalp</option>
            <option value="opening_range_breakout">Opening Range Breakout</option>
            <option value="rsi_divergence">RSI Divergence</option>
          </select>
        </div>
        <div>
          <label className="text-xs text-slate-400 block mb-1">Objective</label>
          <select className="input" value={objective} onChange={e => onObjectiveChange(e.target.value)}>
            <option value="sharpe">Sharpe Ratio</option>
            <option value="sortino">Sortino Ratio</option>
            <option value="calmar">Calmar Ratio</option>
            <option value="profit_factor">Profit Factor</option>
            <option value="return">Total Return</option>
          </select>
        </div>
      </div>
      <div>
        <label className="text-xs text-slate-400 block mb-1">Symbol</label>
        <input className="input" value={symbol} onChange={e => onSymbolChange(e.target.value.toUpperCase())} placeholder="SPY" />
      </div>
      <div className="grid grid-2">
        <div>
          <label className="text-xs text-slate-400 block mb-1">Train Years</label>
          <input className="input" type="number" min={1} max={5} value={trainYears} onChange={e => onTrainYearsChange(parseInt(e.target.value) || 2)} />
        </div>
        <div>
          <label className="text-xs text-slate-400 block mb-1">Test Years</label>
          <input className="input" type="number" min={1} max={3} value={testYears} onChange={e => onTestYearsChange(parseInt(e.target.value) || 1)} />
        </div>
        <div>
          <label className="text-xs text-slate-400 block mb-1">Step Months</label>
          <input className="input" type="number" min={1} max={12} value={stepMonths} onChange={e => onStepMonthsChange(parseInt(e.target.value) || 3)} />
        </div>
        <div>
          <label className="text-xs text-slate-400 block mb-1">Max Trials</label>
          <input className="input" type="number" min={10} max={1000} value={trials} onChange={e => onTrialsChange(parseInt(e.target.value) || 100)} />
        </div>
        <div>
          <label className="text-xs text-slate-400 block mb-1">Capital ($)</label>
          <input className="input" type="number" min={1000} step={1000} value={capital} onChange={e => onCapitalChange(parseInt(e.target.value) || 100000)} />
        </div>
      </div>
      <button className="btn btn-primary w-full" onClick={onSubmit} disabled={loading}>
        {loading ? 'Running...' : submitLabel}
      </button>
    </div>
  )
}
