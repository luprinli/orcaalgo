import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router-dom'
import { strategies } from '../api/client'
import ParamEditor from '../components/ParamEditor'
import type { StrategyValidationResponse, ParamDef } from '../types/api'

interface StrategyFormData {
  name: string
  type: string
  parameters: string
  enabled: boolean
}

interface StrategyFormData {
  name: string
  type: string
  parameters: string
  enabled: boolean
  paramsObj: Record<string, number>
}

function parseParamsJson(json: string): Record<string, number> {
  try {
    const parsed = JSON.parse(json || '{}')
    const obj: Record<string, number> = {}
    for (const [k, v] of Object.entries(parsed)) {
      if (typeof v === 'number') obj[k] = v
    }
    return obj
  } catch { return {} }
}

export default function StrategyEditor() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const [form, setForm] = useState<StrategyFormData>({
    name: '',
    type: 'intraday_mr',
    parameters: '{}',
    enabled: false,
    paramsObj: {},
  })
  const [validation, setValidation] = useState<StrategyValidationResponse | null>(null)
  const [validating, setValidating] = useState(false)
  const [saving, setSaving] = useState(false)
  const [msg, setMsg] = useState('')
  const [createdId, setCreatedId] = useState<string | null>(null)
  const fileRef = useRef<HTMLInputElement | null>(null)
  const [gkrLoading, setGkrLoading] = useState(false)
  const [paramDefs, setParamDefs] = useState<ParamDef[]>([])
  const [allParamDefs, setAllParamDefs] = useState<Record<string, ParamDef[]>>({})
  const [showRawJSON, setShowRawJSON] = useState(false)

  useEffect(() => {
    strategies.paramDefs().then(res => {
      if (res?.defs) {
        setAllParamDefs(res.defs)
      }
    }).catch(() => {})
  }, [])

  useEffect(() => {
    const defs = allParamDefs[form.type]
    setParamDefs(defs ?? [])
  }, [form.type, allParamDefs])

  useEffect(() => {
    if (!id) return
    strategies.get(id).then(s => {
      const paramsStr = s.parameters ? JSON.stringify(s.parameters, null, 2) : '{}'
      const paramsObj = parseParamsJson(paramsStr)
      setForm({
        name: s.name || '',
        type: s.type || 'intraday_mr',
        parameters: paramsStr,
        enabled: s.enabled ?? false,
        paramsObj,
      })
      setCreatedId(s.id)
      setMsg(t('strategyEditor:loadedStrategy', 'Loaded strategy: {{name}}', { name: s.name }))
    }).catch(() => setMsg(t('strategyEditor:failedToLoad', 'Failed to load strategy')))
  }, [id])

  const handleLoadGkr = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setGkrLoading(true)
    setMsg('')
    try {
      const yamlText = await file.text()
      const res = await strategies.fromGkr({ yaml: yamlText })
      if (res.id) {
        const paramsStr = JSON.stringify(res.parameters || {}, null, 2)
        setForm({
          name: res.name || 'gkr-import',
          type: res.type || 'intraday_mr',
          parameters: paramsStr,
          enabled: false,
          paramsObj: parseParamsJson(paramsStr),
        })
        setCreatedId(res.id)
        setMsg(t('strategyEditor:loadedFromGkr', 'Loaded from GKR: {{name}} ({{type}})', { name: res.name, type: res.type }))
      } else if (res.strategy_type) {
        const paramsStr = JSON.stringify(res.parameters || {}, null, 2)
        setForm({
          name: 'gkr-import',
          type: res.strategy_type,
          parameters: paramsStr,
          enabled: false,
          paramsObj: parseParamsJson(paramsStr),
        })
        setMsg(t('strategyEditor:gkrCompiled', 'GKR compiled (no DB): {{type}}', { type: res.strategy_type }))
      }
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t('strategyEditor:gkrLoadFailed', 'GKR load failed'))
    } finally {
      setGkrLoading(false)
      if (fileRef.current) fileRef.current.value = ''
    }
  }

  const updateField = <K extends keyof StrategyFormData>(key: K, value: StrategyFormData[K]) => {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  const handleStructuredParamsChange = (params: Record<string, number>) => {
    setForm(prev => ({ ...prev, paramsObj: params, parameters: JSON.stringify(params, null, 2) }))
  }

  const handleJSONParamsChange = (json: string) => {
    setForm(prev => ({ ...prev, parameters: json, paramsObj: parseParamsJson(json) }))
  }

  const handleValidate = async () => {
    setValidating(true)
    setValidation(null)
    setMsg('')
    try {
      const params = JSON.parse(form.parameters || '{}')
      const res = await strategies.validate({
        name: form.name,
        type: form.type,
        parameters: params,
      })
      setValidation(res)
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t('strategyEditor:validationFailed', 'Validation failed'))
    } finally {
      setValidating(false)
    }
  }

  const handleCreate = async () => {
    if (!form.name) return
    setSaving(true)
    setMsg('')
    try {
      const params = JSON.parse(form.parameters || '{}')
      const res = await strategies.create({
        name: form.name,
        type: form.type,
        parameters: params,
        enabled: form.enabled,
      })
      setCreatedId(res.id)
      setMsg(t('strategyEditor:strategyCreated', 'Strategy "{{name}}" created ({{id}})', { name: res.name, id: res.id }))
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t('strategyEditor:createFailed', 'Create failed'))
    } finally {
      setSaving(false)
    }
  }

  const handleReset = () => {
    setForm({ name: '', type: 'intraday_mr', parameters: '{}', enabled: false, paramsObj: {} })
    setValidation(null)
    setCreatedId(null)
    setMsg('')
  }

  return (
    <div>
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>{t('strategyEditor:title', 'Strategy Editor')}</h1>
        <div className="flex gap-2">
          <input type="file" ref={fileRef} accept=".yaml,.gkr.yaml" style={{ display: 'none' }} onChange={handleLoadGkr} />
          <button className="btn btn-outline" onClick={() => fileRef.current?.click()} disabled={gkrLoading}>
            {gkrLoading ? t('strategyEditor:loading', 'Loading...') : t('strategyEditor:loadGkr', 'Load GKR')}
          </button>
        </div>
      </div>

      <div className="grid-2 mb-4">
        <div className="card">
          <h2>{createdId ? t('strategyEditor:editStrategy', 'Edit Strategy') : t('strategyEditor:newStrategy', 'New Strategy')}</h2>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            <div>
              <label className="text-muted">{t('strategyEditor:name', 'Name')}</label>
              <input
                className="input"
                placeholder="my_strategy"
                value={form.name}
                onChange={(e) => updateField('name', e.target.value)}
              />
            </div>
            <div>
              <label className="text-muted">{t('strategyEditor:type', 'Type')}</label>
              <select className="input" value={form.type} onChange={(e) => updateField('type', e.target.value)}>
                <option value="intraday_mr">{t('strategyEditor:type:intradayMr', 'Intraday Mean Reversion')}</option>
                <option value="opening_range_breakout">{t('strategyEditor:type:openingRangeBreakout', 'Opening Range Breakout')}</option>
                <option value="trend_following">{t('strategyEditor:type:trendFollowing', 'Trend Following')}</option>
                <option value="grid_trading">{t('strategyEditor:type:gridTrading', 'Grid Trading')}</option>
                <option value="session_scalp">{t('strategyEditor:type:sessionScalp', 'Session Scalp')}</option>
                <option value="ma_crossover">{t('strategyEditor:type:maCrossover', 'MA Crossover')}</option>
                <option value="rsi2_reversion">{t('strategyEditor:type:rsi2Reversion', 'RSI-2 Reversion')}</option>
                <option value="donchian_breakout">{t('strategyEditor:type:donchianBreakout', 'Donchian Breakout')}</option>
                <option value="keltner_macd">{t('strategyEditor:type:keltnerMacd', 'Keltner MACD')}</option>
                <option value="ichimoku_cloud">{t('strategyEditor:type:ichimokuCloud', 'Ichimoku Cloud')}</option>
              </select>
            </div>
            <div>
              <div className="flex-between" style={{ marginBottom: 6 }}>
                <label className="text-muted">{t('strategyEditor:parameters', 'Parameters')}</label>
                {paramDefs.length > 0 && (
                  <button className="btn btn-outline" style={{ fontSize: 11, padding: '2px 8px' }} onClick={() => setShowRawJSON(v => !v)}>
                    {showRawJSON ? t('strategyEditor:structured', 'Structured') : t('strategyEditor:rawJson', 'Raw JSON')}
                  </button>
                )}
              </div>
              {showRawJSON || paramDefs.length === 0 ? (
                <textarea
                  className="input"
                  style={{ minHeight: 120, fontFamily: 'monospace', fontSize: 12, resize: 'vertical' }}
                  placeholder='{ "lookback": 20, "threshold": 2.0 }'
                  value={form.parameters}
                  onChange={(e) => handleJSONParamsChange(e.target.value)}
                />
              ) : (
                <ParamEditor
                  defs={paramDefs}
                  initialParams={form.paramsObj}
                  onChange={handleStructuredParamsChange}
                  compact
                />
              )}
            </div>
            <label className="flex gap-2" style={{ alignItems: 'center' }}>
              <input type="checkbox" checked={form.enabled} onChange={(e) => updateField('enabled', e.target.checked)} />
              {t('strategyEditor:enableImmediately', 'Enable immediately')}
            </label>
          </div>
        </div>

        <div className="card">
          <h2>{t('strategyEditor:validation', 'Validation')}</h2>
          <div className="flex gap-2 mb-3">
            <button className="btn btn-outline" onClick={handleValidate} disabled={validating || !form.name}>
              {validating ? t('strategyEditor:validating', 'Validating...') : t('strategyEditor:validate', 'Validate')}
            </button>
            <button className="btn btn-primary" onClick={handleCreate} disabled={saving || !form.name}>
              {saving ? t('strategyEditor:saving', 'Saving...') : t('strategyEditor:createStrategy', 'Create Strategy')}
            </button>
            {createdId && (
              <button className="btn btn-outline" onClick={handleReset}>
                {t('strategyEditor:new', 'New')}
              </button>
            )}
          </div>

          {msg && (
            <p className="text-muted mt-2" style={{ color: msg.includes('created') ? 'var(--success)' : 'var(--danger)' }}>
              {msg}
            </p>
          )}

          {validation && (
            <div className="mt-2">
              <div
                style={{
                  padding: 8,
                  borderRadius: 6,
                  background: validation.valid ? 'rgba(63,185,80,.1)' : 'rgba(218,54,51,.1)',
                  color: validation.valid ? 'var(--success)' : 'var(--danger)',
                  marginBottom: 8,
                }}
              >
                {validation.valid ? t('strategyEditor:strategyValid', '✓ Strategy is valid') : t('strategyEditor:errorsFound', '✗ {{n}} error(s)', { n: validation.errors?.length ?? 0 })}
              </div>

              {validation.errors && validation.errors.length > 0 && (
                <ul style={{ fontSize: 12, padding: '0 0 0 16px', margin: 0 }}>
                  {validation.errors.map((e, i) => (
                    <li key={i} style={{ color: 'var(--danger)', marginBottom: 4 }}>{e}</li>
                  ))}
                </ul>
              )}

              {validation.diagnostics && validation.diagnostics.length > 0 && (
                <div className="mt-2">
                  <span className="text-muted">{t('strategyEditor:diagnostics', 'Diagnostics:')}</span>
                  <pre style={{ fontSize: 11, overflow: 'auto', maxHeight: 200 }}>
                    {JSON.stringify(validation.diagnostics, null, 2)}
                  </pre>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {createdId && (
        <div className="card">
          <h2>{t('strategyEditor:quickActions', 'Quick Actions')}</h2>
          <div className="flex gap-2">
            <button
              className="btn btn-outline"
              onClick={async () => {
                try {
                  await strategies.reload(createdId)
                  setMsg(t('strategyEditor:strategyReloaded', 'Strategy reloaded'))
                } catch (err) {
                  setMsg(err instanceof Error ? err.message : t('strategyEditor:reloadFailed', 'Reload failed'))
                }
              }}
            >
              {t('strategyEditor:reload', 'Reload')}
            </button>
            <button
              className="btn btn-outline"
              onClick={async () => {
                try {
                  const clone = await strategies.clone(createdId)
                  setMsg(t('strategyEditor:clonedAs', 'Cloned as "{{name}}" ({{id}})', { name: clone.name, id: clone.id }))
                } catch (err) {
                  setMsg(err instanceof Error ? err.message : t('strategyEditor:cloneFailed', 'Clone failed'))
                }
              }}
            >
              {t('strategyEditor:clone', 'Clone')}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
