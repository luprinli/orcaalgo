import { useState, useEffect, useCallback } from 'react'
import { settings } from '../api/client'
import type { AppSettings } from '../types/api'

export default function SettingsPage() {
  const [cfg, setCfg] = useState<AppSettings | null>(null)
  const [loading, setLoading] = useState(true)
  const [msg, setMsg] = useState('')
  const [saving, setSaving] = useState(false)

  const fetchSettings = useCallback(async () => {
    try {
      const s = await settings.get()
      setCfg(s)
    } catch { /* ignore */ } finally { setLoading(false) }
  }, [])

  useEffect(() => { fetchSettings() }, [fetchSettings])

  const handleSave = async () => {
    if (!cfg) return
    setSaving(true)
    try { await settings.update(cfg); setMsg('Settings saved') }
    catch (err) { setMsg(err instanceof Error ? err.message : 'Save failed') }
    finally { setSaving(false) }
  }

  const riskSettings = (cfg?.risk as Record<string, unknown>) ?? {}
  const generalSettings = (cfg?.general as Record<string, unknown>) ?? {}

  const updateRisk = (key: string, value: unknown) => {
    setCfg(p => p ? { ...p, risk: { ...p.risk as Record<string, unknown>, [key]: value } } : null)
  }

  const updateGeneral = (key: string, value: unknown) => {
    setCfg(p => p ? { ...p, general: { ...p.general as Record<string, unknown>, [key]: value } } : null)
  }

  if (loading) return <div className="card"><p className="text-muted">Loading settings...</p></div>

  return (
    <div>
      <div className="flex-between mb-4"><h1 style={{ margin: 0 }}>Settings</h1></div>

      {msg && <p className="text-muted mb-4" style={{ color: msg.includes('fail') ? 'var(--danger)' : 'var(--success)' }}>{msg}</p>}

      <div style={{ display: 'flex', flexDirection: 'column', gap: 16, maxWidth: 600 }}>
        <div className="card">
          <h2>Risk Parameters</h2>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {[
              { k: 'max_daily_loss_pct', l: 'Max Daily Loss %', def: 5 },
              { k: 'max_drawdown_pct', l: 'Max Drawdown %', def: 10 },
              { k: 'kelly_fraction', l: 'Kelly Fraction', def: 0.25 },
              { k: 'max_capital_per_trade_pct', l: 'Max Capital Per Trade %', def: 25 },
            ].map(f => (
              <div key={f.k} className="flex-between">
                <label className="text-muted">{f.l}</label>
                <input className="input" style={{ width: 120 }} type="number" step="0.01"
                  value={riskSettings[f.k] != null ? String(riskSettings[f.k]) : ''}
                  onChange={e => updateRisk(f.k, parseFloat(e.target.value) || f.def)}
                  placeholder={String(f.def)} />
              </div>
            ))}
          </div>
        </div>

        <div className="card">
          <h2>General</h2>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            <div className="flex-between">
              <label className="text-muted">Default Timeframe</label>
              <select className="input" style={{ width: 120 }}
                value={String(generalSettings.default_timeframe ?? '15m')}
                onChange={e => updateGeneral('default_timeframe', e.target.value)}>
                {['1m', '5m', '15m', '30m', '1h', '4h', '1d'].map(t => <option key={t} value={t}>{t}</option>)}
              </select>
            </div>
            <div className="flex-between">
              <label className="text-muted">Default Capital</label>
              <input className="input" style={{ width: 150 }} type="number"
                value={generalSettings.default_capital != null ? String(generalSettings.default_capital) : ''}
                onChange={e => updateGeneral('default_capital', parseFloat(e.target.value) || 100000)}
                placeholder="100000" />
            </div>
            <div className="flex-between">
              <label className="text-muted">Data Source</label>
              <select className="input" style={{ width: 150 }}
                value={String(generalSettings.data_source ?? 'alpaca')}
                onChange={e => updateGeneral('data_source', e.target.value)}>
                {['alpaca', 'stooq', 'mock'].map(d => <option key={d} value={d}>{d}</option>)}
              </select>
            </div>
          </div>
        </div>

        <button className="btn btn-primary" onClick={handleSave} disabled={saving} style={{ justifyContent: 'center' }}>
          {saving ? 'Saving...' : 'Save Settings'}
        </button>
      </div>
    </div>
  )
}
