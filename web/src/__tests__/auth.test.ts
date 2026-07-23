import { describe, it, expect } from 'vitest'

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
