import { describe, it, expect } from 'vitest'
import { convertToUTCTime } from '../chartUtils'

describe('Chart Performance Benchmarks', () => {
  function generateCandles(count: number) {
    const candles = []
    let price = 100
    const baseTime = new Date('2024-01-01').getTime() / 1000
    for (let i = 0; i < count; i++) {
      const change = (Math.random() - 0.5) * 2
      candles.push({
        time: baseTime + i * 60,
        open: price,
        high: price + Math.abs(change),
        low: price - Math.abs(change),
        close: price + change,
        volume: Math.floor(Math.random() * 1000000),
      })
      price += change
    }
    return candles
  }

  it('generates 10K candles in under 50ms', () => {
    const start = performance.now()
    const candles = generateCandles(10000)
    const elapsed = performance.now() - start
    expect(candles).toHaveLength(10000)
    expect(elapsed).toBeLessThan(50)
  })

  it('data deduplication handles 10K entries efficiently', () => {
    const candles = generateCandles(10000)
    // Simple dedup — must complete in under 10ms
    const start = performance.now()
    const seen = new Set<number>()
    const deduped = candles.filter(c => {
      if (seen.has(c.time)) return false
      seen.add(c.time)
      return true
    })
    const elapsed = performance.now() - start
    expect(deduped.length).toBeLessThanOrEqual(10000)
    expect(elapsed).toBeLessThan(10)
  })

  it('time conversion handles edge cases', () => {
    // These should not throw
    expect(convertToUTCTime(null as any)).toBeDefined()
    expect(convertToUTCTime('' as any)).toBeDefined()
    expect(convertToUTCTime('2024-01-01T00:00:00Z')).toBeGreaterThan(0)
  })
})
