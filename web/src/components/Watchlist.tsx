import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useWebSocket } from '../hooks/useWebSocket'
import { symbols as symbolsApi } from '../api/client'
import type { WSTickData } from '../types/ws'

interface SymbolItem {
  ticker: string
  exchange: string
  asset_type: string
  id: number
  is_active: boolean
}

interface WatchlistProps {
  onSelectSymbol: (symbol: string) => void
  selectedSymbol?: string
  isOpen: boolean
  onToggle: () => void
}

interface PriceInfo {
  price: number
  prevPrice: number
}

export default function Watchlist({ onSelectSymbol, selectedSymbol, isOpen, onToggle }: WatchlistProps) {
  const { t } = useTranslation()
  const [symbols, setSymbols] = useState<SymbolItem[]>([])
  const [prices, setPrices] = useState<Record<string, PriceInfo>>({})

  useEffect(() => {
    symbolsApi.list()
      .then(d => setSymbols((d ?? []) as unknown as SymbolItem[]))
      .catch(() => {})
  }, [])

  const { connected } = useWebSocket({
    channels: ['ticks'],
    onMessage: (data, channel) => {
      if (channel === 'ticks') {
        const tick = data as WSTickData
        setPrices(prev => {
          const existing = prev[tick.symbol]
          return {
            ...prev,
            [tick.symbol]: {
              price: tick.price,
              prevPrice: existing?.price ?? tick.price,
            },
          }
        })
      }
    },
  })

  const handleSelect = useCallback((ticker: string) => {
    onSelectSymbol(ticker)
  }, [onSelectSymbol])

  const getChangePct = (ticker: string): number | null => {
    const p = prices[ticker]
    if (!p || p.prevPrice === 0 || p.price === p.prevPrice) return null
    return ((p.price - p.prevPrice) / p.prevPrice) * 100
  }

  const panelWidth = isOpen ? 200 : 0
  const activeSymbols = symbols.filter(s => s.is_active).slice(0, 50)

  return (
    <div style={{ position: 'relative', display: 'flex', flexShrink: 0 }}>
      <div
        style={{
          width: panelWidth,
          overflow: 'hidden',
          transition: 'width 0.25s ease',
          background: 'var(--bg-secondary)',
          borderRight: isOpen ? '1px solid var(--border)' : 'none',
          flexShrink: 0,
        }}
      >
        <div style={{ width: 200 }}>
          <div style={{
            padding: '8px 10px 6px',
            fontSize: 11,
            fontWeight: 600,
            color: 'var(--text-secondary)',
            textTransform: 'uppercase',
            letterSpacing: 1,
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}>
            {t('watchlist:title', 'Watchlist')}
            {connected
              ? <span className="badge badge-ok" style={{ fontSize: 9, padding: '1px 5px' }}>{t('common:live', 'LIVE')}</span>
              : <span className="badge badge-err" style={{ fontSize: 9, padding: '1px 5px' }}>{t('common:off', 'OFF')}</span>
            }
          </div>
          <div style={{ maxHeight: 'calc(100vh - 120px)', overflowY: 'auto', overflowX: 'hidden' }}>
            {activeSymbols.map(s => {
              const p = prices[s.ticker]
              const change = getChangePct(s.ticker)
              const isSelected = selectedSymbol === s.ticker
              return (
                <div
                  key={s.id}
                  onClick={() => handleSelect(s.ticker)}
                  role="button"
                  tabIndex={0}
                  onKeyDown={e => { if (e.key === 'Enter') handleSelect(s.ticker) }}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    height: 30,
                    padding: '0 10px',
                    cursor: 'pointer',
                    fontSize: 12,
                    fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace",
                    background: isSelected ? 'var(--bg-hover)' : 'transparent',
                    borderLeft: isSelected ? '2px solid var(--accent)' : '2px solid transparent',
                    transition: 'background 0.1s',
                    outline: 'none',
                  }}
                  onMouseEnter={e => { if (!isSelected) e.currentTarget.style.background = 'var(--bg-hover)' }}
                  onMouseLeave={e => { if (!isSelected) e.currentTarget.style.background = 'transparent' }}
                >
                  <span style={{ fontWeight: 700, color: 'var(--text-primary)' }}>{s.ticker}</span>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <span style={{
                      color: p && change != null
                        ? (change >= 0 ? 'var(--success)' : 'var(--danger)')
                        : 'var(--text-secondary)',
                      fontSize: 11,
                      fontVariantNumeric: 'tabular-nums',
                    }}>
                      {p ? p.price.toFixed(2) : t('common:noData', '--')}
                    </span>
                    {change != null && (
                      <span style={{
                        color: change >= 0 ? 'var(--success)' : 'var(--danger)',
                        fontSize: 10,
                        fontWeight: 600,
                        fontVariantNumeric: 'tabular-nums',
                      }}>
                        {change >= 0 ? '+' : ''}{change.toFixed(2)}%
                      </span>
                    )}
                  </div>
                </div>
              )
            })}
            {activeSymbols.length === 0 && (
              <div style={{ padding: '10px', fontSize: 11, color: 'var(--text-secondary)' }}>
                {t('watchlist:noSymbols', 'No symbols')}
              </div>
            )}
          </div>
        </div>
      </div>

      <button
        onClick={onToggle}
        title={isOpen ? t('watchlist:collapse', 'Collapse watchlist') : t('watchlist:expand', 'Expand watchlist')}
        style={{
          position: 'absolute',
          left: isOpen ? 200 : 0,
          top: '50%',
          transform: 'translateY(-50%)',
          width: 20,
          height: 48,
          background: 'var(--bg-secondary)',
          border: '1px solid var(--border)',
          borderLeft: isOpen ? 'none' : '1px solid var(--border)',
          borderRadius: '0 4px 4px 0',
          cursor: 'pointer',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          color: 'var(--text-secondary)',
          fontSize: 9,
          padding: 0,
          zIndex: 10,
          transition: 'left 0.25s ease, color 0.15s',
          lineHeight: 1,
        }}
      >
        <span style={{ display: 'inline-block', transform: isOpen ? 'none' : 'none' }}>
          {isOpen ? '\u25C0' : '\u25B6'}
        </span>
      </button>
    </div>
  )
}
