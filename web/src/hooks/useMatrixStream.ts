import { useEffect, useRef } from 'react'
import { backtests } from '../api/client'
import { useMatrixStore } from '../stores/matrixStore'

const TERMINAL = new Set(['completed', 'failed', 'cancelled'])

/**
 * useMatrixStream drives incremental matrix result streaming via the `?since=`
 * cursor endpoint — appending deltas into matrixStore instead of re-fetching the
 * whole array every poll (execution-framework plan §5.1/§11.3). The interval is a
 * safety-net cadence; the store upserts by combo key so out-of-order/duplicate
 * deltas are idempotent. Returns nothing; components read matrixStore slices.
 *
 * WS integration (backtest_progress) can later feed the same `applyDelta`; the
 * cursor path remains the reconciliation fallback.
 */
export function useMatrixStream(batchId: string | null, intervalMs = 1500) {
  const applyDelta = useMatrixStore((s) => s.applyDelta)
  const setStatus = useMatrixStore((s) => s.setStatus)
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const inFlight = useRef(false)
  const errCount = useRef(0)

  useEffect(() => {
    if (!batchId) return
    let cancelled = false
    errCount.current = 0

    const tick = async () => {
      if (inFlight.current) return
      inFlight.current = true
      try {
        const seq = useMatrixStore.getState().seq
        const resp = await backtests.matrixResultsSince(batchId, seq)
        if (cancelled) return
        errCount.current = 0 // reset on success
        applyDelta(resp)
        const status = resp.summary?.status ?? resp.status
        if (status && TERMINAL.has(status)) {
          if (timerRef.current) { clearInterval(timerRef.current); timerRef.current = null }
        }
      } catch {
        errCount.current++
        // Three consecutive failures (e.g. 404 for a stale batch, server down)
        // stop the stream rather than polling forever. The resume-on-refresh tab
        // clears when the batch URL is removed.
        if (errCount.current >= 3) {
          if (timerRef.current) { clearInterval(timerRef.current); timerRef.current = null }
          setStatus('failed')
        }
      } finally {
        inFlight.current = false
      }
    }

    void tick()
    timerRef.current = setInterval(tick, intervalMs)

    return () => {
      cancelled = true
      if (timerRef.current) { clearInterval(timerRef.current); timerRef.current = null }
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [batchId, intervalMs])

  return { setStatus }
}
