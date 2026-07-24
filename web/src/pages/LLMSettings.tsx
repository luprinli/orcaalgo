import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import toast from 'react-hot-toast'
import { settings } from '../api/client'
import { PageHeader, PageSection } from '../components/layout'

export default function LLMSettings() {
  const { t } = useTranslation()
  const [provider, setProvider] = useState('openai')
  const [endpoint, setEndpoint] = useState('https://api.openai.com/v1')
  const [model, setModel] = useState('gpt-4o')
  const [apiKey, setApiKey] = useState('')
  const [temperature, setTemperature] = useState(0.3)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [msg, setMsg] = useState('')

  useEffect(() => {
    settings.get()
      .then(cfg => {
        const llm = (cfg as any)?.llm
        if (llm) {
          setProvider(llm.provider || 'openai')
          setEndpoint(llm.endpoint || 'https://api.openai.com/v1')
          setModel(llm.model || 'gpt-4o')
          setTemperature(llm.temperature ?? 0.3)
          if (llm.api_key) setApiKey(llm.api_key)
        }
      })
      .catch(() => setMsg('Could not load LLM settings'))
      .finally(() => setLoading(false))
  }, [])

  const save = async () => {
    setSaving(true)
    setMsg('')
    try {
      await settings.update({
        llm: { provider, endpoint, model, api_key: apiKey || undefined, temperature },
      } as any)
      setMsg('LLM settings saved.')
      toast.success('LLM settings saved')
    } catch (err) {
      setMsg(`Save failed: ${err instanceof Error ? err.message : 'Unknown error'}`)
      toast.error('Failed to save LLM settings')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <div className="p-6"><PageHeader title={t('sidebar:nav.llm', 'LLM Settings')} /><div className="card"><p className="text-muted">Loading...</p></div></div>

  return (
    <div className="p-6 space-y-6">
      <PageHeader title={t('sidebar:nav.llm', 'LLM Settings')} subtitle={t('llm:description', 'Configure LLM provider for strategy analysis and trade commentary.')} />

      {msg && (
        <div className={`card ${msg.includes('failed') ? 'border-l-4' : ''}`} style={msg.includes('failed') ? { borderLeftColor: 'var(--danger)' } : { borderLeftColor: 'var(--success)' }}>
          <p style={{ color: msg.includes('failed') ? 'var(--danger-text)' : 'var(--success)' }}>{msg}</p>
        </div>
      )}

      <PageSection title="Provider Configuration">
        <div className="space-y-4">
          <div>
            <label className="text-xs text-slate-400 block mb-1">Provider</label>
            <select className="input" value={provider} onChange={e => { setProvider(e.target.value); setEndpoint(e.target.value === 'openai' ? 'https://api.openai.com/v1' : e.target.value === 'anthropic' ? 'https://api.anthropic.com/v1' : 'http://localhost:11434/v1'); setModel(e.target.value === 'openai' ? 'gpt-4o' : e.target.value === 'anthropic' ? 'claude-3-opus' : 'llama3') }}>
              <option value="openai">OpenAI</option>
              <option value="anthropic">Anthropic</option>
              <option value="ollama">Ollama (Local)</option>
            </select>
          </div>

          <div>
            <label className="text-xs text-slate-400 block mb-1">API Endpoint</label>
            <input className="input" type="text" value={endpoint} onChange={e => setEndpoint(e.target.value)} />
          </div>

          <div>
            <label className="text-xs text-slate-400 block mb-1">Model</label>
            <input className="input" type="text" value={model} onChange={e => setModel(e.target.value)} placeholder="gpt-4o" />
          </div>

          <div>
            <label className="text-xs text-slate-400 block mb-1">API Key</label>
            <input className="input" type="password" value={apiKey} onChange={e => setApiKey(e.target.value)} placeholder={apiKey ? '••••••••' : 'sk-...'} />
          </div>

          <div>
            <label className="text-xs text-slate-400 block mb-1">Temperature ({temperature})</label>
            <input className="input" type="range" min="0" max="1" step="0.05" value={temperature} onChange={e => setTemperature(parseFloat(e.target.value))} />
          </div>

          <button className="btn btn-primary" onClick={save} disabled={saving}>
            {saving ? 'Saving...' : 'Save LLM Settings'}
          </button>
        </div>
      </PageSection>
    </div>
  )
}
