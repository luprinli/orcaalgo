import { useRef, useEffect } from 'react'
import { useChart, useCandlestickSeries, useHistogramSeries, candlesToData } from './useChart'

interface CandlesChartProps {
  data: Array<{ time: string; open: number; high: number; low: number; close: number; volume: number }>
  height?: number
  title?: string
}

export default function CandlesChart({ data, height = 400, title }: CandlesChartProps) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const chartRef = useChart(containerRef, { height: height + 100 })
  const { setData: setCandles } = useCandlestickSeries(chartRef)
  const { setData: setVolume } = useHistogramSeries(chartRef)

  useEffect(() => {
    if (data.length > 0) {
      const { candlestick, volume } = candlesToData(data)
      setCandles(candlestick)
      setVolume(volume)
    }
  }, [data, setCandles, setVolume])

  return (
    <div className="card">
      {title && <div className="card-header"><h3>{title}</h3></div>}
      <div ref={containerRef} />
    </div>
  )
}
