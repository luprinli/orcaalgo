import { useTimeframeStore, TIMEFRAME_OPTIONS } from '../stores/timeframeStore'

interface TimeframeChipsProps {
  variant?: 'inline' | 'toolbar'
}

export default function TimeframeChips({ variant = 'toolbar' }: TimeframeChipsProps) {
  const { timeframe, setTimeframe } = useTimeframeStore()
  const isToolbar = variant === 'toolbar'

  return (
    <div style={{
      display: 'flex', gap: 2,
      background: isToolbar ? 'var(--bg-secondary)' : undefined,
      borderRadius: 6, padding: isToolbar ? 2 : 0,
    }}>
      {TIMEFRAME_OPTIONS.map(({ label, value }) => (
        <button
          key={value}
          onClick={() => setTimeframe(value)}
          title={label}
          style={{
            padding: isToolbar ? '3px 10px' : '4px 8px',
            fontSize: 11, fontWeight: 600, fontFamily: 'inherit',
            cursor: 'pointer',
            border: 'none', borderRadius: 4,
            background: timeframe === value
              ? 'var(--accent)'
              : 'transparent',
            color: timeframe === value
              ? '#fff'
              : 'var(--text-secondary)',
            transition: 'background .15s, color .15s',
          }}
        >
          {value}
        </button>
      ))}
    </div>
  )
}
