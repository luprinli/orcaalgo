import { useRef, useEffect, useMemo, useState, useCallback } from 'react'
import {
  LineStyle, LineSeries,
  type IPriceLine, type Time, type SeriesMarker,
} from 'lightweight-charts'
import { useChart, useLineSeries, useAreaSeries, convertToUTCTime, equityToLineData } from './useChart'
import { useChartKeyboard } from '../hooks/useChartKeyboard'
import { getChartColors, CHART_LAYOUT, OVERLAY_PALETTE } from './chartConfig'
import CrosshairTooltip from './CrosshairTooltip'

interface EquityPoint {
  time: string
  value: number
}

interface TradeMarker {
  time: string
  side: 'BUY' | 'SELL'
  price: number
  pnl?: number
  label?: string
}

interface EquityOverlay {
  data: EquityPoint[]
  label: string
  color?: string
}

interface EquityCurveChartProps {
  data: EquityPoint[]
  benchmarkData?: EquityPoint[]
  trades?: TradeMarker[]
  overlays?: EquityOverlay[]
  height?: number
  color?: string
  title?: string
}

function computeDrawdown(data: EquityPoint[]): Array<{ time: Time; value: number }> {
  if (data.length === 0) return []
  const sorted = [...data].sort(
    (a, b) => new Date(a.time).getTime() - new Date(b.time).getTime(),
  )
  let peak = sorted[0].value
  const result: Array<{ time: Time; value: number }> = []
  for (const point of sorted) {
    if (point.value > peak) peak = point.value
    const dd = peak > 0 ? ((point.value - peak) / peak) * 100 : 0
    const t = convertToUTCTime(point.time)
    if (t !== 0) {
      result.push({ time: t, value: Math.min(dd, 0) })
    }
  }
  return result
}

