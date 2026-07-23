import { useState } from 'react'
import { useTradingView, INDICATOR_IDS as FALLBACK_IDS } from './TradingViewProvider'
import type { IndicatorSpec } from '../types/api'
import { useIndicatorStore } from '../stores/indicatorStore'
import SymbolSearch from './SymbolSearch'
import TimeframeChips from './TimeframeChips'
import Watchlist from './Watchlist'
import IndicatorConfigModal from './IndicatorConfigModal'
import LiveMonitorChart from '../charts/LiveMonitorChart'

interface TradingChartSectionProps {
  showWatchlist?: boolean
  chartHeight?: number
  chartTitle?: string
}

export default function TradingChartSection({
  showWatchlist = true,
  chartHeight = 400,
  chartTitle: _chartTitle,
}: TradingChartSectionProps) {
  const tv = useTradingView()
  const indicatorStore = useIndicatorStore()
  const activeSpecIds = indicatorStore.all().map(i => i.spec.id)
  const [indicatorMenuOpen, setIndicatorMenuOpen] = useState(false)

  const activeCount = activeSpecIds.length
  const indicatorIds = tv.specs.length > 0 ? tv.specs.map(s => s.id) : FALLBACK_IDS
  const availableSpecs = tv.specs.filter(s => indicatorIds.includes(s.id))

  const handleIndicatorToggle = (spec: IndicatorSpec) => {
    const isActive = activeSpecIds.includes(spec.id)
    if (isActive) {
      const found = indicatorStore.all().find(i => i.spec.id === spec.id)
      if (found) tv.removeIndicator(found._id)
    } else {
      tv.openModal(spec.id)
    }
    setIndicatorMenuOpen(false)
  }

  return (
    <>
      <div className="flex mb-4" style={{ gap: 0, alignItems: 'flex-start' }}>
        {showWatchlist && (
          <Watchlist
            onSelectSymbol={v => tv.setSelectedSymbol(v)}
            selectedSymbol={tv.selectedSymbol}
            isOpen={tv.watchlistOpen}
            onToggle={() => tv.setWatchlistOpen(o => !o)}
          />
        )}
        <div style={{ flex: 1, minWidth: 0 }}>
          <div className="card">
            <div className="flex-between mb-2">
              <h3 style={{ margin: 0 }}>{tv.selectedSymbol} — 1 Week</h3>
              <div className="flex gap-2" style={{ flexWrap: 'wrap', alignItems: 'center', position: 'relative' }}>
                <button
                  className={`btn ${activeCount > 0 ? 'btn-primary' : 'btn-outline'}`}
                  style={{ fontSize: 11, padding: '4px 10px' }}
                  onClick={() => setIndicatorMenuOpen(o => !o)}
                  onBlur={() => setTimeout(() => setIndicatorMenuOpen(false), 200)}
                >
                  Indicators {activeCount > 0 ? `(${activeCount})` : '▾'}
                </button>
                {indicatorMenuOpen && (
                  <div style={{
                    position: 'absolute', top: '100%', left: 0, zIndex: 100,
                    background: 'var(--bg-card)', border: '1px solid var(--border)',
                    borderRadius: 6, padding: '4px 0', minWidth: 160,
                    boxShadow: '0 4px 12px rgba(0,0,0,0.3)', marginTop: 2,
                  }}>
                    {availableSpecs.map(spec => {
                      const isActive = activeSpecIds.includes(spec.id)
                      return (
                        <div
                          key={spec.id}
                          onMouseDown={e => { e.preventDefault(); handleIndicatorToggle(spec) }}
                          style={{
                            padding: '6px 14px', cursor: 'pointer', fontSize: 12,
                            display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                            background: isActive ? 'var(--bg-hover)' : 'transparent',
                            color: isActive ? 'var(--accent-text)' : 'var(--text-primary)',
                          }}
                          onMouseEnter={e => { if (!isActive) e.currentTarget.style.background = 'var(--bg-hover)' }}
                          onMouseLeave={e => { if (!isActive) e.currentTarget.style.background = 'transparent' }}
                        >
                          <span style={{ fontWeight: isActive ? 600 : 400 }}>{spec.name || spec.id.toUpperCase()}</span>
                          {isActive && <span style={{ fontSize: 10, color: 'var(--success)' }}>● Active</span>}
                        </div>
                      )
                    })}
                  </div>
                )}
              </div>
            </div>
            <LiveMonitorChart candles={tv.aggregatedCandles} height={chartHeight} markers={tv.chartMarkers} trades={tv.trades} />
          </div>
        </div>
      </div>

      {tv.modalSpec && (
        <IndicatorConfigModal
          spec={tv.modalSpec}
          initialParams={tv.modalDefaults}
          onApply={tv.handleApplyIndicator}
          onCancel={tv.cancelModal}
        />
      )}
    </>
  )
}

export function TradingToolbar() {
  const tv = useTradingView()

  return (
    <div className="flex gap-2 items-center">
      <SymbolSearch
        value={tv.selectedSymbol}
        onChange={v => tv.setSelectedSymbol(v)}
        placeholder="SPY"
      />
      <button
        className={`btn ${tv.watchlistOpen ? 'btn-primary' : 'btn-outline'}`}
        onClick={() => tv.setWatchlistOpen(o => !o)}
        title="Toggle watchlist"
        style={{ fontSize: 11, padding: '2px 8px' }}
      >
        &#9776;
      </button>
      <TimeframeChips variant="toolbar" />
      <span className={`badge ${tv.wsConnected ? 'badge-ok' : 'badge-err'}`}>
        {tv.wsConnected ? 'WS Connected' : 'WS Disconnected'}
      </span>
    </div>
  )
}
