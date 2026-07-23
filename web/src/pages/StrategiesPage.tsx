import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { strategies } from '../api/client'
import ConfirmDialog from '../components/ConfirmDialog'
import { STRATEGY_CATALOG, type CatalogWithInstance } from '../data/strategyCatalog'
import type { Strategy } from '../types/api'

export default function StrategiesPage() {
  const nav = useNavigate()
  const [dbList, setDbList] = useState<Strategy[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [msg, setMsg] = useState('')
  const [showCreate, setShowCreate] = useState(false)
  const [form, setForm] = useState({ name: '', type: 'intraday_mr', params: '{}', enabled: false })
  const [creating, setCreating] = useState(false)
  const [viewMode, setViewMode] = useState<'catalog' | 'instances'>('catalog')
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)

  useEffect(() => {
    strategies.list()
      .then((res) => setDbList(res.strategies ?? []))
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [])

  const dbByType: Record<string, Strategy[]> = {}
  for (const s of dbList) {
    if (!dbByType[s.type]) dbByType[s.type] = []
    dbByType[s.type].push(s)
  }

  const catalog: CatalogWithInstance[] = STRATEGY_CATALOG.map((entry) => {
    const instances = dbByType[entry.typeKey] ?? []
    return { ...entry, dbInstance: instances.length > 0 ? instances[0] : null }
  })

  const toggleEnabled = async (id: string, current: boolean) => {
    try {
      await strategies.update(id, { enabled: !current })
      setDbList((prev) => prev.map((s) => (s.id === id ? { ...s, enabled: !current } : s)))
      setMsg('Updated')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Update failed')
    }
  }

  const handleCreate = async () => {
    if (!form.name) return
    setCreating(true)
    setError(null)
    try {
      const params = JSON.parse(form.params || '{}')
      const res = await strategies.create({ name: form.name, type: form.type, parameters: params, enabled: form.enabled })
      setMsg(`"${res.name}" created`)
      setShowCreate(false)
      setForm({ name: '', type: 'intraday_mr', params: '{}', enabled: false })
      const refreshed = await strategies.list()
      setDbList(refreshed.strategies ?? [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Create failed')
    } finally {
      setCreating(false)
    }
  }

  const handleDelete = async (id: string) => {
    setConfirmDelete(id)
  }

  const confirmDeleteStrategy = async () => {
    if (!confirmDelete) return
    try {
      await strategies.delete(confirmDelete)
      setDbList((prev) => prev.filter((s) => s.id !== confirmDelete))
      setMsg('Deleted')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Delete failed')
    } finally {
      setConfirmDelete(null)
    }
  }

  const handleClone = async (id: string) => {
    try {
      const clone = await strategies.clone(id)
      setMsg(`Cloned as "${clone.name}"`)
      const refreshed = await strategies.list()
      setDbList(refreshed.strategies ?? [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Clone failed')
    }
  }

  if (loading) return <div className="card"><p className="text-muted">Loading strategies...</p></div>
  if (error) return <div className="card"><p style={{ color: 'var(--danger)' }}>{error}</p></div>

  return (
    <div>
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>Strategies</h1>
        <div className="flex gap-2">
          <button className={`btn ${viewMode === 'catalog' ? 'btn-primary' : 'btn-outline'}`} onClick={() => setViewMode('catalog')}>
            Catalog ({catalog.length})
          </button>
          <button className={`btn ${viewMode === 'instances' ? 'btn-primary' : 'btn-outline'}`} onClick={() => setViewMode('instances')}>
            Instances ({dbList.length})
          </button>
          <button className="btn btn-primary" onClick={() => setShowCreate(true)}>+ New Strategy</button>
        </div>
      </div>

      {msg && <p className="text-muted mb-2" style={{ fontSize: 13, color: 'var(--success)' }}>{msg}</p>}

      {showCreate && (
        <div className="card mb-4">
          <h2>Create Strategy Instance</h2>
          <div className="grid-2">
            <div>
              <label className="text-muted">Display Name</label>
              <input className="input" placeholder="my_strategy" value={form.name} onChange={e => setForm(p => ({ ...p, name: e.target.value }))} />
            </div>
            <div>
              <label className="text-muted">Strategy Type</label>
              <select className="input" value={form.type} onChange={e => setForm(p => ({ ...p, type: e.target.value }))}>
                {STRATEGY_CATALOG.filter(c => c.inEngine).map(c => (
                  <option key={c.typeKey} value={c.typeKey}>{c.displayName}</option>
                ))}
              </select>
            </div>
          </div>
          <div className="mt-2">
            <label className="text-muted">Parameters (JSON)</label>
            <textarea className="input" style={{ minHeight: 60, fontFamily: 'monospace', fontSize: 12, resize: 'vertical' }} placeholder='{ "lookback": 20 }' value={form.params} onChange={e => setForm(p => ({ ...p, params: e.target.value }))} />
          </div>
          <label className="flex gap-2 mt-2" style={{ alignItems: 'center' }}>
            <input type="checkbox" checked={form.enabled} onChange={e => setForm(p => ({ ...p, enabled: e.target.checked }))} />
            Enable immediately
          </label>
          <div className="flex gap-2 mt-2">
            <button className="btn btn-primary" onClick={handleCreate} disabled={creating || !form.name}>{creating ? 'Creating...' : 'Create'}</button>
            <button className="btn btn-outline" onClick={() => setShowCreate(false)}>Cancel</button>
          </div>
        </div>
      )}

      {/* CATALOG VIEW */}
      {viewMode === 'catalog' && (
        <div className="card">
          <div style={{ overflowX: 'auto' }}>
            <table className="data-table">
              <thead>
                <tr>
                  <th>Strategy</th>
                  <th>Type Key</th>
                  <th>Engine</th>
                  <th>GKR</th>
                  <th>DB Instance</th>
                  <th>Parameters</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {catalog.map((c) => (
                  <tr key={c.typeKey} style={{ opacity: c.inEngine || c.dbInstance ? 1 : 0.5 }}>
                    <td><strong>{c.displayName}</strong></td>
                    <td style={{ fontFamily: 'monospace', fontSize: 11 }}>{c.typeKey}</td>
                    <td>
                      <span className={`badge ${c.inEngine ? 'badge-ok' : 'badge-err'}`}>
                        {c.inEngine ? 'Registered' : '—'}
                      </span>
                    </td>
                    <td>
                      {c.hasGkrFile ? <span className="badge badge-ok">YAML</span> : <span className="text-muted">—</span>}
                    </td>
                    <td>
                      {c.dbInstance ? (
                        <span>
                          <span className={`badge ${c.dbInstance.enabled ? 'badge-ok' : 'badge-err'}`}>
                            {c.dbInstance.enabled ? 'Active' : 'Disabled'}
                          </span>
                          <span className="text-muted" style={{ marginLeft: 4, fontSize: 11 }}>{c.dbInstance.name}</span>
                        </span>
                      ) : (
                        <span className="text-muted">None</span>
                      )}
                    </td>
                    <td style={{ fontSize: 11, color: 'var(--text-secondary)', maxWidth: 180 }}>
                      {c.paramDefs}
                    </td>
                    <td>
                      <div className="flex gap-1">
                        {c.dbInstance ? (
                          <>
                            <button className="btn btn-outline" style={{ padding: '2px 8px', fontSize: 11 }} onClick={() => toggleEnabled(c.dbInstance!.id, c.dbInstance!.enabled)}>
                              {c.dbInstance.enabled ? 'Disable' : 'Enable'}
                            </button>
                            <button className="btn btn-outline" style={{ padding: '2px 8px', fontSize: 11 }} onClick={() => handleClone(c.dbInstance!.id)}>
                              Clone
                            </button>
                            <button className="btn btn-outline" style={{ padding: '2px 8px', fontSize: 11, color: 'var(--danger)' }} onClick={() => handleDelete(c.dbInstance!.id)}>
                              Del
                            </button>
                          </>
                        ) : c.inEngine ? (
                          <button className="btn btn-outline" style={{ padding: '2px 8px', fontSize: 11 }} onClick={() => {
                            setForm({ name: c.displayName, type: c.typeKey, params: '{}', enabled: false })
                            setShowCreate(true)
                          }}>Create</button>
                        ) : null}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* INSTANCES VIEW */}
      {viewMode === 'instances' && (
        <>
          {dbList.length === 0 ? (
            <div className="card"><p className="text-muted">No strategy instances. Switch to Catalog view and click "Create" on a strategy type.</p></div>
          ) : (
            <div className="card">
              <div style={{ overflowX: 'auto' }}>
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>Name</th>
                      <th>Type</th>
                      <th>Parameters</th>
                      <th>Enabled</th>
                      <th>Created</th>
                      <th>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {dbList.map((s) => (
                      <tr key={s.id}>
                        <td style={{ cursor: 'pointer' }} onClick={() => nav(`/strategies/${s.id}`)}>
                          <strong>{s.name}</strong>
                        </td>
                        <td style={{ fontFamily: 'monospace', fontSize: 11 }}>{s.type}</td>
                        <td style={{ fontSize: 11, fontFamily: 'monospace', maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {s.parameters ? Object.entries(s.parameters).map(([k, v]) => `${k}:${v}`).join(', ') : '—'}
                        </td>
                        <td>
                          <span className={`badge ${s.enabled ? 'badge-ok' : 'badge-err'}`}>
                            {s.enabled ? 'Enabled' : 'Disabled'}
                          </span>
                        </td>
                        <td style={{ fontSize: 11, color: 'var(--text-secondary)' }}>
                          {s.created_at ? new Date(s.created_at).toLocaleDateString() : '—'}
                        </td>
                        <td>
                          <div className="flex gap-1">
                            <button className="btn btn-outline" style={{ padding: '2px 8px', fontSize: 11 }} onClick={() => toggleEnabled(s.id, s.enabled)}>Toggle</button>
                            <button className="btn btn-outline" style={{ padding: '2px 8px', fontSize: 11 }} onClick={() => handleClone(s.id)}>Clone</button>
                            <button className="btn btn-outline" style={{ padding: '2px 8px', fontSize: 11, color: 'var(--danger)' }} onClick={() => handleDelete(s.id)}>Del</button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </>
      )}

      {confirmDelete && (
        <ConfirmDialog
          title="Delete Strategy"
          message="Delete this strategy instance? This action cannot be undone."
          confirmLabel="Delete"
          danger
          onConfirm={confirmDeleteStrategy}
          onCancel={() => setConfirmDelete(null)}
        />
      )}
    </div>
  )
}
