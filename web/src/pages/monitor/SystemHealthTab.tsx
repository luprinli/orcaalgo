import { useState, useEffect, useCallback } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '../../components/ui/card'
import { Skeleton } from '../../components/ui/skeleton'
import MetricCard from '../../components/MetricCard'

interface PrometheusMetric {
  metric: Record<string, string>
  value: [number, string]
}

interface PrometheusResponse {
  status: string
  data?: { resultType: string; result: PrometheusMetric[] }
}

async function queryPrometheus(query: string): Promise<string> {
  try {
    const res = await fetch(`/api/v1/monitoring/prometheus/query?query=${encodeURIComponent(query)}`)
    const data: PrometheusResponse = await res.json()
    if (data.status !== 'success' || !data.data?.result?.length) return '--'
    return data.data.result[0].value[1]
  } catch {
    return '--'
  }
}

function useMetric(key: string, query: string) {
  const [value, setValue] = useState('--')
  useEffect(() => {
    const fn = () => queryPrometheus(query).then(setValue)
    fn()
    const id = setInterval(fn, 10_000)
    return () => clearInterval(id)
  }, [key, query])
  return value
}

export default function SystemHealthTab() {
  const engineLatencyP95 = useMetric('latency', 'histogram_quantile(0.95, rate(orca_engine_latency_us_bucket[5m]))')
  const rejectRate = useMetric('reject', 'rate(orca_reject_count_total[5m])')
  const overflowCount = useMetric('overflow', 'orca_ring_buffer_overflow_total')
  const brokerStatus = useMetric('broker', 'orca_broker_connected')
  const activeWorkers = useMetric('workers', 'orca_matrix_active_workers')
  const wssConns = useMetric('ws', 'orca_ws_connections')
  const heapMB = useMetric('heap', 'orca_heap_inuse_bytes / 1024 / 1024')
  const dbPool = useMetric('dbpool', 'orca_db_pool_in_use')

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader className="pb-2"><CardTitle className="text-sm">Infrastructure</CardTitle></CardHeader>
        <CardContent>
          <div className="grid grid-cols-4 gap-2">
            <MetricCard label="Engine Latency P95" value={Number(engineLatencyP95).toFixed(0) + '\u00b5s'} format="number" color={Number(engineLatencyP95) > 100 ? 'negative' : 'positive'} />
            <MetricCard label="Reject Rate (5m)" value={Number(rejectRate).toFixed(2) + '/s'} format="number" />
            <MetricCard label="Broker" value={brokerStatus === '1' ? 'Connected' : 'Disconnected'} format="number" color={brokerStatus === '1' ? 'positive' : 'negative'} />
            <MetricCard label="WS Connections" value={Number(wssConns)} format="number" />
            <MetricCard label="Active Workers" value={Number(activeWorkers)} format="number" />
            <MetricCard label="Ring Overflow" value={Number(overflowCount)} format="number" color={Number(overflowCount) > 0 ? 'negative' : 'positive'} />
            <MetricCard label="Heap (MB)" value={Number(heapMB).toFixed(0)} format="number" />
            <MetricCard label="DB Pool Used" value={Number(dbPool)} format="number" />
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
