import { useRef, useEffect, useMemo, useState } from 'react'
import { useChart, useHistogramSeries } from './useChart'
import { convertToUTCTime } from './chartUtils'
import CrosshairTooltip from './CrosshairTooltip'

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
            { label: 'Delta', value: datum.delta?.toFixed(0) ?? '—' },
            { label: 'Buy Vol', value: String(datum.buy_volume ?? '—') },
            { label: 'Sell Vol', value: String(datum.sell_volume ?? '—') },
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
        <div ref={containerRef} role="img" aria-label="CVD chart" />
        {crosshairData && <CrosshairTooltip data={crosshairData} />}
      </div>
    </div>
  )
}
