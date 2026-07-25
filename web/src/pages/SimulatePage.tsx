import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { simulate } from '../api/client'
import ErrorCard from '../components/ErrorCard'
import { Card, CardHeader, CardTitle, CardContent } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'
import { Badge } from '../components/ui/badge'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../components/ui/tabs'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '../components/ui/select'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/table'
import MetricCard from '../components/MetricCard'
import type {
  SimulateGenerateResponse,
  SimulateCalibrateResponse,
  SimulateValidateResponse,
} from '../types/api'

type Tab = 'generate' | 'calibrate' | 'validate' | 'calibrate-regime' | 'ticks' | 'inject-signal' | 'validate-regime'

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
  const [regimeSymbols, setRegimeSymbols] = useState('SPY,QQQ')
  const [regimeResult, setRegimeResult] = useState<Record<string, unknown> | null>(null)
  const [genId, setGenId] = useState('')
  const [tickSymbol, setTickSymbol] = useState('')
  const [tickRate, setTickRate] = useState('10')
  const [tickSpread, setTickSpread] = useState('1')
  const [tickResult, setTickResult] = useState<Record<string, unknown> | null>(null)
  const [injectStrategy, setInjectStrategy] = useState('')
  const [injectStrength, setInjectStrength] = useState('0.7')
  const [injectResult, setInjectResult] = useState<Record<string, unknown> | null>(null)
  const [validRegimeResult, setValidRegimeResult] = useState<{ passed: boolean; regime_persistence?: { passed: boolean }; coverage?: { passed: boolean }; signal_quality?: { passed: boolean } } | null>(null)
  const [loading, setLoading] = useState(false)
  const [msg, setMsg] = useState('')

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

  return (
    <div>
      <h1 className="mb-4">{t('simulate:title', 'Simulation')}</h1>

      <Tabs value={tab} onValueChange={(v) => setTab(v as Tab)} className="mb-4">
        <TabsList>
          <TabsTrigger value="generate">{t('simulate:generate', 'Generate')}</TabsTrigger>
          <TabsTrigger value="calibrate">{t('simulate:calibrate', 'Calibrate')}</TabsTrigger>
          <TabsTrigger value="validate">{t('simulate:validate', 'Validate')}</TabsTrigger>
          <TabsTrigger value="calibrate-regime">{t('simulate:calibrateRegime', 'Regime Calibrate')}</TabsTrigger>
          <TabsTrigger value="ticks">{t('simulate:ticks', 'Ticks')}</TabsTrigger>
          <TabsTrigger value="inject-signal">{t('simulate:injectSignal', 'Inject Signal')}</TabsTrigger>
          <TabsTrigger value="validate-regime">{t('simulate:validateRegime', 'Validate Regime')}</TabsTrigger>
        </TabsList>

        <TabsContent value="generate">
          <Card className="mb-4">
            <CardHeader>
              <CardTitle>{t('simulate:syntheticDataGeneration', 'Synthetic Data Generation')}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-3 mb-3">
                <div>
                  <Label>{t('simulate:numberOfSymbols', 'Number of Symbols')}</Label>
                  <Input type="number" min={1} max={50} value={genSymbols} onChange={(e) => setGenSymbols(Number(e.target.value))} />
                </div>
                <div>
                  <Label>{t('simulate:daysOfData', 'Days of Data')}</Label>
                  <Input type="number" min={30} max={2520} value={genDays} onChange={(e) => setGenDays(Number(e.target.value))} />
                </div>
              </div>

              <h2 className="text-sm font-medium mb-2">{t('simulate:regimeDistribution', 'Regime Distribution (%)')}</h2>
              <div className="grid grid-cols-2 gap-3 mb-3">
                {[
                  { label: t('simulate:calm', 'Calm'), value: regimeCalm, set: setRegimeCalm },
                  { label: t('simulate:trending', 'Trending'), value: regimeTrending, set: setRegimeTrending },
                  { label: t('simulate:highVol', 'High Vol'), value: regimeHighVol, set: setRegimeHighVol },
                  { label: t('simulate:crisis', 'Crisis'), value: regimeCrisis, set: setRegimeCrisis },
                ].map((r) => (
                  <div key={r.label}>
                    <div className="flex items-center justify-between" style={{ marginBottom: 4 }}>
                      <span style={{ fontSize: 12, color: 'var(--muted-foreground)' }}>{r.label}</span>
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

              <h2 className="text-sm font-medium mb-2">{t('simulate:signalInjection', 'Signal Injection')}</h2>
              <div className="flex flex-wrap gap-3 mb-3 items-center">
                <label className="flex items-center gap-1.5 text-sm cursor-pointer">
                  <input type="checkbox" checked={sigTrend} onChange={(e) => setSigTrend(e.target.checked)} />
                  {t('simulate:trend', 'Trend')}
                </label>
                <label className="flex items-center gap-1.5 text-sm cursor-pointer">
                  <input type="checkbox" checked={sigMeanRev} onChange={(e) => setSigMeanRev(e.target.checked)} />
                  {t('simulate:meanReversion', 'Mean Reversion')}
                </label>
                <label className="flex items-center gap-1.5 text-sm cursor-pointer">
                  <input type="checkbox" checked={sigBreakout} onChange={(e) => setSigBreakout(e.target.checked)} />
                  {t('simulate:breakout', 'Breakout')}
                </label>
              </div>
              <div className="max-w-xs">
                <div className="flex items-center justify-between" style={{ marginBottom: 4 }}>
                  <span className="text-sm text-muted-foreground">{t('simulate:signalStrength', 'Signal Strength')}</span>
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
                <Button onClick={runGenerate} disabled={genLoading}>
                  {genLoading ? t('simulate:generating', 'Generating...') : t('simulate:generateButton', 'Generate')}
                </Button>
              </div>
            </CardContent>
          </Card>

          {genError && <ErrorCard message={genError} />}

          {genLoading && (
            <Card>
              <CardContent className="p-6">
                <p className="text-muted-foreground">{t('simulate:generatingData', 'Generating synthetic market data...')}</p>
              </CardContent>
            </Card>
          )}

          {genResult && !genLoading && (
            <Card>
              <CardHeader>
                <CardTitle>{t('simulate:generationComplete', 'Generation Complete')}</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
                  <MetricCard label={t('common:status', 'Status')} value={genResult.status} color="positive" />
                  {genResult.progress != null && (
                    <MetricCard label={t('simulate:progress', 'Progress')} value={genResult.progress} format="percent_raw" color={genResult.progress === 100 ? 'positive' : 'default'} />
                  )}
                </div>
                {genResult.download_url && (
                  <div className="mt-3">
                    <Button asChild>
                      <a href={genResult.download_url} download>
                        {t('simulate:downloadCsv', 'Download CSV')}
                      </a>
                    </Button>
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="calibrate">
          <Card className="mb-4">
            <CardHeader>
              <CardTitle>{t('simulate:hmmCalibration', 'HMM Calibration')}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-3 gap-3 mb-3">
                <div>
                  <Label>{t('common:symbol', 'Symbol')}</Label>
                  <Input value={calSymbol} onChange={(e) => setCalSymbol(e.target.value.toUpperCase())} />
                </div>
                <div>
                  <Label>{t('simulate:timeframe', 'Timeframe')}</Label>
                  <Select value={calTimeframe} onValueChange={setCalTimeframe}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="M1">{t('simulate:tf1min', '1 min')}</SelectItem>
                      <SelectItem value="M5">{t('simulate:tf5min', '5 min')}</SelectItem>
                      <SelectItem value="M15">{t('simulate:tf15min', '15 min')}</SelectItem>
                      <SelectItem value="M30">{t('simulate:tf30min', '30 min')}</SelectItem>
                      <SelectItem value="H1">{t('simulate:tf1hour', '1 hour')}</SelectItem>
                      <SelectItem value="H4">{t('simulate:tf4hour', '4 hour')}</SelectItem>
                      <SelectItem value="1D">{t('simulate:tf1day', '1 day')}</SelectItem>
                      <SelectItem value="1W">{t('simulate:tf1week', '1 week')}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3 mb-3">
                <div>
                  <Label>{t('simulate:startDate', 'Start Date')}</Label>
                  <Input type="date" value={calStart} onChange={(e) => setCalStart(e.target.value)} />
                </div>
                <div>
                  <Label>{t('simulate:endDate', 'End Date')}</Label>
                  <Input type="date" value={calEnd} onChange={(e) => setCalEnd(e.target.value)} />
                </div>
              </div>

              <Button onClick={runCalibrate} disabled={calLoading}>
                {calLoading ? t('simulate:running', 'Running...') : t('simulate:runCalibration', 'Run Calibration')}
              </Button>
            </CardContent>
          </Card>

          {calError && <ErrorCard message={calError} />}

          {calLoading && (
            <Card>
              <CardContent className="p-6">
                <p className="text-muted-foreground">{t('simulate:calibratingHmm', 'Calibrating HMM on market data...')}</p>
              </CardContent>
            </Card>
          )}

          {calResult && !calLoading && (
            <div>
              {calResult.regimes && calResult.regimes.length > 0 && (
                <Card className="mb-4">
                  <CardHeader>
                    <CardTitle>{t('simulate:detectedRegimes', 'Detected Regimes')}</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="flex flex-wrap gap-2">
                      {calResult.regimes.map((r, i) => (
                        <Badge key={i} variant="outline" className="text-trading-success border-trading-success/30 bg-trading-success/10">{r}</Badge>
                      ))}
                    </div>
                  </CardContent>
                </Card>
              )}

              {calResult.state_means && Object.keys(calResult.state_means).length > 0 && (
                <Card className="mb-4">
                  <CardHeader>
                    <CardTitle>{t('simulate:stateMeans', 'State Means')}</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('simulate:state', 'State')}</TableHead>
                          <TableHead>{t('simulate:meanReturn', 'Mean Return')}</TableHead>
                          <TableHead>{t('simulate:volatility', 'Volatility')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {Object.entries(calResult.state_means).map(([state, means]) => (
                          <TableRow key={state}>
                            <TableCell className="font-semibold">{state}</TableCell>
                            <TableCell>{(means[0] * 100).toFixed(4)}%</TableCell>
                            <TableCell>{(means[1] * 100).toFixed(2)}%</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </CardContent>
                </Card>
              )}

              {calResult.transition_matrix && (
                <Card className="mb-4">
                  <CardHeader>
                    <CardTitle>{t('simulate:transitionMatrix', 'Transition Matrix')}</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('simulate:fromTo', 'From \\ To')}</TableHead>
                          {calResult.regimes?.map((r, i) => (
                            <TableHead key={i}>{r}</TableHead>
                          ))}
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {calResult.transition_matrix.map((row, i) => (
                          <TableRow key={i}>
                            <TableCell className="font-semibold">{calResult.regimes?.[i] ?? `S${i}`}</TableCell>
                            {row.map((val, j) => (
                              <TableCell key={j}>{(val * 100).toFixed(1)}%</TableCell>
                            ))}
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </CardContent>
                </Card>
              )}
            </div>
          )}
        </TabsContent>

        <TabsContent value="validate">
          <Card className="mb-4">
            <CardHeader>
              <CardTitle>{t('simulate:validation', 'Validation')}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="max-w-xs">
                <Label>{t('simulate:symbolOptional', 'Symbol (optional)')}</Label>
                <Input value={valSymbol} onChange={(e) => setValSymbol(e.target.value.toUpperCase())} placeholder={t('simulate:allSymbols', 'All symbols')} />
              </div>

              <div className="mt-3">
                <Button onClick={runValidate} disabled={valLoading}>
                  {valLoading ? t('simulate:validating', 'Validating...') : t('simulate:runValidation', 'Run Validation')}
                </Button>
              </div>
            </CardContent>
          </Card>

          {valError && <ErrorCard message={valError} />}

          {valLoading && (
            <Card>
              <CardContent className="p-6">
                <p className="text-muted-foreground">{t('simulate:runningValidationChecks', 'Running validation checks...')}</p>
              </CardContent>
            </Card>
          )}

          {valResult && !valLoading && (
            <div>
              <div className="grid grid-cols-3 gap-3 mb-4">
                <MetricCard label={t('simulate:regimePersistence', 'Regime Persistence')} value={valResult.regime_persistence ? `${(valResult.regime_persistence.score * 100).toFixed(1)}%` : '--'} color={valResult.regime_persistence?.passed ? 'positive' : valResult.regime_persistence ? 'negative' : 'default'} />
                <MetricCard label={t('simulate:coverage', 'Coverage')} value={valResult.coverage ? `${(valResult.coverage.score * 100).toFixed(1)}%` : '--'} color={valResult.coverage?.passed ? 'positive' : valResult.coverage ? 'negative' : 'default'} />
                <MetricCard label={t('simulate:signalQuality', 'Signal Quality')} value={valResult.signal_quality ? `${(valResult.signal_quality.score * 100).toFixed(1)}%` : '--'} color={valResult.signal_quality?.passed ? 'positive' : valResult.signal_quality ? 'negative' : 'default'} />
              </div>

              {valResult.overall_passed != null && (
                <Card>
                  <CardContent className="flex items-center justify-between p-4">
                    <span className="font-semibold">{t('simulate:overall', 'Overall')}</span>
                    <Badge variant={valResult.overall_passed ? 'outline' : 'destructive'}
                      className={valResult.overall_passed ? 'text-trading-success border-trading-success/30 bg-trading-success/10' : ''}>
                      {valResult.overall_passed ? t('simulate:allChecksPassed', 'All Checks Passed') : t('simulate:someChecksFailed', 'Some Checks Failed')}
                    </Badge>
                  </CardContent>
                </Card>
              )}
            </div>
          )}
        </TabsContent>

        <TabsContent value="calibrate-regime">
          <Card>
            <CardHeader><CardTitle>Calibrate Regime Model</CardTitle></CardHeader>
            <CardContent className="flex flex-col gap-3 max-w-[400px]">
              <Input placeholder="Symbols (comma-separated, default: SPY,QQQ)" value={regimeSymbols} onChange={e => setRegimeSymbols(e.target.value)} />
              <Button onClick={async () => { setLoading(true); try { setRegimeResult(await simulate.calibrateRegime({ symbols: regimeSymbols.split(',').map(s => s.trim()).filter(Boolean) })); setMsg('Regime model calibrated') } catch (e) { setMsg(String(e)) } finally { setLoading(false) } }}>{loading ? 'Running...' : 'Calibrate Regime'}</Button>
              {regimeResult && (
                <div className="grid grid-cols-2 gap-3 mt-2">
                  {Object.entries(regimeResult).map(([k, v]) => (
                    <MetricCard key={k} label={k.replace(/_/g, ' ')} value={typeof v === 'number' ? String(v) : typeof v === 'boolean' ? (v ? 'Yes' : 'No') : typeof v === 'object' ? JSON.stringify(v) : String(v ?? '--')} />
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="ticks">
          <Card>
            <CardHeader><CardTitle>Generate Tick Data</CardTitle></CardHeader>
            <CardContent className="flex flex-col gap-3 max-w-[400px]">
              <Input placeholder="Generation ID" value={genId} onChange={e => setGenId(e.target.value)} />
              <Input placeholder="Symbol" value={tickSymbol} onChange={e => setTickSymbol(e.target.value)} />
              <Input placeholder="Ticks per minute (default: 10)" type="number" value={tickRate} onChange={e => setTickRate(e.target.value)} />
              <Input placeholder="Spread bps (default: 1)" type="number" value={tickSpread} onChange={e => setTickSpread(e.target.value)} />
              <Button onClick={async () => { setLoading(true); try { setTickResult(await simulate.generateTicks({ generation_id: genId, symbol: tickSymbol, ticks_per_minute: Number(tickRate) || undefined, spread_bps: Number(tickSpread) || undefined })); setMsg('Ticks generated') } catch (e) { setMsg(String(e)) } finally { setLoading(false) } }}>{loading ? 'Running...' : 'Generate Ticks'}</Button>
              {tickResult && (
                <div className="grid grid-cols-2 gap-3 mt-2">
                  <MetricCard label="Ticks Generated" value={String(tickResult.ticks_generated ?? '--')} format="number" color="positive" />
                  <MetricCard label="Output Path" value={String(tickResult.output_path ?? '--')} />
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="inject-signal">
          <Card>
            <CardHeader><CardTitle>Inject Signal into Data</CardTitle></CardHeader>
            <CardContent className="flex flex-col gap-3 max-w-[400px]">
              <Input placeholder="Generation ID" value={genId} onChange={e => setGenId(e.target.value)} />
              <Input placeholder="Strategy type" value={injectStrategy} onChange={e => setInjectStrategy(e.target.value)} />
              <Input placeholder="Signal strength (0-1, default: 0.7)" type="number" step="0.1" value={injectStrength} onChange={e => setInjectStrength(e.target.value)} />
              <Button onClick={async () => { setLoading(true); try { setInjectResult(await simulate.injectSignal({ generation_id: genId, strategy: injectStrategy, strength: Number(injectStrength) || undefined })); setMsg('Signal injected') } catch (e) { setMsg(String(e)) } finally { setLoading(false) } }}>{loading ? 'Running...' : 'Inject Signal'}</Button>
              {injectResult && (
                <div className="grid grid-cols-2 gap-3 mt-2">
                  <MetricCard label="Injected" value={String(injectResult.injected ?? '--')} color={(injectResult.injected as boolean) ? 'positive' : 'negative'} />
                  <MetricCard label="Signal Count" value={String(injectResult.signal_count ?? '--')} format="number" />
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="validate-regime">
          <Card>
            <CardHeader><CardTitle>Validate Regime Detection</CardTitle></CardHeader>
            <CardContent className="flex flex-col gap-3 max-w-[400px]">
              <Input placeholder="Generation ID" value={genId} onChange={e => setGenId(e.target.value)} />
              <Input placeholder="Symbol" value={tickSymbol} onChange={e => setTickSymbol(e.target.value)} />
              <Button onClick={async () => { setLoading(true); try { setValidRegimeResult(await simulate.validateRegime({ generation_id: genId, symbol: tickSymbol })); setMsg('Validation complete') } catch (e) { setMsg(String(e)) } finally { setLoading(false) } }}>{loading ? 'Running...' : 'Validate Regime'}</Button>
              {validRegimeResult && (
                <div className="space-y-2">
                  <Badge variant={validRegimeResult.passed ? 'success' : 'destructive'}>{validRegimeResult.passed ? 'PASSED' : 'FAILED'}</Badge>
                  <Table><TableHeader><TableRow><TableHead>Check</TableHead><TableHead>Result</TableHead></TableRow></TableHeader><TableBody>
                    <TableRow><TableCell>Regime Persistence</TableCell><TableCell><Badge variant={validRegimeResult.regime_persistence?.passed ? 'success' : 'destructive'} size="sm">{validRegimeResult.regime_persistence?.passed ? 'Pass' : 'Fail'}</Badge></TableCell></TableRow>
                    <TableRow><TableCell>Coverage</TableCell><TableCell><Badge variant={validRegimeResult.coverage?.passed ? 'success' : 'destructive'} size="sm">{validRegimeResult.coverage?.passed ? 'Pass' : 'Fail'}</Badge></TableCell></TableRow>
                    <TableRow><TableCell>Signal Quality</TableCell><TableCell><Badge variant={validRegimeResult.signal_quality?.passed ? 'success' : 'destructive'} size="sm">{validRegimeResult.signal_quality?.passed ? 'Pass' : 'Fail'}</Badge></TableCell></TableRow>
                  </TableBody></Table>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
