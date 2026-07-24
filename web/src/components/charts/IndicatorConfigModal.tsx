import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useIndicatorCompute } from '../../hooks/useIndicator'
import { useIndicatorStore } from '../../stores/indicatorStore'
import { indicators as indicatorsApi } from '../../api/client'
import { IndicatorSpec, IndicatorWithData } from '../../types/api'

interface IndicatorConfigModalProps {
  open: boolean
  onClose: () => void
  candles: Array<{ time: number; open: number; high: number; low: number; close: number; volume: number }>
}

export function IndicatorConfigModal({ open, onClose, candles }: IndicatorConfigModalProps) {
  const { t } = useTranslation()
  const [specs, setSpecs] = useState<IndicatorSpec[]>([])
  const [loaded, setLoaded] = useState(false)
  const { indicators, addIndicator, removeIndicator, updateParameters } = useIndicatorStore()
  const { compute } = useIndicatorCompute()

  // Load available indicator specs on open
  useState(() => {
    if (open && !loaded) {
      indicatorsApi.list()
        .then((data: any) => setSpecs(data?.indicators ?? data ?? []))
        .catch(() => {})
        .finally(() => setLoaded(true))
    }
  })

  if (!open) return null

  const activeIds = Object.keys(indicators)
  const availableSpecs = specs.filter(s => !activeIds.includes(s.id))

  const handleAdd = async (spec: IndicatorSpec) => {
    const id = addIndicator(spec, Object.fromEntries(spec.parameters.map(p => [p.name, p.default])))
    if (candles.length > 0) {
      await compute(id, candles as any)
    }
  }

  return (
    <div style={{
      position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, zIndex: 1000,
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      background: 'rgba(0,0,0,0.6)',
    }} onClick={onClose}>
      <div className="card" style={{ maxWidth: 480, width: '90%', maxHeight: '80vh', overflowY: 'auto' }}
        onClick={e => e.stopPropagation()}>
        <div className="flex-between mb-3">
          <h2 style={{ margin: 0, fontSize: 15 }}>Indicators</h2>
          <button className="btn btn-outline" onClick={onClose} style={{ padding: '4px 10px', fontSize: 18 }}>×</button>
        </div>

        {/* Active indicators */}
        {activeIds.length > 0 && (
          <div className="mb-4">
            <h3 className="text-xs text-slate-400 uppercase mb-2">Active</h3>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              {activeIds.map(id => {
                const ind = indicators[id]
                return (
                  <div key={id} className="flex-between" style={{ padding: '6px 10px', background: 'var(--bg-secondary)', borderRadius: 4 }}>
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

        {/* Available indicators */}
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
                  onClick={() => handleAdd(spec)}>
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
