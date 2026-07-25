import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { strategies } from '../../api/client'
import { STRATEGY_CATALOG } from '../../data/strategyCatalog'
import ParamEditor from '../../components/ParamEditor'
import type { StrategyValidationResponse, ParamDef } from '../../types/api'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Card, CardHeader, CardTitle, CardContent } from '../../components/ui/card'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '../../components/ui/select'
import { Label } from '../../components/ui/label'
import { Textarea } from '../../components/ui/textarea'

interface StrategyFormData {
  name: string
  type: string
  parameters: string
  enabled: boolean
  paramsObj: Record<string, number>
}

interface EditorTabProps {
  id: string | null
  onCreated: () => void
  onBack: () => void
}

function parseParamsJson(json: string): Record<string, number> {
  try {
    const parsed = JSON.parse(json || '{}')
    const obj: Record<string, number> = {}
    for (const [k, v] of Object.entries(parsed)) {
      if (typeof v === 'number') obj[k] = v
    }
    return obj
  } catch {
    return {}
  }
}

export default function EditorTab({ id, onCreated, onBack }: EditorTabProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
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
  const [createdId, setCreatedId] = useState<string | null>(id)
  const fileRef = useRef<HTMLInputElement | null>(null)
  const [gkrLoading, setGkrLoading] = useState(false)
  const [paramDefs, setParamDefs] = useState<ParamDef[]>([])
  const [allParamDefs, setAllParamDefs] = useState<Record<string, ParamDef[]>>({})
  const [showRawJSON, setShowRawJSON] = useState(false)

  const inEngineTypes = STRATEGY_CATALOG.filter((c) => c.inEngine)

  useEffect(() => {
    strategies
      .paramDefs()
      .then((res) => {
        if (res?.defs) setAllParamDefs(res.defs)
      })
      .catch(() => {})
  }, [])

  useEffect(() => {
    const defs = allParamDefs[form.type]
    setParamDefs(defs ?? [])
  }, [form.type, allParamDefs])

  useEffect(() => {
    if (!id) return
    strategies
      .get(id)
      .then((s) => {
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
      })
      .catch(() => setMsg(t('strategyEditor:failedToLoad', 'Failed to load strategy')))
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
        setMsg(
          t('strategyEditor:loadedFromGkr', 'Loaded from GKR: {{name}} ({{type}})', {
            name: res.name,
            type: res.type,
          }),
        )
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
    setForm((prev) => ({ ...prev, paramsObj: params, parameters: JSON.stringify(params, null, 2) }))
  }

  const handleJSONParamsChange = (json: string) => {
    setForm((prev) => ({ ...prev, parameters: json, paramsObj: parseParamsJson(json) }))
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

  const handleSave = async () => {
    if (!form.name) return
    setSaving(true)
    setMsg('')
    try {
      const params = JSON.parse(form.parameters || '{}')
      if (createdId) {
        await strategies.update(createdId, {
          name: form.name,
          type: form.type,
          parameters: params,
          enabled: form.enabled,
        })
        setMsg(t('strategyEditor:strategyUpdated', 'Strategy updated'))
      } else {
        const res = await strategies.create({
          name: form.name,
          type: form.type,
          parameters: params,
          enabled: form.enabled,
        })
        setCreatedId(res.id)
        setMsg(t('strategyEditor:strategyCreated', 'Strategy "{{name}}" created ({{id}})', { name: res.name, id: res.id }))
        onCreated()
      }
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t('strategyEditor:saveFailed', 'Save failed'))
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
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={onBack}>
            {'\u2190'} {t('common:back', 'Back')}
          </Button>
          <h2 className="m-0 text-lg">
            {createdId
              ? t('strategyEditor:editStrategy', 'Edit Strategy')
              : t('strategyEditor:newStrategy', 'New Strategy')}
          </h2>
        </div>
        <div className="flex gap-2">
          <input type="file" ref={fileRef} accept=".yaml,.gkr.yaml" className="hidden" onChange={handleLoadGkr} />
          <Button variant="outline" onClick={() => fileRef.current?.click()} disabled={gkrLoading}>
            {gkrLoading ? t('strategyEditor:loading', 'Loading...') : t('strategyEditor:loadGkr', 'Load GKR')}
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-6 mb-4">
        <Card>
          <CardHeader>
            <CardTitle>
              {createdId ? t('strategyEditor:editStrategy', 'Edit Strategy') : t('strategyEditor:newStrategy', 'New Strategy')}
            </CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <div>
              <Label>{t('strategyEditor:name', 'Name')}</Label>
              <Input
                placeholder="my_strategy"
                value={form.name}
                onChange={(e) => updateField('name', e.target.value)}
              />
            </div>
            <div>
              <Label>{t('strategyEditor:type', 'Type')}</Label>
              <Select value={form.type} onValueChange={(v) => updateField('type', v)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {inEngineTypes.map((c) => (
                    <SelectItem key={c.typeKey} value={c.typeKey}>
                      {c.displayName}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div>
              <div className="flex items-center justify-between mb-2">
                <Label>{t('strategyEditor:parameters', 'Parameters')}</Label>
                {paramDefs.length > 0 && (
                  <Button variant="outline" size="sm" onClick={() => setShowRawJSON((v) => !v)}>
                    {showRawJSON
                      ? t('strategyEditor:structured', 'Structured')
                      : t('strategyEditor:rawJson', 'Raw JSON')}
                  </Button>
                )}
              </div>
              {showRawJSON || paramDefs.length === 0 ? (
                <Textarea
                  className="font-mono text-xs resize-y min-h-[120px]"
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
            <label className="flex items-center gap-2 text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={form.enabled}
                onChange={(e) => updateField('enabled', e.target.checked)}
              />
              {t('strategyEditor:enableImmediately', 'Enable immediately')}
            </label>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t('strategyEditor:validation', 'Validation')}</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <div className="flex gap-2">
              <Button variant="outline" onClick={handleValidate} disabled={validating || !form.name}>
                {validating ? t('strategyEditor:validating', 'Validating...') : t('strategyEditor:validate', 'Validate')}
              </Button>
              <Button onClick={handleSave} disabled={saving || !form.name}>
                {saving
                  ? t('strategyEditor:saving', 'Saving...')
                  : createdId
                    ? t('strategyEditor:save', 'Save')
                    : t('strategyEditor:createStrategy', 'Create Strategy')}
              </Button>
              {createdId && (
                <Button variant="outline" onClick={handleReset}>
                  {t('strategyEditor:new', 'New')}
                </Button>
              )}
            </div>

            {msg && (
              <p
                className={`text-sm ${
                  msg.includes('created') || msg.includes('Loaded') || msg.includes('compiled') || msg.includes('updated')
                    ? 'text-trading-success'
                    : 'text-destructive'
                }`}
              >
                {msg}
              </p>
            )}

            {validation && (
              <div className="space-y-2">
                <div
                  className={`p-2 rounded-md text-sm ${
                    validation.valid
                      ? 'bg-trading-success/10 text-trading-success'
                      : 'bg-destructive/10 text-destructive'
                  }`}
                >
                  {validation.valid
                    ? t('strategyEditor:strategyValid', '\u2713 Strategy is valid')
                    : t('strategyEditor:errorsFound', '\u2717 {{n}} error(s)', { n: validation.errors?.length ?? 0 })}
                </div>

                {validation.errors && validation.errors.length > 0 && (
                  <ul className="text-xs pl-4 m-0 space-y-1">
                    {validation.errors.map((e, i) => (
                      <li key={i} className="text-destructive">
                        {e}
                      </li>
                    ))}
                  </ul>
                )}

                {validation.diagnostics && validation.diagnostics.length > 0 && (
                  <div>
                    <Label>{t('strategyEditor:diagnostics', 'Diagnostics:')}</Label>
                    <pre className="text-xs overflow-auto max-h-[200px] mt-1">
                      {JSON.stringify(validation.diagnostics, null, 2)}
                    </pre>
                  </div>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {createdId && (
        <Card>
          <CardHeader>
            <CardTitle>{t('strategyEditor:quickActions', 'Quick Actions')}</CardTitle>
          </CardHeader>
          <CardContent className="flex gap-2">
            <Button
              variant="outline"
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
            </Button>
            <Button
              variant="outline"
              onClick={async () => {
                try {
                  const clone = await strategies.clone(createdId)
                  setMsg(
                    t('strategyEditor:clonedAs', 'Cloned as "{{name}}" ({{id}})', {
                      name: clone.name,
                      id: clone.id,
                    }),
                  )
                } catch (err) {
                  setMsg(err instanceof Error ? err.message : t('strategyEditor:cloneFailed', 'Clone failed'))
                }
              }}
            >
              {t('strategyEditor:clone', 'Clone')}
            </Button>
            <Button variant="outline" onClick={() => navigate(`/backtest?strategy=${createdId}`)}>
              {t('strategyEditor:quickBacktest', 'Quick Backtest')}
            </Button>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
