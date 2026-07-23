import { useState } from 'react'
import { Link } from 'react-router-dom'

const ALL_STRATEGIES = [
  'intraday_mr', 'opening_range_breakout', 'trend_following', 'grid_trading', 'session_scalp',
  'ma_crossover', 'rsi2_reversion', 'donchian_breakout', 'keltner_macd', 'ichimoku_cloud',
  'pairs_trading', 'volatility_harvesting',
]

const STRATEGY_PARAMS: Record<string, Record<string, { min: number; max: number; step: number; default: number }>> = {
  trend_following: {
    fast_period: { min: 10, max: 50, step: 5, default: 20 },
    slow_period: { min: 50, max: 200, step: 10, default: 100 },
    atr_period: { min: 7, max: 30, step: 1, default: 14 },
    atr_multiplier: { min: 1, max: 5, step: 0.5, default: 2 },
  },
  intraday_mr: {
    entry_z: { min: -3, max: 3, step: 0.5, default: 2 },
    exit_z: { min: 0.1, max: 1.5, step: 0.1, default: 0.5 },
    lookback: { min: 10, max: 50, step: 5, default: 20 },
  },
  opening_range_breakout: {
    range_minutes: { min: 5, max: 60, step: 5, default: 15 },
    entry_buffer_pct: { min: 0.1, max: 2, step: 0.1, default: 0.3 },
  },
}

const OBJECTIVES = ['sharpe', 'sortino', 'calmar', 'profit_factor', 'sharpe / drawdown', 'return / drawdown']

