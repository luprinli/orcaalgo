import { useRef, useEffect, useMemo } from 'react'
import { useChart, useHistogramSeries, convertToUTCTime } from './useChart'

interface CVDBar {
  time: string
  delta: number
  buy_volume: number
  sell_volume: number
}

interface CVDChartProps {
  data: CVDBar[]
  height?: number
  title?: string
}

export default function CVDChart({ data, height = 200, title }: CVDChartProps) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const chartRef = useChart(containerRef, { height })
  const { setData } = useHistogramSeries(chartRef)

  const histogramData = useMemo(() => {
    return data
      .map((bar) => {
        const t = convertToUTCTime(bar.time)
        if (t === 0) return null
        return {
          time: t,
          value: bar.delta,
          color: bar.delta >= 0 ? 'rgba(38, 166, 154, 0.5)' : 'rgba(239, 83, 80, 0.5)',
        }
      })
      .filter((d): d is NonNullable<typeof d> => d !== null)
      .sort((a, b) => (a.time as number) - (b.time as number))
  }, [data])

  useEffect(() => {
    if (histogramData.length > 0) {
      setData(histogramData)
    }
  }, [histogramData, setData])

  return (
    <div className="card">
      {title && <div className="card-header"><h3>{title}</h3></div>}
      <div ref={containerRef} role="img" aria-label="CVD chart" />
    </div>
  )
}
