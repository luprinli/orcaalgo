import { useRef, useEffect } from 'react'
import { type Time } from 'lightweight-charts'
import { useChart, useHistogramSeries } from './useChart'

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

  return (
    <div className="card">
      {title && <div className="card-header"><h3>{title}</h3></div>}
      <div ref={containerRef} role="img" aria-label="Daily returns chart" />
    </div>
  )
}
