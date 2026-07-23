import { create } from 'zustand'
import { TIMEFRAME_OPTIONS } from '../hooks/useCandleAggregation'

function loadPersistedTimeframe(): string {
  try {
    const stored = localStorage.getItem('orca_timeframe')
    if (stored && TIMEFRAME_OPTIONS.some(o => o.value === stored)) {
      return stored
    }
  } catch { /* localStorage unavailable */ }
  return 'M1'
}

interface TimeframeState {
  timeframe: string
  setTimeframe: (tf: string) => void
}

export const useTimeframeStore = create<TimeframeState>((set) => ({
  timeframe: loadPersistedTimeframe(),
  setTimeframe: (timeframe) => {
    try { localStorage.setItem('orca_timeframe', timeframe) } catch { /* localStorage unavailable */ }
    set({ timeframe })
  },
}))

export { TIMEFRAME_OPTIONS }
