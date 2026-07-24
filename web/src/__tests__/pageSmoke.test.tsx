import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string, fallback?: string) => fallback ?? key }),
}))

vi.mock('../api/client', () => ({
  live: { metrics: vi.fn().mockResolvedValue({}), equity: vi.fn().mockResolvedValue([]), trades: vi.fn().mockResolvedValue({ trades: [] }) },
  orders: { list: vi.fn().mockResolvedValue({ orders: [] }), cancel: vi.fn() },
  positions: { list: vi.fn().mockResolvedValue([]) },
  risk: { status: vi.fn().mockResolvedValue({ halted: false, equity: 100000, balance: 100000, daily_pnl_pct: 0, drawdown_used: 0, daily_loss_used: 0, daily_limit_pct: 5, max_dd_pct: 20 }), emergencyStop: vi.fn(), emergencyResume: vi.fn() },
  monitor: { regimeHistory: vi.fn().mockResolvedValue({ history: [] }) },
  settings: { get: vi.fn().mockResolvedValue({}), update: vi.fn() },
  brokers: { list: vi.fn().mockResolvedValue({ brokers: [] }) },
  request: vi.fn().mockResolvedValue([]),
}))

vi.mock('../hooks/useWebSocket', () => ({
  useWebSocket: () => ({ connected: false, lastMessage: null, send: vi.fn() }),
  useWebSocketMulti: () => ({ connected: false, lastMessage: null, send: vi.fn() }),
}))

vi.mock('../hooks/useLiveRiskData', () => ({
  useLiveRiskData: () => ({ riskData: null, connected: false, isHalted: false, error: null, refetch: vi.fn() }),
}))

vi.mock('../charts/EquityCurveChart', () => ({
  default: () => null,
}))

import CommandCenter from '../pages/CommandCenter'
import EmergencyPage from '../pages/EmergencyPage'

describe('Page Smoke Tests', () => {
  it('CommandCenter renders without crashing', () => {
    const { container } = render(<BrowserRouter><CommandCenter /></BrowserRouter>)
    expect(container).toBeTruthy()
  })

  it('EmergencyPage renders without crashing', () => {
    const { container } = render(<BrowserRouter><EmergencyPage /></BrowserRouter>)
    expect(container).toBeTruthy()
  })
})
