import { useEffect, useMemo, useState } from 'react'
import type React from 'react'
import type { IChartApi } from 'lightweight-charts'
import type { TradeSummary } from '../types/api'

export function useTradeTooltip(chartRef: React.MutableRefObject<IChartApi | null>, trades: TradeSummary[] | undefined): { tradeTooltip: TradeSummary | null; setTradeTooltip: React.Dispatch<React.SetStateAction<TradeSummary | null>> } {
  const [tradeTooltip, setTradeTooltip] = useState<TradeSummary | null>(null)

  const tradeMap = useMemo(() => {
    if (!trades || trades.length === 0) return null
    const map = new Map<number, TradeSummary>()
    for (const t of trades) {
      if (t.entry_time) {
        map.set(Math.floor(new Date(t.entry_time).getTime() / 1000), t)
      }
      if (t.exit_time) {
        map.set(Math.floor(new Date(t.exit_time).getTime() / 1000), t)
      }
    }
    return map
  }, [trades])

  useEffect(() => {
    if (!chartRef.current || !tradeMap) return
    const chart = chartRef.current

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const handler = (param: any) => {
      if (!param.time) {
        setTradeTooltip(null)
        return
      }
      const ts = typeof param.time === 'number' ? param.time : 0
      const closest = tradeMap.get(ts)
      if (closest) {
        setTradeTooltip(closest)
      } else {
        setTradeTooltip(null)
      }
    }

    chart.subscribeClick(handler)
    return () => {
      chart.unsubscribeClick(handler)
      setTradeTooltip(null)
    }
  }, [chartRef, tradeMap])

  return { tradeTooltip, setTradeTooltip }
}
