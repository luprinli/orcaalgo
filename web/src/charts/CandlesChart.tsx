import { useRef, useEffect, useMemo, useState } from 'react'
import { type Time } from 'lightweight-charts'
import { useChart, useCandlestickSeries, useHistogramSeries, candlesToData } from './useChart'
import { convertToUTCTime } from './chartUtils'
import CrosshairTooltip from './CrosshairTooltip'

interface CandlesChartProps {
  data: Array<{ time: string; open: number; high: number; low: number; close: number; volume: number }>
  height?: number
  title?: string
}

export default function CandlesChart({ data, height = 400, title }: CandlesChartProps) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const chartRef = useChart(containerRef, { height: height + 100 })
  const { setData: setCandles, update: updateCandle, seriesRef: candleSeriesRef } = useCandlestickSeries(chartRef)
  const { setData: setVolume, update: updateVolume, seriesRef: volumeSeriesRef } = useHistogramSeries(chartRef)

  const prevLenRef = useRef(0)

  useEffect(() => {
    if (data.length > 0) {
      const prevLen = prevLenRef.current
      if (prevLen === 0 || data.length < prevLen) {
        const { candlestick, volume } = candlesToData(data)
        setCandles(candlestick)
        setVolume(volume)
      } else {
        for (let i = prevLen; i < data.length; i++) {
          const d = data[i]
          const t = (new Date(d.time).getTime() / 1000) as Time
          candleSeriesRef.current?.update({ time: t, open: d.open, high: d.high, low: d.low, close: d.close })
          volumeSeriesRef.current?.update({ time: t, value: d.volume, color: d.close >= d.open ? 'rgba(38, 166, 154, 0.5)' : 'rgba(239, 83, 80, 0.5)' })
        }
      }
      prevLenRef.current = data.length
    }
  }, [data, setCandles, setVolume, updateCandle, updateVolume, candleSeriesRef, volumeSeriesRef])

  const dataMap = useMemo(() => {
    const map = new Map<number, any>()
    data.forEach((d: any) => map.set(convertToUTCTime(d.time) as number, d))
    return map
  }, [data])

  const [crosshairData, setCrosshairData] = useState<any>(null)
  const [crosshairPoint, setCrosshairPoint] = useState<{ x: number; y: number } | null>(null)

  useEffect(() => {
    if (!chartRef.current) return
    const chart = chartRef.current
    const handler = (param: any) => {
      if (!param.time || param.point === undefined) {
        setCrosshairData(null)
        setCrosshairPoint(null)
        return
      }
      const datum = dataMap.get(param.time as number)
      if (datum) {
        setCrosshairData({
          timeStr: new Date((param.time as number) * 1000).toLocaleString(),
          rows: [
            { label: 'O', value: datum.open?.toFixed(2) ?? '—' },
            { label: 'H', value: datum.high?.toFixed(2) ?? '—' },
            { label: 'L', value: datum.low?.toFixed(2) ?? '—' },
            { label: 'C', value: datum.close?.toFixed(2) ?? '—' },
            { label: 'V', value: String(datum.volume ?? '—') },
          ],
        })
        setCrosshairPoint({ x: param.point.x, y: param.point.y })
      } else {
        setCrosshairData(null)
        setCrosshairPoint(null)
      }
    }
    chart.subscribeCrosshairMove(handler)
    return () => chart.unsubscribeCrosshairMove(handler)
  }, [chartRef, dataMap])

  return (
    <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4">
      {title && <div className="flex items-center justify-between border-b border-border pb-2 mb-3"><h3>{title}</h3></div>}
      <div style={{ position: 'relative' }}>
        <div ref={containerRef} role="img" aria-label="Candlestick chart" />
        {crosshairData && <CrosshairTooltip data={crosshairData} position={crosshairPoint ?? undefined} />}
      </div>
    </div>
  )
}
