import { ColorType, type DeepPartial, type ChartOptions } from 'lightweight-charts'

function getCSSVar(name: string, fallback: string): string {
  if (typeof window === 'undefined') return fallback
  const style = getComputedStyle(document.documentElement)
  const val = style.getPropertyValue(name).trim()
  return val || fallback
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
  }
}
