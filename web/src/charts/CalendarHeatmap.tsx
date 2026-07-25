import { useState, useMemo } from 'react'
import type { MonthlyReturn } from '../types/api'

interface CalendarHeatmapProps {
  data: MonthlyReturn[]
  height?: number
  onMonthClick?: (year: number, month: number) => void
}

const MONTH_LABELS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

export default function CalendarHeatmap({ data, height = 260, onMonthClick }: CalendarHeatmapProps) {
  const [tooltip, setTooltip] = useState<{ year: number; month: number; ret: number; x: number; y: number } | null>(null)

  const { grouped, minRet, maxRet } = useMemo(() => {
    const g: Record<number, Record<number, number>> = {}
    let min = 0, max = 0
    for (const d of data) {
      if (!g[d.year]) g[d.year] = {}
      g[d.year][d.month] = d.return_pct
      if (d.return_pct < min) min = d.return_pct
      if (d.return_pct > max) max = d.return_pct
    }
    return { grouped: g, minRet: min, maxRet: max }
  }, [data])

  const years = Object.keys(grouped).map(Number).sort()
  if (years.length === 0) return null

  const cellSize = 24
  const gap = 4
  const colW = cellSize + gap
  const rowH = cellSize + gap
  const headerH = 28
  const padding = { top: 8, left: 40, right: 16, bottom: 16 }
  const w = years.length * colW + padding.left + padding.right
  const h = 12 * rowH + headerH + padding.top + padding.bottom

  function getColor(ret: number): string {
    const abs = Math.max(Math.abs(minRet), Math.abs(maxRet))
    if (abs === 0) return 'var(--input)'
    const intensity = Math.min(Math.abs(ret) / abs, 1)
    if (ret > 0) {
      const g = Math.round(200 - intensity * 160)
      return `rgb(${g}, 230, ${g})`
    }
    if (ret < 0) {
      const r = Math.round(200 + intensity * 55)
      return `rgb(${r}, ${170 - intensity * 90}, ${170 - intensity * 90})`
    }
    return 'var(--input)'
  }

  return (
    <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4" style={{ position: 'relative' }}>
      <div className="flex items-center justify-between border-b border-border pb-2 mb-3"><h3>Monthly Returns Heatmap</h3></div>
      <svg width={w} height={Math.min(h, height)} style={{ display: 'block', margin: '0 auto' }} role="img" aria-label="Monthly returns heatmap">
        {years.map((year, yi) => (
          <text key={`y-${year}`} x={padding.left + yi * colW + cellSize / 2} y={padding.top + headerH - 6}
            textAnchor="middle" fill="var(--text-muted)" fontSize={11}>
            {year}
          </text>
        ))}
        {MONTH_LABELS.map((ml, mi) => (
          <text key={`m-${ml}`} x={padding.left - 8} y={padding.top + headerH + mi * rowH + cellSize / 2 + 4}
            textAnchor="end" fill="var(--text-muted)" fontSize={10}>
            {ml}
          </text>
        ))}
        {years.map((year, yi) =>
          MONTH_LABELS.map((_ml, mi) => {
            const month = mi + 1
            const ret = grouped[year]?.[month]
            if (ret === undefined) return null
            const x = padding.left + yi * colW
            const y = padding.top + headerH + mi * rowH
            return (
              <rect
                key={`r-${year}-${month}`}
                x={x} y={y} width={cellSize} height={cellSize} rx={3}
                fill={getColor(ret)}
                stroke={tooltip?.year === year && tooltip?.month === month ? 'var(--chart-line)' : 'transparent'}
                strokeWidth={1.5}
                style={{ cursor: onMonthClick ? 'pointer' : 'default', transition: 'stroke 0.15s' }}
                onMouseEnter={(e) => {
                  const rect = (e.target as SVGRectElement).getBoundingClientRect()
                  setTooltip({ year, month, ret, x: rect.left + rect.width / 2, y: rect.top })
                }}
                onMouseLeave={() => setTooltip(null)}
                onClick={() => onMonthClick?.(year, month)}
              />
            )
          })
        )}
      </svg>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, justifyContent: 'center', marginTop: 8, fontSize: 11, color: 'var(--text-muted)' }}>
        <span style={{ color: 'var(--trading-danger)' }}>{minRet.toFixed(2)}%</span>
        <div style={{ width: 120, height: 12, borderRadius: 3, background: `linear-gradient(to right, rgb(255,${170-90},${170-90}), var(--input), rgb(${200-160}, 230, ${200-160}))` }} />
        <span style={{ color: 'var(--trading-success)' }}>+{maxRet.toFixed(2)}%</span>
      </div>
      {tooltip && (
        <div style={{
          position: 'fixed', left: tooltip.x, top: tooltip.y - 8, transform: 'translate(-50%, -100%)',
          background: 'var(--input)', border: '1px solid var(--border)', borderRadius: 6,
          padding: '6px 10px', fontSize: 12, zIndex: 1000, pointerEvents: 'none', whiteSpace: 'nowrap',
          boxShadow: '0 2px 8px rgba(0,0,0,0.2)',
        }}>
          {MONTH_LABELS[tooltip.month - 1]} {tooltip.year}: <span style={{ color: tooltip.ret >= 0 ? 'var(--trading-success)' : 'var(--trading-danger)' }}>
            {tooltip.ret >= 0 ? '+' : ''}{tooltip.ret.toFixed(2)}%
          </span>
        </div>
      )}
    </div>
  )
}
