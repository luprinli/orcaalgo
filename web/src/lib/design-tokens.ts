// Trading-specific design tokens for the OrcaAlgo dashboard
export const METRIC_GRID_COLS = {
  dashboard: 'grid-cols-3',
  detail: 'grid-cols-5',
  compact: 'grid-cols-4',
} as const

export const CARD_PADDING = 'p-4'
export const SECTION_GAP = 'gap-6'
export const PAGE_PADDING = 'p-6'

// Risk threshold colors
export const RISK_COLORS = {
  safe: 'text-green-400',
  warning: 'text-yellow-400',
  danger: 'text-red-400',
  critical: 'text-red-600',
} as const

// Chart CSS variable bridge (keep existing Lightweight Charts variables)
export const CHART_CSS_VARS = {
  bg: 'var(--chart-bg)',
  text: 'var(--chart-text)',
  grid: 'var(--chart-grid)',
  line: 'var(--chart-line)',
  crosshair: 'var(--chart-crosshair)',
  candleUp: 'var(--candle-up)',
  candleDown: 'var(--candle-down)',
} as const

export type RiskColor = keyof typeof RISK_COLORS
