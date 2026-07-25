import { useEffect, useRef } from 'react'
import { createChart, createSeriesMarkers, LineSeries, CandlestickSeries, HistogramSeries, AreaSeries, LineStyle, type IChartApi, type ISeriesApi, type ISeriesMarkersPluginApi, type Time, type LineData, type CandlestickData, type HistogramData, type AreaData, type DeepPartial, type ChartOptions, type SeriesMarker } from 'lightweight-charts'
import { getChartDefaults } from './chartConfig'

interface ChartBaseProps {
  height?: number
  timeScaleRightOffset?: number
  rightPriceScaleMargins?: { top: number; bottom: number }
  chartOptions?: DeepPartial<ChartOptions>
}

export function useChart(containerRef: React.RefObject<HTMLDivElement | null>, options?: ChartBaseProps) {
  const chartRef = useRef<IChartApi | null>(null)
  const height = options?.height ?? 300

  useEffect(() => {
    if (!containerRef.current) return

    const baseDefaults = getChartDefaults(height)
    const opts: DeepPartial<ChartOptions> = {
      ...baseDefaults,
      timeScale: {
        ...(baseDefaults.timeScale ?? {}),
        ...(options?.timeScaleRightOffset != null ? { rightOffset: options.timeScaleRightOffset } : {}),
      },
      rightPriceScale: {
        ...(baseDefaults.rightPriceScale ?? {}),
        ...(options?.rightPriceScaleMargins ? { scaleMargins: options.rightPriceScaleMargins } : {}),
      },
      ...(options?.chartOptions ?? {}),
    }
    const chart = createChart(containerRef.current, opts)

    chartRef.current = chart

    const resizeObserver = new ResizeObserver(() => {
      if (!containerRef.current || !chartRef.current) return
      chartRef.current.resize(
        containerRef.current.clientWidth,
        height || containerRef.current.clientHeight
      )
    })
    resizeObserver.observe(containerRef.current)
    chart.resize(
      containerRef.current.clientWidth,
      height || containerRef.current.clientHeight
    )

    return () => {
      resizeObserver.disconnect()
      chart.remove()
      chartRef.current = null
    }
  }, [containerRef, height, options?.timeScaleRightOffset, options?.rightPriceScaleMargins, options?.chartOptions])

  return chartRef
}

export function useLineSeries(
  chartRef: React.MutableRefObject<IChartApi | null>,
  color?: string,
  options?: {
    lineWidth?: number
    lineStyle?: number
    lastValueVisible?: boolean
    priceFormat?: { type: string; precision: number; minMove: number }
    priceScaleId?: string
  },
) {
  const seriesRef = useRef<ISeriesApi<'Line'> | null>(null)
  const markersPluginRef = useRef<ISeriesMarkersPluginApi<Time> | null>(null)

  useEffect(() => {
    if (!chartRef.current) return
    const series = chartRef.current.addSeries(LineSeries, {
      color: color ?? '#2962FF',
      lineWidth: options?.lineWidth ?? 2,
      lineStyle: options?.lineStyle ?? LineStyle.Solid,
      lastValueVisible: options?.lastValueVisible ?? true,
      ...(options?.priceScaleId ? { priceScaleId: options.priceScaleId } : {}),
      ...(options?.priceFormat ? { priceFormat: options.priceFormat as Record<string, unknown> } : { type: 'price', precision: 2, minMove: 0.01 }),
    } as Record<string, unknown>)
    seriesRef.current = series
    markersPluginRef.current = createSeriesMarkers<Time>(series)
    return () => {
      markersPluginRef.current?.detach()
      // eslint-disable-next-line react-hooks/exhaustive-deps
      chartRef.current?.removeSeries(series)
      seriesRef.current = null
      markersPluginRef.current = null
    }
  }, [chartRef, color, options?.lineWidth, options?.lineStyle, options?.lastValueVisible, options?.priceScaleId, options?.priceFormat])

  const setData = (data: LineData[]) => {
    if (data.length === 0) return
    seriesRef.current?.setData(data)
  }

  const update = (point: LineData) => {
    seriesRef.current?.update(point)
  }

  const setMarkers = (markers: SeriesMarker<Time>[]) => {
    markersPluginRef.current?.setMarkers(markers)
  }

  return { seriesRef, setData, update, setMarkers }
}

