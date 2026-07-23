import { create } from 'zustand'

export type ToastType = 'success' | 'error' | 'warn' | 'info'
export type ToastPosition = 'top-right' | 'bottom-right'

export interface Toast {
  id: string
  type: ToastType
  message: string
  duration?: number
  position?: ToastPosition
}

interface ToastState {
  toasts: Toast[]
  addToast: (toast: Omit<Toast, 'id'>) => string
  removeToast: (id: string) => void
}

let nextId = 0

export const useToastStore = create<ToastState>((set) => ({
  toasts: [],

  addToast: (toast) => {
    const id = `toast_${++nextId}_${Date.now()}`
    const t: Toast = { ...toast, id }
    set((s) => ({ toasts: [...s.toasts, t] }))

    const duration = toast.duration ?? 4000
    if (duration > 0) {
      setTimeout(() => {
        set((s) => ({ toasts: s.toasts.filter((x) => x.id !== id) }))
      }, duration)
    }

    return id
  },

  removeToast: (id) => {
    set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) }))
  },
}))

export function showToast(type: ToastType, message: string, duration = 4000) {
  return useToastStore.getState().addToast({ type, message, duration })
}
