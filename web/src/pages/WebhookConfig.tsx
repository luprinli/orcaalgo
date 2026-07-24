import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import toast from 'react-hot-toast'
import { settings } from '../api/client'
import { PageHeader, PageSection } from '../components/layout'

export default function WebhookConfig() {
  const { t } = useTranslation()
  const [url, setUrl] = useState('')
  const [secret, setSecret] = useState('')
  const [events, setEvents] = useState<string[]>(['trade', 'signal', 'risk'])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [msg, setMsg] = useState('')

  const eventOptions = [
    { key: 'trade', label: 'Trade Executions' },
    { key: 'signal', label: 'Strategy Signals' },
    { key: 'risk', label: 'Risk Alerts' },
    { key: 'pnl', label: 'P&L Updates' },
    { key: 'regime', label: 'Regime Changes' },
  ]

  useEffect(() => {
    settings.get()
      .then(cfg => {
        const wh = (cfg as any)?.webhook
        if (wh) {
          setUrl(wh.url || '')
          setSecret(wh.secret || '')
          setEvents(wh.events || ['trade', 'signal', 'risk'])
        }
      })
      .catch(() => setMsg('Could not load webhook settings'))
      .finally(() => setLoading(false))
  }, [])

  const save = async () => {
    setSaving(true)
    setMsg('')
    try {
      await settings.update({ webhook: { url, secret: secret || undefined, events } } as any)
      setMsg('Webhook settings saved.')
      toast.success('Webhook settings saved')
    } catch (err) {
      setMsg(`Save failed: ${err instanceof Error ? err.message : 'Unknown error'}`)
      toast.error('Failed to save webhook settings')
    } finally {
      setSaving(false)
    }
  }

  const testFire = async () => {
    if (!url) { setMsg('Enter a webhook URL first'); return }
    setTesting(true)
    setMsg('')
    try {
      const resp = await fetch(url, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ event: 'test', timestamp: new Date().toISOString(), source: 'OrcaAlgo' }) }, )
      if (resp.ok) {
        setMsg('Test webhook sent successfully.')
        toast.success('Test webhook sent')
      } else {
        setMsg(`Webhook returned status ${resp.status}`)
        toast.error(`Webhook returned ${resp.status}`)
      }
    } catch (err) {
      setMsg(`Test failed: ${err instanceof Error ? err.message : 'Connection error'}`)
      toast.error('Webhook test failed')
    } finally {
      setTesting(false)
    }
  }

  if (loading) return <div className="p-6"><PageHeader title={t('sidebar:nav.webhooks', 'Webhooks')} /><div className="card"><p className="text-muted">Loading...</p></div></div>

  return (
    <div className="p-6 space-y-6">
      <PageHeader title={t('sidebar:nav.webhooks', 'Webhooks')} subtitle={t('webhooks:description', 'Configure webhook endpoints for external trade signals and notifications.')} />

      {msg && (
        <div className={`card ${msg.includes('failed') || msg.includes('error') || msg.includes('Could not') ? 'border-l-4' : ''}`} style={(msg.includes('failed') || msg.includes('error') || msg.includes('Could not')) ? { borderLeftColor: 'var(--danger)' } : { borderLeftColor: 'var(--success)' }}>
          <p style={{ color: (msg.includes('failed') || msg.includes('error') || msg.includes('Could not')) ? 'var(--danger-text)' : 'var(--success)' }}>{msg}</p>
        </div>
      )}

      <PageSection title="Webhook Endpoint">
        <div className="space-y-4">
          <div>
            <label className="text-xs text-slate-400 block mb-1">Webhook URL</label>
            <input className="input" type="url" value={url} onChange={e => setUrl(e.target.value)} placeholder="https://hooks.example.com/orca" />
          </div>

          <div>
            <label className="text-xs text-slate-400 block mb-1">Secret (optional)</label>
            <input className="input" type="password" value={secret} onChange={e => setSecret(e.target.value)} placeholder="HMAC signing secret" />
          </div>

          <div className="flex gap-2">
            <button className="btn btn-primary" onClick={save} disabled={saving}>
              {saving ? 'Saving...' : 'Save'}
            </button>
            <button className="btn btn-outline" onClick={testFire} disabled={testing || !url}>
              {testing ? 'Sending...' : 'Test Fire'}
            </button>
          </div>
        </div>
      </PageSection>

      <PageSection title="Event Subscriptions">
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {eventOptions.map(ev => (
            <label key={ev.key} className="flex gap-2" style={{ alignItems: 'center' }}>
              <input type="checkbox" checked={events.includes(ev.key)} onChange={e => setEvents(prev => e.target.checked ? [...prev, ev.key] : prev.filter(k => k !== ev.key))} />
              <span className="text-sm">{ev.label}</span>
            </label>
          ))}
        </div>
      </PageSection>
    </div>
  )
}
