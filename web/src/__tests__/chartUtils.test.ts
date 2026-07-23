import { describe, it, expect } from 'vitest'
import {
  convertToUTCTime,
  equityToLineData,
  candlesToData,
  candlesToVolumeData,
  candleToUpdatable,
} from '../charts/chartUtils'

// ---------------------------------------------------------------------------
// convertToUTCTime
// ---------------------------------------------------------------------------
describe('convertToUTCTime', () => {
  const epochSeconds = (iso: string) => Math.floor(new Date(iso).getTime() / 1000)

  it('converts a valid ISO date to UTC seconds', () => {
    const result = convertToUTCTime('2026-01-15T10:30:00Z')
    expect(result).toBe(epochSeconds('2026-01-15T10:30:00Z'))
    expect(result).toBeGreaterThan(0)
  })

  it('returns 0 for null/undefined (falsy) input', () => {
    expect(convertToUTCTime('' as unknown as string)).toBe(0)
    expect(convertToUTCTime(undefined as unknown as string)).toBe(0)
    expect(convertToUTCTime(null as unknown as string)).toBe(0)
  })

  it('returns 0 for "None" string', () => {
    expect(convertToUTCTime('None')).toBe(0)
  })

  it('returns 0 for "null" string', () => {
    expect(convertToUTCTime('null')).toBe(0)
  })

  it('returns 0 for pre-1970 date (negative timestamp)', () => {
    expect(convertToUTCTime('1969-12-31T23:59:59Z')).toBe(0)
    expect(convertToUTCTime('1960-01-01T00:00:00Z')).toBe(0)
  })

  it('handles far-future dates', () => {
    const result = convertToUTCTime('2099-12-31T23:59:59Z')
    expect(result).toBe(epochSeconds('2099-12-31T23:59:59Z'))
    expect(result).toBeGreaterThan(0)
  })

  it('returns correct value for Unix epoch', () => {
    const result = convertToUTCTime('1970-01-01T00:00:00Z')
    expect(result).toBe(0)
  })

  it('returns 0 for a completely invalid date string', () => {
    expect(convertToUTCTime('not-a-date')).toBe(0)
  })

  it('handles ISO string without timezone (UTC assumed by runtime)', () => {
    const result = convertToUTCTime('2026-06-30T14:45:00')
    const expected = new Date('2026-06-30T14:45:00').getTime()
    expect(result).toBe(Math.floor(expected / 1000))
  })
})

// ---------------------------------------------------------------------------
// equityToLineData
// ---------------------------------------------------------------------------
describe('equityToLineData', () => {
  it('maps and sorts 2+ valid points', () => {
    const points = [
      { time: '2026-02-02T10:00:00Z', value: 100000 },
      { time: '2026-02-01T10:00:00Z', value: 99500 },
      { time: '2026-02-03T10:00:00Z', value: 100500 },
    ]
    const result = equityToLineData(points)
    expect(result).toHaveLength(3)
    // sorted by time ascending
    expect(result[0].value).toBe(99500)
    expect(result[1].value).toBe(100000)
    expect(result[2].value).toBe(100500)
    // time values are numeric (seconds)
    expect(typeof result[0].time).toBe('number')
    expect(result[0].time).toBeGreaterThan(0)
  })

  it('returns empty array for empty input', () => {
    expect(equityToLineData([])).toEqual([])
  })

  it('maps a single valid point', () => {
    const points = [{ time: '2026-03-15T12:00:00Z', value: 150000 }]
    const result = equityToLineData(points)
    expect(result).toHaveLength(1)
    expect(result[0].value).toBe(150000)
  })

  it('filters out points with invalid timestamps', () => {
    const points = [
      { time: 'None', value: 100 },
      { time: 'null', value: 200 },
      { time: '', value: 300 },
      { time: 'not-a-date', value: 400 },
    ]
    const result = equityToLineData(points)
    expect(result).toEqual([])
  })

  it('filters invalid timestamps and keeps valid ones, then sorts', () => {
    const points = [
      { time: 'None', value: 999 },
      { time: '2026-04-20T12:00:00Z', value: 120000 },
      { time: '2026-04-18T12:00:00Z', value: 118000 },
      { time: 'null', value: 888 },
    ]
    const result = equityToLineData(points)
    expect(result).toHaveLength(2)
    expect(result[0].value).toBe(118000)
    expect(result[1].value).toBe(120000)
  })
})