export default function EquityCurveChart({
  data,
  benchmarkData,
  trades,
  overlays,
  height = 300,
  color,
  title,
}: EquityCurveChartProps) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const hwmPriceLineRef = useRef<IPriceLine | null>(null)
  const hasBenchmark = !!(benchmarkData && benchmarkData.length > 0)
  const hasOverlays = !!(overlays && overlays.length > 0)
  const [crosshairData, setCrosshairData] = useState<{
    time: string; equity: number; drawdown: number; overlays?: Array<{ label: string; value: number }>
  } | null>(null)
  const [crosshairPoint, setCrosshairPoint] = useState<{ x: number; y: number } | null>(null)

  const chartRef = useChart(containerRef, {
    height,
    rightPriceScaleMargins: CHART_LAYOUT.EQUITY_SCALE_MARGINS,
  })

  const colors = getChartColors()

  const { setData: setEquity, update: updateEquity, seriesRef: equitySeriesRef, setMarkers: setEquityMarkers } = useLineSeries(
    chartRef,
    color ?? colors.line,
    { lineWidth: 2, priceFormat: { type: 'price', precision: 2, minMove: 0.01 } },
  )

  const { setData: setDrawdown, update: updateDrawdown } = useAreaSeries(chartRef, {
    lineColor: 'rgba(239, 83, 80, 0.6)',
    topColor: 'rgba(239, 83, 80, 0.25)',
    bottomColor: 'rgba(239, 83, 80, 0.02)',
    priceScaleId: 'drawdown',
    priceFormat: { type: 'price', precision: 2, minMove: 0.01 },
  })

  const { setData: setBenchmark, update: updateBenchmark } = useLineSeries(
    chartRef,
    'rgba(255, 165, 0, 0.7)',
    { lineWidth: 1, lineStyle: LineStyle.Dashed, lastValueVisible: false, priceFormat: { type: 'price', precision: 2, minMove: 0.01 } },
  )

  const lineData = useMemo(() => equityToLineData(data), [data])
  const benchLineData = useMemo(
    () => (hasBenchmark ? equityToLineData(benchmarkData!) : []),
    [benchmarkData, hasBenchmark],
  )
  const drawdownData = useMemo(() => computeDrawdown(data), [data])

  const prevEquityLenRef = useRef(0)
  const prevBenchmarkLenRef = useRef(0)
  const prevDrawdownLenRef = useRef(0)

  const overlayLineData = useMemo(() => {
    if (!hasOverlays) return []
    return overlays!.map(o => ({ label: o.label, color: o.color, data: equityToLineData(o.data) }))
  }, [overlays, hasOverlays])

  const drawdownMap = useMemo(() => {
    const map = new Map<number, number>()
    for (const d of drawdownData) {
      map.set(d.time as number, d.value)
    }
    return map
  }, [drawdownData])

  const equityMap = useMemo(() => {
    const map = new Map<number, number>()
    for (const d of lineData) {
      map.set(d.time as number, d.value)
    }
    return map
  }, [lineData])

  const overlayMaps = useMemo(() => {
    if (!hasOverlays) return []
    return overlayLineData.map(o => {
      const map = new Map<number, number>()
      for (const d of o.data) {
        map.set(d.time as number, d.value)
      }
      return map
    })
  }, [overlayLineData, hasOverlays])

  const tradeMarkers: SeriesMarker<Time>[] = useMemo(() => {
    if (!trades || trades.length === 0) return []
    return trades.map((t) => {
      const ts = convertToUTCTime(t.time)
      return {
        time: ts,
        position: t.side === 'SELL' ? 'belowBar' : 'aboveBar',
        color: t.side === 'SELL' ? '#ef5350' : '#26a69a',
        shape: t.side === 'SELL' ? 'arrowDown' : 'arrowUp',
        text: t.side === 'SELL' ? 'S' : 'B',
        size: 1,
      } as SeriesMarker<Time>
    }).sort((a, b) => (a.time as number) - (b.time as number))
  }, [trades])

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleCrosshair = useCallback((param: any) => {
    if (!param.time || param.point === undefined) {
      setCrosshairData(null)
      setCrosshairPoint(null)
      return
    }
    const ts = typeof param.time === 'number' ? param.time : 0
    const eq = equityMap.get(ts)
    const dd = drawdownMap.get(ts)
    if (eq !== undefined) {
      const ov = hasOverlays ? overlayLineData.map((o, i) => ({
        label: o.label,
        value: overlayMaps[i]?.get(ts) ?? 0,
      })) : undefined
      setCrosshairData({
        time: new Date(ts * 1000).toLocaleString(),
        equity: eq,
        drawdown: dd ?? 0,
        overlays: ov,
      })
      setCrosshairPoint({ x: param.point.x, y: param.point.y })
    }
  }, [equityMap, drawdownMap, overlayLineData, overlayMaps, hasOverlays])

  useEffect(() => {
    if (!chartRef.current) return
    const chart = chartRef.current
    chart.subscribeCrosshairMove(handleCrosshair)
    return () => {
      chart.unsubscribeCrosshairMove(handleCrosshair)
      setCrosshairData(null)
    }
  }, [chartRef, handleCrosshair])

  useEffect(() => {
    if (chartRef.current) {
      chartRef.current.priceScale('drawdown').applyOptions({
        scaleMargins: CHART_LAYOUT.DRAWDDOWN_SCALE_MARGINS,
        visible: true,
      })
    }
  }, [chartRef])

  // Overlay series lifecycle
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const overlaySeriesRef = useRef<Map<string, any>>(new Map())
  const prevOverlayLenRef = useRef<Map<string, number>>(new Map())

  useEffect(() => {
    if (!chartRef.current || !hasOverlays) return
    const currentKeys = new Set<string>()

    for (let i = 0; i < overlayLineData.length; i++) {
      const o = overlayLineData[i]
      const key = `overlay_${i}_${o.label}`
      currentKeys.add(key)

      if (overlaySeriesRef.current.has(key)) {
        const series = overlaySeriesRef.current.get(key)
        const prevLen = prevOverlayLenRef.current.get(key) ?? 0
        if (o.data.length > 0) {
          if (prevLen === 0 || o.data.length < prevLen) {
            series.setData(o.data)
          } else {
            for (let j = prevLen; j < o.data.length; j++) {
              series.update(o.data[j])
            }
          }
          prevOverlayLenRef.current.set(key, o.data.length)
        }
        continue
      }

      const overlayColor = o.color ?? OVERLAY_PALETTE[i % OVERLAY_PALETTE.length]
      const series = chartRef.current.addSeries(LineSeries, {
        color: overlayColor,
        lineWidth: 1,
        lineStyle: 1, // dotted
        lastValueVisible: false,
        priceFormat: { type: 'price', precision: 2, minMove: 0.01 },
      })
      series.setData(o.data)
      overlaySeriesRef.current.set(key, series)
      prevOverlayLenRef.current.set(key, o.data.length)
    }

    for (const [key, series] of overlaySeriesRef.current) {
      if (!currentKeys.has(key)) {
        chartRef.current.removeSeries(series)
        overlaySeriesRef.current.delete(key)
        prevOverlayLenRef.current.delete(key)
      }
    }
  }, [overlayLineData, chartRef, hasOverlays])

  useEffect(() => {
    if (tradeMarkers.length > 0) {
      setEquityMarkers(tradeMarkers)
    }
  }, [tradeMarkers, setEquityMarkers])

  useEffect(() => {
    if (lineData.length > 0) {
      const prevLen = prevEquityLenRef.current
      if (prevLen === 0 || lineData.length < prevLen) {
        setEquity(lineData)
      } else {
        for (let i = prevLen; i < lineData.length; i++) {
          updateEquity(lineData[i])
        }
      }
      prevEquityLenRef.current = lineData.length

      if (!hwmPriceLineRef.current && equitySeriesRef.current) {
        const startingEquity = lineData[0]?.value
        if (startingEquity !== undefined && startingEquity > 0) {
          hwmPriceLineRef.current = equitySeriesRef.current.createPriceLine({
            price: startingEquity,
            color: 'rgba(255, 255, 255, 0.4)',
            lineWidth: 1,
            lineStyle: LineStyle.Dashed,
            axisLabelVisible: true,
            title: 'Start',
          })
        }
      }
    }
  }, [lineData, setEquity, updateEquity, equitySeriesRef])

  useEffect(() => {
    if (benchLineData.length > 0) {
      const prevLen = prevBenchmarkLenRef.current
      if (prevLen === 0 || benchLineData.length < prevLen) {
        setBenchmark(benchLineData)
      } else {
        for (let i = prevLen; i < benchLineData.length; i++) {
          updateBenchmark(benchLineData[i])
        }
      }
      prevBenchmarkLenRef.current = benchLineData.length
    }
  }, [benchLineData, setBenchmark, updateBenchmark])

  useEffect(() => {
    if (drawdownData.length > 0) {
      const prevLen = prevDrawdownLenRef.current
      if (prevLen === 0 || drawdownData.length < prevLen) {
        setDrawdown(drawdownData)
      } else {
        for (let i = prevLen; i < drawdownData.length; i++) {
          updateDrawdown(drawdownData[i])
        }
      }
      prevDrawdownLenRef.current = drawdownData.length
    }
  }, [drawdownData, setDrawdown, updateDrawdown])

  useChartKeyboard(chartRef)

  return (
    <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4" style={{ position: 'relative' }}>
      {title && <div className="flex items-center justify-between border-b border-border pb-2 mb-3"><h3>{title}</h3></div>}
      <div ref={containerRef} role="img" aria-label="Equity curve chart" />
      <CrosshairTooltip data={crosshairData && {
        timeStr: crosshairData.time,
        rows: [
          { label: 'Equity', value: crosshairData.equity.toFixed(2) },
          { label: 'Drawdown', value: `${crosshairData.drawdown.toFixed(2)}%`, color: 'var(--trading-danger)' },
          ...(crosshairData.overlays?.map((o, _i) => ({ label: o.label, value: o.value.toFixed(2) })) ?? []),
        ],
      }} position={crosshairPoint ?? undefined} />
      {hasOverlays && (
        <div className="flex gap-3 mt-2" style={{ fontSize: 10, color: 'var(--muted-foreground)', flexWrap: 'wrap' }}>
          {overlays!.map((o, i) => (
            <span key={i}>
              <span style={{ display: 'inline-block', width: 8, height: 8, borderRadius: '50%', background: OVERLAY_PALETTE[i % OVERLAY_PALETTE.length], marginRight: 3, verticalAlign: 'middle' }} />
              {o.label}
            </span>
          ))}
        </div>
      )}
    </div>
  )
}
