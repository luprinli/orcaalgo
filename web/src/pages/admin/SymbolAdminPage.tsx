import { useState, useEffect, useCallback } from 'react'
import ConfirmDialog from '../../components/ConfirmDialog'

export default function SymbolAdminPage() {
  /* eslint-disable @typescript-eslint/no-explicit-any */
  const [symbols, setSymbols] = useState<any[]>([])
  const [providers, setProviders] = useState<any[]>([])
  const [credentials, setCredentials] = useState<any[]>([])
  /* eslint-enable @typescript-eslint/no-explicit-any */
  const [tab, setTab] = useState<'symbols' | 'providers' | 'credentials'>('symbols')
  const [loading, setLoading] = useState(true)
  const [msg, setMsg] = useState('')
  const [showSymbolForm, setShowSymbolForm] = useState(false)
  const [symbolForm, setSymbolForm] = useState({ ticker: '', exchange: 'NASDAQ', asset_type: 'equity', tick_size: 0.01, lot_size: 1 })
  const [showProviderForm, setShowProviderForm] = useState(false)
  const [providerForm, setProviderForm] = useState({ name: '', type: 'broker', driver: 'alpaca', config: '{}' })
  const [showCredForm, setShowCredForm] = useState(false)
  const [credForm, setCredForm] = useState({ provider_id: '', key_label: '', api_key: '', api_secret: '' })
  const [confirmDelete, setConfirmDelete] = useState<{ type: 'symbol' | 'provider'; id: number | string; label: string } | null>(null)

  const fetchSymbols = useCallback(async () => {
    try {
      const res = await fetch('/api/v1/symbols').then(r => r.json())
      setSymbols(res.symbols ?? [])
    } catch { /* ignore */ }
  }, [])

  const fetchProviders = useCallback(async () => {
    try {
      const res = await fetch('/api/v1/providers').then(r => r.json())
      setProviders(res.providers ?? [])
    } catch { /* ignore */ }
  }, [])

  const fetchCredentials = useCallback(async () => {
    try {
      const res = await fetch('/api/v1/credentials').then(r => r.json())
      setCredentials(res.credentials ?? [])
    } catch { /* ignore */ }
  }, [])

  useEffect(() => {
    setLoading(true)
    Promise.all([fetchSymbols(), fetchProviders(), fetchCredentials()]).finally(() => setLoading(false))
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const createSymbol = async () => {
    try {
      await fetch('/api/v1/symbols', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(symbolForm) })
      setMsg(`Symbol ${symbolForm.ticker} created`)
      setShowSymbolForm(false)
      setSymbolForm({ ticker: '', exchange: 'NASDAQ', asset_type: 'equity', tick_size: 0.01, lot_size: 1 })
      fetchSymbols()
    } catch (err) { setMsg(err instanceof Error ? err.message : 'Failed') }
  }

  const deleteSymbol = (id: number) => {
    setConfirmDelete({ type: 'symbol', id, label: `symbol #${id}` })
  }

  const confirmDeleteAction = async () => {
    if (!confirmDelete) return
    try {
      if (confirmDelete.type === 'symbol') {
        await fetch(`/api/v1/symbols/${confirmDelete.id}`, { method: 'DELETE' })
        fetchSymbols()
      } else {
        await fetch(`/api/v1/providers/${confirmDelete.id}`, { method: 'DELETE' })
        fetchProviders()
      }
      setConfirmDelete(null)
    } catch { setMsg('Delete failed'); setConfirmDelete(null) }
  }

  const createProvider = async () => {
    try {
      const config = JSON.parse(providerForm.config)
      await fetch('/api/v1/providers', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ...providerForm, config }) })
      setMsg('Provider created')
      setShowProviderForm(false)
      fetchProviders()
    } catch (err) { setMsg(err instanceof Error ? err.message : 'Failed') }
  }

  const deleteProvider = (id: string) => {
    setConfirmDelete({ type: 'provider', id, label: `provider ${id}` })
  }

  const testProvider = async (id: string) => {
    try {
      const res = await fetch(`/api/v1/providers/${id}/test`, { method: 'POST' }).then(r => r.json())
      setMsg(res.reachable ? `Provider ${id} reachable (${res.latency_ms}ms)` : `Provider ${id} unreachable: ${res.error}`)
    } catch (err) { setMsg(err instanceof Error ? err.message : 'Test failed') }
  }

  const createCredential = async () => {
    try {
      await fetch('/api/v1/credentials', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(credForm) })
      setMsg('Credential stored')
      setShowCredForm(false)
      setCredForm({ provider_id: '', key_label: '', api_key: '', api_secret: '' })
      fetchCredentials()
    } catch (err) { setMsg(err instanceof Error ? err.message : 'Failed') }
  }

  if (loading) return <div className="card"><p className="text-muted">Loading...</p></div>

  return (
    <div>
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>Symbol &amp; Provider Admin</h1>
      </div>
      <div className="flex gap-2 mb-4">
        {(['symbols', 'providers', 'credentials'] as const).map(t => (
          <button key={t} className={`btn ${tab === t ? 'btn-primary' : 'btn-outline'}`} onClick={() => setTab(t)}>
            {t.charAt(0).toUpperCase() + t.slice(1)}
          </button>
        ))}
      </div>

      {msg && <p className="text-muted mb-2" style={{ color: msg.includes('fail') ? 'var(--danger)' : 'var(--success)' }}>{msg}</p>}

      {tab === 'symbols' && (
        <div>
          <button className="btn btn-primary mb-3" onClick={() => setShowSymbolForm(true)}>+ Symbol</button>
          {showSymbolForm && (
            <div className="card mb-4" style={{ maxWidth: 400 }}>
              <h2>New Symbol</h2>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                <input className="input" placeholder="Ticker" value={symbolForm.ticker} onChange={e => setSymbolForm(p => ({ ...p, ticker: e.target.value.toUpperCase() }))} />
                <input className="input" placeholder="Exchange" value={symbolForm.exchange} onChange={e => setSymbolForm(p => ({ ...p, exchange: e.target.value }))} />
                <select className="input" value={symbolForm.asset_type} onChange={e => setSymbolForm(p => ({ ...p, asset_type: e.target.value }))}>
                  <option value="equity">Equity</option><option value="forex">Forex</option><option value="crypto">Crypto</option><option value="futures">Futures</option>
                </select>
                <div className="grid-2">
                  <div><label className="text-muted">Tick Size</label><input className="input" type="number" step="0.001" value={symbolForm.tick_size} onChange={e => setSymbolForm(p => ({ ...p, tick_size: parseFloat(e.target.value) }))} /></div>
                  <div><label className="text-muted">Lot Size</label><input className="input" type="number" value={symbolForm.lot_size} onChange={e => setSymbolForm(p => ({ ...p, lot_size: parseFloat(e.target.value) }))} /></div>
                </div>
                <div className="flex gap-2">
                  <button className="btn btn-primary" onClick={createSymbol}>Create</button>
                  <button className="btn btn-outline" onClick={() => setShowSymbolForm(false)}>Cancel</button>
                </div>
              </div>
            </div>
          )}
          <div className="card">
            {symbols.length === 0 ? <p className="text-muted">No symbols</p> : (
              <table className="data-table">
                <thead><tr><th>Ticker</th><th>Exchange</th><th>Type</th><th>Active</th><th>Last Price</th><th>Actions</th></tr></thead>
                <tbody>
                  {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                  {symbols.map((s: any) => (
                    <tr key={s.id}>
                      <td><strong>{s.ticker}</strong></td><td>{s.exchange}</td><td>{s.asset_type}</td>
                      <td><span className={`badge ${s.is_active ? 'badge-ok' : 'badge-err'}`}>{s.is_active ? 'Active' : 'Inactive'}</span></td>
                      <td>{s.last_price ? `$${s.last_price}` : '--'}</td>
                      <td><button className="btn btn-outline" style={{ padding: '2px 8px', fontSize: 11, color: 'var(--danger)' }} onClick={() => deleteSymbol(s.id)}>Delete</button></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}

      {tab === 'providers' && (
        <div>
          <button className="btn btn-primary mb-3" onClick={() => setShowProviderForm(true)}>+ Provider</button>
          {showProviderForm && (
            <div className="card mb-4" style={{ maxWidth: 400 }}>
              <h2>New Provider</h2>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                <input className="input" placeholder="Name" value={providerForm.name} onChange={e => setProviderForm(p => ({ ...p, name: e.target.value }))} />
                <select className="input" value={providerForm.type} onChange={e => setProviderForm(p => ({ ...p, type: e.target.value }))}>
                  <option value="broker">Broker</option><option value="data">Data</option><option value="llm">LLM</option>
                </select>
                <input className="input" placeholder="Driver (alpaca, polygon, etc)" value={providerForm.driver} onChange={e => setProviderForm(p => ({ ...p, driver: e.target.value }))} />
                <div><label className="text-muted">Config (JSON)</label><textarea className="input" style={{ minHeight: 80, fontFamily: 'monospace', fontSize: 12 }} value={providerForm.config} onChange={e => setProviderForm(p => ({ ...p, config: e.target.value }))} /></div>
                <div className="flex gap-2">
                  <button className="btn btn-primary" onClick={createProvider}>Create</button>
                  <button className="btn btn-outline" onClick={() => setShowProviderForm(false)}>Cancel</button>
                </div>
              </div>
            </div>
          )}
          <div className="card">
            {providers.length === 0 ? <p className="text-muted">No providers</p> : (
              <table className="data-table">
                <thead><tr><th>ID</th><th>Name</th><th>Type</th><th>Driver</th><th>Enabled</th><th>Actions</th></tr></thead>
                <tbody>
                  {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                  {providers.map((p: any) => (
                    <tr key={p.id}>
                      <td style={{ fontFamily: 'monospace', fontSize: 11 }}>{p.id}</td><td>{p.name}</td><td>{p.type}</td><td>{p.driver}</td>
                      <td><span className={`badge ${p.is_enabled ? 'badge-ok' : 'badge-err'}`}>{p.is_enabled ? 'Yes' : 'No'}</span></td>
                      <td><div className="flex gap-1">
                        <button className="btn btn-outline" style={{ padding: '2px 8px', fontSize: 11 }} onClick={() => testProvider(p.id)}>Test</button>
                        <button className="btn btn-outline" style={{ padding: '2px 8px', fontSize: 11, color: 'var(--danger)' }} onClick={() => deleteProvider(p.id)}>Delete</button>
                      </div></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}

      {tab === 'credentials' && (
        <div>
          <button className="btn btn-primary mb-3" onClick={() => setShowCredForm(true)}>+ Credential</button>
          {showCredForm && (
            <div className="card mb-4" style={{ maxWidth: 400 }}>
              <h2>Store Credential</h2>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                <input className="input" placeholder="Provider ID" value={credForm.provider_id} onChange={e => setCredForm(p => ({ ...p, provider_id: e.target.value }))} />
                <input className="input" placeholder="Key Label" value={credForm.key_label} onChange={e => setCredForm(p => ({ ...p, key_label: e.target.value }))} />
                <input className="input" placeholder="API Key" value={credForm.api_key} onChange={e => setCredForm(p => ({ ...p, api_key: e.target.value }))} />
                <input className="input" placeholder="API Secret" type="password" value={credForm.api_secret} onChange={e => setCredForm(p => ({ ...p, api_secret: e.target.value }))} />
                <div className="flex gap-2">
                  <button className="btn btn-primary" onClick={createCredential}>Store</button>
                  <button className="btn btn-outline" onClick={() => setShowCredForm(false)}>Cancel</button>
                </div>
              </div>
            </div>
          )}
          <div className="card">
            {credentials.length === 0 ? <p className="text-muted">No credentials stored</p> : (
              <table className="data-table">
                <thead><tr><th>Provider</th><th>Key Label</th><th>Actions</th></tr></thead>
                <tbody>
                  {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                  {credentials.map((c: any) => (
                    <tr key={c.id}>
                      <td>{c.provider_id}</td><td>{c.key_label}</td>
                      <td><button className="btn btn-outline" style={{ padding: '2px 8px', fontSize: 11 }} onClick={async () => { await fetch(`/api/v1/credentials/${c.id}/rotate`, { method: 'PUT' }); setMsg('Credential rotated') }}>Rotate</button></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}

      {confirmDelete && (
        <ConfirmDialog
          title={`Delete ${confirmDelete.type}`}
          message={`Delete ${confirmDelete.label}? This action cannot be undone.`}
          confirmLabel="Delete"
          danger
          onConfirm={confirmDeleteAction}
          onCancel={() => setConfirmDelete(null)}
        />
      )}
    </div>
  )
}
