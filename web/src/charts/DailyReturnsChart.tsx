import { useRef, useEffect, useMemo, useState } from 'react'
import { type Time } from 'lightweight-charts'
import { useChart, useHistogramSeries } from './useChart'
import { convertToUTCTime } from './chartUtils'
import CrosshairTooltip from './CrosshairTooltip'

interface DailyReturnsChartProps {
  data: Array<{ date: string; return_pct: number }>
  height?: number
  title?: string
}

export default function DailyReturnsChart({ data, height = 200, title }: DailyReturnsChartProps) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const chartRef = useChart(containerRef, { height })
  const { setData, update } = useHistogramSeries(chartRef)

  const prevLenRef = useRef(0)

  const histogramData = useMemo(() => {
    return data.map((d) => ({
      time: (new Date(d.date).getTime() / 1000) as Time,
      value: d.return_pct,
      color: d.return_pct >= 0 ? 'rgba(38, 166, 154, 0.5)' : 'rgba(239, 83, 80, 0.5)',
    }))
  }, [data])

  useEffect(() => {
    if (histogramData.length > 0) {
      const prevLen = prevLenRef.current
      if (prevLen === 0 || histogramData.length < prevLen) {
        setData(histogramData)
      } else {
        for (let i = prevLen; i < histogramData.length; i++) {
          update(histogramData[i])
        }
      }
      prevLenRef.current = histogramData.length
    }
  }, [histogramData, setData, update])

  const dataMap = useMemo(() => {
    const map = new Map<number, any>()
    data.forEach((d: any) => map.set(convertToUTCTime(d.date) as number, d))
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
            { label: 'Return %', value: datum.return_pct?.toFixed(4) ?? '—' },
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
        <div ref={containerRef} role="img" aria-label="Daily returns chart" />
        {crosshairData && <CrosshairTooltip data={crosshairData} position={crosshairPoint ?? undefined} />}
      </div>
    </div>
  )
}
