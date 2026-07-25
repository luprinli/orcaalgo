import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import type { ParamDef } from '../types/api'

interface ParamEditorProps {
  defs: ParamDef[]
  initialParams?: Record<string, number>
  onChange: (params: Record<string, number>) => void
  compact?: boolean
}

function groupBy<T>(items: T[], keyFn: (item: T) => string): Record<string, T[]> {
  const groups: Record<string, T[]> = {}
  for (const item of items) {
    const key = keyFn(item)
    if (!groups[key]) groups[key] = []
    groups[key].push(item)
  }
  return groups
}

export default function ParamEditor({ defs, initialParams = {}, onChange, compact = false }: ParamEditorProps) {
  const { t } = useTranslation()
  const [params, setParams] = useState<Record<string, number>>(() => {
    const initial: Record<string, number> = {}
    for (const d of defs) {
      initial[d.name] = initialParams[d.name] ?? d.default
    }
    return initial
  })

  const updateParam = useCallback((name: string, value: number) => {
    setParams(prev => {
      const next = { ...prev, [name]: value }
      onChange(next)
      return next
    })
  }, [onChange])

  const handleReset = useCallback((d: ParamDef) => {
    updateParam(d.name, d.default)
  }, [updateParam])

  const grouped = groupBy(defs, d => d.group || 'General')
  const groupOrder = Object.keys(grouped)

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: compact ? 6 : 14 }}>
      {groupOrder.map(group => (
        <div key={group}>
          <div className="text-muted" style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: 1, marginBottom: compact ? 4 : 8 }}>
            {group}
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: compact ? '1fr' : 'repeat(auto-fill, minmax(220px, 1fr))', gap: compact ? 4 : 10 }}>
            {grouped[group].map(d => {
              const value = params[d.name] ?? d.default
              const isDefault = value === d.default
              const displayPrecision = d.type === 'integer' ? 0 : (d.step < 1 ? String(d.step).split('.')[1]?.length ?? 1 : 1)

              return (
                <div key={d.name} style={{
                  border: `1px solid var(--border)`,
                  borderRadius: 'var(--radius-sm)',
                  padding: compact ? '6px 8px' : '8px 10px',
                  background: !isDefault ? 'rgba(63,185,80,.05)' : 'var(--muted)',
                }}>
                  <div className="flex-between" style={{ marginBottom: 4 }}>
                    <label
                      title={d.description}
                      style={{ fontSize: 11, fontWeight: 600, color: 'var(--foreground)', cursor: 'help' }}
                    >
                      {d.name}
                    </label>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      <input
                        type="number"
                        style={{
                          width: 70, fontSize: 11, fontFamily: 'monospace',
                          background: 'var(--input)', border: '1px solid var(--border)',
                          borderRadius: 3, color: 'var(--foreground)',
                          padding: '2px 5px', textAlign: 'right',
                        }}
                        value={d.type === 'integer' ? Math.round(value) : Number(value.toFixed(displayPrecision))}
                        min={d.min}
                        max={d.max}
                        step={d.type === 'integer' ? Math.max(1, Math.round(d.step)) : d.step}
                        onChange={e => {
                          const v = parseFloat(e.target.value)
                          if (!isNaN(v) && v >= d.min && v <= d.max) {
                            updateParam(d.name, d.type === 'integer' ? Math.round(v) : v)
                          }
                        }}
                      />
                      {!isDefault && (
                        <button
                          className="btn btn-outline"
                          title={t('components:paramEditor.resetToDefault', 'Reset to default')}
                          style={{ fontSize: 9, padding: '1px 5px' }}
                          onClick={() => handleReset(d)}
                        >
                          R
                        </button>
                      )}
                    </div>
                  </div>
                  <input
                    type="range"
                    style={{ width: '100%', height: 4, cursor: 'pointer', accentColor: 'var(--accent)' }}
                    value={value}
                    min={d.min}
                    max={d.max}
                    step={d.type === 'integer' ? Math.max(1, Math.round(d.step)) : d.step}
                    onChange={e => {
                      const v = parseFloat(e.target.value)
                      updateParam(d.name, d.type === 'integer' ? Math.round(v) : v)
                    }}
                  />
                  <div className="flex-between" style={{ fontSize: 10, color: 'var(--muted-foreground)', marginTop: 2 }}>
                    <span>{d.type === 'integer' ? Math.round(d.min) : Number(d.min.toFixed(displayPrecision))}</span>
                    <span>{d.type === 'integer' ? Math.round(d.max) : Number(d.max.toFixed(displayPrecision))}</span>
                  </div>
                  {d.description && !compact && (
                    <div style={{ fontSize: 10, color: 'var(--muted-foreground)', marginTop: 2, fontStyle: 'italic' }}>
                      {d.description}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        </div>
      ))}
    </div>
  )
}
