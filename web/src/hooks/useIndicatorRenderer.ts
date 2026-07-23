import { useEffect, useRef } from 'react'
import type React from 'react'
import { LineSeries, HistogramSeries, type IChartApi, type Time } from 'lightweight-charts'
import { useIndicatorStore } from '../stores/indicatorStore'

export function useIndicatorRenderer(chartRef: React.MutableRefObject<IChartApi | null>, indicatorVersions: string): void {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const indicatorSeriesRef = useRef<Map<string, any>>(new Map())

  useEffect(() => {
    const allIndicators = useIndicatorStore.getState().all()
    for (const indicator of allIndicators) {
      if (!indicator.result) continue

      const spec = indicator.spec
      const result = indicator.result
      const paneIndex = indicator.paneIndex

      for (const output of spec.outputs) {
        const seriesKey = `${indicator._id}_${output.name}`
        if (indicatorSeriesRef.current.has(seriesKey)) continue

        const plotOpts = output.plotOptions ?? { color: '#ffffff', lineWidth: 2 }
        const seriesOptions: Record<string, unknown> = {
          color: plotOpts.color ?? '#ffffff',
          lineWidth: plotOpts.lineWidth ?? 2,
          lastValueVisible: true,
          priceLineVisible: false,
        }
        if (plotOpts.precision !== undefined) {
          seriesOptions.priceFormat = {
            type: 'price', precision: plotOpts.precision,
            minMove: plotOpts.minMove ?? 0.01,
          }
        }

        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        let series: any
        if (output.type === 'histogram') {
          series = chartRef.current?.addSeries(HistogramSeries, {
            color: plotOpts.color ?? '#ffffff',
          }, paneIndex)
        } else {
          series = chartRef.current?.addSeries(LineSeries, seriesOptions, paneIndex)
        }

        if (series) {
          const points = result.data
            .filter(p => p.values[output.name] !== undefined && isFinite(p.values[output.name]))
            .map(p => ({ time: p.time as Time, value: p.values[output.name] }))
          series.setData(points)
          indicatorSeriesRef.current.set(seriesKey, series)
        }
      }
    }

    const currentIds = new Set(useIndicatorStore.getState().all().map(i => i._id))
    for (const [key, series] of indicatorSeriesRef.current) {
      const indId = key.split('_').slice(0, 2).join('_')
      if (!currentIds.has(indId) && !currentIds.has(key.split('_')[0])) {
        chartRef.current?.removeSeries(series)
        indicatorSeriesRef.current.delete(key)
      }
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [indicatorVersions, chartRef])
}
