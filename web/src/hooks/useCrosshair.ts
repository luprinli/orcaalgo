import { useEffect, useMemo, useRef, useState } from 'react'
import type React from 'react'
import type { IChartApi, Time } from 'lightweight-charts'
import { useIndicatorStore } from '../stores/indicatorStore'
import type { Candle } from '../types/api'

export interface CrosshairDatum {
  time: Time
  timeStr: string
  ohlcv: { o: number; h: number; l: number; c: number; v: number } | null
  indicators: Array<{ name: string; values: Array<{ key: string; value: number }> }>
}

export function useCrosshair(chartRef: React.MutableRefObject<IChartApi | null>, candles: Candle[], indicatorIds: string): { crosshairData: CrosshairDatum | null } {
  const [crosshairData, setCrosshairData] = useState<CrosshairDatum | null>(null)

  const candleTimeMap = useMemo(() => {
    const map = new Map<number, { o: number; h: number; l: number; c: number; v: number }>()
    for (const c of candles) {
      map.set(Math.floor(new Date(c.time).getTime() / 1000), { o: c.open, h: c.high, l: c.low, c: c.close, v: c.volume })
    }
    return map
  }, [candles])

  const indicatorMapRef = useRef<Map<string, Map<number, Record<string, number>>>>(new Map())

  const indicators = useIndicatorStore(s => s.indicators)

  useEffect(() => {
    const all = Object.values(indicators)
    const map = new Map<string, Map<number, Record<string, number>>>()
    all.forEach(ind => {
      if (!ind.result) return
      const timeMap = new Map<number, Record<string, number>>()
      ind.result.data.forEach(d => {
        timeMap.set(d.time, d.values)
      })
      map.set(ind._id, timeMap)
    })
    indicatorMapRef.current = map
  }, [indicators])

  useEffect(() => {
    if (!chartRef.current) return
    const chart = chartRef.current

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const handler = (param: any) => {
      if (!param.time) {
        setCrosshairData(null)
        return
      }
      const ts = typeof param.time === 'number' ? param.time : 0
      const ohlcv = candleTimeMap.get(ts) ?? null

      const indicatorsArr: Array<{ name: string; values: Array<{ key: string; value: number }> }> = []
      for (const indicator of useIndicatorStore.getState().all()) {
        if (!indicator.result) continue
        const pointValues = indicatorMapRef.current.get(indicator._id)?.get(ts)
        if (pointValues) {
          const values = Object.entries(pointValues).map(([key, value]) => ({ key, value }))
          if (values.length > 0) {
            indicatorsArr.push({ name: indicator.spec.name, values })
          }
        }
      }

      setCrosshairData({
        time: param.time,
        timeStr: ts > 0 ? new Date(ts * 1000).toLocaleString() : '',
        ohlcv,
        indicators: indicatorsArr,
      })
    }

    chart.subscribeCrosshairMove(handler)
    return () => {
      chart.unsubscribeCrosshairMove(handler)
      setCrosshairData(null)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [chartRef, candleTimeMap, indicatorIds])

  return { crosshairData }
}
