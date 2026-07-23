import { useState, useCallback } from 'react'
import { admin } from '../../api/client'
import ConfirmDialog from '../../components/ConfirmDialog'

export default function AdminPage() {
  const [tab, setTab] = useState<'health' | 'users' | 'audit' | 'errors' | 'email' | 'seed'>('health')
  /* eslint-disable @typescript-eslint/no-explicit-any */
  const [health, setHealth] = useState<any>(null)
  const [systemHealth, setSystemHealth] = useState<any>(null)
  const [users, setUsers] = useState<any[]>([])
  const [auditLogs, setAuditLogs] = useState<any[]>([])
  const [errorLogs, setErrorLogs] = useState<any[]>([])
  const [seedInfo, setSeedInfo] = useState<any>(null)
  /* eslint-enable @typescript-eslint/no-explicit-any */
  const [loading, setLoading] = useState(false)
  const [msg, setMsg] = useState('')
  const [emailForm, setEmailForm] = useState({ host: '', port: '587', username: '', password: '', from: '', from_name: '' })
  const [confirmSeed, setConfirmSeed] = useState(false)

  const fetchHealth = useCallback(async () => {
    setLoading(true)
    try {
      const [h, sh] = await Promise.all([admin.health(), admin.systemHealth()])
      setHealth(h)
      setSystemHealth(sh)
    } catch { setMsg('Failed to load health') }
    finally { setLoading(false) }
  }, [])

  const fetchUsers = useCallback(async () => {
    setLoading(true)
    try {
      const res = await admin.users()
      setUsers(res.users ?? [])
    } catch { setMsg('Failed to load users') }
    finally { setLoading(false) }
  }, [])

  const fetchAudit = useCallback(async (component?: string) => {
    setLoading(true)
    try {
      const res = await admin.auditLogs({ component, limit: 50 })
      setAuditLogs(res)
    } catch { setMsg('Failed to load audit logs') }
    finally { setLoading(false) }
  }, [])

  const fetchErrors = useCallback(async (params?: { severity?: string; component?: string }) => {
    setLoading(true)
    try {
      const res = await admin.errorLogs({ ...params, limit: 50 })
      setErrorLogs(res)
    } catch { setMsg('Failed to load error logs') }
    finally { setLoading(false) }
  }, [])

  const fetchSeedInfo = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/v1/admin/info').then(r => r.json())
      setSeedInfo(res)
    } catch { /* ignore */ }
    finally { setLoading(false) }
  }, [])

  const handleSeed = () => {
    setConfirmSeed(true)
  }

  const confirmSeedDatabase = async () => {
    setConfirmSeed(false)
    try {
      const res = await admin.seed(true) as { seeded?: boolean }
      setMsg(res?.seeded ? 'Database seeded successfully' : 'Seed failed')
      fetchSeedInfo()
    } catch (err) {
      setMsg(err instanceof Error ? err.message : 'Seed failed')
    }
  }

  const handleUserToggle = async (userId: string, enable: boolean) => {
    try {
      if (enable) {
        const res = await fetch(`/api/v1/admin/users/${userId}/enable`, { method: 'PUT' }).then(r => r.json())
        setMsg(res?.enabled ? 'User enabled' : 'Failed')
      } else {
        const res = await fetch(`/api/v1/admin/users/${userId}/disable`, { method: 'PUT' }).then(r => r.json())
        setMsg(res?.disabled ? 'User disabled' : 'Failed')
      }
      fetchUsers()
    } catch (err) {
      setMsg(err instanceof Error ? err.message : 'Toggle failed')
    }
  }

  const handleTestEmail = async () => {
    try {
      const res = await fetch('/api/v1/admin/email/test', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(emailForm) }).then(r => r.json())
      setMsg(res.ok ? 'Email test successful' : `Email test failed: ${res.error}`)
    } catch (err) {
      setMsg(err instanceof Error ? err.message : 'Email test failed')
    }
  }

  const handleSaveEmail = async () => {
    try {
      const res = await fetch('/api/v1/admin/email/config', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(emailForm) }).then(r => r.json())
      setMsg(res.ok ? 'Email config saved' : 'Failed')
    } catch (err) {
      setMsg(err instanceof Error ? err.message : 'Save failed')
    }
  }

  return (
    <div>
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>Admin</h1>
      </div>

      <div className="flex gap-2 flex-wrap mb-4">
        {(['health', 'users', 'audit', 'errors', 'email', 'seed'] as const).map(t => (
          <button key={t} className={`btn ${tab === t ? 'btn-primary' : 'btn-outline'}`} onClick={() => { setTab(t); setMsg('') }}>
            {t.charAt(0).toUpperCase() + t.slice(1)}
          </button>
        ))}
      </div>

      {msg && <p className="text-muted mb-2" style={{ color: msg.includes('fail') ? 'var(--danger)' : 'var(--success)' }}>{msg}</p>}

      {tab === 'health' && (
        <div>
          <button className="btn btn-outline mb-3" onClick={fetchHealth}>{loading ? 'Loading...' : 'Refresh Health'}</button>
          {health && (
            <div className="card mb-4">
              <h2>Health</h2>
              <div className="metric-grid">
                <div className="metric-card"><div className="metric-label">Status</div><div className="metric-value" style={{ color: health.healthy ? 'var(--success)' : 'var(--danger)' }}>{health.healthy ? 'Healthy' : 'Unhealthy'}</div></div>
                <div className="metric-card"><div className="metric-label">Database</div><div className="metric-value" style={{ fontSize: 14 }}>{health.components?.database ?? '--'}</div></div>
              </div>
              <p className="text-muted mt-2">{health.timestamp ? new Date(health.timestamp).toLocaleString() : ''}</p>
            </div>
          )}
          {systemHealth && (
            <div className="card">
              <h2>System Health</h2>
              <div className="metric-grid">
                {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                {Object.entries(systemHealth.checks ?? {}).map(([key, val]: [string, any]) => (
                  <div key={key} className="metric-card">
                    <div className="metric-label">{key}</div>
                    <div className="metric-value" style={{ fontSize: 14, color: val.status === 'ok' ? 'var(--success)' : 'var(--danger)' }}>{val.status}</div>
                    <div className="text-muted">{val.message}</div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {tab === 'users' && (
        <div>
          <button className="btn btn-outline mb-3" onClick={fetchUsers}>{loading ? 'Loading...' : 'Refresh Users'}</button>
          <div className="card">
            {users.length === 0 ? <p className="text-muted">No users</p> : (
              <table className="data-table">
                <thead><tr><th>Username</th><th>Email</th><th>Roles</th><th>Verified</th><th>2FA</th><th>Active</th><th>Actions</th></tr></thead>
                <tbody>
                  {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                  {users.map((u: any) => (
                    <tr key={u.id}>
                      <td>{u.username}</td><td>{u.email}</td>
                      <td>{(u.roles ?? []).join(', ')}</td>
                      <td><span className={`badge ${u.is_verified ? 'badge-ok' : 'badge-err'}`}>{u.is_verified ? 'Yes' : 'No'}</span></td>
                      <td><span className={`badge ${u.totp_enabled ? 'badge-ok' : 'badge-err'}`}>{u.totp_enabled ? 'On' : 'Off'}</span></td>
                      <td><span className={`badge ${u.is_active ? 'badge-ok' : 'badge-err'}`}>{u.is_active ? 'Active' : 'Disabled'}</span></td>
                      <td><button className="btn btn-outline" style={{ padding: '2px 8px', fontSize: 11 }} onClick={() => handleUserToggle(u.id, !u.is_active)}>{u.is_active ? 'Disable' : 'Enable'}</button></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}

      {tab === 'audit' && (
        <div>
          <button className="btn btn-outline mb-3" onClick={() => fetchAudit()}>{loading ? 'Loading...' : 'Refresh Audit'}</button>
          <div className="card" style={{ maxHeight: 600, overflowY: 'auto' }}>
            {auditLogs.length === 0 ? <p className="text-muted">No audit logs</p> : (
              <table className="data-table">
                <thead><tr><th>Time</th><th>Action</th><th>Resource</th><th>User</th></tr></thead>
                <tbody>
                  {auditLogs.map((l: { id?: string; created_at?: string; action?: string; resource_type?: string; resource_id?: string; user_id?: string }, i: number) => (
                    <tr key={l.id ?? i}>
                      <td style={{ fontSize: 11 }}>{l.created_at ? new Date(l.created_at).toLocaleString() : '--'}</td>
                      <td>{l.action ?? '--'}</td>
                      <td>{(l.resource_type ?? '') + (l.resource_id ? ': ' + l.resource_id : '')}</td>
                      <td>{l.user_id?.slice(0, 8) ?? '--'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}

      {tab === 'errors' && (
        <div>
          <button className="btn btn-outline mb-3" onClick={() => fetchErrors()}>{loading ? 'Loading...' : 'Refresh Errors'}</button>
          <div className="card" style={{ maxHeight: 600, overflowY: 'auto' }}>
            {errorLogs.length === 0 ? <p className="text-muted">No error logs</p> : (
              <table className="data-table">
                <thead><tr><th>Time</th><th>Severity</th><th>Component</th><th>Message</th><th>Resolved</th></tr></thead>
                <tbody>
                  {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                  {errorLogs.map((e: any, i: number) => (
                    <tr key={e.id ?? i}>
                      <td style={{ fontSize: 11 }}>{e.timestamp ? new Date(e.timestamp).toLocaleString() : '--'}</td>
                      <td><span className={`badge ${e.severity === 'error' || e.severity === 'critical' ? 'badge-err' : 'badge-warn'}`}>{e.severity}</span></td>
                      <td>{e.component}</td><td>{e.message}</td>
                      <td>{e.resolved != null ? (e.resolved ? 'Yes' : 'No') : '--'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}

      {tab === 'email' && (
        <div className="card" style={{ maxWidth: 450 }}>
          <h2>Email Configuration</h2>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {(['host', 'port', 'username', 'password', 'from', 'from_name'] as const).map(f => (
              <div key={f}><label className="text-muted">{f.replace('_', ' ')}</label>
                <input className="input" type={f === 'password' ? 'password' : 'text'} value={emailForm[f]} onChange={e => setEmailForm(p => ({ ...p, [f]: e.target.value }))} />
              </div>
            ))}
            <div className="flex gap-2">
              <button className="btn btn-outline" onClick={handleTestEmail}>Test</button>
              <button className="btn btn-primary" onClick={handleSaveEmail}>Save</button>
            </div>
          </div>
        </div>
      )}

      {tab === 'seed' && (
        <div>
          <div className="flex gap-2 mb-3">
            <button className="btn btn-outline" onClick={fetchSeedInfo}>{loading ? 'Loading...' : 'Info'}</button>
            <button className="btn btn-danger" onClick={handleSeed}>Seed Database</button>
          </div>
          {seedInfo && (
            <div className="card">
              <h2>Database Status</h2>
              <div className="metric-grid">
                {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                {Object.entries(seedInfo).filter(([k]) => k !== 'admin_credentials').map(([key, val]: [string, any]) => (
                  <div key={key} className="metric-card">
                    <div className="metric-label">{key.replace(/_/g, ' ')}</div><div className="metric-value" style={{ fontSize: 16 }}>{val}</div>
                  </div>
                ))}
              </div>
              {seedInfo.admin_credentials && (
                <div className="mt-3" style={{ padding: 8, background: 'rgba(210,153,34,.1)', borderRadius: 6 }}>
                  <span className="text-muted">Admin: {seedInfo.admin_credentials.username} / {seedInfo.admin_credentials.password}</span>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {confirmSeed && (
        <ConfirmDialog
          title="Seed Database"
          message="This will reset the database to its initial state. All existing data will be replaced. Are you sure?"
          confirmLabel="Seed Database"
          danger
          onConfirm={confirmSeedDatabase}
          onCancel={() => setConfirmSeed(false)}
        />
      )}
    </div>
  )
}
