import { useEffect, useRef, useState, useCallback } from 'react'
import type React from 'react'
import { LineSeries, type IChartApi, type ISeriesApi, type Time } from 'lightweight-charts'

export function useDrawingTool(
  chartRef: React.MutableRefObject<IChartApi | null>,
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  candleSeriesRef: React.MutableRefObject<ISeriesApi<'Candlestick'> | null>,
  containerRef: React.MutableRefObject<HTMLDivElement | null>,
) {
  const [drawingMode, setDrawingMode] = useState(false)
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const priceLinesRef = useRef<Set<any>>(new Set())
  const pendingPointRef = useRef<{ time: Time; price: number } | null>(null)
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const drawingClickRef = useRef<((param: any) => void) | null>(null)

  useEffect(() => {
    if (!chartRef.current) return
    const chart = chartRef.current

    if (!drawingMode) {
      if (containerRef.current) containerRef.current.style.cursor = ''
      if (drawingClickRef.current) {
        chart.unsubscribeClick(drawingClickRef.current)
        drawingClickRef.current = null
      }
      pendingPointRef.current = null
      return
    }

    if (containerRef.current) containerRef.current.style.cursor = 'crosshair'

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const handler = (param: any) => {
      if (!param.point || !param.time) return
      const series = candleSeriesRef.current;
      if (!series) return
      const price = series.coordinateToPrice(param.point.y as number)
      if (price == null || isNaN(price as number)) return

      if (!pendingPointRef.current) {
        pendingPointRef.current = { time: param.time as Time, price }
        return
      }

      const p0 = pendingPointRef.current
      pendingPointRef.current = null

      const tlSeries = chart.addSeries(LineSeries, {
        color: '#f85149',
        lineWidth: 1,
        priceLineVisible: false,
        lastValueVisible: false,
      })
      tlSeries.setData([
        { time: p0.time, value: p0.price },
        { time: param.time as Time, value: price },
      ])
      priceLinesRef.current.add(tlSeries)
    }

    drawingClickRef.current = handler
    chart.subscribeClick(handler)

    return () => {
      chart.unsubscribeClick(handler)
      drawingClickRef.current = null
      pendingPointRef.current = null
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [drawingMode, chartRef])

  const clearDrawingLines = useCallback(() => {
    for (const line of priceLinesRef.current) {
      chartRef.current?.removeSeries(line)
    }
    priceLinesRef.current.clear()
  }, [chartRef])

  return { drawingMode, setDrawingMode, clearDrawingLines, priceLinesRef }
}
