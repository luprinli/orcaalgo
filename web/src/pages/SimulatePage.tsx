import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { simulate } from '../api/client'
import ErrorCard from '../components/ErrorCard'
import type {
  SimulateGenerateResponse,
  SimulateCalibrateResponse,
  SimulateValidateResponse,
} from '../types/api'

type Tab = 'generate' | 'calibrate' | 'validate'

export default function SimulatePage() {
  const { t } = useTranslation()
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
      setGenError(err instanceof Error ? err.message : t('simulate:generationFailed', 'Generation failed'))
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
      setCalError(err instanceof Error ? err.message : t('simulate:calibrationFailed', 'Calibration failed'))
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
      setValError(err instanceof Error ? err.message : t('simulate:validationFailed', 'Validation failed'))
    } finally {
      setValLoading(false)
    }
  }, [valSymbol])

  const tabs: { id: Tab; label: string }[] = [
    { id: 'generate', label: t('simulate:generate', 'Generate') },
    { id: 'calibrate', label: t('simulate:calibrate', 'Calibrate') },
    { id: 'validate', label: t('simulate:validate', 'Validate') },
  ]

  return (
    <div>
      <h1 style={{ margin: '0 0 16px' }}>{t('simulate:title', 'Simulation')}</h1>

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
            <h2>{t('simulate:syntheticDataGeneration', 'Synthetic Data Generation')}</h2>
            <div className="grid-2 mb-3">
              <div>
                <label className="metric-label" style={{ display: 'block', marginBottom: 4 }}>{t('simulate:numberOfSymbols', 'Number of Symbols')}</label>
                <input className="input" type="number" min={1} max={50} value={genSymbols} onChange={(e) => setGenSymbols(Number(e.target.value))} />
              </div>
              <div>
                <label className="metric-label" style={{ display: 'block', marginBottom: 4 }}>{t('simulate:daysOfData', 'Days of Data')}</label>
                <input className="input" type="number" min={30} max={2520} value={genDays} onChange={(e) => setGenDays(Number(e.target.value))} />
              </div>
            </div>

            <h2 style={{ fontSize: 13 }}>{t('simulate:regimeDistribution', 'Regime Distribution (%)')}</h2>
            <div className="grid-2 mb-3">
              {[
                { label: t('simulate:calm', 'Calm'), value: regimeCalm, set: setRegimeCalm },
                { label: t('simulate:trending', 'Trending'), value: regimeTrending, set: setRegimeTrending },
                { label: t('simulate:highVol', 'High Vol'), value: regimeHighVol, set: setRegimeHighVol },
                { label: t('simulate:crisis', 'Crisis'), value: regimeCrisis, set: setRegimeCrisis },
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

            <h2 style={{ fontSize: 13 }}>{t('simulate:signalInjection', 'Signal Injection')}</h2>
            <div className="flex flex-wrap gap-3 mb-3" style={{ alignItems: 'center' }}>
              <label style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, cursor: 'pointer' }}>
                <input type="checkbox" checked={sigTrend} onChange={(e) => setSigTrend(e.target.checked)} />
                {t('simulate:trend', 'Trend')}
              </label>
              <label style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, cursor: 'pointer' }}>
                <input type="checkbox" checked={sigMeanRev} onChange={(e) => setSigMeanRev(e.target.checked)} />
                {t('simulate:meanReversion', 'Mean Reversion')}
              </label>
              <label style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, cursor: 'pointer' }}>
                <input type="checkbox" checked={sigBreakout} onChange={(e) => setSigBreakout(e.target.checked)} />
                {t('simulate:breakout', 'Breakout')}
              </label>
            </div>
            <div style={{ maxWidth: 300 }}>
              <div className="flex-between" style={{ marginBottom: 4 }}>
                <span className="metric-label">{t('simulate:signalStrength', 'Signal Strength')}</span>
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
                {genLoading ? t('simulate:generating', 'Generating...') : t('simulate:generateButton', 'Generate')}
              </button>
            </div>
          </div>

          {genError && <ErrorCard message={genError} />}

          {genLoading && (
            <div className="card">
              <p className="text-muted">{t('simulate:generatingData', 'Generating synthetic market data...')}</p>
            </div>
          )}

          {genResult && !genLoading && (
            <div className="card">
              <h2>{t('simulate:generationComplete', 'Generation Complete')}</h2>
              <div className="metric-grid">
                <div className="metric-card">
                  <div className="metric-label">{t('common:status', 'Status')}</div>
                  <div className="metric-value" style={{ color: 'var(--success)' }}>{genResult.status}</div>
                </div>
                {genResult.progress != null && (
                  <div className="metric-card">
                    <div className="metric-label">{t('simulate:progress', 'Progress')}</div>
                    <div className="metric-value">{genResult.progress}%</div>
                  </div>
                )}
              </div>
              {genResult.download_url && (
                <div className="mt-3">
                  <a href={genResult.download_url} className="btn btn-primary" download>
                    {t('simulate:downloadCsv', 'Download CSV')}
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
            <h2>{t('simulate:hmmCalibration', 'HMM Calibration')}</h2>
            <div className="grid-3 mb-3">
              <div>
                <label className="metric-label" style={{ display: 'block', marginBottom: 4 }}>{t('common:symbol', 'Symbol')}</label>
                <input className="input" value={calSymbol} onChange={(e) => setCalSymbol(e.target.value.toUpperCase())} />
              </div>
              <div>
                <label className="metric-label" style={{ display: 'block', marginBottom: 4 }}>{t('simulate:timeframe', 'Timeframe')}</label>
                <select className="input" value={calTimeframe} onChange={(e) => setCalTimeframe(e.target.value)}>
                  <option value="M1">{t('simulate:tf1min', '1 min')}</option>
                  <option value="M5">{t('simulate:tf5min', '5 min')}</option>
                  <option value="M15">{t('simulate:tf15min', '15 min')}</option>
                  <option value="M30">{t('simulate:tf30min', '30 min')}</option>
                  <option value="H1">{t('simulate:tf1hour', '1 hour')}</option>
                  <option value="H4">{t('simulate:tf4hour', '4 hour')}</option>
                  <option value="1D">{t('simulate:tf1day', '1 day')}</option>
                  <option value="1W">{t('simulate:tf1week', '1 week')}</option>
                </select>
              </div>
            </div>
            <div className="grid-2 mb-3">
              <div>
                <label className="metric-label" style={{ display: 'block', marginBottom: 4 }}>{t('simulate:startDate', 'Start Date')}</label>
                <input className="input" type="date" value={calStart} onChange={(e) => setCalStart(e.target.value)} />
              </div>
              <div>
                <label className="metric-label" style={{ display: 'block', marginBottom: 4 }}>{t('simulate:endDate', 'End Date')}</label>
                <input className="input" type="date" value={calEnd} onChange={(e) => setCalEnd(e.target.value)} />
              </div>
            </div>

            <button className="btn btn-primary" onClick={runCalibrate} disabled={calLoading}>
              {calLoading ? t('simulate:running', 'Running...') : t('simulate:runCalibration', 'Run Calibration')}
            </button>
          </div>

          {calError && <ErrorCard message={calError} />}

          {calLoading && (
            <div className="card">
              <p className="text-muted">{t('simulate:calibratingHmm', 'Calibrating HMM on market data...')}</p>
            </div>
          )}

          {calResult && !calLoading && (
            <div>
              {calResult.regimes && calResult.regimes.length > 0 && (
                <div className="card mb-4">
                  <h2>{t('simulate:detectedRegimes', 'Detected Regimes')}</h2>
                  <div className="flex flex-wrap gap-2">
                    {calResult.regimes.map((r, i) => (
                      <span key={i} className="badge badge-ok">{r}</span>
                    ))}
                  </div>
                </div>
              )}

              {calResult.state_means && Object.keys(calResult.state_means).length > 0 && (
                <div className="card mb-4">
                  <h2>{t('simulate:stateMeans', 'State Means')}</h2>
                  <div style={{ overflowX: 'auto' }}>
                    <table>
                      <thead>
                        <tr>
                          <th>{t('simulate:state', 'State')}</th>
                          <th>{t('simulate:meanReturn', 'Mean Return')}</th>
                          <th>{t('simulate:volatility', 'Volatility')}</th>
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
                  <h2>{t('simulate:transitionMatrix', 'Transition Matrix')}</h2>
                  <div style={{ overflowX: 'auto' }}>
                    <table>
                      <thead>
                        <tr>
                          <th>{t('simulate:fromTo', 'From \\ To')}</th>
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
            <h2>{t('simulate:validation', 'Validation')}</h2>
            <div style={{ maxWidth: 300 }}>
              <label className="metric-label" style={{ display: 'block', marginBottom: 4 }}>{t('simulate:symbolOptional', 'Symbol (optional)')}</label>
              <input className="input" value={valSymbol} onChange={(e) => setValSymbol(e.target.value.toUpperCase())} placeholder={t('simulate:allSymbols', 'All symbols')} />
            </div>

            <div className="mt-3">
              <button className="btn btn-primary" onClick={runValidate} disabled={valLoading}>
                {valLoading ? t('simulate:validating', 'Validating...') : t('simulate:runValidation', 'Run Validation')}
              </button>
            </div>
          </div>

          {valError && <ErrorCard message={valError} />}

          {valLoading && (
            <div className="card">
              <p className="text-muted">{t('simulate:runningValidationChecks', 'Running validation checks...')}</p>
            </div>
          )}

          {valResult && !valLoading && (
            <div>
              <div className="grid-3 mb-4">
                <div className="metric-card">
                  <div className="metric-label">{t('simulate:regimePersistence', 'Regime Persistence')}</div>
                  <div className="metric-value" style={{ fontSize: 16 }}>
                    {valResult.regime_persistence ? (
                      <>
                        {(valResult.regime_persistence.score * 100).toFixed(1)}%
                        <span className={`badge ${valResult.regime_persistence.passed ? 'badge-ok' : 'badge-err'}`} style={{ marginLeft: 8 }}>
                          {valResult.regime_persistence.passed ? t('simulate:pass', 'Pass') : t('simulate:fail', 'Fail')}
                        </span>
                      </>
                    ) : (
                      '--'
                    )}
                  </div>
                </div>
                <div className="metric-card">
                  <div className="metric-label">{t('simulate:coverage', 'Coverage')}</div>
                  <div className="metric-value" style={{ fontSize: 16 }}>
                    {valResult.coverage ? (
                      <>
                        {(valResult.coverage.score * 100).toFixed(1)}%
                        <span className={`badge ${valResult.coverage.passed ? 'badge-ok' : 'badge-err'}`} style={{ marginLeft: 8 }}>
                          {valResult.coverage.passed ? t('simulate:pass', 'Pass') : t('simulate:fail', 'Fail')}
                        </span>
                      </>
                    ) : (
                      '--'
                    )}
                  </div>
                </div>
                <div className="metric-card">
                  <div className="metric-label">{t('simulate:signalQuality', 'Signal Quality')}</div>
                  <div className="metric-value" style={{ fontSize: 16 }}>
                    {valResult.signal_quality ? (
                      <>
                        {(valResult.signal_quality.score * 100).toFixed(1)}%
                        <span className={`badge ${valResult.signal_quality.passed ? 'badge-ok' : 'badge-err'}`} style={{ marginLeft: 8 }}>
                          {valResult.signal_quality.passed ? t('simulate:pass', 'Pass') : t('simulate:fail', 'Fail')}
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
                    <span style={{ fontWeight: 600 }}>{t('simulate:overall', 'Overall')}</span>
                    <span className={`badge ${valResult.overall_passed ? 'badge-ok' : 'badge-err'}`}>
                      {valResult.overall_passed ? t('simulate:allChecksPassed', 'All Checks Passed') : t('simulate:someChecksFailed', 'Some Checks Failed')}
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
