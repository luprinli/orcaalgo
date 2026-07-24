import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import toast from 'react-hot-toast'
import { request } from '../api/client'
import { PageHeader, PageSection } from '../components/layout'

interface Credential {
  id: string
  name: string
  provider_type: string
  api_key?: string
  secret_key?: string
  created_at?: string
  last_rotated?: string
}

export default function CredentialManagement() {
  const { t } = useTranslation()
  const [credentials, setCredentials] = useState<Credential[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [name, setName] = useState('')
  const [providerType, setProviderType] = useState('alpaca')
  const [apiKey, setApiKey] = useState('')
  const [secretKey, setSecretKey] = useState('')
  const [saving, setSaving] = useState(false)
  const [msg, setMsg] = useState('')

  const fetchCredentials = async () => {
    try {
      setLoading(true)
      setError(null)
      const data = await request<Credential[]>('GET', '/api/v1/credentials')
      setCredentials(data ?? [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load credentials')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchCredentials() }, [])

  const createCredential = async () => {
    if (!name || !apiKey) { setMsg('Name and API Key are required'); return }
    setSaving(true)
    setMsg('')
    try {
      await request('POST', '/api/v1/credentials', { name, provider_type: providerType, api_key: apiKey, secret_key: secretKey || undefined })
      setShowForm(false)
      setName('')
      setApiKey('')
      setSecretKey('')
      setProviderType('alpaca')
      toast.success('Credential created')
      fetchCredentials()
    } catch (err) {
      setMsg(`Create failed: ${err instanceof Error ? err.message : 'Unknown error'}`)
      toast.error('Failed to create credential')
    } finally {
      setSaving(false)
    }
  }

  const rotateCredential = async (id: string) => {
    try {
      await request('PUT', `/api/v1/credentials/${id}/rotate`)
      toast.success('Credential rotated')
      fetchCredentials()
    } catch (err) {
      toast.error('Rotation failed')
    }
  }

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        title={t('sidebar:nav.credentials', 'Credentials')}
        subtitle={t('credentials:description', 'Manage API keys and broker credentials.')}
        actions={
          <button className="btn btn-primary" onClick={() => setShowForm(!showForm)}>
            {showForm ? 'Cancel' : '+ New Credential'}
          </button>
        }
      />

      {error && (
        <div className="card" style={{ borderLeftColor: 'var(--danger)', borderLeftWidth: 4, borderLeftStyle: 'solid' }}>
          <p style={{ color: 'var(--danger-text)' }}>{error}</p>
          <button className="btn btn-outline mt-2" onClick={fetchCredentials}>Retry</button>
        </div>
      )}

      {showForm && (
        <PageSection title="New Credential">
          <div className="space-y-3">
            <input className="input" placeholder="Credential name" value={name} onChange={e => setName(e.target.value)} />
            <select className="input" value={providerType} onChange={e => setProviderType(e.target.value)}>
              <option value="alpaca">Alpaca</option>
              <option value="ibkr">Interactive Brokers</option>
              <option value="tiingo">Tiingo</option>
              <option value="polygon">Polygon</option>
              <option value="openai">OpenAI</option>
              <option value="custom">Custom</option>
            </select>
            <input className="input" type="password" placeholder="API Key" value={apiKey} onChange={e => setApiKey(e.target.value)} />
            <input className="input" type="password" placeholder="Secret Key (optional)" value={secretKey} onChange={e => setSecretKey(e.target.value)} />
            {msg && <p className="text-sm" style={{ color: 'var(--danger-text)' }}>{msg}</p>}
            <button className="btn btn-primary" onClick={createCredential} disabled={saving}>
              {saving ? 'Creating...' : 'Create Credential'}
            </button>
          </div>
        </PageSection>
      )}

      <PageSection title="Stored Credentials">
        {loading ? (
          <p className="text-muted">Loading credentials...</p>
        ) : credentials.length === 0 ? (
          <p className="text-muted">No credentials stored. Add your first API key above.</p>
        ) : (
          <div style={{ overflowX: 'auto' }}>
            <table>
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Provider</th>
                  <th>API Key</th>
                  <th>Created</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {credentials.map(c => (
                  <tr key={c.id}>
                    <td className="font-medium text-white">{c.name}</td>
                    <td><span className="badge badge-warn">{c.provider_type}</span></td>
                    <td className="text-muted font-mono text-xs">{c.api_key ? `${c.api_key.slice(0, 8)}...` : '—'}</td>
                    <td className="text-muted text-xs">{c.created_at ? new Date(c.created_at).toLocaleDateString() : '—'}</td>
                    <td>
                      <button className="btn btn-outline text-xs" onClick={() => rotateCredential(c.id)}>Rotate</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </PageSection>
    </div>
  )
}
