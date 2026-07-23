import { useState, useCallback } from 'react'
import { simulate } from '../api/client'
import ErrorCard from '../components/ErrorCard'
import type {
  SimulateGenerateResponse,
  SimulateCalibrateResponse,
  SimulateValidateResponse,
} from '../types/api'

type Tab = 'generate' | 'calibrate' | 'validate'

export default function SimulatePage() {
  const [tab, setTab] = useState<Tab>('generate')

  const [genSymbols, setGenSymbols] = useState(5)
  const [genDays, setGenDays] = useState(252)
  const [regimeCalm, setRegimeCalm] = useState(40)
  const [regimeTrending, setRegimeTrending] = useState(30)
  const [regimeHighVol, setRegimeHighVol] = useState(20)
  const [regimeCrisis, setRegimeCrisis] = useState(10)
  const [sigTrend, setSigTrend] = useState(true)
  const [sigMeanRev, setSigMeanRev] = useState(false)
  const [sigBreakout, setSigBreakout] = useState(false)
  const [sigStrength, setSigStrength] = useState(50)
  const [genResult, setGenResult] = useState<SimulateGenerateResponse | null>(null)
  const [genLoading, setGenLoading] = useState(false)
  const [genError, setGenError] = useState<string | null>(null)

  const [calSymbol, setCalSymbol] = useState('SPY')
  const [calTimeframe, setCalTimeframe] = useState('1D')
  const [calStart, setCalStart] = useState('2024-01-01')
  const [calEnd, setCalEnd] = useState('2024-12-31')
  const [calResult, setCalResult] = useState<SimulateCalibrateResponse | null>(null)
  const [calLoading, setCalLoading] = useState(false)
  const [calError, setCalError] = useState<string | null>(null)

  const [valSymbol, setValSymbol] = useState('')
  const [valResult, setValResult] = useState<SimulateValidateResponse | null>(null)
  const [valLoading, setValLoading] = useState(false)
  const [valError, setValError] = useState<string | null>(null)

  const runGenerate = useCallback(async () => {
    setGenLoading(true)
    setGenError(null)
    setGenResult(null)
    try {
      const res = await simulate.generate({
        symbols: genSymbols,
        days: genDays,
        regimes: {
          calm: regimeCalm,
          trending: regimeTrending,
          highVol: regimeHighVol,
          crisis: regimeCrisis,
        },
        signals: {
          trend: sigTrend,
          meanReversion: sigMeanRev,
          breakout: sigBreakout,
          strength: sigStrength,
        },
      })
      if (res.error) {
        setGenError(res.error)
      } else {
        setGenResult(res)
      }
    } catch (err) {
      setGenError(err instanceof Error ? err.message : 'Generation failed')
    } finally {
      setGenLoading(false)
    }
  }, [genSymbols, genDays, regimeCalm, regimeTrending, regimeHighVol, regimeCrisis, sigTrend, sigMeanRev, sigBreakout, sigStrength])

  const runCalibrate = useCallback(async () => {
    setCalLoading(true)
    setCalError(null)
    setCalResult(null)
    try {
      const res = await simulate.calibrate({
        symbol: calSymbol,
        timeframe: calTimeframe,
        start_date: calStart,
        end_date: calEnd,
      })
      if (res.error) {
        setCalError(res.error)
      } else {
        setCalResult(res)
      }
    } catch (err) {
      setCalError(err instanceof Error ? err.message : 'Calibration failed')
    } finally {
      setCalLoading(false)
    }
  }, [calSymbol, calTimeframe, calStart, calEnd])

  const runValidate = useCallback(async () => {
    setValLoading(true)
    setValError(null)
    setValResult(null)
    try {
      const res = await simulate.validate({ symbol: valSymbol || undefined })
      if (res.error) {
        setValError(res.error)
      } else {
        setValResult(res)
      }
    } catch (err) {
      setValError(err instanceof Error ? err.message : 'Validation failed')
    } finally {
      setValLoading(false)
    }
  }, [valSymbol])

  const tabs: { id: Tab; label: string }[] = [
    { id: 'generate', label: 'Generate' },
    { id: 'calibrate', label: 'Calibrate' },
    { id: 'validate', label: 'Validate' },
  ]

  return (
    <div>
      <h1 style={{ margin: '0 0 16px' }}>Simulation</h1>

      <div className="flex gap-2 mb-4">
        {tabs.map((t) => (
          <button
            key={t.id}
            className={`btn ${tab === t.id ? 'btn-primary' : 'btn-outline'}`}
            onClick={() => setTab(t.id)}
          >
            {t.label}
          </button>
        ))}
      </div>

      {tab === 'generate' && (
        <div>
          <div className="card mb-4">
            <h2>Synthetic Data Generation</h2>
            <div className="grid-2 mb-3">
              <div>
                <label className="metric-label" style={{ display: 'block', marginBottom: 4 }}>Number of Symbols</label>
                <input className="input" type="number" min={1} max={50} value={genSymbols} onChange={(e) => setGenSymbols(Number(e.target.value))} />
              </div>
              <div>
                <label className="metric-label" style={{ display: 'block', marginBottom: 4 }}>Days of Data</label>
                <input className="input" type="number" min={30} max={2520} value={genDays} onChange={(e) => setGenDays(Number(e.target.value))} />
              </div>
            </div>

            <h2 style={{ fontSize: 13 }}>Regime Distribution (%)</h2>
            <div className="grid-2 mb-3">
              {[
                { label: 'Calm', value: regimeCalm, set: setRegimeCalm },
                { label: 'Trending', value: regimeTrending, set: setRegimeTrending },
                { label: 'High Vol', value: regimeHighVol, set: setRegimeHighVol },
                { label: 'Crisis', value: regimeCrisis, set: setRegimeCrisis },
              ].map((r) => (
                <div key={r.label}>
                  <div className="flex-between" style={{ marginBottom: 4 }}>
                    <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>{r.label}</span>
                    <span style={{ fontSize: 12, fontWeight: 600 }}>{r.value}%</span>
                  </div>
                  <input
                    type="range"
                    min={0}
                    max={100}
                    value={r.value}
                    onChange={(e) => r.set(Number(e.target.value))}
                    style={{ width: '100%' }}
                  />
                </div>
              ))}
            </div>

            <h2 style={{ fontSize: 13 }}>Signal Injection</h2>
            <div className="flex flex-wrap gap-3 mb-3" style={{ alignItems: 'center' }}>
              <label style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, cursor: 'pointer' }}>
                <input type="checkbox" checked={sigTrend} onChange={(e) => setSigTrend(e.target.checked)} />
                Trend
              </label>
              <label style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, cursor: 'pointer' }}>
                <input type="checkbox" checked={sigMeanRev} onChange={(e) => setSigMeanRev(e.target.checked)} />
                Mean Reversion
              </label>
              <label style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, cursor: 'pointer' }}>
                <input type="checkbox" checked={sigBreakout} onChange={(e) => setSigBreakout(e.target.checked)} />
                Breakout
              </label>
            </div>
            <div style={{ maxWidth: 300 }}>
              <div className="flex-between" style={{ marginBottom: 4 }}>
                <span className="metric-label">Signal Strength</span>
                <span style={{ fontSize: 12, fontWeight: 600 }}>{sigStrength}%</span>
              </div>
              <input
                type="range"
                min={0}
                max={100}
                value={sigStrength}
                onChange={(e) => setSigStrength(Number(e.target.value))}
                style={{ width: '100%' }}
              />
            </div>

            <div className="mt-3">
              <button className="btn btn-primary" onClick={runGenerate} disabled={genLoading}>
                {genLoading ? 'Generating...' : 'Generate'}
              </button>
            </div>
          </div>

          {genError && <ErrorCard message={genError} />}

          {genLoading && (
            <div className="card">
              <p className="text-muted">Generating synthetic market data...</p>
            </div>
          )}

          {genResult && !genLoading && (
            <div className="card">
              <h2>Generation Complete</h2>
              <div className="metric-grid">
                <div className="metric-card">
                  <div className="metric-label">Status</div>
                  <div className="metric-value" style={{ color: 'var(--success)' }}>{genResult.status}</div>
                </div>
                {genResult.progress != null && (
                  <div className="metric-card">
                    <div className="metric-label">Progress</div>
                    <div className="metric-value">{genResult.progress}%</div>
                  </div>
                )}
              </div>
              {genResult.download_url && (
                <div className="mt-3">
                  <a href={genResult.download_url} className="btn btn-primary" download>
                    Download CSV
                  </a>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {tab === 'calibrate' && (
        <div>
          <div className="card mb-4">
            <h2>HMM Calibration</h2>
            <div className="grid-3 mb-3">
              <div>
                <label className="metric-label" style={{ display: 'block', marginBottom: 4 }}>Symbol</label>
                <input className="input" value={calSymbol} onChange={(e) => setCalSymbol(e.target.value.toUpperCase())} />
              </div>
              <div>
                <label className="metric-label" style={{ display: 'block', marginBottom: 4 }}>Timeframe</label>
                <select className="input" value={calTimeframe} onChange={(e) => setCalTimeframe(e.target.value)}>
                  <option value="M1">1 min</option>
                  <option value="M5">5 min</option>
                  <option value="M15">15 min</option>
                  <option value="M30">30 min</option>
                  <option value="H1">1 hour</option>
                  <option value="H4">4 hour</option>
                  <option value="1D">1 day</option>
                  <option value="1W">1 week</option>
                </select>
              </div>
            </div>
            <div className="grid-2 mb-3">
              <div>
                <label className="metric-label" style={{ display: 'block', marginBottom: 4 }}>Start Date</label>
                <input className="input" type="date" value={calStart} onChange={(e) => setCalStart(e.target.value)} />
              </div>
              <div>
                <label className="metric-label" style={{ display: 'block', marginBottom: 4 }}>End Date</label>
                <input className="input" type="date" value={calEnd} onChange={(e) => setCalEnd(e.target.value)} />
              </div>
            </div>

            <button className="btn btn-primary" onClick={runCalibrate} disabled={calLoading}>
              {calLoading ? 'Running...' : 'Run Calibration'}
            </button>
          </div>

          {calError && <ErrorCard message={calError} />}

          {calLoading && (
            <div className="card">
              <p className="text-muted">Calibrating HMM on market data...</p>
            </div>
          )}

          {calResult && !calLoading && (
            <div>
              {calResult.regimes && calResult.regimes.length > 0 && (
                <div className="card mb-4">
                  <h2>Detected Regimes</h2>
                  <div className="flex flex-wrap gap-2">
                    {calResult.regimes.map((r, i) => (
                      <span key={i} className="badge badge-ok">{r}</span>
                    ))}
                  </div>
                </div>
              )}

              {calResult.state_means && Object.keys(calResult.state_means).length > 0 && (
                <div className="card mb-4">
                  <h2>State Means</h2>
                  <div style={{ overflowX: 'auto' }}>
                    <table>
                      <thead>
                        <tr>
                          <th>State</th>
                          <th>Mean Return</th>
                          <th>Volatility</th>
                        </tr>
                      </thead>
                      <tbody>
                        {Object.entries(calResult.state_means).map(([state, means]) => (
                          <tr key={state}>
                            <td style={{ fontWeight: 600 }}>{state}</td>
                            <td>{(means[0] * 100).toFixed(4)}%</td>
                            <td>{(means[1] * 100).toFixed(2)}%</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}

              {calResult.transition_matrix && (
                <div className="card mb-4">
                  <h2>Transition Matrix</h2>
                  <div style={{ overflowX: 'auto' }}>
                    <table>
                      <thead>
                        <tr>
                          <th>From \ To</th>
                          {calResult.regimes?.map((r, i) => (
                            <th key={i}>{r}</th>
                          ))}
                        </tr>
                      </thead>
                      <tbody>
                        {calResult.transition_matrix.map((row, i) => (
                          <tr key={i}>
                            <td style={{ fontWeight: 600 }}>{calResult.regimes?.[i] ?? `S${i}`}</td>
                            {row.map((val, j) => (
                              <td key={j}>{(val * 100).toFixed(1)}%</td>
                            ))}
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {tab === 'validate' && (
        <div>
          <div className="card mb-4">
            <h2>Validation</h2>
            <div style={{ maxWidth: 300 }}>
              <label className="metric-label" style={{ display: 'block', marginBottom: 4 }}>Symbol (optional)</label>
              <input className="input" value={valSymbol} onChange={(e) => setValSymbol(e.target.value.toUpperCase())} placeholder="All symbols" />
            </div>

            <div className="mt-3">
              <button className="btn btn-primary" onClick={runValidate} disabled={valLoading}>
                {valLoading ? 'Validating...' : 'Run Validation'}
              </button>
            </div>
          </div>

          {valError && <ErrorCard message={valError} />}

          {valLoading && (
            <div className="card">
              <p className="text-muted">Running validation checks...</p>
            </div>
          )}

          {valResult && !valLoading && (
            <div>
              <div className="grid-3 mb-4">
                <div className="metric-card">
                  <div className="metric-label">Regime Persistence</div>
                  <div className="metric-value" style={{ fontSize: 16 }}>
                    {valResult.regime_persistence ? (
                      <>
                        {(valResult.regime_persistence.score * 100).toFixed(1)}%
                        <span className={`badge ${valResult.regime_persistence.passed ? 'badge-ok' : 'badge-err'}`} style={{ marginLeft: 8 }}>
                          {valResult.regime_persistence.passed ? 'Pass' : 'Fail'}
                        </span>
                      </>
                    ) : (
                      '--'
                    )}
                  </div>
                </div>
                <div className="metric-card">
                  <div className="metric-label">Coverage</div>
                  <div className="metric-value" style={{ fontSize: 16 }}>
                    {valResult.coverage ? (
                      <>
                        {(valResult.coverage.score * 100).toFixed(1)}%
                        <span className={`badge ${valResult.coverage.passed ? 'badge-ok' : 'badge-err'}`} style={{ marginLeft: 8 }}>
                          {valResult.coverage.passed ? 'Pass' : 'Fail'}
                        </span>
                      </>
                    ) : (
                      '--'
                    )}
                  </div>
                </div>
                <div className="metric-card">
                  <div className="metric-label">Signal Quality</div>
                  <div className="metric-value" style={{ fontSize: 16 }}>
                    {valResult.signal_quality ? (
                      <>
                        {(valResult.signal_quality.score * 100).toFixed(1)}%
                        <span className={`badge ${valResult.signal_quality.passed ? 'badge-ok' : 'badge-err'}`} style={{ marginLeft: 8 }}>
                          {valResult.signal_quality.passed ? 'Pass' : 'Fail'}
                        </span>
                      </>
                    ) : (
                      '--'
                    )}
                  </div>
                </div>
              </div>

              {valResult.overall_passed != null && (
                <div className="card">
                  <div className="flex-between">
                    <span style={{ fontWeight: 600 }}>Overall</span>
                    <span className={`badge ${valResult.overall_passed ? 'badge-ok' : 'badge-err'}`}>
                      {valResult.overall_passed ? 'All Checks Passed' : 'Some Checks Failed'}
                    </span>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
