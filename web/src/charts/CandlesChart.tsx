import { useRef, useEffect, useMemo, useState } from 'react'
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
  const { setData: setCandles } = useCandlestickSeries(chartRef)
  const { setData: setVolume } = useHistogramSeries(chartRef)

  useEffect(() => {
    if (data.length > 0) {
      const { candlestick, volume } = candlesToData(data)
      setCandles(candlestick)
      setVolume(volume)
    }
  }, [data, setCandles, setVolume])

  const dataMap = useMemo(() => {
    const map = new Map<number, any>()
    data.forEach((d: any) => map.set(convertToUTCTime(d.time) as number, d))
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
            { label: 'O', value: datum.open?.toFixed(2) ?? '—' },
            { label: 'H', value: datum.high?.toFixed(2) ?? '—' },
            { label: 'L', value: datum.low?.toFixed(2) ?? '—' },
            { label: 'C', value: datum.close?.toFixed(2) ?? '—' },
            { label: 'V', value: String(datum.volume ?? '—') },
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
        <div ref={containerRef} />
        {crosshairData && <CrosshairTooltip data={crosshairData} />}
      </div>
    </div>
  )
}
