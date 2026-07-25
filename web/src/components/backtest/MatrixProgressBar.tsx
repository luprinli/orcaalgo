import { useMatrixStore } from '../../stores/matrixStore'
import ChunkTracker from './ChunkTracker'

function fmtEta(seconds: number): string {
  if (!seconds || seconds <= 0 || !isFinite(seconds)) return '—'
  const s = Math.round(seconds)
  if (s < 60) return `${s}s`
  const m = Math.floor(s / 60)
  const rem = s % 60
  if (m < 60) return `${m}m ${rem}s`
  const h = Math.floor(m / 60)
  return `${h}h ${m % 60}m`
}

const STATUS_COLOR: Record<string, string> = {
  running: 'var(--accent)',
  queued: 'var(--muted-foreground)',
  completed: 'var(--trading-success)',
  failed: 'var(--trading-danger)',
  cancelled: 'var(--trading-danger)',
}

/**
 * MatrixProgressBar renders live telemetry (%, throughput, ETA, current task)
 * from matrixStore — parity with the backend's per-combo streaming (plan §11.3).
 * Subscribes only to the telemetry/status slices to avoid table re-renders.
 */
export default function MatrixProgressBar() {
  const status = useMatrixStore((s) => s.status)
  const t = useMatrixStore((s) => s.telemetry)

  if (status === 'idle') return null

  // Floor percent: the local "completed/total" is authoritative (avoids showing
  // 100.0% from a server "percent" derived via a different rounding/EMA).
  const pct = t.total > 0 ? Math.min(99.9, (t.completed / t.total) * 100) : 0
  const color = STATUS_COLOR[status] ?? 'var(--accent)'

  return (
    <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4 mb-3" style={{ padding: '10px 14px' }}>
      <div className="flex-between mb-1" style={{ fontSize: 12 }}>
        <span style={{ fontWeight: 600, textTransform: 'capitalize', color }}>
          {status}{t.phase === 'screening' ? ' · screening' : ''}
        </span>
        <span className="text-muted">
          {t.completed}/{t.total} combos{t.failed > 0 ? ` · ${t.failed} failed` : ''}{t.skipped > 0 ? ` · ${t.skipped} screened out` : ''}
        </span>
      </div>

      <div style={{ height: 8, background: 'var(--bg-tertiary)', borderRadius: 4, overflow: 'hidden' }}>
        <div style={{
          height: '100%', width: `${pct}%`, background: color,
          transition: 'width .4s ease', borderRadius: 4,
        }} />
      </div>

      <div className="flex gap-3 mt-1" style={{ fontSize: 11, flexWrap: 'wrap' }}>
        <span>{pct.toFixed(1)}%</span>
        <span className="text-muted">throughput <b>{(t.throughputPerMin || 0).toFixed(0)}</b>/min</span>
        <span className="text-muted">ETA <b>{fmtEta(t.etaSeconds)}</b></span>
        <ChunkTracker />
        {t.current && (
          <span className="text-muted">
            current <b>{t.current.strategy}</b> · {t.current.symbol} · {t.current.timeframe}
          </span>
        )}
        {t.bestStrategy && (
          <span className="text-muted" style={{ marginLeft: 'auto' }}>
            best Sharpe <b>{t.bestSharpe.toFixed(2)}</b> ({t.bestStrategy}/{t.bestSymbol})
          </span>
        )}
      </div>
    </div>
  )
}
