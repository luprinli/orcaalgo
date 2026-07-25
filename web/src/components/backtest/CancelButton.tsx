import { useState } from 'react'
import { backtests } from '../../api/client'
import { useMatrixStore } from '../../stores/matrixStore'

/**
 * CancelButton stops a running matrix batch (execution-framework plan §4.1/§11.3).
 * Optimistically flips status to `cancelled`; partial results remain visible.
 */
export default function CancelButton({ batchId }: { batchId: string }) {
  const status = useMatrixStore((s) => s.status)
  const setStatus = useMatrixStore((s) => s.setStatus)
  const [busy, setBusy] = useState(false)

  if (status !== 'running' && status !== 'queued') return null

  const onCancel = async () => {
    setBusy(true)
    setStatus('cancelled') // optimistic
    try {
      await backtests.cancelMatrix(batchId)
    } catch {
      // even if the request fails, the poll will reconcile the true status
    } finally {
      setBusy(false)
    }
  }

  return (
    <button
      className="btn btn-outline"
      onClick={onCancel}
      disabled={busy}
      style={{ fontSize: 11, padding: '4px 12px', color: 'var(--trading-danger)', borderColor: 'var(--trading-danger)' }}
    >
      {busy ? 'Cancelling…' : 'Cancel Run'}
    </button>
  )
}
