import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, act } from '@testing-library/react'

const mocks = vi.hoisted(() => {
  const crosshairState = { callback: null as ((param: unknown) => void) | null }
  const sharedSetMarkers = vi.fn()

  const mockChart = {
    addSeries: vi.fn(() => ({
      setData: vi.fn(),
      createPriceLine: vi.fn(() => ({})),
    })),
    removeSeries: vi.fn(),
    subscribeCrosshairMove: vi.fn((cb: (param: unknown) => void) => { crosshairState.callback = cb }),
    unsubscribeCrosshairMove: vi.fn(() => { crosshairState.callback = null }),
    timeScale: vi.fn(() => ({
      fitContent: vi.fn(),
      applyOptions: vi.fn(),
      scrollPosition: vi.fn(() => 0),
      scrollToPosition: vi.fn(),
      options: vi.fn(() => ({ barSpacing: 10 })),
    })),
    priceScale: vi.fn(() => ({ applyOptions: vi.fn() })),
    remove: vi.fn(),
    applyOptions: vi.fn(),
  }

  const mockChartRef = { current: mockChart }
  const mockUseChart = vi.fn(() => mockChartRef)

  return { mockChart, mockChartRef, sharedSetMarkers, crosshairState, mockUseChart }
})

vi.mock('lightweight-charts', () => ({
  createChart: vi.fn(() => mocks.mockChart),
  createSeriesMarkers: vi.fn(() => ({ setMarkers: mocks.sharedSetMarkers })),
  LineStyle: { Solid: 0, Dotted: 1, Dashed: 2 },
  LineSeries: {},
  AreaSeries: {},
  HistogramSeries: {},
  CandlestickSeries: {},
  ColorType: { Solid: 'solid' },
}))

vi.mock('../charts/useChart', () => ({
  useChart: mocks.mockUseChart,
  useLineSeries: vi.fn(() => ({
    setData: vi.fn(),
    update: vi.fn(),
    setMarkers: mocks.sharedSetMarkers,
    seriesRef: { current: null },
  })),
  useAreaSeries: vi.fn(() => ({
    setData: vi.fn(),
    update: vi.fn(),
    seriesRef: { current: null },
  })),
  useCandlestickSeries: vi.fn(() => ({
    setData: vi.fn(),
    update: vi.fn(),
    setMarkers: vi.fn(),
    seriesRef: { current: null },
  })),
  useHistogramSeries: vi.fn(() => ({
    setData: vi.fn(),
    update: vi.fn(),
    seriesRef: { current: null },
  })),
  equityToLineData: vi.fn((points: Array<{ time: string; value: number }>) =>
    points.map((p) => ({
      time: new Date(p.time).getTime() / 1000,
      value: p.value,
    }))
  ),
  convertToUTCTime: vi.fn((time: string) => {
    const d = new Date(time)
    return isNaN(d.getTime()) ? 0 : Math.floor(d.getTime() / 1000)
  }),
}))

vi.mock('../hooks/useChartKeyboard', () => ({
  useChartKeyboard: vi.fn(),
}))

vi.mock('../charts/chartConfig', () => ({
  getChartColors: vi.fn(() => ({
    background: '#1a1a2e',
    text: '#d1d4dc',
    grid: '#2a2a3e',
    line: '#2962FF',
    crosshair: '#758696',
    up: '#26a69a',
    down: '#ef5350',
  })),
  getChartDefaults: vi.fn(() => ({
    height: 300,
    layout: { background: { type: 'solid', color: '#1a1a2e' }, textColor: '#d1d4dc' },
    grid: { vertLines: { color: '#2a2a3e' }, horzLines: { color: '#2a2a3e' } },
    crosshair: { mode: 1 },
    timeScale: { timeVisible: true },
    rightPriceScale: { borderColor: '#2a2a3e' },
  })),
}))

import EquityCurveChart from '../charts/EquityCurveChart'

beforeEach(() => {
  vi.clearAllMocks()
  mocks.crosshairState.callback = null
  vi.stubGlobal('ResizeObserver', vi.fn(() => ({ observe: vi.fn(), disconnect: vi.fn() })))
})

