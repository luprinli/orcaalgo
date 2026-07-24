import { useEffect, useRef, useCallback } from 'react'

interface AdaptivePollingOptions {
  minInterval?: number      // minimum poll interval in ms (default: 5000)
  maxInterval?: number      // maximum poll interval in ms (default: 60000)
  idleMultiplier?: number   // multiplier when browser tab is hidden (default: 4)
  activeMultiplier?: number // multiplier during market hours (default: 1)
  backoffFactor?: number    // exponential backoff factor on unchanged data (default: 1.5)
  enabled?: boolean         // whether polling is active (default: true)
}

/**
 * Adaptive polling hook — replaces fixed-interval setInterval patterns.
 *
 * - Slows down when the browser tab is hidden (saves bandwidth)
 * - Speeds up during market hours (configurable)
 * - Backs off exponentially when consecutive responses are unchanged
 * - Automatically cleans up on unmount
 *
 * Example:
 *   useAdaptivePolling(fetchMetrics, {
 *     minInterval: 5000,
 *     maxInterval: 30000,
 *     idleMultiplier: 4,
 *   })
 */
export function useAdaptivePolling(
  fetcher: () => Promise<unknown>,
  options: AdaptivePollingOptions = {}
) {
  const {
    minInterval = 5000,
    maxInterval = 60000,
    idleMultiplier = 4,
    activeMultiplier = 1,
    backoffFactor = 1.5,
    enabled = true,
  } = options

  const intervalRef = useRef(minInterval)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const lastResultRef = useRef<unknown>(undefined)
  const isMarketHoursRef = useRef(true)

  // Detect market hours: 9:30 AM - 4:00 PM ET, Mon-Fri
  const checkMarketHours = useCallback((): boolean => {
    const now = new Date()
    const etOffset = -4 * 60 // EDT offset in minutes (simplified; would need ET timezone for production)
    const utcMinutes = now.getUTCHours() * 60 + now.getUTCMinutes()
    const etMinutes = (utcMinutes + etOffset + 24 * 60) % (24 * 60)

    const day = now.getUTCDay()
    const isWeekday = day >= 1 && day <= 5
    const marketOpen = 9 * 60 + 30 // 9:30 AM
    const marketClose = 16 * 60    // 4:00 PM

    return isWeekday && etMinutes >= marketOpen && etMinutes < marketClose
  }, [])

  const getBaseInterval = useCallback((): number => {
    // Apply idle multiplier when tab is hidden
    const multiplier = document.hidden ? idleMultiplier : activeMultiplier
    // Apply market hours adjustment
    isMarketHoursRef.current = checkMarketHours()
    const marketMult = isMarketHoursRef.current ? 1 : 2
    return minInterval * multiplier * marketMult
  }, [minInterval, idleMultiplier, activeMultiplier, checkMarketHours])

  useEffect(() => {
    if (!enabled) {
      if (timerRef.current) {
        clearTimeout(timerRef.current)
        timerRef.current = null
      }
      return
    }

    let stopped = false

    const poll = async () => {
      if (stopped) return

      try {
        const result = await fetcher()

        if (!stopped) {
          // Compare with last result using JSON serialization
          const currentJson = JSON.stringify(result)
          const lastJson = JSON.stringify(lastResultRef.current)

          if (currentJson === lastJson) {
            // Data unchanged: apply exponential backoff
            intervalRef.current = Math.min(
              intervalRef.current * backoffFactor,
              maxInterval
            )
          } else {
            // Data changed: reset to base interval
            intervalRef.current = getBaseInterval()
            lastResultRef.current = result
          }
        }
      } catch {
        // On error, increase interval to avoid hammering the server
        intervalRef.current = Math.min(intervalRef.current * 2, maxInterval)
      }

      if (!stopped) {
        timerRef.current = setTimeout(poll, intervalRef.current)
      }
    }

    // Initial poll
    intervalRef.current = getBaseInterval()
    poll()

    // Visibility change handler — reset interval when tab becomes visible
    const handleVisibilityChange = () => {
      if (!document.hidden && !stopped && enabled) {
        intervalRef.current = getBaseInterval()
      }
    }
    document.addEventListener('visibilitychange', handleVisibilityChange)

    return () => {
      stopped = true
      if (timerRef.current) {
        clearTimeout(timerRef.current)
        timerRef.current = null
      }
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
  }, [fetcher, enabled, maxInterval, backoffFactor, getBaseInterval])

  // Expose manual reset for imperative use
  const reset = useCallback(() => {
    intervalRef.current = getBaseInterval()
    lastResultRef.current = undefined
  }, [getBaseInterval])

  const isMarketHours = isMarketHoursRef.current

  return { reset, isMarketHours }
}
