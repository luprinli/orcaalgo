import { useState, useEffect } from 'react'
import type { IndicatorSpec, IndicatorParamDef } from '../types/api'

interface Props {
  spec: IndicatorSpec
  initialParams: Record<string, number | string>
  onApply: (params: Record<string, number | string>) => void
  onCancel: () => void
}

export default function IndicatorConfigModal({ spec, initialParams, onApply, onCancel }: Props) {
  const [params, setParams] = useState<Record<string, number | string>>({ ...initialParams })

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCancel()
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onCancel])

  const handleNumericChange = (param: IndicatorParamDef, value: string) => {
    if (value === '' || value === '-') {
      setParams(prev => ({ ...prev, [param.name]: value }))
      return
    }
    const num = param.type === 'int' ? parseInt(value, 10) : parseFloat(value)
    if (isNaN(num)) return
    if (param.min !== undefined && num < param.min) return
    if (param.max !== undefined && num > param.max) return
    setParams(prev => ({ ...prev, [param.name]: num }))
  }

  const handleReset = (param: IndicatorParamDef) => {
    setParams(prev => ({ ...prev, [param.name]: param.default }))
  }

  const handleApply = () => {
    const cleaned: Record<string, number | string> = {}
    for (const param of spec.parameters) {
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
    onApply(cleaned)
  }

  return (
    <div
      style={{
        position: 'fixed', inset: 0, zIndex: 1000,
        background: 'rgba(0,0,0,0.6)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
      }}
      onClick={(e) => { if (e.target === e.currentTarget) onCancel() }}
    >
      <div className="card" style={{ width: 420, maxWidth: '90vw', maxHeight: '80vh', overflowY: 'auto' }}>
        <h2 style={{ margin: '0 0 2px' }}>{spec.name}</h2>
        <p className="text-muted" style={{ margin: '0 0 16px' }}>{spec.description}</p>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          {spec.parameters.map(param => {
            const currentVal = params[param.name]
            const isModified = currentVal !== param.default
            return (
              <div key={param.name}>
                <div className="flex-between" style={{ marginBottom: 4 }}>
                  <label style={{ fontSize: 12, color: 'var(--text-secondary)', textTransform: 'capitalize' }}>
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
          <button className="btn btn-outline" onClick={onCancel}>Cancel</button>
          <button className="btn btn-primary" onClick={handleApply}>Apply</button>
        </div>
      </div>
    </div>
  )
}
