// Trading-specific design tokens for the OrcaAlgo dashboard

// Risk threshold colors — maps to trading semantic CSS variables
export const RISK_COLORS = {
  safe: 'text-trading-success',
  warning: 'text-trading-warning',
  danger: 'text-trading-danger',
  critical: 'text-destructive',
} as const;

// Chart CSS variable bridge (consumed by chartConfig.ts & Lightweight Charts)
export const CHART_CSS_VARS = {
  bg: 'var(--chart-bg)',
  text: 'var(--chart-text)',
  grid: 'var(--chart-grid)',
  line: 'var(--chart-line)',
  crosshair: 'var(--chart-crosshair)',
  candleUp: 'var(--candle-up)',
  candleDown: 'var(--candle-down)',
} as const;

export type RiskColor = keyof typeof RISK_COLORS;
