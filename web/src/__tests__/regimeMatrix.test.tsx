import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import RegimeActivationMatrix from '../components/backtest/RegimeActivationMatrix'

describe('RegimeActivationMatrix', () => {
  it('renders all 18 strategies', () => {
    render(<RegimeActivationMatrix editable={false} />)
    expect(screen.getByText('Grid Trading')).toBeTruthy()
    expect(screen.getByText('Vol-Adjusted Grid')).toBeTruthy()
    expect(screen.getByText('Trend Following')).toBeTruthy()
    expect(screen.getByText('Session Scalp')).toBeTruthy()
    expect(screen.getByText('Intraday MR')).toBeTruthy()
    expect(screen.getByText('VWAP MR')).toBeTruthy()
    expect(screen.getByText('ORB (5m)')).toBeTruthy()
    expect(screen.getByText('ORB (15m)')).toBeTruthy()
    expect(screen.getByText('Pairs Trading')).toBeTruthy()
    expect(screen.getByText('Vol Harvesting')).toBeTruthy()
    expect(screen.getByText('Dragon Trend')).toBeTruthy()
    expect(screen.getByText('Volume Scalp')).toBeTruthy()
    expect(screen.getByText('VIX Futures Carry')).toBeTruthy()
    expect(screen.getByText('MA Crossover')).toBeTruthy()
    expect(screen.getByText('RSI(2) Reversion')).toBeTruthy()
    expect(screen.getByText('Donchian Breakout')).toBeTruthy()
    expect(screen.getByText('Keltner MACD')).toBeTruthy()
    expect(screen.getByText('Ichimoku Cloud')).toBeTruthy()
  })

  it('renders all 4 regime labels', () => {
    render(<RegimeActivationMatrix editable={false} />)
    expect(screen.getByText('Calm')).toBeTruthy()
    expect(screen.getByText('Trending')).toBeTruthy()
    expect(screen.getByText('High Vol')).toBeTruthy()
    expect(screen.getByText('Crisis')).toBeTruthy()
  })

  it('shows title', () => {
    render(<RegimeActivationMatrix editable={false} />)
    expect(screen.getByText('Strategy vs Regime Activation Matrix')).toBeTruthy()
  })

  it('shows read-only message when not editable', () => {
    render(<RegimeActivationMatrix editable={false} />)
    expect(screen.getByText(/Read-only/)).toBeTruthy()
  })

  it('switches are disabled when not editable', () => {
    render(<RegimeActivationMatrix editable={false} />)
    const switches = document.querySelectorAll('button[role="switch"]')
    expect(switches.length).toBe(72) // 18 strategies × 4 regimes
    switches.forEach((el) => {
      expect(el.getAttribute('disabled')).toBe('')
    })
  })
})
