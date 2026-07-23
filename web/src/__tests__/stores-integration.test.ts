import { describe, it, expect, beforeEach } from 'vitest'

describe('authStore integration', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('full auth lifecycle: login → store → retrieve → logout', async () => {
    const { useAuthStore, hydrateAuth } = await import('../stores/authStore')

    // Start clean
    expect(useAuthStore.getState().token).toBeNull()

    // Simulate login
    const loginData = { token: 'access-123', refresh: 'refresh-456', roles: ['trader'] }
    useAuthStore.getState().setAuth(loginData.token, loginData.refresh, loginData.roles)

    // Verify state
    expect(useAuthStore.getState().token).toBe('access-123')
    expect(useAuthStore.getState().refresh).toBe('refresh-456')
    expect(useAuthStore.getState().roles).toEqual(['trader'])
    expect(useAuthStore.getState().isAuthenticated()).toBe(true)

    // Verify persisted
    const raw = localStorage.getItem('orca_auth')
    expect(raw).not.toBeNull()
    const parsed = JSON.parse(raw!)
    expect(parsed.token).toBe('access-123')

    // Simulate page reload — clear store, hydrate from localStorage
    useAuthStore.getState().clearAuth()
    expect(useAuthStore.getState().token).toBeNull()

    // Re-set localStorage (simulating page reload where it's still there)
    localStorage.setItem('orca_auth', JSON.stringify(loginData))

    hydrateAuth()
    expect(useAuthStore.getState().token).toBe('access-123')
    expect(useAuthStore.getState().isAuthenticated()).toBe(true)

    // Logout
    useAuthStore.getState().clearAuth()
    expect(useAuthStore.getState().token).toBeNull()
    expect(useAuthStore.getState().isAuthenticated()).toBe(false)
    expect(localStorage.getItem('orca_auth')).toBeNull()
  })

  it('handles corrupted localStorage gracefully', async () => {
    localStorage.setItem('orca_auth', 'not-valid-json{{{')
    const { hydrateAuth } = await import('../stores/authStore')
    // Should not throw; hydrateAuth catches JSON parse errors
    expect(() => hydrateAuth()).not.toThrow()
  })
})

describe('wsStore integration', () => {
  it('full WS lifecycle: disconnect → connect → receive data → disconnect', async () => {
    const { useWSStore } = await import('../stores/wsStore')

    // Start disconnected
    expect(useWSStore.getState().connected).toBe(false)

    // Connect
    useWSStore.getState().setConnected(true)
    expect(useWSStore.getState().connected).toBe(true)

    // Receive risk data
    useWSStore.getState().setRisk({
      halted: false,
      equity: 105000,
      daily_pnl_pct: 1.5,
      regime: 1,
      reason: '',
      connections: 0,
      confidence: 0.85,
      vix: 18,
      sentiment: 0.3,
      daily_loss_used: 20,
      drawdown_used: 10,
      daily_limit_pct: 5,
      max_dd_pct: 10,
      consistency_multiplier: 1.0,
      tick_count: 500,
      balance: 100000,
    })

    expect(useWSStore.getState().risk?.equity).toBe(105000)
    expect(useWSStore.getState().risk?.regime).toBe(1)
    expect(useWSStore.getState().risk?.halted).toBe(false)

    // Receive performance data
    useWSStore.getState().setPerformance({
      timestamp: '2026-06-30T16:00:00Z',
      equity: 105000,
      balance: 100000,
      daily_pnl: 5000,
      daily_pnl_pct: 1.5,
      drawdown_pct: 2,
      max_drawdown_pct: 8,
      sharpe: 2.1,
      sortino: 3.0,
      win_rate: 0.65,
      profit_factor: 1.8,
      num_trades: 50,
    })

    expect(useWSStore.getState().performance?.sharpe).toBe(2.1)

    // Disconnect
    useWSStore.getState().setConnected(false)
    expect(useWSStore.getState().connected).toBe(false)
    // Data persists
    expect(useWSStore.getState().risk).not.toBeNull()
  })
})
