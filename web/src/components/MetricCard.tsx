import { useRef, useState } from 'react'
import { formatNumber, formatUSD } from '../lib/format'

export interface MetricCardProps {
  label: string
  value: string | number
  format?: 'number' | 'percent' | 'percent_raw' | 'currency' | 'decimal'
  color?: 'default' | 'positive' | 'negative' | 'auto'
  tooltip?: string
  trend?: 'up' | 'down' | 'neutral'
  onClick?: () => void
  skeleton?: boolean
}

export default function MetricCard({
  label,
  value,
  format = 'decimal',
  color = 'default',
  tooltip,
  trend,
  onClick,
  skeleton,
}: MetricCardProps) {
  const [showTooltip, setShowTooltip] = useState(false)
  const tipRef = useRef<HTMLDivElement | null>(null)

  if (skeleton) {
    return (
      <div className="metric-card skeleton-pulse" style={{ opacity: 0.5 }}>
        <div className="metric-label" style={{ height: 12, width: '60%', background: 'var(--bg-input)', borderRadius: 4, marginBottom: 6 }} />
        <div className="metric-value" style={{ height: 20, width: '40%', background: 'var(--bg-input)', borderRadius: 4 }} />
      </div>
    )
  }

  const num = typeof value === 'string' ? parseFloat(value) : value

  const formatted = (() => {
    if (typeof value === 'string' && isNaN(num)) return value
    switch (format) {
      case 'number': return formatNumber(num, 0)
      case 'percent': return typeof value === 'number' ? `${(num * 100).toFixed(1)}%` : value
      case 'percent_raw': return typeof value === 'number' ? `${num.toFixed(1)}%` : value
      case 'currency': return formatUSD(num)
      default: return formatNumber(num, 2)
    }
  })()

  const resolvedColor = (() => {
    if (color === 'default') return undefined
    if (color === 'auto') {
      if (typeof value === 'string') return undefined
      return num >= 0 ? 'var(--success)' : 'var(--danger)'
    }
    return color === 'positive' ? 'var(--success)' : color === 'negative' ? 'var(--danger)' : undefined
  })()

  const trendIcon = trend === 'up' ? '↑' : trend === 'down' ? '↓' : ''

  return (
    <div
      className="metric-card"
      onClick={onClick}
      style={{ cursor: onClick ? 'pointer' : undefined, position: 'relative' }}
      onMouseEnter={() => setShowTooltip(true)}
      onMouseLeave={() => setShowTooltip(false)}
    >
      <div className="metric-label">
        {label}
        {trendIcon && <span style={{ marginLeft: 4, fontSize: 10 }}>{trendIcon}</span>}
      </div>
      <div className="metric-value" style={{ color: resolvedColor }}>
        {formatted}
      </div>
      {tooltip && showTooltip && (
        <div
          ref={tipRef}
          style={{
            position: 'absolute', bottom: '100%', left: '50%', transform: 'translateX(-50%)',
            background: 'var(--bg-input)', border: '1px solid var(--border)', borderRadius: 4,
            padding: '4px 8px', fontSize: 11, whiteSpace: 'nowrap', zIndex: 100,
            marginBottom: 4, pointerEvents: 'none',
          }}
        >
          {tooltip}
        </div>
      )}
    </div>
  )
}
