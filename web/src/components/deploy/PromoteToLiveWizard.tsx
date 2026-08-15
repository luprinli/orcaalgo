import { useState, useEffect, useCallback } from 'react'
import { strategies, accounts, orchestrator, strategyStatus, symbols as symbolsApi, backtests } from '../../api/client'
import type { DeployStrategyResponse, PreflightResponse, Strategy } from '../../types/api'
import OrchestrationProgressBar from '../backtest/OrchestrationProgressBar'
import { useOrchestrationPoll } from '../../hooks/useOrchestrationPoll'

interface Props {
  strategyName: string
  backtestId: string
  sharpe: number
  maxDD: number
  passProb: number
  profitFactor: number
  onClose: () => void
  onDeployed: () => void
}

import { TIMEFRAME_OPTIONS } from '../../data/constants'

const GATES = { sharpeMin: 1.0, maxDDMax: 8.0, passProbMin: 80, profitFactorMin: 1.5 }

interface OrchRow {
  strategy_id: string
  symbol: string
  timeframe: string
}

export default function PromoteToLiveWizard({
  strategyName, backtestId, sharpe, maxDD, passProb, profitFactor, onClose, onDeployed,
}: Props) {
  const [step, setStep] = useState(1)
  const [account, setAccount] = useState('paper')
  const [capitalPct, setCapitalPct] = useState(25)
  const [deploying, setDeploying] = useState(false)
  const [preflight, setPreflight] = useState<PreflightResponse | null>(null)
  const [preflightLoading, setPreflightLoading] = useState(false)
  const [gateOverride, setGateOverride] = useState(false)
  const [benchmarkPassed, setBenchmarkPassed] = useState<boolean | null>(null)
  const [deployResult, setDeployResult] = useState<DeployStrategyResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [accountBalances, setAccountBalances] = useState<Record<string, number>>({})

  const [orchMode, setOrchMode] = useState(false)
  const [orchRows, setOrchRows] = useState<OrchRow[]>(() => [
    { strategy_id: strategyName, symbol: "", timeframe: "4h" },
  ])
  const [orchRebalance, setOrchRebalance] = useState("20")
  const [orchKelly, setOrchKelly] = useState("0.25")
  const [orchCorrBrake, setOrchCorrBrake] = useState(false)
  const [orchStartDate, setOrchStartDate] = useState("2024-01-01")
  const [orchEndDate, setOrchEndDate] = useState("2025-12-31")
  const [orchCapital, setOrchCapital] = useState("100000")
  const [orchFriction, setOrchFriction] = useState<"realistic" | "idealized">("realistic")
  const [orchMaxPct, setOrchMaxPct] = useState("2")
  const [availableStrategies, setAvailableStrategies] = useState<Strategy[]>([])
  const [availableSymbols, setAvailableSymbols] = useState<string[]>([])
  const [orchRunId, setOrchRunId] = useState<string | null>(null)
  const [orchStartTs, setOrchStartTs] = useState(0)

  const orchPoll = useOrchestrationPoll(orchRunId, () => {
    setDeployResult({ account_id: 'orchestration', capital_allocation_pct: parseFloat(orchCapital) ? 50 : 0, backtest_id: orchRunId ?? '' } as any)
    setOrchRunId(null)
  })

  useEffect(() => {
    (async () => {
      try {
        const list = await accounts.list()
        const bal: Record<string, number> = {}
        const accts: unknown[] = (list as any)?.accounts ?? (Array.isArray(list) ? list : [])
        for (const a of accts as any[]) {
          if (a?.account_id && (a?.balance ?? a?.equity)) bal[a.account_id] = a.balance ?? a.equity ?? 0
        }
        setAccountBalances(bal)
      } catch { /* optional */ }
    })()
  }, [])

  useEffect(() => {
    (async () => {
      try {
        const r = await strategies.list()
        setAvailableStrategies(r.strategies ?? [])
      } catch { /* optional */ }
    })()
  }, [])

  useEffect(() => {
    (async () => {
      try {
        const r = await symbolsApi.list() as unknown as { symbols?: { ticker: string }[] }
        setAvailableSymbols((r.symbols ?? []).map(s => s.ticker))
      } catch { /* optional */ }
    })()
  }, [])

  useEffect(() => {
    let active = true
    backtests.benchmarkEval(backtestId).then(b => {
      if (active) setBenchmarkPassed(b.passed)
    }).catch(() => {
      if (active) setBenchmarkPassed(null)
    })
    return () => { active = false }
  }, [backtestId])

  const checks = [
    { label: 'Sharpe Ratio', required: `≥ ${GATES.sharpeMin}`, actual: sharpe.toFixed(2), passed: sharpe >= GATES.sharpeMin },
    { label: 'Max Drawdown', required: `≤ ${GATES.maxDDMax}%`, actual: `${maxDD.toFixed(1)}%`, passed: maxDD <= GATES.maxDDMax },
    { label: 'Pass Probability', required: `≥ ${GATES.passProbMin}%`, actual: `${passProb.toFixed(0)}%`, passed: passProb >= GATES.passProbMin },
    { label: 'Profit Factor', required: `≥ ${GATES.profitFactorMin}`, actual: profitFactor.toFixed(2), passed: profitFactor >= GATES.profitFactorMin },
  ]
  const allPass = gateOverride || (checks.every((c) => c.passed) && benchmarkPassed !== false)

  const handleRunPreflight = async () => {
    setPreflightLoading(true)
    setError(null)
    try {
      const result = await strategies.preflight()
      setPreflight(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Pre-flight check failed')
    } finally { setPreflightLoading(false) }
  }

  const handleSingleDeploy = async () => {
    setDeploying(true)
    setError(null)
    try {
      const result = await strategies.deploy({
        strategy_name: strategyName,
        backtest_id: backtestId,
        account_id: account,
        capital_allocation_pct: capitalPct,
      })
      setDeployResult(result)
      onDeployed()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Deployment failed')
    } finally { setDeploying(false) }
  }

  const handleOrchDeploy = async () => {
    setDeploying(true)
    setError(null)
    try {
      const filtered = orchRows.filter(r => r.strategy_id)
      const config = {
        strategies: filtered.map(r => ({
          strategy_id: r.strategy_id,
          symbol: r.symbol,
          timeframe: r.timeframe,
        })),
        start_date: orchStartDate,
        end_date: orchEndDate,
        initial_capital: parseFloat(orchCapital) || 100000,
        rebalance_bars: parseInt(orchRebalance, 10) || 20,
        kelly_fraction: parseFloat(orchKelly) || 0.25,
        max_position_pct: parseFloat(orchMaxPct) / 100 || 0.02,
        enable_correlation_brake: orchCorrBrake,
        correlation_threshold: 0.6,
        friction_model: orchFriction,
      }
      const result = await orchestrator.submit(config as any)

      const seen = new Set<string>()
      for (const r of filtered) {
        if (!seen.has(r.strategy_id)) {
          seen.add(r.strategy_id)
          try { await strategyStatus.promote(r.strategy_id, "Orchestration deploy", 50, result.run_id) } catch { /* non-critical */ }
        }
      }

      setOrchRunId(result.run_id)
      setOrchStartTs(Date.now())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Orchestration deployment failed')
    } finally { setDeploying(false) }
  }

  const addOrchRow = useCallback(() => {
    setOrchRows(prev => [...prev, {
      strategy_id: availableStrategies[0]?.id ?? "grid_trading",
      symbol: "JPN225",
      timeframe: "1h",
    }])
  }, [availableStrategies])

  const removeOrchRow = useCallback((i: number) => {
    if (orchRows.length <= 1) return
    setOrchRows(prev => prev.filter((_, idx) => idx !== i))
  }, [orchRows])

  const updateOrchRow = useCallback((i: number, f: keyof OrchRow, v: string) => {
    setOrchRows(prev => prev.map((r, idx) => idx === i ? { ...r, [f]: v } : r))
  }, [])

  const fillSupplementary = useCallback(() => {
    const s = availableSymbols
    const recommended = [
      { strategy_id: "grid_trading", symbol: s[0] || "", timeframe: "1h" },
      { strategy_id: "rsi2_reversion", symbol: s[0] || "", timeframe: "1h" },
    ]
    setOrchRows(prev => [...prev, ...recommended.filter(r => !prev.some(p => p.strategy_id === r.strategy_id))])
  }, [availableSymbols])

  const showOrchDeployed = deployResult && orchMode

  return (
    <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000 }}>
      <div style={{ background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 'var(--radius)', padding: 24, maxWidth: orchMode ? 620 : 520, width: '100%', maxHeight: '80vh', overflow: 'auto' }}>
        <div className="flex-between mb-3">
          <h2 style={{ margin: 0 }}>{orchMode ? 'Deploy Orchestration Set' : `Promote: ${strategyName}`}</h2>
          <button onClick={onClose} style={{ border: 'none', background: 'none', color: 'var(--muted-foreground)', cursor: 'pointer', fontSize: 18 }}>✕</button>
        </div>

        <div className="flex gap-1 mb-3">
          {[1, 2, 3].map((i) => (
            <div key={i} style={{ flex: 1, height: 3, borderRadius: 2, background: step >= i ? 'var(--accent)' : 'var(--border)' }} />
          ))}
        </div>

        {error && (
          <div style={{ background: 'rgba(218,54,51,.1)', border: '1px solid var(--trading-danger)', marginBottom: 12, padding: 8, borderRadius: 6 }}>
            <span style={{ color: 'var(--trading-danger)', fontSize: 13 }}>{error}</span>
          </div>
        )}

        {step === 1 && (
          <div>
            <h3>Step 1: Quality Gates</h3>
            {checks.map((c) => (
              <div key={c.label} className="flex-between" style={{ padding: '8px 0', borderBottom: '1px solid var(--border)' }}>
                <span>{c.label}</span>
                <span className="text-muted">{c.required}</span>
                <span style={{ color: c.passed ? 'var(--trading-success)' : 'var(--trading-danger)', fontWeight: 600 }}>
                  {c.passed ? '✓' : '✗'} {c.actual}
                </span>
              </div>
            ))}
            <label className="flex gap-2 mt-3" style={{ alignItems: 'center' }}>
              <input type="checkbox" checked={gateOverride} onChange={(e) => setGateOverride(e.target.checked)} />
              Override gates (skip quality checks)
            </label>
            <div className="flex-between" style={{ padding: '8px 0', borderBottom: '1px solid var(--border)' }}>
              <span>Benchmark Filter</span>
              <span className="text-muted">must pass (vs. SPY)</span>
              <span style={{ color: benchmarkPassed === false ? 'var(--trading-danger)' : benchmarkPassed === true ? 'var(--trading-success)' : 'var(--trading-warning, #d29922)', fontWeight: 600 }}>
                {benchmarkPassed === null ? '…' : benchmarkPassed ? '✓ pass' : '✗ fail'}
              </span>
            </div>
          </div>
        )}

        {step === 2 && (
          <div>
            <h3>Step 2: Pre-Flight Checklist</h3>
            {!preflight ? (
              <button className="btn btn-outline" onClick={handleRunPreflight} disabled={preflightLoading}>
                {preflightLoading ? 'Running...' : 'Run Pre-Flight'}
              </button>
            ) : (
              <div className="mt-2">
                <div style={{ padding: 8, borderRadius: 6, background: preflight.passed ? 'rgba(63,185,80,.1)' : 'rgba(218,54,51,.1)', color: preflight.passed ? 'var(--trading-success)' : 'var(--trading-danger)', marginBottom: 8 }}>
                  {preflight.passed ? `✓ All checks passed (${preflight.passed_count}/${preflight.checks.length})` : `✗ ${preflight.failed_count} check(s) failed`}
                </div>
                {preflight.checks.map((c) => (
                  <div key={c.name} className="flex-between" style={{ fontSize: 12, padding: '2px 0' }}>
                    <span>{c.name}</span>
                    <span style={{ color: c.status === 'pass' ? 'var(--trading-success)' : c.status === 'warn' ? 'var(--trading-warning, #d29922)' : 'var(--trading-danger)' }}>
                      {c.status === 'pass' ? '✓' : c.status === 'warn' ? '△' : '✗'} {c.message}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {step === 3 && (
          <div>
            <h3>Step 3: Deploy</h3>

            {!showOrchDeployed && !orchPoll.status.includes('completed') && (
              <label className="flex gap-2 mb-3" style={{ alignItems: 'center', cursor: 'pointer' }}>
                <input type="checkbox" checked={orchMode} onChange={(e) => setOrchMode(e.target.checked)} />
                <span style={{ fontSize: 13, fontWeight: 600 }}>Deploy as orchestration set</span>
              </label>
            )}

            {showOrchDeployed && (
              <div className="mt-2">
                <div style={{ padding: 8, borderRadius: 6, background: 'rgba(63,185,80,.1)', color: 'var(--trading-success)', marginBottom: 8 }}>
                  ✓ Orchestration set deployed
                </div>
                <div className="text-muted" style={{ fontSize: 13 }}>Run ID: {orchRunId ?? (deployResult as any)?.backtest_id?.slice(0, 8)}...</div>
              </div>
            )}

            {orchMode && !showOrchDeployed ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                <div style={{ border: '1px solid var(--border)', borderRadius: 6, padding: 10, background: 'var(--bg-card)' }}>
                  <p className="text-xs font-semibold mb-2">Strategy Pairs ({orchRows.length})</p>
                  {orchRows.map((row, i) => (
                    <div key={i} className="flex gap-2 mb-1.5">
                      <select className="input" style={{ flex: 1, fontSize: 12, padding: '2px 4px' }} value={row.strategy_id} onChange={(e) => updateOrchRow(i, 'strategy_id', e.target.value)}>
                        {availableStrategies.map(s => <option key={s.id} value={s.id}>{s.name || s.id}</option>)}
                        {!availableStrategies.some(s => s.id === strategyName) && <option value={strategyName}>{strategyName}</option>}
                      </select>
                      <select className="input" style={{ width: 80, fontSize: 12, padding: '2px 4px' }} value={row.symbol} onChange={(e) => updateOrchRow(i, 'symbol', e.target.value)}>
                        {availableSymbols.map(s => <option key={s} value={s}>{s}</option>)}
                      </select>
                      <select className="input" style={{ width: 60, fontSize: 12, padding: '2px 4px' }} value={row.timeframe} onChange={(e) => updateOrchRow(i, 'timeframe', e.target.value)}>
                        {TIMEFRAME_OPTIONS.map(t => <option key={t} value={t}>{t}</option>)}
                      </select>
                      {orchRows.length > 1 && (
                        <button onClick={() => removeOrchRow(i)} style={{ border: 'none', background: 'none', color: 'var(--trading-danger)', cursor: 'pointer', fontSize: 14, padding: '0 4px' }}>✕</button>
                      )}
                    </div>
                  ))}
                  <div className="flex gap-2 mt-1">
                    <button className="btn btn-outline" style={{ fontSize: 11, padding: '2px 8px' }} onClick={addOrchRow}>+ Add</button>
                    <button className="btn btn-outline" style={{ fontSize: 11, padding: '2px 8px' }} onClick={fillSupplementary}>+ Recommended</button>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-2" style={{ fontSize: 12 }}>
                  <div>
                    <label className="text-xs">Start Date</label>
                    <input type="date" className="input" style={{ width: '100%', fontSize: 12, padding: '2px 4px' }} value={orchStartDate} onChange={(e) => setOrchStartDate(e.target.value)} />
                  </div>
                  <div>
                    <label className="text-xs">End Date</label>
                    <input type="date" className="input" style={{ width: '100%', fontSize: 12, padding: '2px 4px' }} value={orchEndDate} onChange={(e) => setOrchEndDate(e.target.value)} />
                  </div>
                  <div>
                    <label className="text-xs">Capital</label>
                    <input type="number" className="input" style={{ width: '100%', fontSize: 12, padding: '2px 4px' }} value={orchCapital} onChange={(e) => setOrchCapital(e.target.value)} />
                  </div>
                  <div>
                    <label className="text-xs">Rebalance Bars</label>
                    <input type="number" className="input" style={{ width: '100%', fontSize: 12, padding: '2px 4px' }} value={orchRebalance} onChange={(e) => setOrchRebalance(e.target.value)} />
                  </div>
                  <div>
                    <label className="text-xs">Kelly Fraction</label>
                    <input type="number" step="0.05" min="0" max="1" className="input" style={{ width: '100%', fontSize: 12, padding: '2px 4px' }} value={orchKelly} onChange={(e) => setOrchKelly(e.target.value)} />
                  </div>
                  <div>
                    <label className="text-xs">Max Position: {orchMaxPct}%</label>
                    <input type="range" min={0.5} max={20} step={0.5} style={{ width: '100%' }} value={orchMaxPct} onChange={(e) => setOrchMaxPct(e.target.value)} />
                  </div>
                  <div style={{ display: 'flex', alignItems: 'flex-end', gap: 6 }}>
                    <label className="flex gap-1" style={{ alignItems: 'center', fontSize: 11 }}>
                      <input type="checkbox" checked={orchCorrBrake} onChange={(e) => setOrchCorrBrake(e.target.checked)} />
                      Corr. Brake
                    </label>
                    <select className="input" style={{ fontSize: 11, padding: '2px 4px' }} value={orchFriction} onChange={(e) => setOrchFriction(e.target.value as any)}>
                      <option value="realistic">Realistic</option>
                      <option value="idealized">Idealized</option>
                    </select>
                  </div>
                </div>

                {orchRunId && orchPoll.status !== 'idle' && (
                  <OrchestrationProgressBar status={orchPoll.status} startTime={orchStartTs} />
                )}

                <button className="btn btn-primary" onClick={handleOrchDeploy} disabled={deploying || orchRows.every(r => !r.strategy_id)} style={{ justifyContent: 'center' }}>
                  {deploying ? 'Deploying...' : 'Deploy Orchestration Set'}
                </button>
              </div>
            ) : !orchMode ? (
              <div>
                <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4" style={{ background: 'var(--bg-card)', marginBottom: 12, padding: 10 }}>
                  <p className="text-sm font-semibold mb-1">Strategy: {strategyName}</p>
                  <p className="text-xs text-muted-foreground">Backtest: {backtestId}</p>
                </div>
                {deployResult ? (
                  <div className="mt-2">
                    <div style={{ padding: 8, borderRadius: 6, background: 'rgba(63,185,80,.1)', color: 'var(--trading-success)', marginBottom: 8 }}>
                      ✓ Deployed successfully to {deployResult.account_id}
                    </div>
                    <div className="text-muted" style={{ fontSize: 13 }}>Allocation: {deployResult.capital_allocation_pct}% · Backtest: {deployResult.backtest_id}</div>
                  </div>
                ) : (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                    <select className="input" value={account} onChange={(e) => setAccount(e.target.value)}>
                      <option value="paper">Alpaca Paper</option>
                      <option value="alpaca">Alpaca Live</option>
                    </select>
                    {accountBalances[account] != null && (
                      <div className="text-xs" style={{ color: 'var(--trading-success)' }}>Available: ${accountBalances[account]?.toLocaleString() ?? '--'}</div>
                    )}
                    <label>
                      Capital: {capitalPct}%
                      <input type="range" min={5} max={100} value={capitalPct} onChange={(e) => setCapitalPct(Number(e.target.value))} style={{ width: '100%' }} />
                    </label>
                    <div className="text-muted">Daily Loss 5% · Max DD 10% · Kelly k=0.25</div>
                    <button className="btn btn-primary" onClick={handleSingleDeploy} disabled={deploying} style={{ justifyContent: 'center' }}>
                      {deploying ? 'Deploying...' : 'Deploy to Live'}
                    </button>
                  </div>
                )}
              </div>
            ) : null}
          </div>
        )}

        <div className="flex-between mt-3">
          <button className="btn btn-outline" onClick={onClose}>{deployResult ? 'Close' : 'Cancel'}</button>
          <div className="flex gap-2">
            {step > 1 && !deployResult && (
              <button className="btn btn-outline" onClick={() => setStep((s) => s - 1)}>Back</button>
            )}
            {step < 3 && !deployResult && (
              <button className="btn btn-primary" onClick={() => setStep((s) => s + 1)} disabled={step === 1 && !allPass}>Next</button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
