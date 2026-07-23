import { describe, it, expect } from 'vitest'
import { hydrateAuth, useAuthStore } from '../stores/authStore'

describe('authStore', () => {
  it('setAuth updates token and persists to localStorage', () => {
    const store = useAuthStore.getState()
    store.setAuth('test-token', 'test-refresh', ['user'])

    expect(useAuthStore.getState().token).toBe('test-token')
    expect(useAuthStore.getState().refresh).toBe('test-refresh')
    expect(useAuthStore.getState().roles).toEqual(['user'])
  })

  it('clearAuth resets state and removes localStorage', () => {
    const store = useAuthStore.getState()
    store.setAuth('test-token', 'test-refresh', ['user'])
    store.clearAuth()

    expect(useAuthStore.getState().token).toBeNull()
    expect(useAuthStore.getState().refresh).toBeNull()
    expect(useAuthStore.getState().roles).toEqual([])
  })

  it('isAuthenticated returns correct status', () => {
    useAuthStore.getState().clearAuth()
    expect(useAuthStore.getState().isAuthenticated()).toBe(false)

    useAuthStore.getState().setAuth('tok', 'ref', [])
    expect(useAuthStore.getState().isAuthenticated()).toBe(true)
  })

  it('hydrateAuth reads from localStorage', () => {
    useAuthStore.getState().clearAuth()
    const data = { token: 'stored', refresh: 'ref', roles: ['admin'] }
    localStorage.setItem('orca_auth', JSON.stringify(data))
    hydrateAuth()
    expect(useAuthStore.getState().token).toBe('stored')
    expect(useAuthStore.getState().refresh).toBe('ref')
    expect(useAuthStore.getState().roles).toEqual(['admin'])
  })
})
