import { useState, useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { symbols as symbolsApi } from '../api/client'

interface SymbolItem {
  ticker: string
  exchange: string
  asset_type: string
  id: number
  is_active: boolean
}

interface SymbolSearchProps {
  value: string
  onChange: (symbol: string) => void
  placeholder?: string
  style?: React.CSSProperties
}

export default function SymbolSearch({ value, onChange, placeholder, style }: SymbolSearchProps) {
  const { t } = useTranslation()
  const [symbols, setSymbols] = useState<SymbolItem[]>([])
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState(value)
  const containerRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    setQuery(value)
  }, [value])

  useEffect(() => {
    symbolsApi.list()
      .then(d => {
        const all = (d ?? []) as unknown as SymbolItem[]
        setSymbols(all)
      })
      .catch(() => {})
  }, [])

  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  const filtered = query
    ? symbols.filter(s => s.ticker.toLowerCase().includes(query.toLowerCase()) && s.is_active)
    : symbols.filter(s => s.is_active).slice(0, 20)

  const handleSelect = (ticker: string) => {
    setQuery(ticker)
    setOpen(false)
    onChange(ticker)
  }

  return (
    <div ref={containerRef} style={{ position: 'relative', ...style }}>
      <input
        className="input"
        style={{ width: 100, boxSizing: 'border-box' }}
        value={query}
        onChange={e => {
          const v = e.target.value.toUpperCase()
          setQuery(v)
          setOpen(true)
        }}
        onFocus={() => setOpen(true)}
        placeholder={placeholder ?? t('components:symbolSearch.placeholder', 'Search symbols...')}
      />
      {open && filtered.length > 0 && (
        <div
          style={{
            position: 'absolute',
            top: '100%',
            left: 0,
            right: 0,
            maxHeight: 200,
            overflowY: 'auto',
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius-sm)',
            zIndex: 50,
            boxShadow: '0 4px 12px rgba(0,0,0,0.3)',
          }}
        >
          {filtered.map(s => (
            <div
              key={s.id}
              onClick={() => handleSelect(s.ticker)}
              style={{
                padding: '6px 10px',
                cursor: 'pointer',
                fontSize: 13,
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                borderBottom: '1px solid var(--border)',
              }}
              onMouseEnter={e => (e.currentTarget.style.background = 'var(--bg-hover)')}
              onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
            >
              <span style={{ fontWeight: 600 }}>{s.ticker}</span>
              <span style={{ fontSize: 10, color: 'var(--muted-foreground)' }}>{s.exchange}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