export function useCandlestickSeries(
  chartRef: React.MutableRefObject<IChartApi | null>,
  options?: { upColor?: string; downColor?: string },
) {
  const seriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null)
  const markersPluginRef = useRef<ISeriesMarkersPluginApi<Time> | null>(null)

  useEffect(() => {
    if (!chartRef.current) return
    const upColor = options?.upColor ?? (getComputedStyle(document.documentElement).getPropertyValue('--candle-up').trim() || '#26a69a')
    const downColor = options?.downColor ?? (getComputedStyle(document.documentElement).getPropertyValue('--candle-down').trim() || '#ef5350')
    const series = chartRef.current.addSeries(CandlestickSeries, {
      upColor,
      downColor,
      borderUpColor: upColor,
      borderDownColor: downColor,
      wickUpColor: upColor,
      wickDownColor: downColor,
    })
    seriesRef.current = series
    markersPluginRef.current = createSeriesMarkers<Time>(series)
    return () => {
      markersPluginRef.current?.detach()
      // eslint-disable-next-line react-hooks/exhaustive-deps
      chartRef.current?.removeSeries(series)
      seriesRef.current = null
      markersPluginRef.current = null
    }
  }, [chartRef, options])

  const setData = (data: CandlestickData[]) => {
    if (data.length === 0) return
    seriesRef.current?.setData(data)
  }

  const update = (point: CandlestickData) => {
    seriesRef.current?.update(point)
  }

  const setMarkers = (markers: SeriesMarker<Time>[]) => {
    markersPluginRef.current?.setMarkers(markers)
  }

  return { seriesRef, setData, update, setMarkers }
}

export function useHistogramSeries(
  chartRef: React.MutableRefObject<IChartApi | null>,
  color?: string,
  options?: { priceScaleId?: string },
) {
  const seriesRef = useRef<ISeriesApi<'Histogram'> | null>(null)

  useEffect(() => {
    if (!chartRef.current) return
    const series = chartRef.current.addSeries(HistogramSeries, {
      color: color ?? '#26a69a',
      priceFormat: { type: 'volume' },
      ...(options?.priceScaleId ? { priceScaleId: options.priceScaleId } : {}),
    })
    seriesRef.current = series
    return () => {
      // eslint-disable-next-line react-hooks/exhaustive-deps
      chartRef.current?.removeSeries(series)
      seriesRef.current = null
    }
  }, [chartRef, color, options?.priceScaleId])

  const setData = (data: HistogramData[]) => {
    if (data.length === 0) return
    seriesRef.current?.setData(data)
  }

  const update = (point: HistogramData) => {
    seriesRef.current?.update(point)
  }

  return { seriesRef, setData, update }
}

export function useAreaSeries(
  chartRef: React.MutableRefObject<IChartApi | null>,
  options?: {
    lineColor?: string
    topColor?: string
    bottomColor?: string
    lineWidth?: number
    lastValueVisible?: boolean
    priceScaleId?: string
    priceFormat?: { type: string; precision: number; minMove: number }
  },
) {
  const seriesRef = useRef<ISeriesApi<'Area'> | null>(null)

  useEffect(() => {
    if (!chartRef.current) return
    const series = chartRef.current.addSeries(AreaSeries, {
      lineColor: options?.lineColor ?? '#26a69a',
      topColor: options?.topColor ?? 'rgba(38, 166, 154, 0.15)',
      bottomColor: options?.bottomColor ?? 'rgba(38, 166, 154, 0.02)',
      lineWidth: options?.lineWidth ?? 1,
      lastValueVisible: options?.lastValueVisible ?? false,
      ...(options?.priceScaleId ? { priceScaleId: options.priceScaleId } : {}),
      ...(options?.priceFormat ? { priceFormat: options.priceFormat as Record<string, unknown> } : {}),
    } as Record<string, unknown>)
    seriesRef.current = series
    return () => {
      // eslint-disable-next-line react-hooks/exhaustive-deps
      chartRef.current?.removeSeries(series)
      seriesRef.current = null
    }
  }, [chartRef, options?.lineColor, options?.topColor, options?.bottomColor, options?.lineWidth, options?.lastValueVisible, options?.priceScaleId, options?.priceFormat])

  const setData = (data: AreaData[]) => {
    if (data.length === 0) return
    seriesRef.current?.setData(data)
  }

  const update = (point: AreaData) => {
    seriesRef.current?.update(point)
  }

  return { seriesRef, setData, update }
}

export { convertToUTCTime, equityToLineData, candlesToData, candlesToVolumeData, candleToUpdatable } from './chartUtils'
