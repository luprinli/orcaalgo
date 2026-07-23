import { useEffect, useState } from 'react'
import { system } from '../../api/client'
import type { SystemHealth } from '../../types/api'

function gaugeColor(ratio: number): string {
  if (ratio >= 0.9) return 'var(--danger)'
  if (ratio >= 0.7) return 'var(--warning, #d29922)'
  return 'var(--success)'
}

function Gauge({ label, value, max, unit }: { label: string; value: number; max: number; unit: string }) {
  const ratio = max > 0 ? Math.min(1, value / max) : 0
  const color = gaugeColor(ratio)
  return (
    <div style={{ minWidth: 120 }}>
      <div className="flex-between" style={{ fontSize: 10 }}>
        <span className="text-muted">{label}</span>
        <span style={{ color }}>{value.toFixed(0)}/{max.toFixed(0)}{unit}</span>
      </div>
      <div style={{ height: 5, background: 'var(--bg-tertiary)', borderRadius: 3, overflow: 'hidden', marginTop: 2 }}>
        <div style={{ height: '100%', width: `${ratio * 100}%`, background: color, borderRadius: 3, transition: 'width .4s ease' }} />
      </div>
    </div>
  )
}

/**
 * ResourceGauges polls /system/health and shows heap, DB-pool, and worker
 * utilization with amber/red thresholds + a near-capacity banner — so users can
 * see when the engine is loaded (execution-framework plan §2.2/§5.2/§11.3).
 * Polls only while `active` to avoid idle chatter.
 */
export default function ResourceGauges({ active }: { active: boolean }) {
  const [health, setHealth] = useState<SystemHealth | null>(null)

  useEffect(() => {
    if (!active) return
    let cancelled = false
    const tick = async () => {
      try {
        const h = await system.health()
        if (!cancelled) setHealth(h)
      } catch { /* transient */ }
    }
    void tick()
    const id = setInterval(tick, 3000)
    return () => { cancelled = true; clearInterval(id) }
  }, [active])

  if (!health) return null

  return (
    <div className="card mb-3" style={{ padding: '8px 14px' }}>
      {health.near_capacity && (
        <div style={{ color: 'var(--danger)', fontSize: 11, marginBottom: 6 }}>
          Engine near capacity — new heavy runs may be slow or queued.
        </div>
      )}
      <div className="flex gap-4" style={{ flexWrap: 'wrap' }}>
        <Gauge label="Heap" value={health.heap_inuse_mb} max={health.heap_budget_mb} unit="MB" />
        <Gauge label="DB pool" value={health.db_pool_in_use} max={health.db_pool_max} unit="" />
        <div style={{ fontSize: 10 }} className="text-muted">
          workers <b>{health.matrix_workers}</b> · goroutines <b>{health.num_goroutine}</b> · CPU <b>{health.num_cpu}</b>
        </div>
      </div>
    </div>
  )
}
