import { useMatrixStore } from '../../stores/matrixStore'

/**
 * ChunkTracker surfaces the server's chunk progression (index/total) so users can
 * see large matrices advancing through bounded chunks (execution-framework plan
 * §3.4/§11.3). Hidden until chunk telemetry is available.
 */
export default function ChunkTracker() {
  const status = useMatrixStore((s) => s.status)
  const chunkIndex = useMatrixStore((s) => s.telemetry.chunkIndex)
  const chunkTotal = useMatrixStore((s) => s.telemetry.chunkTotal)

  if (status === 'idle' || chunkTotal <= 1) return null

  return (
    <span className="text-muted" style={{ fontSize: 11 }}>
      chunk <b>{chunkIndex}</b>/<b>{chunkTotal}</b>
    </span>
  )
}
