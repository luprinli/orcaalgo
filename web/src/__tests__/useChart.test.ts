import { describe, it, expect } from 'vitest'
import { convertToUTCTime, equityToLineData, candlesToData } from '../charts/useChart'

describe('convertToUTCTime', () => {
  it('converts ISO string to unix timestamp', () => {
    const result = convertToUTCTime('2024-01-15T10:30:00Z')
    expect(typeof result).toBe('number')
    expect(result).toBe(1705314600)
  })

  it('handles date-only string', () => {
    const result = convertToUTCTime('2024-01-01')
    expect(typeof result).toBe('number')
  })

  // ── Boundary conditions (regression for NaN crash) ──

  it('returns 0 for empty string', () => {
    expect(convertToUTCTime('')).toBe(0)
  })

  it('returns 0 for "None" string', () => {
    expect(convertToUTCTime('None')).toBe(0)
  })

  it('returns 0 for "null" string', () => {
    expect(convertToUTCTime('null')).toBe(0)
  })

  it('returns 0 for arbitrary invalid date strings', () => {
    expect(convertToUTCTime('not-a-date')).toBe(0)
  })

  it('returns 0 for undefined-like inputs', () => {
    expect(convertToUTCTime('undefined')).toBe(0)
  })
})

describe('equityToLineData', () => {
  it('converts equity points to LineData', () => {
    const input = [
      { time: '2024-01-01T00:00:00Z', value: 100000 },
      { time: '2024-01-02T00:00:00Z', value: 101000 },
    ]
    const result = equityToLineData(input)
    expect(result).toHaveLength(2)
    expect(result[0]).toHaveProperty('time')
    expect(result[0]).toHaveProperty('value')
    expect(result[0].value).toBe(100000)
  })

  it('returns empty array for empty input', () => {
    expect(equityToLineData([])).toEqual([])
  })

  // ── Boundary conditions (regression for NaN crash) ──

  it('filters out points with empty time strings', () => {
    const input = [
      { time: '', value: 100000 },
      { time: '2024-01-03T00:00:00Z', value: 100000 },
    ]
    expect(equityToLineData(input)).toHaveLength(1)
  })

  it('filters out points with "None" time', () => {
    const input = [{ time: 'None', value: 100000 }]
    expect(equityToLineData(input)).toEqual([])
  })

  it('sorts output by time ascending', () => {
    const input = [
      { time: '2024-01-05T00:00:00Z', value: 100 },
      { time: '2024-01-01T00:00:00Z', value: 100 },
    ]
    const result = equityToLineData(input)
    expect(result).toHaveLength(2)
    expect((result[0].time as number)).toBeLessThan((result[1].time as number))
  })

  it('returns empty array when all points filtered out', () => {
    const input = [
      { time: '', value: 100 },
      { time: 'None', value: 200 },
    ]
    expect(equityToLineData(input)).toEqual([])
  })
})

describe('candlesToData', () => {
  it('converts candle array to candlestick and volume data', () => {
    const input = [
      { time: '2024-01-01T00:00:00Z', open: 100, high: 105, low: 99, close: 104, volume: 1000 },
    ]
    const result = candlesToData(input)
    expect(result.candlestick).toHaveLength(1)
    expect(result.volume).toHaveLength(1)
    expect(result.candlestick[0].open).toBe(100)
    expect(result.candlestick[0].close).toBe(104)
    expect(result.volume[0].value).toBe(1000)
  })

  it('colors volume green for bullish candle, red for bearish', () => {
    const bullish = candlesToData([
      { time: '2024-01-01T00:00:00Z', open: 100, high: 105, low: 99, close: 104, volume: 1000 },
    ])
    expect(bullish.volume[0].color).toContain('38, 166, 154')

    const bearish = candlesToData([
      { time: '2024-01-01T00:00:00Z', open: 104, high: 105, low: 99, close: 100, volume: 1000 },
    ])
    expect(bearish.volume[0].color).toContain('239, 83, 80')
  })
})
