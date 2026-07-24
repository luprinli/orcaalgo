import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import toast from 'react-hot-toast'
import { settings } from '../api/client'
import { PageHeader, PageSection } from '../components/layout'

interface NotificationPrefs {
  email: boolean
  push: boolean
  telegram: boolean
  email_address?: string
  telegram_chat_id?: string
}

export default function NotificationSettings() {
  const { t } = useTranslation()
  const [prefs, setPrefs] = useState<NotificationPrefs>({
    email: true,
    push: true,
    telegram: false,
    email_address: '',
    telegram_chat_id: '',
  })
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [msg, setMsg] = useState('')

  useEffect(() => {
    settings.get()
      .then(cfg => {
        const notif = (cfg as any)?.notifications
        if (notif) {
          setPrefs({
            email: notif.email ?? true,
            push: notif.push ?? true,
            telegram: notif.telegram ?? false,
            email_address: notif.email_address || '',
            telegram_chat_id: notif.telegram_chat_id || '',
          })
        }
      })
      .catch(() => setMsg('Could not load notification settings'))
      .finally(() => setLoading(false))
  }, [])

  const save = async () => {
    setSaving(true)
    setMsg('')
    try {
      await settings.update({ notifications: prefs } as any)
      setMsg('Notification preferences saved.')
      toast.success('Notification settings saved')
    } catch (err) {
      setMsg(`Save failed: ${err instanceof Error ? err.message : 'Unknown error'}`)
      toast.error('Failed to save notification settings')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <div className="p-6"><PageHeader title={t('sidebar:nav.notifications', 'Notifications')} /><div className="card"><p className="text-muted">Loading...</p></div></div>

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        title={t('sidebar:nav.notifications', 'Notification Settings')}
        subtitle="Configure how you receive trading alerts, risk notifications, and system events."
      />

      {msg && (
        <div className={`card ${msg.includes('failed') || msg.includes('Could not') ? 'border-l-4' : ''}`}
          style={(msg.includes('failed') || msg.includes('Could not')) ? { borderLeftColor: 'var(--danger)' } : { borderLeftColor: 'var(--success)' }}>
          <p style={{ color: (msg.includes('failed') || msg.includes('Could not')) ? 'var(--danger-text)' : 'var(--success)' }}>{msg}</p>
        </div>
      )}

      <PageSection title={t('notification:channels', 'Notification Channels')}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <label className="flex gap-2" style={{ alignItems: 'center' }}>
            <input type="checkbox" checked={prefs.email} onChange={e => setPrefs(p => ({ ...p, email: e.target.checked }))} />
            <div>
              <div className="font-medium text-white text-sm">{t('notification:emailAlerts', 'Email Alerts')}</div>
              <div className="text-muted text-xs">Daily summaries, risk alerts, trade confirmations</div>
            </div>
          </label>

          {prefs.email && (
            <div className="ml-5">
              <input className="input" type="email" placeholder="your@email.com" value={prefs.email_address || ''}
                onChange={e => setPrefs(p => ({ ...p, email_address: e.target.value }))}
                style={{ maxWidth: 300 }} />
            </div>
          )}

          <label className="flex gap-2" style={{ alignItems: 'center' }}>
            <input type="checkbox" checked={prefs.push} onChange={e => setPrefs(p => ({ ...p, push: e.target.checked }))} />
            <div>
              <div className="font-medium text-white text-sm">{t('notification:pushNotifications', 'Push Notifications')}</div>
              <div className="text-muted text-xs">Browser push notifications for real-time events</div>
            </div>
          </label>

          <label className="flex gap-2" style={{ alignItems: 'center' }}>
            <input type="checkbox" checked={prefs.telegram} onChange={e => setPrefs(p => ({ ...p, telegram: e.target.checked }))} />
            <div>
              <div className="font-medium text-white text-sm">{t('notification:telegramAlerts', 'Telegram Alerts')}</div>
              <div className="text-muted text-xs">Instant alerts via Telegram bot</div>
            </div>
          </label>

          {prefs.telegram && (
            <div className="ml-5">
              <input className="input" type="text" placeholder="Telegram Chat ID" value={prefs.telegram_chat_id || ''}
                onChange={e => setPrefs(p => ({ ...p, telegram_chat_id: e.target.value }))}
                style={{ maxWidth: 300 }} />
            </div>
          )}
        </div>

        <button className="btn btn-primary mt-4" onClick={save} disabled={saving}>
          {saving ? 'Saving...' : 'Save Preferences'}
        </button>
      </PageSection>

      <PageSection title="Alert Triggers">
        <p className="text-muted text-sm mb-3">These notifications are sent based on the following events:</p>
        <div className="grid-2 text-sm text-slate-400">
          <div>• Trade executed or cancelled</div>
          <div>• Daily loss limit approaching</div>
          <div>• Strategy signal generated</div>
          <div>• Max drawdown approaching</div>
          <div>• Regime change detected</div>
          <div>• Kill-switch activated</div>
          <div>• System health degraded</div>
          <div>• Backtest completed</div>
        </div>
      </PageSection>
    </div>
  )
}
