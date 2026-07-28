import { useEffect, useRef } from 'react'
import toast from 'react-hot-toast'
import { useWebSocket } from './useWebSocket'
import type { WSAlertData } from '../types/ws'

export function useAlertToast() {
  const seenRef = useRef<Set<string>>(new Set())

  useWebSocket({
    channels: 'alerts',
    onMessage: (data) => {
      const alert = data as WSAlertData
      if (!alert?.name) return

      const key = `${alert.name}:${alert.active}`
      if (seenRef.current.has(key)) return
      seenRef.current.add(key)

      if (alert.active) {
        switch (alert.severity) {
          case 'critical':
            toast.error(alert.summary, { duration: 0 })
            break
          case 'warning':
            toast(alert.summary, { icon: '\u26A0', duration: 8000 })
            break
          default:
            toast(alert.summary, { icon: '\u2139', duration: 5000 })
        }
      } else if (alert.resolved_at) {
        toast.success(`${alert.summary} — resolved`, { duration: 5000 })
      }

      setTimeout(() => seenRef.current.delete(key), 60_000)
    },
  })
}
