import type { Candle } from '../types/api'

// eslint-disable-next-line react-refresh/only-export-components
export function formatVolume(v: number): string {
  if (v >= 1e6) return (v / 1e6).toFixed(1) + 'M'
  if (v >= 1e3) return (v / 1e3).toFixed(1) + 'K'
  return String(v)
}

function formatPrice(p: number): string {
  if (p >= 1) return p.toFixed(2)
  if (p >= 0.01) return p.toFixed(4)
  return p.toFixed(6)
}

export function OHLCVHeader({ candle }: { candle?: Candle }) {
  if (!candle) {
    return (
      <div style={{ padding: '6px 10px', borderBottom: '1px solid var(--border)', display: 'flex', gap: 16, fontSize: 11, fontFamily: 'monospace', color: 'var(--text-secondary)', background: 'var(--bg-secondary)' }}>
        <span>Waiting for data...</span>
      </div>
    )
  }
  const change = candle.close - candle.open
  const changePct = candle.open !== 0 ? (change / candle.open) * 100 : 0
  const color = change >= 0 ? 'var(--success)' : 'var(--danger)'

  return (
    <div style={{ padding: '6px 10px', borderBottom: '1px solid var(--border)', display: 'flex', gap: 16, fontSize: 11, fontFamily: 'monospace', color: 'var(--text-secondary)', background: 'var(--bg-secondary)', flexWrap: 'wrap' }}>
      <span>O: <span style={{ color: 'var(--text-primary)' }}>{formatPrice(candle.open)}</span></span>
      <span>H: <span style={{ color: 'var(--text-primary)' }}>{formatPrice(candle.high)}</span></span>
      <span>L: <span style={{ color: 'var(--text-primary)' }}>{formatPrice(candle.low)}</span></span>
      <span>C: <span style={{ color: 'var(--text-primary)' }}>{formatPrice(candle.close)}</span></span>
      <span>V: <span style={{ color: 'var(--text-primary)' }}>{formatVolume(candle.volume)}</span></span>
      <span style={{ color }}>{change >= 0 ? '+' : ''}{changePct.toFixed(2)}%</span>
    </div>
  )
}
