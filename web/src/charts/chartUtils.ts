import type { Time, LineData, CandlestickData, HistogramData } from 'lightweight-charts'

export function convertToUTCTime(dateStr: string): Time {
  if (!dateStr || dateStr === 'None' || dateStr === 'null') {
    return 0 as Time
  }
  const d = new Date(dateStr)
  const ts = d.getTime()
  if (Number.isNaN(ts) || ts < 0) {
    return 0 as Time
  }
  return Math.floor(ts / 1000) as Time
}

function dedupeByTime<T extends { time: Time }>(items: T[]): T[] {
  const seen = new Set<number>()
  return items.filter(item => {
    const t = item.time as number
    if (seen.has(t)) return false
    seen.add(t)
    return true
  })
}

export function equityToLineData(points: Array<{ time: string; value: number }>): LineData[] {
  const mapped: LineData[] = []
  for (const p of points) {
    const t = convertToUTCTime(p.time)
    if (t === 0) continue
    mapped.push({ time: t, value: p.value })
  }
  mapped.sort((a, b) => (a.time as number) - (b.time as number))
  return dedupeByTime(mapped)
}

export function candlesToData(candles: Array<{ time: string; open: number; high: number; low: number; close: number; volume: number }>) {
  const candlestick: CandlestickData[] = candles.map((c) => ({
    time: convertToUTCTime(c.time),
    open: c.open,
    high: c.high,
    low: c.low,
    close: c.close,
  })).sort((a, b) => (a.time as number) - (b.time as number))

  const volume: HistogramData[] = candles.map((c) => ({
    time: convertToUTCTime(c.time),
    value: c.volume,
    color: c.close >= c.open ? 'rgba(38, 166, 154, 0.3)' : 'rgba(239, 83, 80, 0.3)',
  })).sort((a, b) => (a.time as number) - (b.time as number))

  return { candlestick: dedupeByTime(candlestick), volume: dedupeByTime(volume) }
}

export function candlesToVolumeData(candles: Array<{ time: string; open: number; high: number; low: number; close: number; volume: number }>): HistogramData[] {
  const data = candles.map((c) => ({
    time: convertToUTCTime(c.time),
    value: c.volume,
    color: c.close >= c.open ? 'rgba(38,166,154,0.3)' : 'rgba(239,83,80,0.3)',
  })).sort((a, b) => (a.time as number) - (b.time as number))
  return dedupeByTime(data)
}

export function candleToUpdatable(candle: { time: string; open: number; high: number; low: number; close: number }): { time: Time; open: number; high: number; low: number; close: number } {
  return {
    time: convertToUTCTime(candle.time),
    open: candle.open,
    high: candle.high,
    low: candle.low,
    close: candle.close,
  }
}
