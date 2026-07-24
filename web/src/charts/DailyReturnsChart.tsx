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
  const { setData } = useHistogramSeries(chartRef)

  useEffect(() => {
    if (data.length > 0) {
      setData(
        data.map((d) => ({
          time: (new Date(d.date).getTime() / 1000) as Time,
          value: d.return_pct,
          color: d.return_pct >= 0 ? 'rgba(38, 166, 154, 0.5)' : 'rgba(239, 83, 80, 0.5)',
        })),
      )
    }
  }, [data, setData])

  const dataMap = useMemo(() => {
    const map = new Map<number, any>()
    data.forEach((d: any) => map.set(convertToUTCTime(d.date) as number, d))
    return map
  }, [data])

  const [crosshairData, setCrosshairData] = useState<any>(null)

  useEffect(() => {
    if (!chartRef.current) return
    const chart = chartRef.current
    const handler = (param: any) => {
      if (!param.time) {
        setCrosshairData(null)
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
      } else {
        setCrosshairData(null)
      }
    }
    chart.subscribeCrosshairMove(handler)
    return () => chart.unsubscribeCrosshairMove(handler)
  }, [chartRef, dataMap])

  return (
    <div className="card">
      {title && <div className="card-header"><h3>{title}</h3></div>}
      <div style={{ position: 'relative' }}>
        <div ref={containerRef} role="img" aria-label="Daily returns chart" />
        {crosshairData && <CrosshairTooltip data={crosshairData} />}
      </div>
    </div>
  )
}