export default function BacktestPage() {
  const [mode, setMode] = useState<'backtest' | 'optimize'>('backtest')
  const [matrixMode, setMatrixMode] = useState<'matrix' | 'single'>('matrix')
  const [strategies, setStrategies] = useState<string[]>(['ma_crossover'])
  const [symbols, setSymbols] = useState('SPX500,NAS100')
  const [start, setStart] = useState('2024-01-01')
  const [end, setEnd] = useState('2024-06-30')
  const [capital, setCapital] = useState('100000')
  const [result, setResult] = useState<Record<string, unknown> | null>(null)
  const [loading, setLoading] = useState(false)

  const [optStrategy, setOptStrategy] = useState('trend_following')
  const [optObjective, setOptObjective] = useState('sharpe / drawdown')

  const toggleStrat = (s: string) => { setStrategies(p => p.includes(s) ? p.filter(x => x !== s) : [...p, s]) }

  const currentParams = STRATEGY_PARAMS[optStrategy] ?? {}

  const run = async () => {
    setLoading(true)
    try {
      const body = mode === 'backtest'
        ? { strategy_ids: strategies, symbols: symbols.split(',').map(s => s.trim()), start_date: start, end_date: end, capital: parseFloat(capital) }
        : { strategy_ids: [optStrategy], symbols: symbols.split(',').map(s => s.trim()), start_date: start, end_date: end, capital: parseFloat(capital), optimize: true }
      const r = await fetch('/api/v1/backtests', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })
      setResult(await r.json())
    } catch { setResult({ error: 'Backtest failed' }) } finally { setLoading(false) }
  }

  return <div>
    <div className="flex-between mb-4"><h1 style={{ margin: 0 }}>Backtest Runner</h1><Link to="/backtest/history" className="btn btn-outline">History</Link></div>

    <div className="flex gap-2 mb-4">
      <button className={`btn ${mode === 'backtest' ? 'btn-primary' : 'btn-outline'}`} onClick={() => setMode('backtest')}>Backtest</button>
      <button className={`btn ${mode === 'optimize' ? 'btn-primary' : 'btn-outline'}`} onClick={() => setMode('optimize')}>Optimize</button>
    </div>

    <div className="card mb-4">
      <h2>{mode === 'backtest' ? 'Backtest Configuration' : 'Optimization Configuration'}</h2>

      {mode === 'backtest' ? (
        <>
          <div className="flex gap-3 mb-3">
            <label className="flex gap-2" style={{ alignItems: 'center', cursor: 'pointer' }}>
              <input type="radio" name="run_mode" checked={matrixMode === 'matrix'} onChange={() => setMatrixMode('matrix')} /> Matrix
            </label>
            <label className="flex gap-2" style={{ alignItems: 'center', cursor: 'pointer' }}>
              <input type="radio" name="run_mode" checked={matrixMode === 'single'} onChange={() => setMatrixMode('single')} /> Single
            </label>
            <span className="text-muted">Daily (1d)</span>
            <span className="text-muted">Hourly (1h)</span>
            <span className="text-muted">5-Minute (5m)</span>
          </div>
          <div className="grid-2">
            <div><label className="text-muted">Start Date</label><input className="input" type="date" value={start} onChange={e => setStart(e.target.value)} /></div>
            <div><label className="text-muted">End Date</label><input className="input" type="date" value={end} onChange={e => setEnd(e.target.value)} /></div>
            <div><label className="text-muted">Symbols</label><input className="input" placeholder="EURUSD, BTCUSD, US30" value={symbols} onChange={e => setSymbols(e.target.value)} /></div>
            <div><label className="text-muted">Capital ($)</label><input className="input" type="number" value={capital} onChange={e => setCapital(e.target.value)} /></div>
          </div>
          <div className="mt-3"><label className="text-muted">Strategies</label>
            <div className="flex flex-wrap gap-2 mt-1" style={{ flexDirection: 'column' }}>
              {ALL_STRATEGIES.map(s => (
                <label key={s} className="flex gap-2" style={{ alignItems: 'center', padding: '4px 0', cursor: 'pointer' }}>
                  <input type="checkbox" checked={strategies.includes(s)} onChange={() => toggleStrat(s)} />
                  <span style={{ fontSize: 13 }}>{s}</span>
                </label>
              ))}
            </div>
          </div>
          <div className="text-muted mt-3">{matrixMode === 'single' ? `Will run 1 backtest` : `Will run ${strategies.length * symbols.split(',').length} backtests`}</div>
          <button className="btn btn-primary mt-2" onClick={run} disabled={loading || strategies.length === 0} style={{ justifyContent: 'center', width: '100%' }}>
            {loading ? 'Running...' : 'Run Backtest'}
          </button>
        </>
      ) : (
        <>
          <select className="input mb-3" value={optStrategy} onChange={e => setOptStrategy(e.target.value)}>
            {ALL_STRATEGIES.map(s => <option key={s} value={s}>{s.replace(/_/g, ' ')}</option>)}
          </select>
          <select className="input mb-3" value={optObjective} onChange={e => setOptObjective(e.target.value)}>
            {OBJECTIVES.map(o => <option key={o} value={o}>{o.replace(/_/g, ' ')}</option>)}
          </select>
          <div className="mb-3"><span className="text-muted" style={{ fontWeight: 600 }}>Search Space</span></div>
          {Object.keys(currentParams).length > 0 ? (
            <table className="data-table mb-3">
              <thead><tr><th>Parameter</th><th>Min</th><th>Max</th><th>Step</th></tr></thead>
              <tbody>
                {Object.entries(currentParams).map(([name, def]) => (
                  <tr key={name}>
                    <td>{name}</td><td>{def.min}</td><td>{def.max}</td><td>{def.step}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : <p className="text-muted">No parameter search space defined for this strategy.</p>}
          <button className="btn btn-primary" onClick={run} disabled={loading} style={{ justifyContent: 'center', width: '100%' }}>
            {loading ? 'Running...' : 'Run Optimization'}
          </button>
        </>
      )}
    </div>

    {result && <div className="card">
      <h2>Backtest Results</h2>
      {result.error ? <p className="text-muted" style={{ color: 'var(--danger)' }}>{String(result.error)}</p> : <div className="metric-grid">
        <div className="metric-card"><div className="metric-label orca-metric-card__label">Sharpe</div><div className="metric-value orca-metric-card__value">{Number(result.sharpe_ratio || 0).toFixed(2)}</div></div>
        <div className="metric-card"><div className="metric-label orca-metric-card__label">Max DD</div><div className="metric-value orca-metric-card__value">{Number(result.max_drawdown || 0).toFixed(1)}%</div></div>
        <div className="metric-card"><div className="metric-label orca-metric-card__label">Win Rate</div><div className="metric-value orca-metric-card__value">{Number(result.win_rate || 0).toFixed(1)}%</div></div>
        <div className="metric-card"><div className="metric-label orca-metric-card__label">Trades</div><div className="metric-value orca-metric-card__value">{Number(result.num_trades || 0)}</div></div>
        <div className="metric-card"><div className="metric-label orca-metric-card__label">Profit Factor</div><div className="metric-value orca-metric-card__value">{Number(result.profit_factor || 0).toFixed(2)}</div></div>
      </div>}
    </div>}
  </div>
}
