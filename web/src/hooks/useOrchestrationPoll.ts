import { useState, useEffect, useRef, useCallback } from "react"
import { orchestrator } from "../api/client"

interface OrchestrationPollState {
  status: string
  error: string | null
}

export function useOrchestrationPoll(runId: string | null, onComplete?: (id: string) => void) {
  const [state, setState] = useState<OrchestrationPollState>({ status: "idle", error: null })
  const consecutiveErrors = useRef(0)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const onCompleteRef = useRef(onComplete)
  onCompleteRef.current = onComplete

  const stop = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current)
      intervalRef.current = null
    }
  }, [])

  useEffect(() => {
    if (!runId) {
      setState({ status: "idle", error: null })
      return
    }

    setState({ status: "running", error: null })
    consecutiveErrors.current = 0

    const poll = async () => {
      try {
        const r: any = await orchestrator.get(runId)
        if (r?.status === "completed") {
          stop()
          setState({ status: "completed", error: null })
          onCompleteRef.current?.(runId)
          return
        }
        if (r?.status === "failed" || r?.status === "cancelled") {
          stop()
          setState({ status: r.status, error: r.error ?? "Run failed" })
          return
        }
        consecutiveErrors.current = 0
        setState({ status: "running", error: null })
      } catch {
        consecutiveErrors.current++
        if (consecutiveErrors.current >= 3) {
          stop()
          setState({ status: "failed", error: "Polling failed after 3 errors" })
        }
      }
    }

    poll()
    intervalRef.current = setInterval(poll, 2000)

    return () => {
      stop()
    }
  }, [runId, stop])

  return state
}
