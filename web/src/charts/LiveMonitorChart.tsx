import { useEffect, useRef, useCallback, useState, useMemo } from 'react'
import {
  type Time, type CandlestickData,
  type SeriesMarker,
} from 'lightweight-charts'
import { useChart, useCandlestickSeries, useHistogramSeries } from './useChart'
import { candlesToVolumeData } from './chartUtils'
import { useChartUpdate } from '../hooks/useChartUpdate'
import { useIndicatorStore } from '../stores/indicatorStore'
import { useChartKeyboard } from '../hooks/useChartKeyboard'
import { useIndicatorCompute } from '../hooks/useIndicator'
import { useCrosshair } from '../hooks/useCrosshair'
import { useTradeTooltip } from '../hooks/useTradeTooltip'
import { useDrawingTool } from '../hooks/useDrawingTool'
import { useIndicatorRenderer } from '../hooks/useIndicatorRenderer'
import { OHLCVHeader } from '../components/OHLCVHeader'
import ChartOverlayButtons from '../components/ChartOverlayButtons'
import TimeframeChips from '../components/TimeframeChips'
import { useTimeframeStore } from '../stores/timeframeStore'
import type { Candle, TradeSummary, IndicatorSpec, IndicatorWithData } from '../types/api'

interface LiveMonitorChartProps {
  candles: Candle[]
  symbol: string
  range: string
  onSymbolChange: (symbol: string) => void
  onRangeChange: (range: string) => void
  onLoad: () => void
  indicatorSpecs: IndicatorSpec[]
  height?: number
  markers?: SeriesMarker<Time>[]
  trades?: TradeSummary[]
  loading?: boolean
  error?: string | null
}

const RANGES = ['1D', '1W', '1M', '3M', '1Y', 'ALL']
const INDICATOR_IDS = ['sma', 'ema', 'rsi', 'macd', 'bbands', 'atr']
const DEFAULT_PARAMS: Record<string, Record<string, number | string>> = {
  sma: { period: 20, source: 'close' },
  ema: { period: 20, source: 'close' },
  rsi: { period: 14, source: 'close' },
  macd: { fast: 12, slow: 26, signal: 9, source: 'close' },
  bbands: { period: 20, std_dev: 2, source: 'close' },
  atr: { period: 14 },
}

