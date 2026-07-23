import { describe, it, expect } from 'vitest'

// Test that the useWebSocket hook exports work at the module level.
// The hook itself requires jsdom + WebSocket mock which we validate
// via the existing useWebSocket.test.ts protocol tests.

describe('useWebSocket hook — lifecycle simulation', () => {
  it('export is importable', async () => {
    const mod = await import('../hooks/useWebSocket')
    expect(mod.useWebSocket).toBeDefined()
    expect(mod.useWebSocketMulti).toBeDefined()
  })

  it('useWebSocket is a function', async () => {
    const { useWebSocket } = await import('../hooks/useWebSocket')
    expect(typeof useWebSocket).toBe('function')
  })

  it('useWebSocketMulti is a function', async () => {
    const { useWebSocketMulti } = await import('../hooks/useWebSocket')
    expect(typeof useWebSocketMulti).toBe('function')
  })
})
