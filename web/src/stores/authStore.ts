import { create } from 'zustand'

interface AuthState {
  token: string | null
  refresh: string | null
  roles: string[]
  setAuth: (token: string, refresh: string, roles: string[]) => void
  clearAuth: () => void
  isAuthenticated: () => boolean
}

export const useAuthStore = create<AuthState>((set, get) => ({
  token: null,
  refresh: null,
  roles: [],
  setAuth: (token, refresh, roles) => {
    const data = { token, refresh, roles }
    localStorage.setItem('orca_auth', JSON.stringify(data))
    set(data)
  },
  clearAuth: () => {
    localStorage.removeItem('orca_auth')
    set({ token: null, refresh: null, roles: [] })
  },
  isAuthenticated: () => get().token !== null,
}))

export function hydrateAuth() {
  try {
    const raw = localStorage.getItem('orca_auth')
    if (!raw) return
    const parsed = JSON.parse(raw)
    if (parsed.token) {
      useAuthStore.getState().setAuth(parsed.token, parsed.refresh || '', parsed.roles || [])
    }
  } catch {
    localStorage.removeItem('orca_auth')
  }
}
