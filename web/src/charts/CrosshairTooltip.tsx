export interface CrosshairTooltipRow {
  label: string
  value: string
  color?: string
}

export interface CrosshairTooltipDetail {
  name: string
  values: Array<{ key: string; value: string }>
}

interface CrosshairTooltipProps {
  data: {
    timeStr: string
    rows: CrosshairTooltipRow[]
    detail?: CrosshairTooltipDetail[]
  } | null
  position?: { x: number; y: number }
}

export default function CrosshairTooltip({ data, position }: CrosshairTooltipProps) {
  if (!data) return null

  return (
    <div style={{
      position: 'absolute', top: position ? position.y + 10 : 4, left: position ? position.x + 10 : 4, zIndex: 50,
      background: 'var(--chart-tooltip-bg)',
      border: '1px solid var(--border)',
      borderRadius: 6, padding: '8px 10px',
      fontSize: 11, fontFamily: 'monospace',
      color: 'var(--muted-foreground)',
      pointerEvents: 'none',
      boxShadow: '0 2px 8px rgba(0,0,0,0.3)',
      maxWidth: 280,
    }}>
      <div style={{ color: 'var(--foreground)', fontWeight: 600, marginBottom: 4, fontSize: 10 }}>
        {data.timeStr}
      </div>
      {data.rows.map((row, i) => (
        <div key={i}>
          {row.label}{' '}
          <span style={{ color: row.color ?? 'var(--foreground)' }}>{row.value}</span>
        </div>
      ))}
      {data.detail && data.detail.map((section) => (
        <div key={section.name} style={{ marginTop: 4, marginBottom: 2 }}>
          <span style={{ color: 'var(--accent)', fontWeight: 500 }}>{section.name}</span>
          {section.values.map((v) => (
            <span key={v.key} style={{ marginLeft: 6 }}>
              {v.key} <span style={{ color: 'var(--foreground)' }}>{v.value}</span>
            </span>
          ))}
        </div>
      ))}
    </div>
  )
}
