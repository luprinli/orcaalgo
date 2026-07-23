import { useMemo } from 'react'
import type { Candle } from '../types/api'

const TIMEFRAME_MINUTES: Record<string, number> = {
  M1: 1, M5: 5, M15: 15, M30: 30,
  H1: 60, H4: 240, D1: 1440, W1: 10080,
}

export function useCandleAggregation(candles: Candle[], timeframe: string): Candle[] {
  return useMemo(() => {
    const minutes = TIMEFRAME_MINUTES[timeframe]
    if (!minutes || minutes <= 1) return candles

    const tfMs = minutes * 60 * 1000
    const buckets = new Map<number, Candle>()

    for (const c of candles) {
      const ts = new Date(c.time).getTime()
      const bucketTs = Math.floor(ts / tfMs) * tfMs

      const existing = buckets.get(bucketTs)
      if (!existing) {
        buckets.set(bucketTs, {
          time: new Date(bucketTs).toISOString(),
          open: c.open,
          high: c.high,
          low: c.low,
          close: c.close,
          volume: c.volume,
        })
      } else {
        existing.high = Math.max(existing.high, c.high)
        existing.low = Math.min(existing.low, c.low)
        existing.close = c.close
        existing.volume += c.volume
      }
    }

    return Array.from(buckets.entries())
      .sort(([a], [b]) => a - b)
      .map(([_, candle]) => candle)
  }, [candles, timeframe])
}

export function getTimeframeMinutes(timeframe: string): number {
  return TIMEFRAME_MINUTES[timeframe] ?? 1
}

export const TIMEFRAME_OPTIONS = Object.keys(TIMEFRAME_MINUTES).map((tf) => ({
  label: tf,
  value: tf,
}))
