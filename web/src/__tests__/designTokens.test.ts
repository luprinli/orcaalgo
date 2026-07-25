import { describe, it, expect } from 'vitest'
import { RISK_COLORS, CHART_CSS_VARS } from '../lib/design-tokens'

describe('design-tokens', () => {
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
})
