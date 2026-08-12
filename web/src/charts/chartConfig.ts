import { ColorType, type DeepPartial, type ChartOptions } from 'lightweight-charts'

export const CHART_LAYOUT = {
  CANDLE_SCALE_MARGINS: { top: 0.1, bottom: 0.2 },
  EQUITY_SCALE_MARGINS: { top: 0.02, bottom: 0.25 },
  VOLUME_SCALE_MARGINS: { top: 0.85, bottom: 0 },
  DRAWDDOWN_SCALE_MARGINS: { top: 0.75, bottom: 0 },
} as const

export const OVERLAY_PALETTE = ['#3fb950', '#d29922', '#da3633', '#8b949e', '#f0883e', '#58a6ff', '#bc8cff', '#ff7b72']
export const COMPARE_COLORS = ['#3fb950', '#d29922', '#58a6ff', '#da3633', '#f0883e', '#bc8cff', '#8b949e', '#ff7b72']

export const CHART_DIMENSIONS = {
  DEFAULT_CHART_HEIGHT: 500,
  EQUITY_CHART_HEIGHT: 300,
  DD_CHART_HEIGHT: 200,
} as const

function getCSSVar(name: string, fallback: string): string {
  if (typeof window === 'undefined') return fallback
  try {
    const style = getComputedStyle(document.documentElement)
    const val = style.getPropertyValue(name).trim()
    if (!val) return fallback
    // OKLCH colors are not supported by lightweight-charts — fall back
    if (val.startsWith('oklch(')) return fallback
    return val
  } catch {
    return fallback
  }
}

export interface ChartColors {
  background: string
  text: string
  grid: string
  line: string
  crosshair: string
  up: string
  down: string
}

export function getChartColors(): ChartColors {
  return {
    background: getCSSVar('--chart-bg', '#1a1a2e'),
    text:       getCSSVar('--chart-text', '#d1d4dc'),
    grid:       getCSSVar('--chart-grid', '#2a2a3e'),
    line:       getCSSVar('--chart-line', '#2962FF'),
    crosshair:  getCSSVar('--chart-crosshair', '#758696'),
    up:         getCSSVar('--candle-up', '#26a69a'),
    down:       getCSSVar('--candle-down', '#ef5350'),
  }
}

export function getChartDefaults(height: number, colors?: Partial<ChartColors>): DeepPartial<ChartOptions> {
  const c = { ...getChartColors(), ...colors }
  return {
    height,
    layout: {
      background: { type: ColorType.Solid, color: c.background },
      textColor: c.text,
    },
    grid: {
      vertLines: { color: c.grid },
      horzLines: { color: c.grid },
    },
    crosshair: {
      mode: 1,
      vertLine: { color: c.crosshair, width: 1, style: 2, labelBackgroundColor: c.crosshair },
      horzLine: { color: c.crosshair, width: 1, style: 2, labelBackgroundColor: c.crosshair },
    },
    timeScale: {
      borderColor: c.grid,
      timeVisible: true,
      secondsVisible: false,
    },
    rightPriceScale: {
      borderColor: c.grid,
    },
    localization: {
      locale: 'en-US',
      dateFormat: 'yyyy-MM-dd',
    },
  }
}
