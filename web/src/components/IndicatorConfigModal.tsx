import { useState, useEffect, useRef, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useIndicatorCompute } from '../hooks/useIndicator'
import { useIndicatorStore } from '../stores/indicatorStore'
import { indicators as indicatorsApi } from '../api/client'
import type { IndicatorSpec, IndicatorParamDef } from '../types/api'

interface IndicatorConfigModalProps {
  open: boolean
  onClose: () => void
  candles: Array<{ time: number; open: number; high: number; low: number; close: number; volume: number }>
}

export default function IndicatorConfigModal({ open, onClose, candles }: IndicatorConfigModalProps) {
  const { t } = useTranslation()
  const [specs, setSpecs] = useState<IndicatorSpec[]>([])
  const [loaded, setLoaded] = useState(false)
  const [configuringSpec, setConfiguringSpec] = useState<IndicatorSpec | null>(null)
  const [params, setParams] = useState<Record<string, number | string>>({})
  const { indicators, addIndicator, removeIndicator } = useIndicatorStore()
  const { compute } = useIndicatorCompute()

  const onCloseRef = useRef(onClose)
  onCloseRef.current = onClose

  useEffect(() => {
    if (open && !loaded) {
      indicatorsApi.list()
        .then((data: any) => setSpecs(data?.indicators ?? data ?? []))
        .catch(() => {})
        .finally(() => setLoaded(true))
    }
  }, [open, loaded])

  useEffect(() => {
    if (!open) {
      setConfiguringSpec(null)
      setParams({})
      return
    }
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (configuringSpec) {
          setConfiguringSpec(null)
        } else {
          onCloseRef.current()
        }
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [open, configuringSpec])

  if (!open) return null

  const activeIds = Object.keys(indicators)
  const availableSpecs = specs.filter(s => !activeIds.includes(s.id))

  const handleStartConfig = useCallback((spec: IndicatorSpec) => {
    const defaults: Record<string, number | string> = {}
    for (const p of spec.parameters) {
      defaults[p.name] = p.default
    }
    setParams(defaults)
    setConfiguringSpec(spec)
  }, [])

  const handleNumericChange = useCallback((param: IndicatorParamDef, value: string) => {
    if (value === '' || value === '-') {
      setParams(prev => ({ ...prev, [param.name]: value }))
      return
    }
    const num = param.type === 'int' ? parseInt(value, 10) : parseFloat(value)
    if (isNaN(num)) return
    if (param.min !== undefined && num < param.min) return
    if (param.max !== undefined && num > param.max) return
    setParams(prev => ({ ...prev, [param.name]: num }))
  }, [])

  const handleReset = useCallback((param: IndicatorParamDef) => {
    setParams(prev => ({ ...prev, [param.name]: param.default }))
  }, [])

  const handleApply = useCallback(async () => {
    if (!configuringSpec) return
    const cleaned: Record<string, number | string> = {}
    for (const param of configuringSpec.parameters) {
      const val = params[param.name]
      if (val === '' || val === '-' || val === undefined) {
        cleaned[param.name] = param.default
      } else if (param.type === 'int') {
        cleaned[param.name] = typeof val === 'string' ? parseInt(val, 10) : val
      } else if (param.type === 'float') {
        cleaned[param.name] = typeof val === 'string' ? parseFloat(val) : val
      } else {
        cleaned[param.name] = String(val)
      }
    }
    const id = addIndicator(configuringSpec, cleaned)
    if (candles.length > 0) {
      await compute(id, candles as any)
    }
    setConfiguringSpec(null)
  }, [configuringSpec, params, addIndicator, compute, candles])

  const handleAddQuick = useCallback(async (spec: IndicatorSpec) => {
    const defaults: Record<string, number | string> = {}
    for (const p of spec.parameters) {
      defaults[p.name] = p.default
    }
    const id = addIndicator(spec, defaults)
    if (candles.length > 0) {
      await compute(id, candles as any)
    }
  }, [addIndicator, compute, candles])

  if (configuringSpec) {
    return (
      <div
        style={{
          position: 'fixed', inset: 0, zIndex: 1000,
          background: 'rgba(0,0,0,0.6)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }}
        onClick={(e) => { if (e.target === e.currentTarget) setConfiguringSpec(null) }}
      >
        <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4" style={{ width: 420, maxWidth: '90vw', maxHeight: '80vh', overflowY: 'auto' }}>
          <div className="flex-between" style={{ marginBottom: 2 }}>
            <h2 style={{ margin: 0 }}>{configuringSpec.name}</h2>
            <button className="btn btn-outline" onClick={() => setConfiguringSpec(null)} style={{ padding: '4px 10px', fontSize: 18 }}>×</button>
          </div>
          <p className="text-muted" style={{ margin: '0 0 16px' }}>{configuringSpec.description}</p>

          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {configuringSpec.parameters.map(param => {
              const currentVal = params[param.name]
              const isModified = currentVal !== param.default
              return (
                <div key={param.name}>
                  <div className="flex-between" style={{ marginBottom: 4 }}>
                    <label style={{ fontSize: 12, color: 'var(--muted-foreground)', textTransform: 'capitalize' }}>
                      {param.name}
                      {param.description && (
                        <span className="text-muted" style={{ marginLeft: 6, textTransform: 'none' }}>({param.description})</span>
                      )}
                    </label>
                    <button
                      className="btn btn-outline"
                      style={{ fontSize: 10, padding: '1px 6px', opacity: isModified ? 1 : 0.4 }}
                      onClick={() => handleReset(param)}
                      disabled={!isModified}
                    >
                      Reset
                    </button>
                  </div>
                  {param.options ? (
                    <select
                      className="input"
                      value={String(currentVal ?? param.default)}
                      onChange={e => setParams(prev => ({ ...prev, [param.name]: e.target.value }))}
                    >
                      {param.options.map(opt => (
                        <option key={opt} value={opt}>{opt}</option>
                      ))}
                    </select>
                  ) : param.type === 'string' ? (
                    <input
                      className="input"
                      type="text"
                      value={String(currentVal ?? param.default)}
                      onChange={e => setParams(prev => ({ ...prev, [param.name]: e.target.value }))}
                    />
                  ) : (
                    <input
                      className="input"
                      type="number"
                      value={currentVal ?? param.default}
                      min={param.min}
                      max={param.max}
                      step={param.type === 'int' ? (param.step ?? 1) : (param.step ?? 0.1)}
                      onChange={e => handleNumericChange(param, e.target.value)}
                    />
                  )}
                </div>
              )
            })}
          </div>

          <div className="flex gap-2" style={{ justifyContent: 'flex-end', marginTop: 20 }}>
            <button className="btn btn-outline" onClick={() => setConfiguringSpec(null)}>Cancel</button>
            <button className="btn btn-primary" onClick={handleApply}>Apply</button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div
      style={{
        position: 'fixed', inset: 0, zIndex: 1000,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        background: 'rgba(0,0,0,0.6)',
      }}
      onClick={(e) => { if (e.target === e.currentTarget) onClose() }}
    >
      <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4" style={{ maxWidth: 480, width: '90%', maxHeight: '80vh', overflowY: 'auto' }}>
        <div className="flex-between mb-3">
          <h2 style={{ margin: 0, fontSize: 15 }}>Indicators</h2>
          <button className="btn btn-outline" onClick={onClose} style={{ padding: '4px 10px', fontSize: 18 }}>×</button>
        </div>

        {activeIds.length > 0 && (
          <div className="mb-4">
            <h3 className="text-xs text-slate-400 uppercase mb-2">Active</h3>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              {activeIds.map(id => {
                const ind = indicators[id]
                return (
                  <div key={id} className="flex-between" style={{ padding: '6px 10px', background: 'var(--muted)', borderRadius: 4 }}>
                    <span className="text-sm text-white">{ind.spec.name || id}</span>
                    <button className="btn btn-outline text-xs" onClick={() => removeIndicator(id)} style={{ padding: '2px 8px' }}>
                      Remove
                    </button>
                  </div>
                )
              })}
            </div>
          </div>
        )}

        <div>
          <h3 className="text-xs text-slate-400 uppercase mb-2">Add Indicator</h3>
          {availableSpecs.length === 0 ? (
            <p className="text-muted text-xs">All available indicators are active.</p>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
              {availableSpecs.map(spec => (
                <button key={spec.id}
                  className="btn btn-outline"
                  style={{ justifyContent: 'flex-start', textAlign: 'left', padding: '8px 12px' }}
                  onClick={() => spec.parameters.length > 0 ? handleStartConfig(spec) : handleAddQuick(spec)}>
                  <div>
                    <div className="text-sm text-white">{spec.name}</div>
                    {spec.description && <div className="text-muted text-xs">{spec.description}</div>}
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
