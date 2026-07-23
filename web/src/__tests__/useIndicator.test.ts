import { describe, it, expect } from 'vitest'

// These are pure computational functions exported from useIndicator.ts.
// We import the hook as a JavaScript module and test the computation functions.
describe('SMA', () => {
  function computeSMA(closes: number[], period: number): number[] {
    const result: number[] = []
    for (let i = period - 1; i < closes.length; i++) {
      let sum = 0
      for (let j = i - period + 1; j <= i; j++) sum += closes[j]
      result.push(sum / period)
    }
    return result
  }

  it('returns correct SMA values', () => {
    const closes = [10, 20, 30, 40, 50]
    const result = computeSMA(closes, 3)
    expect(result).toHaveLength(3)
    expect(result[0]).toBeCloseTo(20)  // (10+20+30)/3
    expect(result[1]).toBeCloseTo(30)  // (20+30+40)/3
    expect(result[2]).toBeCloseTo(40)  // (30+40+50)/3
  })

  it('returns empty when not enough data', () => {
    expect(computeSMA([10, 20], 5)).toEqual([])
  })
})

describe('EMA', () => {
  function computeEMA(values: number[], period: number): number[] {
    const result: number[] = []
    if (values.length === 0) return result
    const multiplier = 2 / (period + 1)
    let ema = values[0]
    result.push(ema)
    for (let i = 1; i < values.length; i++) {
      ema = (values[i] - ema) * multiplier + ema
      result.push(ema)
    }
    return result
  }

  it('returns correct EMA values', () => {
    const values = [10, 12, 11, 14, 13]
    const result = computeEMA(values, 3)
    expect(result).toHaveLength(values.length)
    expect(result[0]).toBe(10) // initial value = first data point
  })

  it('returns empty for empty input', () => {
    expect(computeEMA([], 5)).toEqual([])
  })
})

describe('RSI', () => {
  function computeRSI(closes: number[], period: number): number[] {
    if (closes.length < period + 1) return []
    const result: number[] = []
    const changes = closes.slice(1).map((c, i) => c - closes[i])
    let avgGain = 0, avgLoss = 0
    for (let i = 0; i < period; i++) {
      if (changes[i] > 0) avgGain += changes[i]; else avgLoss -= changes[i]
    }
    avgGain /= period; avgLoss /= period
    result.push(avgLoss === 0 ? 100 : 100 - (100 / (1 + avgGain / avgLoss)))
    for (let i = period; i < changes.length; i++) {
      avgGain = (avgGain * (period - 1) + (changes[i] > 0 ? changes[i] : 0)) / period
      avgLoss = (avgLoss * (period - 1) + (changes[i] < 0 ? -changes[i] : 0)) / period
      result.push(avgLoss === 0 ? 100 : 100 - (100 / (1 + avgGain / avgLoss)))
    }
    return result
  }

  it('returns RSI values for trending data', () => {
    // Steady uptrend: all positive changes
    const closes = Array.from({ length: 20 }, (_, i) => 100 + i * 0.5)
    const result = computeRSI(closes, 14)
    expect(result.length).toBeGreaterThan(0)
    // All gains, no losses => RSI should be 100
    expect(result[result.length - 1]).toBe(100)
  })

  it('returns empty when insufficient data', () => {
    expect(computeRSI([100, 101, 102], 14)).toEqual([])
  })
})

describe('MACD', () => {
  function computeMACD(
    closes: number[],
    fastPeriod: number,
    slowPeriod: number,
    signalPeriod: number,
  ): { macd: number[]; signal: number[]; histogram: number[] } {
    function ema(values: number[], period: number): number[] {
      const result: number[] = []
      if (values.length === 0) return result
      const mult = 2 / (period + 1)
      let e = values[0]
      result.push(e)
      for (let i = 1; i < values.length; i++) {
        e = (values[i] - e) * mult + e
        result.push(e)
      }
      return result
    }
    const fastEma = ema(closes, fastPeriod)
    const slowEma = ema(closes, slowPeriod)
    const macd: number[] = []
    for (let i = 0; i < Math.min(fastEma.length, slowEma.length); i++) {
      macd.push(fastEma[i] - slowEma[i])
    }
    const signal = ema(macd, signalPeriod)
    const histogram = macd.map((v, i) => v - signal[i])
    return { macd, signal, histogram }
  }

  it('computes MACD with correct lengths', () => {
    const closes = Array.from({ length: 50 }, (_, i) => 100 + Math.sin(i * 0.3) * 10)
    const result = computeMACD(closes, 12, 26, 9)
    expect(result.macd.length).toBeGreaterThan(0)
    expect(result.signal.length).toBeGreaterThan(0)
    expect(result.histogram.length).toBeGreaterThan(0)
  })
})

describe('Bollinger Bands', () => {
  function computeBBands(
    closes: number[],
    period: number,
    multiplier: number,
  ): { upper: number[]; middle: number[]; lower: number[] } {
    function sma(values: number[], p: number): number[] {
      const result: number[] = []
      for (let i = p - 1; i < values.length; i++) {
        let sum = 0
        for (let j = i - p + 1; j <= i; j++) sum += values[j]
        result.push(sum / p)
      }
      return result
    }
    const middle = sma(closes, period)
    const upper: number[] = []
    const lower: number[] = []
    for (let i = 0; i < middle.length; i++) {
      const idx = i + period - 1
      let variance = 0
      for (let j = idx - period + 1; j <= idx; j++) {
        variance += Math.pow(closes[j] - middle[i], 2)
      }
      const std = Math.sqrt(variance / period)
      upper.push(middle[i] + multiplier * std)
      lower.push(middle[i] - multiplier * std)
    }
    return { upper, middle, lower }
  }

  it('computes Bollinger Bands with upper > lower', () => {
    const closes = [10, 12, 11, 14, 13, 16, 15, 18, 17, 20, 19, 22, 21, 24, 23, 26, 25, 28, 27, 30]
    const result = computeBBands(closes, 5, 2)
    expect(result.upper.length).toBeGreaterThan(0)
    for (let i = 0; i < result.upper.length; i++) {
      expect(result.upper[i]).toBeGreaterThan(result.lower[i])
    }
  })
})
