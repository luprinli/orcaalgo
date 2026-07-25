import { useState, useEffect } from 'react'
import { submitOptimizationRun, getOptimizationStatus, getOptimizationResults, listOptimizationRuns, OptimizeConfig, OptimizationResult, OptimizationStatus } from '../api/optimize'
import { Card, CardHeader, CardTitle, CardContent } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '../components/ui/select'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/table'
import MetricCard from '../components/MetricCard'

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
    <div className="flex flex-col gap-6">
      <Card>
        <CardHeader>
          <CardTitle>Optimization</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <Label>Strategy</Label>
              <Select value={strategyId} onValueChange={setStrategyId}>
                <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="intraday_mr">Intraday Mean Reversion</SelectItem>
                  <SelectItem value="trend_following">Trend Following</SelectItem>
                  <SelectItem value="opening_range_breakout">Opening Range Breakout</SelectItem>
                  <SelectItem value="grid_trading">Grid Trading</SelectItem>
                  <SelectItem value="session_scalp">Session Scalp</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>Symbol</Label>
              <Input type="text" value={symbol} onChange={e => setSymbol(e.target.value)} />
            </div>
            <div>
              <Label>Objective</Label>
              <Select value={objective} onValueChange={v => setObjective(v as OptimizeConfig['objective'])}>
                <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="sharpe">Sharpe Ratio</SelectItem>
                  <SelectItem value="sortino">Sortino Ratio</SelectItem>
                  <SelectItem value="profit_factor">Profit Factor</SelectItem>
                  <SelectItem value="win_rate">Win Rate</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>Max Combinations</Label>
              <Input type="number" value={trials} onChange={e => setTrials(Number(e.target.value))} min={1} />
            </div>
            <div>
              <Label>Train Years</Label>
              <Input type="number" value={trainYears} onChange={e => setTrainYears(Number(e.target.value))} min={1} />
            </div>
            <div>
              <Label>Test Years</Label>
              <Input type="number" value={testYears} onChange={e => setTestYears(Number(e.target.value))} min={1} />
            </div>
            <div>
              <Label>Step Months</Label>
              <Input type="number" value={stepMonths} onChange={e => setStepMonths(Number(e.target.value))} min={1} />
            </div>
            <div>
              <Label>Capital</Label>
              <Input type="number" value={capital} onChange={e => setCapital(Number(e.target.value))} min={1000} />
            </div>
          </div>

          {Object.keys(params).length > 0 && (
            <div className="mt-4">
              <h3 className="text-sm mb-2" style={{ color: 'var(--foreground)' }}>Parameter Ranges</h3>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead className="text-right">Min</TableHead>
                    <TableHead className="text-right">Max</TableHead>
                    <TableHead className="text-right">Step</TableHead>
                    <TableHead className="text-right">Default</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {Object.entries(params).map(([name, def]) => (
                    <TableRow key={name}>
                      <TableCell>{name}</TableCell>
                      <TableCell className="text-right">{def.min}</TableCell>
                      <TableCell className="text-right">{def.max}</TableCell>
                      <TableCell className="text-right">{def.step}</TableCell>
                      <TableCell className="text-right">{def.default}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}

          <Button className="mt-5 px-6 py-2.5" onClick={handleRun} disabled={loading}>
            {loading ? 'Running...' : 'Run Optimization'}
          </Button>
        </CardContent>
      </Card>

      {status && status.status === 'running' && (
        <Card>
          <CardHeader>
            <CardTitle>Progress</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold" style={{ color: 'var(--accent-text)' }}>
              {status.progress ?? 0}%
            </div>
            <div className="relative h-2 rounded mt-3" style={{ background: 'var(--bg-card)' }}>
              <div className="h-full rounded transition-all duration-1000" style={{ width: `${status.progress ?? 0}%`, background: 'var(--accent)' }} />
            </div>
            {status.elapsed_seconds && <div className="text-xs mt-1" style={{ color: 'var(--muted-foreground)' }}>Elapsed: {status.elapsed_seconds}s</div>}
          </CardContent>
        </Card>
      )}

      {result && (
        <Card>
          <CardHeader>
            <CardTitle>Results</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-3">
              <MetricCard label="Best Metric" value={result.best_metric?.toFixed(4) ?? '-'} color="positive" />
              <MetricCard label="Total Trials" value={result.trials?.length ?? (result.trials ? 1 : 0)} format="number" />
              {/* eslint-disable @typescript-eslint/no-explicit-any */}
              <MetricCard label="Avg OOS Sharpe" value={(result as any).avg_oos_sharpe?.toFixed(3) ?? '-'} />
              <MetricCard label="Windows Passed" value={`${(result as any).windows_passed ?? '-'} / ${(result as any).windows_total ?? '-'}`} />
              {/* eslint-enable @typescript-eslint/no-explicit-any */}
            </div>
            {result.best_params && Object.keys(result.best_params).length > 0 && (
              <div className="mt-4">
                <h3 className="text-sm mb-2" style={{ color: 'var(--foreground)' }}>Best Parameters</h3>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Parameter</TableHead>
                      <TableHead className="text-right">Value</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {Object.entries(result.best_params).map(([k, v]) => (
                      <TableRow key={k}>
                        <TableCell>{k}</TableCell>
                        <TableCell className="text-right font-bold" style={{ color: 'var(--trading-success)' }}>{v.toFixed(4)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
