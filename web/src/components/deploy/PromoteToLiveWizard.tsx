import { useState, useEffect } from 'react'
import { strategies, accounts } from '../../api/client'
import type { DeployStrategyResponse, PreflightResponse } from '../../types/api'

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

const GATES = { sharpeMin: 1.0, maxDDMax: 8.0, passProbMin: 80, profitFactorMin: 1.5 }

export default function PromoteToLiveWizard({
  strategyName,
  backtestId,
  sharpe,
  maxDD,
  passProb,
  profitFactor,
  onClose,
  onDeployed,
}: Props) {
  const [step, setStep] = useState(1)
  const [account, setAccount] = useState('paper')
  const [capitalPct, setCapitalPct] = useState(25)
  const [deploying, setDeploying] = useState(false)
  const [preflight, setPreflight] = useState<PreflightResponse | null>(null)
  const [preflightLoading, setPreflightLoading] = useState(false)
  const [gateOverride, setGateOverride] = useState(false)
  const [deployResult, setDeployResult] = useState<DeployStrategyResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [accountBalances, setAccountBalances] = useState<Record<string, number>>({})

  useEffect(() => {
    (async () => {
      try {
        const list = await accounts.list()
        const bal: Record<string, number> = {}
        const accts: unknown[] = (list as any)?.accounts ?? (Array.isArray(list) ? list : [])
        for (const a of accts as any[]) {
          if (a?.account_id && (a?.balance ?? a?.equity)) {
            bal[a.account_id] = a.balance ?? a.equity ?? 0
          }
        }
        setAccountBalances(bal)
      } catch { /* balance fetch is optional */ }
    })()
  }, [])

  const checks = [
    { label: 'Sharpe Ratio', required: `≥ ${GATES.sharpeMin}`, actual: sharpe.toFixed(2), passed: sharpe >= GATES.sharpeMin },
    { label: 'Max Drawdown', required: `≤ ${GATES.maxDDMax}%`, actual: `${maxDD.toFixed(1)}%`, passed: maxDD <= GATES.maxDDMax },
    { label: 'Pass Probability', required: `≥ ${GATES.passProbMin}%`, actual: `${passProb.toFixed(0)}%`, passed: passProb >= GATES.passProbMin },
    { label: 'Profit Factor', required: `≥ ${GATES.profitFactorMin}`, actual: profitFactor.toFixed(2), passed: profitFactor >= GATES.profitFactorMin },
  ]
  const allPass = gateOverride || checks.every((c) => c.passed)

  const handleRunPreflight = async () => {
    setPreflightLoading(true)
    setError(null)
    try {
      const result = await strategies.preflight()
      setPreflight(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Pre-flight check failed')
    } finally {
      setPreflightLoading(false)
    }
  }

  const handleDeploy = async () => {
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
    } finally {
      setDeploying(false)
    }
  }

  return (
    <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000 }}>
      <div style={{ background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 'var(--radius)', padding: 24, maxWidth: 520, width: '100%', maxHeight: '80vh', overflow: 'auto' }}>
        <div className="flex-between mb-3">
          <h2 style={{ margin: 0 }}>Promote: {strategyName}</h2>
          <button onClick={onClose} style={{ border: 'none', background: 'none', color: 'var(--muted-foreground)', cursor: 'pointer', fontSize: 18 }}>✕</button>
        </div>

        <div className="flex gap-1 mb-3">
          {[1, 2, 3].map((i) => (
            <div key={i} style={{ flex: 1, height: 3, borderRadius: 2, background: step >= i ? 'var(--accent)' : 'var(--border)' }} />
          ))}
        </div>

        {error && (
          <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4" style={{ background: 'rgba(218,54,51,.1)', border: '1px solid var(--trading-danger)', marginBottom: 12, padding: 8 }}>
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
                <div
                  style={{
                    padding: 8,
                    borderRadius: 6,
                    background: preflight.passed ? 'rgba(63,185,80,.1)' : 'rgba(218,54,51,.1)',
                    color: preflight.passed ? 'var(--trading-success)' : 'var(--trading-danger)',
                    marginBottom: 8,
                  }}
                >
                  {preflight.passed
                    ? `✓ All checks passed (${preflight.passed_count}/${preflight.checks.length})`
                    : `✗ ${preflight.failed_count} check(s) failed`}
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
            <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4" style={{ background: 'var(--bg-card)', marginBottom: 12, padding: 10 }}>
              <p className="text-sm font-semibold mb-1">Strategy: {strategyName}</p>
              <p className="text-xs text-muted-foreground">Backtest: {backtestId}</p>
            </div>
            {deployResult ? (
              <div className="mt-2">
                <div style={{ padding: 8, borderRadius: 6, background: 'rgba(63,185,80,.1)', color: 'var(--trading-success)', marginBottom: 8 }}>
                  ✓ Deployed successfully to {deployResult.account_id}
                </div>
                <div className="text-muted" style={{ fontSize: 13 }}>
                  Allocation: {deployResult.capital_allocation_pct}% · Backtest: {deployResult.backtest_id}
                </div>
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
                <button className="btn btn-primary" onClick={handleDeploy} disabled={deploying} style={{ justifyContent: 'center' }}>
                  {deploying ? 'Deploying...' : 'Deploy to Live'}
                </button>
              </div>
            )}
          </div>
        )}

        <div className="flex-between mt-3">
          <button className="btn btn-outline" onClick={onClose}>
            {deployResult ? 'Close' : 'Cancel'}
          </button>
          <div className="flex gap-2">
            {step > 1 && !deployResult && (
              <button className="btn btn-outline" onClick={() => setStep((s) => s - 1)}>Back</button>
            )}
            {step < 3 && !deployResult && (
              <button className="btn btn-primary" onClick={() => setStep((s) => s + 1)} disabled={step === 1 && !allPass}>
                Next
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
