import { describe, it, expect, beforeEach } from 'vitest'
import { getRequestHeaders, markRequestComplete } from '../api/middleware'

describe('localStorage auth helpers', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('stores and retrieves auth data', () => {
    const authData = { token: 'abc', refresh: 'def', roles: ['admin'] }
    localStorage.setItem('orca_auth', JSON.stringify(authData))
    const raw = localStorage.getItem('orca_auth')
    expect(raw).not.toBeNull()
    const parsed = JSON.parse(raw!)
    expect(parsed.token).toBe('abc')
    expect(parsed.refresh).toBe('def')
    expect(parsed.roles).toEqual(['admin'])
  })

  it('removes auth data on clear', () => {
    localStorage.setItem('orca_auth', JSON.stringify({ token: 'abc', refresh: 'def', roles: [] }))
    localStorage.removeItem('orca_auth')
    expect(localStorage.getItem('orca_auth')).toBeNull()
  })

  it('returns null for missing token', () => {
    expect(localStorage.getItem('orca_auth')).toBeNull()
  })
})

describe('token extraction from localStorage (getAuth / getToken)', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('extracts token from JSON object', async () => {
    localStorage.setItem('orca_auth', JSON.stringify({ token: 'abc123', refresh: 'def', roles: ['admin'] }))
    const { getToken } = await import('../api/client')
    expect(getToken()).toBe('abc123')
  })

  it('extracts token from raw string', async () => {
    localStorage.setItem('orca_auth', 'raw-jwt-token-string')
    const { getToken } = await import('../api/client')
    expect(getToken()).toBe('raw-jwt-token-string')
  })

  it('returns null for corrupted localStorage', async () => {
    localStorage.setItem('orca_auth', '{broken')
    const { getToken } = await import('../api/client')
    expect(getToken()).toBeNull()
  })

  it('returns null for empty/missing key', async () => {
    const { getToken } = await import('../api/client')
    expect(getToken()).toBeNull()
  })

  it('getAuth returns null when key is missing', async () => {
    const { getAuth } = await import('../api/client')
    expect(getAuth()).toBeNull()
  })

  it('getAuth parses full JSON object', async () => {
    localStorage.setItem('orca_auth', JSON.stringify({ token: 'tok', refresh: 'ref', roles: ['admin'] }))
    const { getAuth } = await import('../api/client')
    const auth = getAuth()
    expect(auth).not.toBeNull()
    expect(auth!.token).toBe('tok')
    expect(auth!.refresh).toBe('ref')
    expect(auth!.roles).toEqual(['admin'])
  })

  it('getAuth wraps raw string into AuthData shape', async () => {
    localStorage.setItem('orca_auth', 'bare-jwt')
    const { getAuth } = await import('../api/client')
    const auth = getAuth()
    expect(auth).not.toBeNull()
    expect(auth!.token).toBe('bare-jwt')
    expect(auth!.refresh).toBe('')
    expect(auth!.roles).toEqual([])
  })
})

describe('getRequestHeaders', () => {
  it('includes Authorization header when token is provided', () => {
    const headers = getRequestHeaders('abc123')
    expect(headers['Authorization']).toBe('Bearer abc123')
  })

  it('always includes X-Request-ID', () => {
    const headers = getRequestHeaders('abc123')
    expect(headers['X-Request-ID']).toBeTruthy()
    expect(headers['X-Request-ID']).toMatch(/^orca-ui-/)
  })

  it('does NOT include Authorization when token is null', () => {
    const headers = getRequestHeaders(null)
    expect(headers['Authorization']).toBeUndefined()
  })

  it('does NOT include Authorization when no token argument is passed', () => {
    const headers = getRequestHeaders()
    expect(headers['Authorization']).toBeUndefined()
  })

  it('does NOT include Authorization when token is empty string', () => {
    const headers = getRequestHeaders('')
    expect(headers['Authorization']).toBeUndefined()
  })
})

describe('markRequestComplete', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('is a function', () => {
    expect(typeof markRequestComplete).toBe('function')
  })

  it('does not throw when called with no arguments', () => {
    expect(() => markRequestComplete()).not.toThrow()
  })

  it('clears auth from localStorage on 401 status', () => {
    localStorage.setItem('orca_auth', JSON.stringify({ token: 'abc', refresh: 'def', roles: [] }))
    markRequestComplete(401)
    expect(localStorage.getItem('orca_auth')).toBeNull()
  })

  it('does NOT clear auth on non-401 status', () => {
    localStorage.setItem('orca_auth', JSON.stringify({ token: 'abc', refresh: 'def', roles: [] }))
    markRequestComplete(200)
    expect(localStorage.getItem('orca_auth')).not.toBeNull()
  })

  it('does NOT clear auth when called without status', () => {
    localStorage.setItem('orca_auth', JSON.stringify({ token: 'abc', refresh: 'def', roles: [] }))
    markRequestComplete()
    expect(localStorage.getItem('orca_auth')).not.toBeNull()
  })
})

describe('auth flow integration', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('setAuth stores and getToken retrieves', async () => {
    const { setAuth, getToken } = await import('../api/client')
    setAuth({ token: 'tok1', refresh: 'ref1', roles: ['admin'] })
    expect(getToken()).toBe('tok1')
  })

  it('clearAuth removes stored token', async () => {
    const { setAuth, clearAuth, getToken } = await import('../api/client')
    setAuth({ token: 'tok1', refresh: 'ref1', roles: ['admin'] })
    clearAuth()
    expect(getToken()).toBeNull()
  })

  it('header pair: setAuth + getRequestHeaders(token) includes Bearer', async () => {
    const { setAuth, getToken } = await import('../api/client')
    setAuth({ token: 'int-token', refresh: 'ref', roles: ['user'] })
    const headers = getRequestHeaders(getToken())
    expect(headers['Authorization']).toBe('Bearer int-token')
  })
})
