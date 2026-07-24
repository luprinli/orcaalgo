import { useState, useEffect } from 'react'
import { submitOptimizationRun, getOptimizationStatus, getOptimizationResults, listOptimizationRuns, OptimizeConfig, OptimizationResult, OptimizationStatus } from '../api/optimize'

const POLL_INTERVAL = 2000

const STRATEGY_PARAMS: Record<string, Record<string, { min: number; max: number; step: number; default: number }>> = {
  intraday_mr: {
    entry_z: { min: -3, max: 3, step: 0.5, default: 2 },
    exit_z: { min: 0.1, max: 1.5, step: 0.1, default: 0.5 },
    lookback: { min: 10, max: 50, step: 5, default: 20 },
  },
  trend_following: {
    fast_ma: { min: 10, max: 50, step: 5, default: 20 },
    slow_ma: { min: 50, max: 200, step: 10, default: 100 },
    atr_period: { min: 7, max: 30, step: 1, default: 14 },
  },
}

export default function OptimizationPanel() {
  const [strategyId, setStrategyId] = useState('intraday_mr')
  const [objective, setObjective] = useState<OptimizeConfig['objective']>('sharpe')
  const [symbol, setSymbol] = useState('SPY')
  const [trainYears, setTrainYears] = useState(2)
  const [testYears, setTestYears] = useState(1)
  const [stepMonths, setStepMonths] = useState(3)
  const [trials, setTrials] = useState(100)
  const [capital, setCapital] = useState(100000)

  const [runId, setRunId] = useState<string | null>(null)
  const [status, setStatus] = useState<OptimizationStatus | null>(null)
  const [result, setResult] = useState<OptimizationResult | null>(null)
  const [loading, setLoading] = useState(false)
  const [, setPreviousRuns] = useState<unknown[]>([])

  const params = STRATEGY_PARAMS[strategyId] ?? {}

  useEffect(() => {
    listOptimizationRuns().then(r => setPreviousRuns(r || [])).catch(() => {})
  }, [])

  useEffect(() => {
    if (!runId) return
    setLoading(true)
    const interval = setInterval(async () => {
      const s = await getOptimizationStatus(runId)
      setStatus(s)
      if (s.status === 'completed' || s.status === 'failed') {
        clearInterval(interval)
        if (s.status === 'completed') {
          const r = await getOptimizationResults(runId)
          setResult(r)
        }
        setLoading(false)
      }
    }, POLL_INTERVAL)
    return () => clearInterval(interval)
  }, [runId])

  const handleRun = async () => {
    const constraints: Record<string, { min: number; max: number; step: number }> = {}
    Object.entries(params).forEach(([name, def]) => {
      constraints[name] = { min: def.min, max: def.max, step: def.step }
    })

    const config: OptimizeConfig = {
      strategy_id: strategyId,
      objective,
      max_combinations: trials,
      train_years: trainYears,
      test_years: testYears,
      step_months: stepMonths,
      symbols: [symbol],
      capital,
      parameters: constraints,
    }

    const run = await submitOptimizationRun(config)
    setRunId(run.run_id)
    setResult(null)
    setStatus(null)
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
      <div className="card">
        <h2>Optimization</h2>
        <div className="grid grid-2" style={{ marginTop: 12 }}>
          <div>
            <label style={{ fontSize: 12, color: 'var(--text-secondary)', display: 'block', marginBottom: 4 }}>Strategy</label>
            <select value={strategyId} onChange={e => setStrategyId(e.target.value)} style={{ width: '100%' }}>
              <option value="intraday_mr">Intraday Mean Reversion</option>
              <option value="trend_following">Trend Following</option>
              <option value="opening_range_breakout">Opening Range Breakout</option>
              <option value="grid_trading">Grid Trading</option>
              <option value="session_scalp">Session Scalp</option>
            </select>
          </div>
          <div>
            <label style={{ fontSize: 12, color: 'var(--text-secondary)', display: 'block', marginBottom: 4 }}>Symbol</label>
            <input type="text" value={symbol} onChange={e => setSymbol(e.target.value)} style={{ width: '100%' }} />
          </div>
          <div>
            <label style={{ fontSize: 12, color: 'var(--text-secondary)', display: 'block', marginBottom: 4 }}>Objective</label>
            {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
            <select value={objective} onChange={e => setObjective(e.target.value as any)} style={{ width: '100%' }}>
              <option value="sharpe">Sharpe Ratio</option>
              <option value="sortino">Sortino Ratio</option>
              <option value="profit_factor">Profit Factor</option>
              <option value="win_rate">Win Rate</option>
            </select>
          </div>
          <div>
            <label style={{ fontSize: 12, color: 'var(--text-secondary)', display: 'block', marginBottom: 4 }}>Max Combinations</label>
            <input type="number" value={trials} onChange={e => setTrials(Number(e.target.value))} style={{ width: '100%' }} min={1} />
          </div>
          <div>
            <label style={{ fontSize: 12, color: 'var(--text-secondary)', display: 'block', marginBottom: 4 }}>Train Years</label>
            <input type="number" value={trainYears} onChange={e => setTrainYears(Number(e.target.value))} style={{ width: '100%' }} min={1} />
          </div>
          <div>
            <label style={{ fontSize: 12, color: 'var(--text-secondary)', display: 'block', marginBottom: 4 }}>Test Years</label>
            <input type="number" value={testYears} onChange={e => setTestYears(Number(e.target.value))} style={{ width: '100%' }} min={1} />
          </div>
          <div>
            <label style={{ fontSize: 12, color: 'var(--text-secondary)', display: 'block', marginBottom: 4 }}>Step Months</label>
            <input type="number" value={stepMonths} onChange={e => setStepMonths(Number(e.target.value))} style={{ width: '100%' }} min={1} />
          </div>
          <div>
            <label style={{ fontSize: 12, color: 'var(--text-secondary)', display: 'block', marginBottom: 4 }}>Capital</label>
            <input type="number" value={capital} onChange={e => setCapital(Number(e.target.value))} style={{ width: '100%' }} min={1000} />
          </div>
        </div>

        {Object.keys(params).length > 0 && (
          <div style={{ marginTop: 16 }}>
            <h3 style={{ fontSize: 13, color: 'var(--text-primary)', marginBottom: 8 }}>Parameter Ranges</h3>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
              <thead>
                <tr style={{ borderBottom: '1px solid var(--border)', color: 'var(--text-secondary)' }}>
                  <th style={{ textAlign: 'left', padding: '4px 8px' }}>Name</th>
                  <th style={{ textAlign: 'right', padding: '4px 8px' }}>Min</th>
                  <th style={{ textAlign: 'right', padding: '4px 8px' }}>Max</th>
                  <th style={{ textAlign: 'right', padding: '4px 8px' }}>Step</th>
                  <th style={{ textAlign: 'right', padding: '4px 8px' }}>Default</th>
                </tr>
              </thead>
              <tbody>
                {Object.entries(params).map(([name, def]) => (
                  <tr key={name} style={{ borderBottom: '1px solid var(--bg-card)' }}>
                    <td style={{ padding: '4px 8px' }}>{name}</td>
                    <td style={{ textAlign: 'right', padding: '4px 8px' }}>{def.min}</td>
                    <td style={{ textAlign: 'right', padding: '4px 8px' }}>{def.max}</td>
                    <td style={{ textAlign: 'right', padding: '4px 8px' }}>{def.step}</td>
                    <td style={{ textAlign: 'right', padding: '4px 8px' }}>{def.default}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        <button
          className="kill-switch-btn"
          style={{ background: 'var(--accent)', marginTop: 20, width: 'auto', padding: '10px 24px' }}
          onClick={handleRun}
          disabled={loading}
        >
          {loading ? 'Running...' : 'Run Optimization'}
        </button>
      </div>

      {status && status.status === 'running' && (
        <div className="card">
          <h2>Progress</h2>
          <div style={{ fontSize: 28, fontWeight: 700, color: 'var(--accent-text)' }}>
            {status.progress ?? 0}%
          </div>
          <div style={{ position: 'relative', height: 8, background: 'var(--bg-card)', borderRadius: 4, marginTop: 12 }}>
            <div style={{ width: `${status.progress ?? 0}%`, height: '100%', background: 'var(--accent)', borderRadius: 4, transition: 'width 1s' }} />
          </div>
          {status.elapsed_seconds && <div style={{ fontSize: 12, color: 'var(--text-secondary)', marginTop: 4 }}>Elapsed: {status.elapsed_seconds}s</div>}
        </div>
      )}

      {result && (
        <div className="card">
          <h2>Results</h2>
          <div className="grid grid-2" style={{ marginTop: 8 }}>
            <div className="metric">
              <div className="label">Best Metric</div>
              <div className="value green">{result.best_metric?.toFixed(4) ?? '-'}</div>
            </div>
            <div className="metric">
              <div className="label">Total Trials</div>
              <div className="value">{result.trials?.length ?? (result.trials ? 1 : 0)}</div>
            </div>
            {/* eslint-disable @typescript-eslint/no-explicit-any */}
            <div className="metric">
              <div className="label">Avg OOS Sharpe</div>
              <div className="value">{(result as any).avg_oos_sharpe?.toFixed(3) ?? '-'}</div>
            </div>
            <div className="metric">
              <div className="label">Windows Passed</div>
              <div className="value">{(result as any).windows_passed ?? '-'} / {(result as any).windows_total ?? '-'}</div>
            </div>
            {/* eslint-enable @typescript-eslint/no-explicit-any */}
          </div>
          {result.best_params && Object.keys(result.best_params).length > 0 && (
            <div style={{ marginTop: 16 }}>
              <h3 style={{ fontSize: 13, color: 'var(--text-primary)', marginBottom: 8 }}>Best Parameters</h3>
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
                <thead>
                  <tr style={{ borderBottom: '1px solid var(--border)', color: 'var(--text-secondary)' }}>
                    <th style={{ textAlign: 'left', padding: '4px 8px' }}>Parameter</th>
                    <th style={{ textAlign: 'right', padding: '4px 8px' }}>Value</th>
                  </tr>
                </thead>
                <tbody>
                  {Object.entries(result.best_params).map(([k, v]) => (
                    <tr key={k} style={{ borderBottom: '1px solid var(--bg-card)' }}>
                      <td style={{ padding: '4px 8px' }}>{k}</td>
                      <td style={{ textAlign: 'right', padding: '4px 8px', fontWeight: 700, color: 'var(--success)' }}>{v.toFixed(4)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
