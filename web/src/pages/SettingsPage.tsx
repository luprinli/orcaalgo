import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { settings, llm } from '../api/client'
import type { AppSettings, LLMKey } from '../types/api'
import { PageHeader } from '../components/layout'
import { Button } from '../components/ui/button'
import { Badge } from '../components/ui/badge'
import { Card, CardContent, CardFooter, CardHeader, CardTitle, CardDescription } from '../components/ui/card'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '../components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs'

// ── Shared settings shape ──────────────────────────────────────────────
interface FullSettings {
  risk: Record<string, number | undefined>
  general: Record<string, unknown>
  webhook: { url: string; secret: string; events: string[] }
  notifications: {
    email: boolean; push: boolean; telegram: boolean
    email_address: string; telegram_chat_id: string
  }
  llm: {
    provider: string; endpoint: string; model: string
    api_key: string; temperature: number
  }
}

const DEFAULTS: FullSettings = {
  risk: {
    max_daily_loss_pct: 5,
    max_drawdown_pct: 10,
    kelly_fraction: 0.25,
    max_capital_per_trade_pct: 25,
  },
  general: { default_timeframe: '15m', default_capital: 100000, data_source: 'alpaca' },
  webhook: { url: '', secret: '', events: ['trade', 'signal', 'risk'] },
  notifications: {
    email: true, push: true, telegram: false,
    email_address: '', telegram_chat_id: '',
  },
  llm: {
    provider: 'openai', endpoint: 'https://api.openai.com/v1',
    model: 'gpt-4o', api_key: '', temperature: 0.3,
  },
}

function applySection<K extends keyof FullSettings>(
  cfg: FullSettings, section: K, patch: Partial<FullSettings[K]>,
): FullSettings {
  return { ...cfg, [section]: { ...cfg[section], ...patch } as FullSettings[K] }
}

// ── Tab content components ─────────────────────────────────────────────

