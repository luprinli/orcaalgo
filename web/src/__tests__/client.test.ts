import { describe, it, expect, vi, beforeEach } from 'vitest'

const mockFetch = vi.fn()
const originalFetch = globalThis.fetch

beforeEach(() => {
  mockFetch.mockReset()
  localStorage.clear()
  localStorage.setItem('orca_auth', JSON.stringify({ token: 'test-token', refresh: 'ref', roles: ['user'] }))
})

// We need to import client after setting up localStorage
describe('API client — auth headers', () => {
  it('injects Authorization header from localStorage', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ strategies: [] }),
    })

    // Dynamically import after setting up mocks
    const { strategies } = await import('../api/client')
    await strategies.list()

    expect(globalThis.fetch).toHaveBeenCalledWith(
      '/api/v1/strategies',
      expect.objectContaining({
        method: 'GET',
        headers: expect.objectContaining({
          'Content-Type': 'application/json',
          Authorization: 'Bearer test-token',
        }),
      }),
    )
    globalThis.fetch = originalFetch
  })

  it('does not send auth header when no token exists', async () => {
    localStorage.clear()
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ strategies: [] }),
    })

    const { strategies } = await import('../api/client')
    await strategies.list()

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const callHeaders = (globalThis.fetch as any).mock.calls[0][1].headers
    expect(callHeaders.Authorization).toBeUndefined()
    expect(callHeaders['Content-Type']).toBe('application/json')
    globalThis.fetch = originalFetch
  })
})

describe('API client — error handling', () => {
  it('throws on non-ok response with error body', async () => {
    localStorage.clear()
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      json: () => Promise.resolve({ error: 'Internal server error' }),
    })

    const { strategies } = await import('../api/client')
    await expect(strategies.list()).rejects.toThrow('Internal server error')
    globalThis.fetch = originalFetch
  })

  it('throws on non-ok response without error body', async () => {
    localStorage.clear()
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      json: () => Promise.reject(new Error('not json')),
    })

    const { strategies } = await import('../api/client')
    await expect(strategies.list()).rejects.toThrow('HTTP 404')
    globalThis.fetch = originalFetch
  })
})

describe('API client — POST requests', () => {
  it('sends JSON body with POST', async () => {
    localStorage.clear()
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ id: 'bt-1' }),
    })

    const { backtests } = await import('../api/client')
    await backtests.run({
      symbols: ['AAPL'],
      start_date: '2024-01-01',
      end_date: '2024-06-30',
    })

    expect(globalThis.fetch).toHaveBeenCalledWith(
      '/api/v1/backtests',
      expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('AAPL'),
      }),
    )
    globalThis.fetch = originalFetch
  })
})