export default function LiveMonitorChart({
  candles, symbol, range, onSymbolChange, onRangeChange, onLoad,
  indicatorSpecs, height = 500, markers, trades, loading, error,
}: LiveMonitorChartProps) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const { timeframe } = useTimeframeStore()
  const [isFullscreen, setIsFullscreen] = useState(false)
  const fullscreenContainerRef = useRef<HTMLDivElement | null>(null)
  const [indicatorOpen, setIndicatorOpen] = useState(false)
  const [paramEditor, setParamEditor] = useState<{ indicatorId: string; spec: IndicatorSpec; params: Record<string, number | string> } | null>(null)

  const chartOptions = useMemo(() => ({
    height,
    timeScaleRightOffset: 5,
    rightPriceScaleMargins: { top: 0.1, bottom: 0.2 },
  }), [height])

  const chartRef = useChart(containerRef, chartOptions)

  const { setData: setCandleData, setMarkers: setCandleMarkers, seriesRef: candleSeriesRef } = useCandlestickSeries(chartRef)
  const { setData: setVolumeData } = useHistogramSeries(chartRef, undefined, { priceScaleId: 'volume' })
  const { enqueue } = useChartUpdate()
  const rawIndicators = useIndicatorStore(s => s.indicators)
  const indicatorIds = useMemo(
    () => Object.values(rawIndicators).map(i => i._id).sort().join(','),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [JSON.stringify(Object.values(rawIndicators).map(i => i._id))],
  )
  const indicatorVersions = useMemo(
    () => Object.values(rawIndicators).map(i => `${i._id}:${i.dataVersion}`).join(';'),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [JSON.stringify(Object.values(rawIndicators).map(i => ({ _id: i._id, v: i.dataVersion })))],
  )
  const { compute } = useIndicatorCompute()

  // Configure volume price scale
  useEffect(() => {
    if (!chartRef.current) return
    try {
      chartRef.current.priceScale('volume').applyOptions({
        scaleMargins: { top: 0.85, bottom: 0 },
      })
    } catch { /* volume scale may not exist yet */ }
  }, [chartRef])

  // Refit chart when timeframe changes
  useEffect(() => {
    if (chartRef.current) {
      chartRef.current.timeScale().fitContent()
    }
  }, [timeframe, chartRef])

  const { crosshairData } = useCrosshair(chartRef, candles, indicatorIds)

  const prevCandlesLenRef = useRef(0)

  // Candlestick + volume data
  useEffect(() => {
    if (!candles.length) return
    const prevLen = prevCandlesLenRef.current
    prevCandlesLenRef.current = candles.length

    enqueue(() => {
      if (prevLen === 0 || prevLen !== candles.length - 1) {
        const candleData: CandlestickData[] = candles.map((c) => ({
          time: (new Date(c.time).getTime() / 1000) as Time,
          open: c.open, high: c.high, low: c.low, close: c.close,
        }))
        setCandleData(candleData)
        setVolumeData(candlesToVolumeData(candles))
      } else {
        const c = candles[candles.length - 1]
        const lastTime = (new Date(c.time).getTime() / 1000) as Time
        candleSeriesRef.current?.update({ time: lastTime, open: c.open, high: c.high, low: c.low, close: c.close })
      }

    })
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [candles, enqueue, setCandleData, setVolumeData])

  // Markers
  useEffect(() => {
    if (!markers || !markers.length) return
    setCandleMarkers(markers)
  }, [markers, setCandleMarkers])

  // Indicator computation — guarded against re-entrancy
  const computingRef = useRef(new Set<string>())

  useEffect(() => {
    if (candles.length === 0) return
    const allIndicators = useIndicatorStore.getState().all()
    for (const indicator of allIndicators) {
      if (indicator.loading || computingRef.current.has(indicator._id)) continue
      computingRef.current.add(indicator._id)
      compute(indicator._id, candles).finally(() => {
        computingRef.current.delete(indicator._id)
      })
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [candles.length, indicatorIds])

  useIndicatorRenderer(chartRef, indicatorVersions)

  const addIndicator = useCallback((indicatorId: string) => {
    const spec = indicatorSpecs.find(s => s.id === indicatorId)
    if (!spec) return
    const params = DEFAULT_PARAMS[indicatorId] ?? {}
    const id = useIndicatorStore.getState().addIndicator(spec, params)
    compute(id, candles)
    setIndicatorOpen(false)
  }, [indicatorSpecs, compute, candles])

  const removeIndicator = useCallback((id: string) => {
    useIndicatorStore.getState().removeIndicator(id)
  }, [])

  const openParamEditor = useCallback((indicatorId: string) => {
    const indicator = useIndicatorStore.getState().getById(indicatorId)
    if (!indicator) return
    setParamEditor({ indicatorId, spec: indicator.spec, params: { ...indicator.parameters } })
    setIndicatorOpen(false)
  }, [])

  const applyParams = useCallback(() => {
    if (!paramEditor) return
    const id = paramEditor.indicatorId
    useIndicatorStore.getState().updateParameters(id, paramEditor.params)
    compute(id, candles)
    setParamEditor(null)
  }, [paramEditor, compute, candles])

  const activeSpecIds = rawIndicators ? Object.values(rawIndicators).map((i: IndicatorWithData) => i.spec.id) : []

  const toggleFullscreen = useCallback(() => {
    if (!fullscreenContainerRef.current) return
    if (document.fullscreenElement) {
      document.exitFullscreen().catch(() => {})
    } else {
      fullscreenContainerRef.current.requestFullscreen().catch(() => {})
    }
  }, [])

  const { drawingMode, setDrawingMode, clearDrawingLines, priceLinesRef } = useDrawingTool(chartRef, candleSeriesRef, containerRef)

  useEffect(() => {
    const handleFullscreenChange = () => {
      setIsFullscreen(!!document.fullscreenElement)
    }
    document.addEventListener('fullscreenchange', handleFullscreenChange)
    return () => document.removeEventListener('fullscreenchange', handleFullscreenChange)
  }, [])

  useChartKeyboard(chartRef, { enabled: !drawingMode })

  const { tradeTooltip, setTradeTooltip } = useTradeTooltip(chartRef, trades)

  const exportPNG = useCallback(() => {
    const chart = chartRef.current
    if (chart) {
      const canvas = chart.takeScreenshot()
      const a = document.createElement('a')
      a.href = canvas.toDataURL()
      a.download = `chart_${new Date().toISOString().slice(0, 19).replace(/:/g, '-')}.png`
      a.click()
    }
  }, [chartRef])

  const latestCandle = candles.length > 0 ? candles[candles.length - 1] : undefined

  return (
    <div ref={fullscreenContainerRef} style={{ width: '100%', borderRadius: 8, overflow: 'hidden', position: 'relative', background: isFullscreen ? 'var(--chart-bg)' : undefined }}>
      {/* Toolbar — symbol + timeframe + range + indicators */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '4px 8px', background: 'var(--muted)', borderBottom: '1px solid var(--border)', flexWrap: 'wrap' }}>
        <input
          value={symbol}
          onChange={e => onSymbolChange(e.target.value.toUpperCase())}
          onKeyDown={e => { if (e.key === 'Enter') onLoad() }}
          placeholder="Symbol"
          style={{
            width: 80, padding: '3px 6px', fontSize: 12, fontFamily: 'monospace', fontWeight: 600,
            background: 'var(--card)', color: 'var(--foreground)', border: '1px solid var(--border)',
            borderRadius: 4, outline: 'none',
          }}
        />
        <div style={{ display: 'flex', gap: 1 }}>
          {RANGES.map(r => (
            <button
              key={r}
              onClick={() => onRangeChange(r)}
              style={{
                padding: '3px 8px', fontSize: 11, fontWeight: 600, fontFamily: 'inherit',
                border: 'none', borderRadius: 4, cursor: 'pointer',
                background: range === r ? 'var(--accent)' : 'transparent',
                color: range === r ? '#fff' : 'var(--muted-foreground)',
              }}
            >
              {r}
            </button>
          ))}
        </div>
        <div style={{ position: 'relative' }}>
          <button
            onClick={() => setIndicatorOpen(v => !v)}
            style={{
              padding: '3px 8px', fontSize: 11, fontFamily: 'inherit',
              border: '1px solid var(--border)', borderRadius: 4, cursor: 'pointer',
              background: 'var(--card)', color: 'var(--foreground)',
            }}
          >
            Indicators ▾
          </button>
          {indicatorOpen && (
            <div style={{
              position: 'absolute', top: '100%', left: 0, zIndex: 60,
              background: 'var(--card)', border: '1px solid var(--border)', borderRadius: 6,
              padding: 4, minWidth: 140, marginTop: 2, boxShadow: '0 4px 12px rgba(0,0,0,0.4)',
            }}>
              {INDICATOR_IDS.map(id => {
                const spec = indicatorSpecs.find(s => s.id === id)
                const isActive = activeSpecIds.includes(id)
                const instance = isActive ? useIndicatorStore.getState().all().find(i => i.spec.id === id) : null
                return (
                  <div
                    key={id}
                    style={{
                      display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                      padding: '4px 8px', fontSize: 12, cursor: 'pointer',
                      borderRadius: 4, color: 'var(--foreground)',
                      background: isActive ? 'var(--accent)' : 'transparent',
                    }}
                    onMouseEnter={e => { if (!isActive) e.currentTarget.style.background = 'var(--muted)' }}
                    onMouseLeave={e => { if (!isActive) e.currentTarget.style.background = 'transparent' }}
                  >
                    <span onClick={() => isActive ? removeIndicator(instance?._id ?? '') : addIndicator(id)} style={{ flex: 1 }}>
                      {spec?.name ?? id.toUpperCase()}
                    </span>
                    {isActive && (
                      <span
                        onClick={(e) => { e.stopPropagation(); openParamEditor(instance?._id ?? '') }}
                        style={{ fontSize: 12, cursor: 'pointer', opacity: 0.6, marginLeft: 4 }}
                        title="Edit parameters"
                      >⚙</span>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </div>
        <div style={{ flex: 1 }} />
        <TimeframeChips variant="toolbar" />
        {loading && <span style={{ fontSize: 11, color: 'var(--muted-foreground)' }}>Loading...</span>}
      </div>
      {error && (
        <div style={{ padding: '4px 8px', fontSize: 11, color: 'var(--trading-danger)', background: 'rgba(239,83,80,0.1)', borderBottom: '1px solid var(--trading-danger)' }}>
          {error}
        </div>
      )}
      {paramEditor && (
        <div style={{ padding: '8px', background: 'var(--muted)', borderBottom: '1px solid var(--border)' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
            <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--foreground)' }}>{paramEditor.spec.name}</span>
            {paramEditor.spec.parameters.map(p => (
              <label key={p.name} style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 11 }}>
                <span style={{ color: 'var(--muted-foreground)' }}>{p.name}</span>
                <input
                  type="number"
                  min={p.min}
                  max={p.max}
                  step={p.step ?? 1}
                  value={String(paramEditor.params[p.name] ?? p.default)}
                  onChange={e => setParamEditor(prev => prev ? { ...prev, params: { ...prev.params, [p.name]: Number(e.target.value) } } : null)}
                  style={{
                    width: 50, padding: '2px 4px', fontSize: 11, fontFamily: 'monospace',
                    background: 'var(--card)', color: 'var(--foreground)',
                    border: '1px solid var(--border)', borderRadius: 3, outline: 'none',
                  }}
                />
              </label>
            ))}
            <button
              onClick={applyParams}
              style={{
                padding: '3px 10px', fontSize: 11, fontFamily: 'inherit', fontWeight: 600,
                background: 'var(--accent)', color: '#fff', border: 'none', borderRadius: 4, cursor: 'pointer',
              }}
            >
              Apply
            </button>
            <button
              onClick={() => setParamEditor(null)}
              style={{
                padding: '3px 8px', fontSize: 11, fontFamily: 'inherit',
                background: 'transparent', color: 'var(--muted-foreground)', border: '1px solid var(--border)', borderRadius: 4, cursor: 'pointer',
              }}
            >
              Cancel
            </button>
          </div>
        </div>
      )}
      <OHLCVHeader candle={latestCandle} />
      <div ref={containerRef} role="img" aria-label="Live trading chart" style={{ width: '100%', position: 'relative' }}>
        <ChartOverlayButtons
          isFullscreen={isFullscreen}
          onToggleFullscreen={toggleFullscreen}
          onExportPNG={exportPNG}
          drawingMode={drawingMode}
          onToggleDrawing={() => setDrawingMode(v => !v)}
          lineCount={priceLinesRef.current.size}
          onClearLines={clearDrawingLines}
        />
        {crosshairData && (
          <div style={{
            position: 'absolute', top: 4, left: 4, zIndex: 50,
            background: 'var(--chart-tooltip-bg)',
            border: '1px solid var(--border)',
            borderRadius: 6, padding: '8px 10px',
            fontSize: 11, fontFamily: 'monospace',
            color: 'var(--muted-foreground)',
            minWidth: 180, maxWidth: 280,
            pointerEvents: 'none',
            boxShadow: '0 2px 8px rgba(0,0,0,0.3)',
          }}>
            <div style={{ color: 'var(--foreground)', fontWeight: 600, marginBottom: 4, fontSize: 10 }}>
              {crosshairData.timeStr || '—'}
            </div>
            {crosshairData.ohlcv ? (
              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: crosshairData.indicators.length > 0 ? 6 : 0 }}>
                <span>O <span style={{ color: 'var(--foreground)' }}>{crosshairData.ohlcv.o.toFixed(4)}</span></span>
                <span>H <span style={{ color: 'var(--foreground)' }}>{crosshairData.ohlcv.h.toFixed(4)}</span></span>
                <span>L <span style={{ color: 'var(--foreground)' }}>{crosshairData.ohlcv.l.toFixed(4)}</span></span>
                <span>C <span style={{ color: 'var(--foreground)' }}>{crosshairData.ohlcv.c.toFixed(4)}</span></span>
                <span>V <span style={{ color: 'var(--foreground)' }}>{crosshairData.ohlcv.v.toLocaleString()}</span></span>
              </div>
            ) : (
              <span className="text-muted">No data</span>
            )}
            {crosshairData.indicators.map((ind) => (
              <div key={ind.name} style={{ marginBottom: 2 }}>
                <span style={{ color: 'var(--accent)', fontWeight: 500 }}>{ind.name}</span>
                {ind.values.map((v) => (
                  <span key={v.key} style={{ marginLeft: 6 }}>
                    {v.key} <span style={{ color: 'var(--foreground)' }}>{v.value.toFixed(4)}</span>
                  </span>
                ))}
              </div>
            ))}
          </div>
        )}
        {tradeTooltip && (
          <div style={{
            position: 'absolute', top: 36, right: 8, zIndex: 50,
            background: 'var(--chart-tooltip-bg)',
            border: '1px solid var(--border)',
            borderRadius: 6, padding: '10px 12px',
            fontSize: 11, fontFamily: 'monospace',
            color: 'var(--muted-foreground)',
            minWidth: 180,
            boxShadow: '0 2px 8px rgba(0,0,0,0.3)',
          }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6 }}>
              <span style={{ color: tradeTooltip.side === 'BUY' ? 'var(--trading-success)' : 'var(--trading-danger)', fontWeight: 600 }}>
                {tradeTooltip.side} {tradeTooltip.symbol}
              </span>
              <button
                onClick={() => setTradeTooltip(null)}
                style={{ background: 'none', border: 'none', color: 'var(--muted-foreground)', cursor: 'pointer', fontSize: 14, lineHeight: 1, padding: 0 }}
                aria-label="Close trade detail"
              >
                ×
              </button>
            </div>
            <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', marginBottom: 4 }}>
              <span>Entry <span style={{ color: 'var(--foreground)' }}>${tradeTooltip.entry_price?.toFixed(2)}</span></span>
              <span>Exit <span style={{ color: 'var(--foreground)' }}>${tradeTooltip.exit_price?.toFixed(2)}</span></span>
            </div>
            <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
              <span>P&L <span style={{ color: (tradeTooltip.pnl ?? 0) >= 0 ? 'var(--trading-success)' : 'var(--trading-danger)', fontWeight: 600 }}>
                ${tradeTooltip.pnl?.toFixed(2)} ({tradeTooltip.pnl_pct?.toFixed(2)}%)
              </span></span>
              <span>Qty <span style={{ color: 'var(--foreground)' }}>{tradeTooltip.quantity}</span></span>
              <span>Dur <span style={{ color: 'var(--foreground)' }}>{tradeTooltip.hold_duration?.toFixed(1)}h</span></span>
            </div>
            <div style={{ marginTop: 4, fontSize: 10, color: 'var(--muted-foreground)' }}>
              {tradeTooltip.exit_reason && <span>Exit: {tradeTooltip.exit_reason}</span>}
              {tradeTooltip.mae !== undefined && <span style={{ marginLeft: 8 }}>MAE: ${tradeTooltip.mae?.toFixed(2)}</span>}
              {tradeTooltip.mfe !== undefined && <span style={{ marginLeft: 8 }}>MFE: ${tradeTooltip.mfe?.toFixed(2)}</span>}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