function TradingTab({
  cfg, setCfg,
}: { cfg: FullSettings; setCfg: React.Dispatch<React.SetStateAction<FullSettings>> }) {
  const { t } = useTranslation()

  const updateRisk = (key: string, value: unknown) =>
    setCfg(p => applySection(p, 'risk', { [key]: value as number | undefined }))

  const updateGeneral = (key: string, value: unknown) =>
    setCfg(p => applySection(p, 'general', { [key]: value }))

  const riskFields = [
    { k: 'max_daily_loss_pct', l: t('settings:maxDailyLossPct', 'Max Daily Loss %'), def: 5 },
    { k: 'max_drawdown_pct', l: t('settings:maxDrawdownPct', 'Max Drawdown %'), def: 10 },
    { k: 'kelly_fraction', l: t('settings:kellyFraction', 'Kelly Fraction'), def: 0.25 },
    { k: 'max_capital_per_trade_pct', l: t('settings:maxCapitalPerTradePct', 'Max Capital Per Trade %'), def: 25 },
  ]

  return (
    <div className="flex flex-col space-y-4 max-w-[600px]">
      {/* ── Risk Parameters ── */}
      <Card>
        <CardHeader>
          <CardTitle>{t('settings:riskParameters', 'Risk Parameters')}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col space-y-2.5">
            {riskFields.map(f => (
              <div key={f.k} className="flex items-center justify-between">
                <Label htmlFor={`risk-${f.k}`}>{f.l}</Label>
                <Input
                  id={`risk-${f.k}`} className="w-[120px]" type="number" step="0.01"
                  value={cfg.risk[f.k] != null ? String(cfg.risk[f.k]) : ''}
                  onChange={e => updateRisk(f.k, parseFloat(e.target.value) || f.def)}
                  placeholder={String(f.def)}
                />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* ── General Settings ── */}
      <Card>
        <CardHeader>
          <CardTitle>{t('settings:general', 'General')}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col space-y-2.5">
            <div className="flex items-center justify-between">
              <Label htmlFor="gen-default_timeframe">{t('settings:defaultTimeframe', 'Default Timeframe')}</Label>
              <Select
                value={String(cfg.general.default_timeframe ?? '15m')}
                onValueChange={v => updateGeneral('default_timeframe', v)}
              >
                <SelectTrigger className="w-[120px]"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {['1m', '5m', '15m', '30m', '1h', '4h', '1d'].map(tf => (
                    <SelectItem key={tf} value={tf}>{tf}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="flex items-center justify-between">
              <Label htmlFor="gen-default_capital">{t('settings:defaultCapital', 'Default Capital')}</Label>
              <Input
                id="gen-default_capital" className="w-[150px]" type="number"
                value={cfg.general.default_capital != null ? String(cfg.general.default_capital) : ''}
                onChange={e => updateGeneral('default_capital', parseFloat(e.target.value) || 100000)}
                placeholder="100000"
              />
            </div>

            <div className="flex items-center justify-between">
              <Label htmlFor="gen-data_source">{t('settings:dataSource', 'Data Source')}</Label>
              <Select
                value={String(cfg.general.data_source ?? 'alpaca')}
                onValueChange={v => updateGeneral('data_source', v)}
              >
                <SelectTrigger className="w-[150px]"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {['alpaca', 'stooq', 'mock'].map(d => (
                    <SelectItem key={d} value={d}>{d.charAt(0).toUpperCase() + d.slice(1)}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

// ── Webhooks Tab ────────────────────────────────────────────────────────

function WebhooksTab({
  cfg, setCfg, msg, setMsg,
}: {
  cfg: FullSettings
  setCfg: React.Dispatch<React.SetStateAction<FullSettings>>
  msg: string; setMsg: (m: string) => void
}) {
  const { t } = useTranslation()
  const [testing, setTesting] = useState(false)

  const eventOptions = [
    { key: 'trade', label: 'Trade Executions' },
    { key: 'signal', label: 'Strategy Signals' },
    { key: 'risk', label: 'Risk Alerts' },
    { key: 'pnl', label: 'P&L Updates' },
    { key: 'regime', label: 'Regime Changes' },
  ]

  const setUrl = (url: string) => setCfg(p => applySection(p, 'webhook', { url }))
  const setSecret = (secret: string) => setCfg(p => applySection(p, 'webhook', { secret }))
  const setEvents = (fn: (prev: string[]) => string[]) =>
    setCfg(p => applySection(p, 'webhook', { events: fn(p.webhook.events) }))

  const testFire = async () => {
    const url = cfg.webhook.url
    if (!url) { setMsg('Enter a webhook URL first'); return }
    setTesting(true)
    setMsg('')
    try {
      const resp = await fetch(url, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ event: 'test', timestamp: new Date().toISOString(), source: 'OrcaAlgo' }),
      })
      if (resp.ok) setMsg('Test webhook sent successfully.')
      else setMsg(`Webhook returned status ${resp.status}`)
    } catch (err) {
      setMsg(`Test failed: ${err instanceof Error ? err.message : 'Connection error'}`)
    } finally { setTesting(false) }
  }

  return (
    <div className="flex flex-col space-y-4 max-w-[600px]">
      {/* ── Webhook Endpoint ── */}
      <Card>
        <CardHeader>
          <CardTitle>{t('sidebar:nav.webhooks', 'Webhook Endpoint')}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div>
              <Label htmlFor="webhook-url" className="text-xs text-muted-foreground block mb-1">Webhook URL</Label>
              <Input
                id="webhook-url" type="url" value={cfg.webhook.url}
                onChange={e => setUrl(e.target.value)}
                placeholder="https://hooks.example.com/orca"
              />
            </div>

            <div>
              <Label htmlFor="webhook-secret" className="text-xs text-muted-foreground block mb-1">Secret (optional)</Label>
              <Input
                id="webhook-secret" type="password" value={cfg.webhook.secret}
                onChange={e => setSecret(e.target.value)}
                placeholder="HMAC signing secret"
              />
            </div>

            <Button variant="outline" onClick={testFire} disabled={testing || !cfg.webhook.url}>
              {testing ? 'Sending...' : 'Test Fire'}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* ── Event Subscriptions ── */}
      <Card>
        <CardHeader>
          <CardTitle>Event Subscriptions</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col space-y-2">
            {eventOptions.map(ev => (
              <label key={ev.key} className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={cfg.webhook.events.includes(ev.key)}
                  onChange={e => setEvents(prev => e.target.checked ? [...prev, ev.key] : prev.filter(k => k !== ev.key))}
                />
                <span className="text-sm">{ev.label}</span>
              </label>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

// ── Notifications Tab ────────────────────────────────────────────────────

function NotificationsTab({
  cfg, setCfg,
}: { cfg: FullSettings; setCfg: React.Dispatch<React.SetStateAction<FullSettings>> }) {
  const { t } = useTranslation()
  const [testing, setTesting] = useState(false)
  const [testMsg, setTestMsg] = useState('')

  const updateNotif = (patch: Partial<FullSettings['notifications']>) =>
    setCfg(p => applySection(p, 'notifications', patch))

  const handleTest = async () => {
    setTesting(true)
    setTestMsg('')
    try {
      const res = await settings.testNotification()
      setTestMsg(res.success ? 'Test notification sent successfully' : `Failed: ${res.message}`)
    } catch (err) {
      setTestMsg(err instanceof Error ? err.message : 'Test failed')
    } finally { setTesting(false) }
  }

  return (
    <div className="flex flex-col space-y-4 max-w-[600px]">
      {/* ── Notification Channels ── */}
      <Card>
        <CardHeader>
          <CardTitle>{t('notification:channels', 'Notification Channels')}</CardTitle>
          <CardDescription>
            {t('sidebar:nav.notifications', 'Configure how you receive trading alerts, risk notifications, and system events.')}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col space-y-3">
            <label className="flex items-center gap-2">
              <input
                type="checkbox" checked={cfg.notifications.email}
                onChange={e => updateNotif({ email: e.target.checked })}
              />
              <div>
                <div className="font-medium text-foreground text-sm">{t('notification:emailAlerts', 'Email Alerts')}</div>
                <div className="text-muted-foreground text-xs">Daily summaries, risk alerts, trade confirmations</div>
              </div>
            </label>

            {cfg.notifications.email && (
              <div className="ml-5">
                <Input
                  type="email" placeholder="your@email.com" value={cfg.notifications.email_address}
                  onChange={e => updateNotif({ email_address: e.target.value })}
                  className="max-w-[300px]"
                />
              </div>
            )}

            <label className="flex items-center gap-2">
              <input
                type="checkbox" checked={cfg.notifications.push}
                onChange={e => updateNotif({ push: e.target.checked })}
              />
              <div>
                <div className="font-medium text-foreground text-sm">{t('notification:pushNotifications', 'Push Notifications')}</div>
                <div className="text-muted-foreground text-xs">Browser push notifications for real-time events</div>
              </div>
            </label>

            <label className="flex items-center gap-2">
              <input
                type="checkbox" checked={cfg.notifications.telegram}
                onChange={e => updateNotif({ telegram: e.target.checked })}
              />
              <div>
                <div className="font-medium text-foreground text-sm">{t('notification:telegramAlerts', 'Telegram Alerts')}</div>
                <div className="text-muted-foreground text-xs">Instant alerts via Telegram bot</div>
              </div>
            </label>

            {cfg.notifications.telegram && (
              <div className="ml-5">
                <Input
                  type="text" placeholder="Telegram Chat ID" value={cfg.notifications.telegram_chat_id}
                  onChange={e => updateNotif({ telegram_chat_id: e.target.value })}
                  className="max-w-[300px]"
                />
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* ── Alert Triggers ── */}
      <Card>
        <CardHeader>
          <CardTitle>Alert Triggers</CardTitle>
        </CardHeader>
        <CardContent>
          <CardDescription className="mb-3">These notifications are sent based on the following events:</CardDescription>
          <div className="grid grid-cols-2 text-sm text-muted-foreground">
            <div>Trade executed or cancelled</div>
            <div>Daily loss limit approaching</div>
            <div>Strategy signal generated</div>
            <div>Max drawdown approaching</div>
            <div>Regime change detected</div>
            <div>Kill-switch activated</div>
            <div>System health degraded</div>
            <div>Backtest completed</div>
          </div>
        </CardContent>
      </Card>

      {testMsg && (
        <Card className={testMsg.includes('fail') || testMsg.includes('Failed') ? 'border-l-4 border-l-destructive' : 'border-l-4 border-l-trading-success'}>
          <CardContent className="text-sm py-2">{testMsg}</CardContent>
        </Card>
      )}

      <Card>
        <CardFooter>
          <Button onClick={handleTest} disabled={testing}>
            {testing ? t('settings:testing', 'Testing...') : 'Test Notification'}
          </Button>
        </CardFooter>
      </Card>
    </div>
  )
}

// ── LLM Tab ─────────────────────────────────────────────────────────────

function LLMTab({
  cfg, setCfg,
}: { cfg: FullSettings; setCfg: React.Dispatch<React.SetStateAction<FullSettings>> }) {
  const { t } = useTranslation()
  const [testing, setTesting] = useState(false)
  const [testMsg, setTestMsg] = useState('')
  const [keys, setKeys] = useState<LLMKey[]>([])
  const [savingKey, setSavingKey] = useState(false)

  const loadKeys = useCallback(async () => {
    try {
      const res = await llm.listKeys()
      setKeys(res.keys ?? [])
    } catch {
      setKeys([])
    }
  }, [])

  useEffect(() => { loadKeys() }, [loadKeys])

  const updateLLM = (patch: Partial<FullSettings['llm']>) =>
    setCfg(p => applySection(p, 'llm', patch))

  const setProvider = (v: string) => {
    const defaults: Record<string, { endpoint: string; model: string }> = {
      openai:    { endpoint: 'https://api.openai.com/v1',      model: 'gpt-4o' },
      anthropic: { endpoint: 'https://api.anthropic.com/v1',   model: 'claude-3-opus' },
      ollama:    { endpoint: 'http://localhost:11434/v1',      model: 'llama3' },
    }
    const d = defaults[v] ?? { endpoint: v, model: '' }
    setCfg(p => applySection(p, 'llm', { provider: v, endpoint: d.endpoint, model: d.model }))
  }

  return (
    <div className="flex flex-col space-y-4 max-w-[600px]">
      <Card>
        <CardHeader>
          <CardTitle>{t('sidebar:nav.llm', 'LLM Settings')}</CardTitle>
          <CardDescription>
            {t('llm:description', 'Configure LLM provider for strategy analysis and trade commentary.')}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div>
              <Label htmlFor="llm-provider" className="text-xs text-muted-foreground block mb-1">Provider</Label>
              <Select value={cfg.llm.provider} onValueChange={setProvider}>
                <SelectTrigger id="llm-provider"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="openai">OpenAI</SelectItem>
                  <SelectItem value="anthropic">Anthropic</SelectItem>
                  <SelectItem value="ollama">Ollama (Local)</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div>
              <Label htmlFor="llm-endpoint" className="text-xs text-muted-foreground block mb-1">API Endpoint</Label>
              <Input
                id="llm-endpoint" type="text" value={cfg.llm.endpoint}
                onChange={e => updateLLM({ endpoint: e.target.value })}
              />
            </div>

            <div>
              <Label htmlFor="llm-model" className="text-xs text-muted-foreground block mb-1">Model</Label>
              <Input
                id="llm-model" type="text" value={cfg.llm.model}
                onChange={e => updateLLM({ model: e.target.value })}
                placeholder="gpt-4o"
              />
            </div>

            <div>
              <Label htmlFor="llm-api-key" className="text-xs text-muted-foreground block mb-1">API Key</Label>
              <Input
                id="llm-api-key" type="password" value={cfg.llm.api_key}
                onChange={e => updateLLM({ api_key: e.target.value })}
                placeholder={cfg.llm.api_key ? '••••••••' : 'sk-...'}
              />
            </div>

            <div>
              <Label htmlFor="llm-temperature" className="text-xs text-muted-foreground block mb-1">
                Temperature ({cfg.llm.temperature})
              </Label>
              <Input
                id="llm-temperature" type="range" min={0} max={1} step={0.05}
                value={cfg.llm.temperature}
                onChange={e => updateLLM({ temperature: parseFloat(e.target.value) })}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t('llm:storedKeys', 'Stored API Keys (BYOK)')}</CardTitle>
          <CardDescription>
            {t('llm:storedKeysDesc', 'Keys are encrypted at rest and shown masked. One key per provider.')}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {keys.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t('llm:noKeys', 'No stored keys. Save your key below.')}</p>
          ) : (
            <div className="space-y-2">
              {keys.map(k => (
                <div key={k.provider} className="flex items-center justify-between border rounded p-2">
                  <div className="text-sm">
                    <span className="font-medium">{k.provider}</span>
                    <span className="text-muted-foreground ml-2">{k.masked_suffix}</span>
                    {k.model && <span className="text-muted-foreground ml-2 text-xs">{k.model}</span>}
                  </div>
                  <Button variant="ghost" size="sm" onClick={async () => { try { await llm.deleteKey(k.provider); await loadKeys() } catch (e) { setTestMsg(e instanceof Error ? e.message : 'Delete failed') } }}>
                    {t('common:delete', 'Delete')}
                  </Button>
                </div>
              ))}
            </div>
          )}
          <div className="flex gap-2">
            <Button
              variant="outline"
              disabled={savingKey || !cfg.llm.api_key}
              onClick={async () => {
                setSavingKey(true)
                try {
                  await llm.addKey({ provider: cfg.llm.provider, api_key: cfg.llm.api_key, base_url: cfg.llm.endpoint, model: cfg.llm.model })
                  await loadKeys()
                } catch (e) {
                  setTestMsg(e instanceof Error ? e.message : 'Save failed')
                } finally { setSavingKey(false) }
              }}
            >
              {savingKey ? 'Saving...' : 'Save Key'}
            </Button>
          </div>
        </CardContent>
      </Card>

      {testMsg && (
        <Card className={testMsg.includes('fail') || testMsg.includes('Failed') ? 'border-l-4 border-l-destructive' : 'border-l-4 border-l-trading-success'}>
          <CardContent className="text-sm py-2">{testMsg}</CardContent>
        </Card>
      )}

      <Button
        onClick={async () => {
          setTesting(true)
          setTestMsg('')
          try {
            const res = await settings.testLLM(cfg.llm.provider, cfg.llm.api_key, cfg.llm.endpoint, cfg.llm.model)
            setTestMsg(res.reachable ? `Connected — ${res.response}` : 'Unreachable')
          } catch (err) {
            setTestMsg(err instanceof Error ? err.message : 'Test failed')
          } finally { setTesting(false) }
        }}
        disabled={testing || !cfg.llm.api_key}
        variant="outline"
      >
        {testing ? 'Testing...' : 'Test Connection'}
      </Button>
    </div>
  )
}

// ── Main Page ───────────────────────────────────────────────────────────

export default function SettingsPage() {
  const { t } = useTranslation()
  const [cfg, setCfg] = useState<FullSettings>(DEFAULTS)
  const [loading, setLoading] = useState(true)
  const [msg, setMsg] = useState('')
  const [saving, setSaving] = useState(false)

  const fetchSettings = useCallback(async () => {
    try {
      const s = await settings.get()
      const raw = s as Record<string, any>
      setCfg({
        risk:          { ...DEFAULTS.risk,          ...(raw.risk          as Record<string, number | undefined> ?? {}) },
        general:       { ...DEFAULTS.general,       ...(raw.general       as Record<string, unknown> ?? {}) },
        webhook:       { ...DEFAULTS.webhook,       ...(raw.webhook       as Record<string, unknown> ?? {}) },
        notifications: { ...DEFAULTS.notifications, ...(raw.notifications as Record<string, unknown> ?? {}) },
        llm:           { ...DEFAULTS.llm,           ...(raw.llm           as Record<string, unknown> ?? {}) },
      })
    } catch { /* ignore */ } finally { setLoading(false) }
  }, [])

  useEffect(() => { fetchSettings() }, [fetchSettings])

  const handleSave = async () => {
    setSaving(true)
    setMsg('')
    try {
      await settings.update(cfg as unknown as AppSettings)
      setMsg(t('settings:updated', 'Settings saved'))
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t('settings:updateFailed', 'Save failed'))
    } finally { setSaving(false) }
  }

  if (loading) {
    return (
      <div>
        <PageHeader title={t('settings:title', 'Settings')} />
        <Card><CardContent className="pt-6"><CardDescription>{t('settings:loading', 'Loading settings...')}</CardDescription></CardContent></Card>
      </div>
    )
  }

  const isError = msg.includes('fail') || msg.includes('Fail')

  return (
    <div>
      <PageHeader title={t('settings:title', 'Settings')} />

      <Tabs defaultValue="trading" className="max-w-[680px]">
        <TabsList variant="line" className="mb-4">
          <TabsTrigger value="trading" variant="line">
            <Badge variant="outline" size="sm" className="mr-1.5">R</Badge>
            Trading
          </TabsTrigger>
          <TabsTrigger value="webhooks" variant="line">
            <Badge variant="outline" size="sm" className="mr-1.5">W</Badge>
            Webhooks
          </TabsTrigger>
          <TabsTrigger value="notifications" variant="line">
            <Badge variant="outline" size="sm" className="mr-1.5">N</Badge>
            Notifications
          </TabsTrigger>
          <TabsTrigger value="llm" variant="line">
            <Badge variant="outline" size="sm" className="mr-1.5">L</Badge>
            LLM
          </TabsTrigger>
        </TabsList>

        {msg && (
          <Card className={`mb-4 ${isError ? 'border-l-4 border-l-destructive' : 'border-l-4 border-l-green-500'}`}>
            <CardContent className={isError ? 'text-destructive text-sm py-2' : 'text-green-500 text-sm py-2'}>{msg}</CardContent>
          </Card>
        )}

        <TabsContent value="trading">
          <TradingTab cfg={cfg} setCfg={setCfg} />
        </TabsContent>

        <TabsContent value="webhooks">
          <WebhooksTab cfg={cfg} setCfg={setCfg} msg={msg} setMsg={setMsg} />
        </TabsContent>

        <TabsContent value="notifications">
          <NotificationsTab cfg={cfg} setCfg={setCfg} />
        </TabsContent>

        <TabsContent value="llm">
          <LLMTab cfg={cfg} setCfg={setCfg} />
        </TabsContent>

        <Card className="mt-2">
          <CardFooter>
            <Button onClick={handleSave} disabled={saving}>
              {saving ? t('settings:updating', 'Saving...') : t('settings:updateButton', 'Save Settings')}
            </Button>
          </CardFooter>
        </Card>
      </Tabs>
    </div>
  )
}
