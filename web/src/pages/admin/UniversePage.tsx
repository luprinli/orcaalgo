import { useState, useEffect, useCallback } from 'react'
import { universe } from '../../api/client'

export default function UniversePage() {
  /* eslint-disable @typescript-eslint/no-explicit-any */
  const [symbols, setSymbols] = useState<any[]>([])
  const [configs, setConfigs] = useState<any[]>([])
  /* eslint-enable @typescript-eslint/no-explicit-any */
  const [loading, setLoading] = useState(true)
  const [msg, setMsg] = useState('')
  const [overrideTicker, setOverrideTicker] = useState('')
  const [overrideAction, setOverrideAction] = useState<'add' | 'remove'>('add')
  const [showConfigForm, setShowConfigForm] = useState(false)
  const [configForm, setConfigForm] = useState({ name: '', profile_id: 'default', asset_class_filters: '{}', dynamic_triggers: '{}' })

  const fetchUniverse = useCallback(async () => {
    try {
      const [cur, cfg] = await Promise.all([
        universe.current(),
        universe.configs(),
      ])
      setSymbols(cur.symbols ?? [])
      setConfigs(cfg.configs ?? [])
    } catch { /* ignore */ }
    finally { setLoading(false) }
  }, [])

  useEffect(() => { fetchUniverse() }, [fetchUniverse])

  const handleOverride = async () => {
    if (!overrideTicker) return
    try {
      await universe.override(overrideTicker.toUpperCase(), overrideAction)
      setMsg(`${overrideAction === 'add' ? 'Added' : 'Removed'} ${overrideTicker.toUpperCase()}`)
      setOverrideTicker('')
      fetchUniverse()
    } catch (err) { setMsg(err instanceof Error ? err.message : 'Override failed') }
  }

  const handleRefresh = async () => {
    try {
      const res = await universe.refresh()
      setMsg(`Universe refreshed: ${res.total} symbols`)
      fetchUniverse()
    } catch (err) { setMsg(err instanceof Error ? err.message : 'Refresh failed') }
  }

  const createConfig = async () => {
    try {
      await universe.createConfig({
        name: configForm.name, profile_id: configForm.profile_id,
        asset_class_filters: JSON.parse(configForm.asset_class_filters),
        dynamic_triggers: JSON.parse(configForm.dynamic_triggers),
      })
      setMsg('Config created')
      setShowConfigForm(false)
      fetchUniverse()
    } catch (err) { setMsg(err instanceof Error ? err.message : 'Failed') }
  }

  const activateConfig = async (id: string) => {
    try {
      await universe.activateConfig(id)
      setMsg('Config activated')
      fetchUniverse()
    } catch (err) { setMsg(err instanceof Error ? err.message : 'Failed') }
  }

  if (loading) return <div className="card"><p className="text-muted">Loading universe...</p></div>

  return (
    <div>
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>Universe Management</h1>
        <button className="btn btn-outline" onClick={handleRefresh}>Refresh</button>
      </div>

      {msg && <p className="text-muted mb-2" style={{ color: msg.includes('fail') ? 'var(--danger)' : 'var(--success)' }}>{msg}</p>}

      <div className="grid-2 mb-4">
        <div className="card">
          <h2>Current Universe ({symbols.length})</h2>
          <div className="flex gap-2 mb-3">
            <input className="input" placeholder="Ticker" value={overrideTicker} onChange={e => setOverrideTicker(e.target.value.toUpperCase())} onKeyDown={e => e.key === 'Enter' && handleOverride()} />
            {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
            <select className="input" style={{ width: 90 }} value={overrideAction} onChange={e => setOverrideAction(e.target.value as any)}>
              <option value="add">Add</option><option value="remove">Remove</option>
            </select>
            <button className="btn btn-primary" onClick={handleOverride} disabled={!overrideTicker}>Go</button>
          </div>
          <div style={{ maxHeight: 400, overflowY: 'auto' }}>
            {symbols.length === 0 ? <p className="text-muted">Empty universe</p> : (
              <table className="data-table">
                <thead><tr><th>Ticker</th><th>Exchange</th><th>Type</th><th>Active</th><th>Price</th></tr></thead>
                <tbody>
                  {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                  {symbols.map((s: any) => (
                    <tr key={s.id}>
                      <td><strong>{s.ticker}</strong></td><td>{s.exchange}</td><td>{s.asset_type}</td>
                      <td><span className={`badge ${s.is_active ? 'badge-ok' : 'badge-err'}`}>{s.is_active ? 'Active' : 'Inactive'}</span></td>
                      <td>{s.last_price ? `$${s.last_price}` : '--'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>

        <div className="card">
          <div className="flex-between mb-3">
            <h2 style={{ margin: 0 }}>Configs</h2>
            <button className="btn btn-primary" onClick={() => setShowConfigForm(true)}>+ Config</button>
          </div>

          {showConfigForm && (
            <div className="mb-4" style={{ display: 'flex', flexDirection: 'column', gap: 10, paddingBottom: 12, borderBottom: '1px solid var(--border)' }}>
              <input className="input" placeholder="Name" value={configForm.name} onChange={e => setConfigForm(p => ({ ...p, name: e.target.value }))} />
              <input className="input" placeholder="Profile ID" value={configForm.profile_id} onChange={e => setConfigForm(p => ({ ...p, profile_id: e.target.value }))} />
              <div><label className="text-muted">Asset Class Filters (JSON)</label><textarea className="input" style={{ minHeight: 60, fontFamily: 'monospace', fontSize: 12 }} value={configForm.asset_class_filters} onChange={e => setConfigForm(p => ({ ...p, asset_class_filters: e.target.value }))} /></div>
              <div><label className="text-muted">Dynamic Triggers (JSON)</label><textarea className="input" style={{ minHeight: 60, fontFamily: 'monospace', fontSize: 12 }} value={configForm.dynamic_triggers} onChange={e => setConfigForm(p => ({ ...p, dynamic_triggers: e.target.value }))} /></div>
              <div className="flex gap-2">
                <button className="btn btn-primary" onClick={createConfig}>Create</button>
                <button className="btn btn-outline" onClick={() => setShowConfigForm(false)}>Cancel</button>
              </div>
            </div>
          )}

          {configs.length === 0 ? <p className="text-muted">No configs</p> : (
            <table className="data-table">
              <thead><tr><th>Name</th><th>Profile</th><th>Active</th><th>Actions</th></tr></thead>
              <tbody>
                {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                {configs.map((c: any) => (
                  <tr key={c.ID}>
                    <td>{c.Name}</td><td>{c.ProfileID}</td>
                    <td>{c.IsActive ? <span className="badge badge-ok">Active</span> : '—'}</td>
                    <td>{!c.IsActive && <button className="btn btn-outline" style={{ padding: '2px 8px', fontSize: 11 }} onClick={() => activateConfig(c.ID)}>Activate</button>}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  )
}
