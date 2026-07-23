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
import type { Candle, TradeSummary } from '../types/api'

interface LiveMonitorChartProps {
  candles: Candle[]
  height?: number
  markers?: SeriesMarker<Time>[]
  trades?: TradeSummary[]
}

export default function LiveMonitorChart({ candles, height = 500, markers, trades }: LiveMonitorChartProps) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const [isFullscreen, setIsFullscreen] = useState(false)
  const fullscreenContainerRef = useRef<HTMLDivElement | null>(null)

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

      chartRef.current?.timeScale().fitContent()
    })
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [candles, enqueue, setCandleData, setVolumeData, chartRef])

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
      <OHLCVHeader candle={latestCandle} />
      <div ref={containerRef} role="img" aria-label="Candlestick chart with indicators" style={{ width: '100%', position: 'relative' }}>
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
            color: 'var(--text-secondary)',
            minWidth: 180, maxWidth: 280,
            pointerEvents: 'none',
            boxShadow: '0 2px 8px rgba(0,0,0,0.3)',
          }}>
            <div style={{ color: 'var(--text-primary)', fontWeight: 600, marginBottom: 4, fontSize: 10 }}>
              {crosshairData.timeStr || '—'}
            </div>
            {crosshairData.ohlcv ? (
              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: crosshairData.indicators.length > 0 ? 6 : 0 }}>
                <span>O <span style={{ color: 'var(--text-primary)' }}>{crosshairData.ohlcv.o.toFixed(4)}</span></span>
                <span>H <span style={{ color: 'var(--text-primary)' }}>{crosshairData.ohlcv.h.toFixed(4)}</span></span>
                <span>L <span style={{ color: 'var(--text-primary)' }}>{crosshairData.ohlcv.l.toFixed(4)}</span></span>
                <span>C <span style={{ color: 'var(--text-primary)' }}>{crosshairData.ohlcv.c.toFixed(4)}</span></span>
                <span>V <span style={{ color: 'var(--text-primary)' }}>{crosshairData.ohlcv.v.toLocaleString()}</span></span>
              </div>
            ) : (
              <span className="text-muted">No data</span>
            )}
            {crosshairData.indicators.map((ind) => (
              <div key={ind.name} style={{ marginBottom: 2 }}>
                <span style={{ color: 'var(--accent)', fontWeight: 500 }}>{ind.name}</span>
                {ind.values.map((v) => (
                  <span key={v.key} style={{ marginLeft: 6 }}>
                    {v.key} <span style={{ color: 'var(--text-primary)' }}>{v.value.toFixed(4)}</span>
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
            color: 'var(--text-secondary)',
            minWidth: 180,
            boxShadow: '0 2px 8px rgba(0,0,0,0.3)',
          }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6 }}>
              <span style={{ color: tradeTooltip.side === 'BUY' ? 'var(--success)' : 'var(--danger)', fontWeight: 600 }}>
                {tradeTooltip.side} {tradeTooltip.symbol}
              </span>
              <button
                onClick={() => setTradeTooltip(null)}
                style={{ background: 'none', border: 'none', color: 'var(--text-secondary)', cursor: 'pointer', fontSize: 14, lineHeight: 1, padding: 0 }}
                aria-label="Close trade detail"
              >
                ×
              </button>
            </div>
            <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', marginBottom: 4 }}>
              <span>Entry <span style={{ color: 'var(--text-primary)' }}>${tradeTooltip.entry_price?.toFixed(2)}</span></span>
              <span>Exit <span style={{ color: 'var(--text-primary)' }}>${tradeTooltip.exit_price?.toFixed(2)}</span></span>
            </div>
            <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
              <span>P&L <span style={{ color: (tradeTooltip.pnl ?? 0) >= 0 ? 'var(--success)' : 'var(--danger)', fontWeight: 600 }}>
                ${tradeTooltip.pnl?.toFixed(2)} ({tradeTooltip.pnl_pct?.toFixed(2)}%)
              </span></span>
              <span>Qty <span style={{ color: 'var(--text-primary)' }}>{tradeTooltip.quantity}</span></span>
              <span>Dur <span style={{ color: 'var(--text-primary)' }}>{tradeTooltip.hold_duration?.toFixed(1)}h</span></span>
            </div>
            <div style={{ marginTop: 4, fontSize: 10, color: 'var(--text-secondary)' }}>
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