describe('EquityCurveChart', () => {
  it('renders without crashing', () => {
    const { container } = render(
      <EquityCurveChart data={[{ time: '2024-01-01T00:00:00Z', value: 100 }]} />,
    )
    expect(container.querySelector('[role="img"]')).toBeTruthy()
  })

  it('renders title when provided', () => {
    render(
      <EquityCurveChart
        data={[{ time: '2024-01-01T00:00:00Z', value: 100 }]}
        title="Test Strategy"
      />,
    )
    expect(screen.getByText('Test Strategy')).toBeTruthy()
  })

  it('renders legend when overlays are present', () => {
    const overlays = [
      {
        label: 'SMA 50',
        data: [{ time: '2024-01-01T00:00:00Z', value: 99 }],
        color: '#3fb950',
      },
      {
        label: 'SMA 200',
        data: [{ time: '2024-01-01T00:00:00Z', value: 98 }],
        color: '#d29922',
      },
    ]

    render(
      <EquityCurveChart
        data={[{ time: '2024-01-01T00:00:00Z', value: 100 }]}
        overlays={overlays}
      />,
    )

    expect(screen.getByText('SMA 50')).toBeTruthy()
    expect(screen.getByText('SMA 200')).toBeTruthy()
  })

  it('does not render legend when overlays are empty', () => {
    render(
      <EquityCurveChart
        data={[{ time: '2024-01-01T00:00:00Z', value: 100 }]}
        overlays={[]}
      />,
    )

    expect(screen.queryByText('SMA 50')).toBeNull()
  })

  it('overlays prop causes extra line series to be added', () => {
    const overlays = [
      {
        label: 'SMA 50',
        data: [{ time: '2024-01-01T00:00:00Z', value: 99 }],
      },
      {
        label: 'BB Upper',
        data: [{ time: '2024-01-01T00:00:00Z', value: 105 }],
      },
      {
        label: 'BB Lower',
        data: [{ time: '2024-01-01T00:00:00Z', value: 95 }],
      },
    ]

    render(
      <EquityCurveChart
        data={[{ time: '2024-01-01T00:00:00Z', value: 100 }]}
        overlays={overlays}
      />,
    )

    expect(mocks.mockChart.addSeries).toHaveBeenCalledTimes(3)
  })

  it('no extra addSeries calls when overlays are empty', () => {
    render(
      <EquityCurveChart
        data={[{ time: '2024-01-01T00:00:00Z', value: 100 }]}
      />,
    )

    expect(mocks.mockChart.addSeries).not.toHaveBeenCalled()
  })

  it('trades prop generates trade markers via setMarkers', () => {
    const trades = [
      { time: '2024-01-02T00:00:00Z', side: 'BUY' as const, price: 101 },
      { time: '2024-01-03T00:00:00Z', side: 'SELL' as const, price: 102 },
    ]

    render(
      <EquityCurveChart
        data={[
          { time: '2024-01-01T00:00:00Z', value: 100 },
          { time: '2024-01-02T00:00:00Z', value: 101 },
          { time: '2024-01-03T00:00:00Z', value: 102 },
        ]}
        trades={trades}
      />,
    )

    expect(mocks.sharedSetMarkers).toHaveBeenCalled()
    const markers = mocks.sharedSetMarkers.mock.calls[0][0]
    expect(markers).toHaveLength(2)
    expect(markers[0].position).toBe('aboveBar')
    expect(markers[0].text).toBe('B')
    expect(markers[1].position).toBe('belowBar')
    expect(markers[1].text).toBe('S')
  })

  it('crosshair move updates displayed data', () => {
    const data = [
      { time: '2024-01-01T00:00:00Z', value: 100 },
      { time: '2024-01-02T00:00:00Z', value: 105 },
    ]
    const overlays = [
      {
        label: 'Benchmark',
        data: [{ time: '2024-01-02T00:00:00Z', value: 103 }],
        color: '#3fb950',
      },
    ]

    render(
      <EquityCurveChart data={data} overlays={overlays} />,
    )

    const ts2024_01_02 = Math.floor(new Date('2024-01-02T00:00:00Z').getTime() / 1000)

    act(() => {
      mocks.crosshairState.callback!({
        time: ts2024_01_02,
        point: { x: 200, y: 150 },
      })
    })

    expect(screen.getByText('105.00')).toBeTruthy()
    expect(screen.getByText(/0\.00\s*%/)).toBeTruthy()
    const benchmarkElements = screen.getAllByText('Benchmark')
    expect(benchmarkElements).toHaveLength(2)
    expect(screen.getByText('103.00')).toBeTruthy()
  })

  it('crosshair clears when param has no time', () => {
    const data = [{ time: '2024-01-01T00:00:00Z', value: 100 }]

    render(<EquityCurveChart data={data} />)

    const ts = Math.floor(new Date('2024-01-01T00:00:00Z').getTime() / 1000)

    act(() => {
      mocks.crosshairState.callback!({ time: ts, point: { x: 100, y: 100 } })
    })
    expect(screen.getByText('100.00')).toBeTruthy()

    act(() => {
      mocks.crosshairState.callback!({})
    })
    // Crosshair data div is removed
    expect(screen.queryByText('100.00')).toBeNull()
    expect(screen.queryByText(/0\.00\s*%/)).toBeNull()
  })

  it('compact mode passes height=150 to useChart', () => {
    render(
      <EquityCurveChart
        data={[{ time: '2024-01-01T00:00:00Z', value: 100 }]}
        height={150}
      />,
    )

    expect(mocks.mockUseChart).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({ height: 150 }),
    )
  })

  it('renders chart with role img for accessibility', () => {
    const { container } = render(
      <EquityCurveChart data={[{ time: '2024-01-01T00:00:00Z', value: 100 }]} />,
    )
    const chart = container.querySelector('[role="img"]')
    expect(chart).toBeTruthy()
    expect(chart!.getAttribute('aria-label')).toBe('Equity curve chart')
  })
})
