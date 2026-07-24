import { describe, it, expect } from 'vitest'
import { METRIC_GRID_COLS, CARD_PADDING, SECTION_GAP, PAGE_PADDING, RISK_COLORS, CHART_CSS_VARS } from '../lib/design-tokens'

describe('design-tokens', () => {
  it('METRIC_GRID_COLS has expected keys', () => {
    expect(METRIC_GRID_COLS.dashboard).toBe('grid-cols-3')
    expect(METRIC_GRID_COLS.detail).toBe('grid-cols-5')
    expect(METRIC_GRID_COLS.compact).toBe('grid-cols-4')
  })

  it('RISK_COLORS has all risk levels', () => {
    expect(RISK_COLORS.safe).toBeTruthy()
    expect(RISK_COLORS.warning).toBeTruthy()
    expect(RISK_COLORS.danger).toBeTruthy()
    expect(RISK_COLORS.critical).toBeTruthy()
  })

  it('CHART_CSS_VARS references CSS variables', () => {
    Object.values(CHART_CSS_VARS).forEach(v => {
      expect(v).toMatch(/^var\(/)
    })
  })

  it('CARD_PADDING and SECTION_GAP are strings', () => {
    expect(typeof CARD_PADDING).toBe('string')
    expect(typeof SECTION_GAP).toBe('string')
    expect(typeof PAGE_PADDING).toBe('string')
  })
})
