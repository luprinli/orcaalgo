/**
 * OrcaAlgo Trading Theme — Design Tokens
 *
 * Navy blue financial dashboard palette:
 * - Dark mode: deep navy backgrounds with vibrant blue accents
 * - Light mode: crisp white/navy with high contrast
 * - oklch color space: perceptually uniform, light/dark adaptive
 *
 * All token values match CSS custom properties in index.css :root/.dark.
 */

/* ================================================================
   TRADING-SPECIFIC SEMANTIC COLORS
   ================================================================ */

export const TRADING_COLORS = {
  /** Positive PnL, winning trades, compliance pass */
  success: 'oklch(0.65 0.18 150)',
  /** Negative PnL, losing trades, compliance fail */
  danger: 'oklch(0.58 0.2 27)',
  /** Warning thresholds, marginal results */
  warning: 'oklch(0.72 0.15 85)',
  /** Informational, neutral */
  info: 'oklch(0.55 0.18 260)',

  /** Bid side / buying */
  bid: 'oklch(0.65 0.18 150)',
  /** Ask side / selling */
  ask: 'oklch(0.58 0.2 27)',

  /** Regime: bull / trending up */
  regimeBull: 'oklch(0.65 0.18 150)',
  /** Regime: bear / trending down */
  regimeBear: 'oklch(0.58 0.2 27)',
  /** Regime: sideways / mean-reverting */
  regimeNeutral: 'oklch(0.72 0.15 85)',
  /** Regime: crisis / high volatility */
  regimeCrisis: 'oklch(0.55 0.22 27)',

  /** VIX low / calm */
  vixLow: 'oklch(0.65 0.18 150)',
  /** VIX elevated */
  vixElevated: 'oklch(0.72 0.15 85)',
  /** VIX spike / panic */
  vixSpike: 'oklch(0.58 0.2 27)',

  /** Navy blue brand colors */
  navyDark: 'oklch(0.14 0.04 255)',
  navyMid: 'oklch(0.22 0.04 255)',
  navyLight: 'oklch(0.94 0.01 255)',
  blueAccent: 'oklch(0.55 0.18 255)',
}

/* ================================================================
   TYPOGRAPHY — Tabular numerics for financial data
   ================================================================ */

export const TYPOGRAPHY = {
  /** Geist Sans — primary font */
  fontFamily: "'Geist Sans', ui-sans-serif, system-ui, sans-serif",

  /** Geist Mono — prices, quantities, order IDs */
  fontFamilyMono: "'Geist Mono', ui-monospace, monospace",

  body: { fontSize: '13px', lineHeight: '1.5' },
  table: { fontSize: '12px', lineHeight: '1.4' },
  label: {
    fontSize: '10px',
    fontWeight: '600',
    textTransform: 'uppercase' as const,
    letterSpacing: '0.5px',
  },
  value: {
    fontSize: '18px',
    fontWeight: '700',
    fontVariantNumeric: 'tabular-nums' as const,
  },
  h1: { fontSize: '20px', fontWeight: '700' },
  h2: { fontSize: '15px', fontWeight: '600' },
  h3: { fontSize: '13px', fontWeight: '600' },
  tabular: { fontVariantNumeric: 'tabular-nums' as const },
} as const;

/* ================================================================
   SPACING — High-density compact layout
   ================================================================ */

export const SPACING = {
  page: '16px',
  card: '12px',
  section: '12px',
  element: '8px',
  tight: '4px',
  radius: 'var(--radius)',
  radiusSm: 'var(--radius-sm)',
  buttonHeight: '32px',
  buttonHeightSm: '24px',
  buttonHeightLg: '40px',
  inputHeight: '32px',
  tableRowHeight: '28px',
} as const;

/* ================================================================
   CHART TOKENS — Lightweight Charts integration
   ================================================================ */

export const CHART_TOKENS = {
  bg: 'var(--chart-bg)',
  text: 'var(--chart-text)',
  grid: 'var(--chart-grid)',
  crosshair: 'var(--chart-crosshair)',
  line: 'var(--chart-line)',
  area: 'var(--chart-area-fill)',
  candleUp: 'var(--candle-up)',
  candleDown: 'var(--candle-down)',
  candleWick: 'var(--chart-text)',
  volumeUp: 'var(--chart-volume-up)',
  volumeDown: 'var(--chart-volume-down)',
  mcBand: 'var(--chart-mc-band)',
  benchmark: 'var(--chart-benchmark)',
  tradeEntry: 'var(--chart-trade-entry)',
  tradeExit: 'var(--chart-trade-exit)',
} as const;

/* ================================================================
   SHADCN UI COMPONENT OVERRIDES — High-density defaults
   ================================================================ */

export const COMPONENT_DEFAULTS = {
  button: {
    defaultSize: 'sm' as const,
    defaultVariant: 'default' as const,
  },
  card: { padding: '12px', radius: 'var(--radius)' },
  input: { height: '32px', fontSize: '13px' },
  table: { rowHeight: '28px', fontSize: '12px', headerFontSize: '10px' },
  badge: { fontSize: '10px', padding: '1px 6px' },
  tabs: { height: '32px', fontSize: '12px' },
} as const;