// ---------------------------------------------------------------------------
// candlesToData
// ---------------------------------------------------------------------------
describe('candlesToData', () => {
  const bullishCandle = {
    time: '2026-05-01T10:00:00Z',
    open: 100,
    high: 110,
    low: 95,
    close: 108,
    volume: 5000,
  }
  const bearishCandle = {
    time: '2026-05-01T11:00:00Z',
    open: 108,
    high: 112,
    low: 102,
    close: 104,
    volume: 8000,
  }
  const neutralCandle = {
    time: '2026-05-01T12:00:00Z',
    open: 104,
    high: 106,
    low: 103,
    close: 104,
    volume: 3000,
  }

  it('returns candlestick and volume arrays with correct length', () => {
    const result = candlesToData([bullishCandle, bearishCandle, neutralCandle])
    expect(result.candlestick).toHaveLength(3)
    expect(result.volume).toHaveLength(3)
  })

  it('maps candlestick fields correctly', () => {
    const result = candlesToData([bullishCandle])
    const cs = result.candlestick[0]
    expect(cs.open).toBe(100)
    expect(cs.high).toBe(110)
    expect(cs.low).toBe(95)
    expect(cs.close).toBe(108)
    expect(typeof cs.time).toBe('number')
    expect(cs.time).toBeGreaterThan(0)
  })

  it('maps volume fields correctly', () => {
    const result = candlesToData([bullishCandle])
    const vol = result.volume[0]
    expect(vol.value).toBe(5000)
    expect(typeof vol.time).toBe('number')
  })

  it('applies green volume color for bullish candle (close >= open)', () => {
    const result = candlesToData([bullishCandle])
    expect(result.volume[0].color).toBe('rgba(38, 166, 154, 0.3)')
  })

  it('applies red volume color for bearish candle (close < open)', () => {
    const result = candlesToData([bearishCandle])
    expect(result.volume[0].color).toBe('rgba(239, 83, 80, 0.3)')
  })

  it('applies green volume color for neutral/flat candle (close >= open)', () => {
    const result = candlesToData([neutralCandle])
    expect(result.volume[0].color).toBe('rgba(38, 166, 154, 0.3)')
  })

  it('handles empty input', () => {
    const result = candlesToData([])
    expect(result.candlestick).toEqual([])
    expect(result.volume).toEqual([])
  })
})

// ---------------------------------------------------------------------------
// candlesToVolumeData
// ---------------------------------------------------------------------------
describe('candlesToVolumeData', () => {
  const bullishCandle = {
    time: '2026-06-15T14:00:00Z',
    open: 200,
    high: 210,
    low: 198,
    close: 208,
    volume: 12000,
  }
  const bearishCandle = {
    time: '2026-06-15T15:00:00Z',
    open: 208,
    high: 212,
    low: 195,
    close: 197,
    volume: 15000,
  }

  it('converts multiple candles to histogram data', () => {
    const result = candlesToVolumeData([bullishCandle, bearishCandle])
    expect(result).toHaveLength(2)
  })

  it('maps time to numeric seconds', () => {
    const result = candlesToVolumeData([bullishCandle])
    expect(typeof result[0].time).toBe('number')
    expect(result[0].time).toBeGreaterThan(0)
  })

  it('maps volume value', () => {
    const result = candlesToVolumeData([bullishCandle])
    expect(result[0].value).toBe(12000)
  })

  it('applies green color for close >= open (bullish)', () => {
    const result = candlesToVolumeData([bullishCandle])
    expect(result[0].color).toBe('rgba(38,166,154,0.3)')
  })

  it('applies red color for close < open (bearish)', () => {
    const result = candlesToVolumeData([bearishCandle])
    expect(result[0].color).toBe('rgba(239,83,80,0.3)')
  })

  it('applies green color for flat candle (close === open)', () => {
    const flatCandle = {
      time: '2026-06-15T16:00:00Z',
      open: 197,
      high: 200,
      low: 195,
      close: 197,
      volume: 5000,
    }
    const result = candlesToVolumeData([flatCandle])
    expect(result[0].color).toBe('rgba(38,166,154,0.3)')
  })

  it('returns empty array for empty input', () => {
    expect(candlesToVolumeData([])).toEqual([])
  })
})

// ---------------------------------------------------------------------------
// candleToUpdatable
// ---------------------------------------------------------------------------
describe('candleToUpdatable', () => {
  it('converts time string to numeric Time', () => {
    const candle = {
      time: '2026-07-01T09:30:00Z',
      open: 450,
      high: 455,
      low: 448,
      close: 453,
    }
    const result = candleToUpdatable(candle)
    expect(typeof result.time).toBe('number')
    const expectedSec = Math.floor(new Date('2026-07-01T09:30:00Z').getTime() / 1000)
    expect(result.time).toBe(expectedSec)
  })

  it('preserves OHLC fields', () => {
    const candle = {
      time: '2026-07-01T10:00:00Z',
      open: 300,
      high: 315,
      low: 298,
      close: 310,
    }
    const result = candleToUpdatable(candle)
    expect(result.open).toBe(300)
    expect(result.high).toBe(315)
    expect(result.low).toBe(298)
    expect(result.close).toBe(310)
  })

  it('does not mutate the input object', () => {
    const candle = {
      time: '2026-07-01T11:00:00Z',
      open: 500,
      high: 520,
      low: 495,
      close: 515,
    }
    const original = { ...candle }
    candleToUpdatable(candle)
    expect(candle).toEqual(original)
  })
})
