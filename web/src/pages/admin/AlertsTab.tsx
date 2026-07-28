import { useState, useEffect, useCallback } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '../../components/ui/card'
import { Button } from '../../components/ui/button'
import { Badge } from '../../components/ui/badge'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '../../components/ui/select'
import { Skeleton } from '../../components/ui/skeleton'
import ErrorCard from '../../components/ErrorCard'

interface AlertmanagerAlert {
  fingerprint: string
  labels: Record<string, string>
  annotations: { summary?: string; description?: string }
  startsAt: string
  status: { state: string }
}

const SILENCE_DURATIONS = [
  { label: '1 hour', value: '1h' },
  { label: '4 hours', value: '4h' },
  { label: '1 day', value: '1d' },
  { label: '1 week', value: '1w' },
]

function severityVariant(severity: string): 'destructive' | 'warning' | 'default' {
  if (severity === 'critical') return 'destructive'
  if (severity === 'warning') return 'warning'
  return 'default'
}

export default function AlertsTab() {
  const [alerts, setAlerts] = useState<AlertmanagerAlert[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [silenceDuration, setSilenceDuration] = useState('1h')

  const fetchAlerts = useCallback(async () => {
    try {
      const res = await fetch('/api/v1/monitoring/alerts')
      setAlerts(await res.json())
      setError(false)
    } catch {
      setError(true)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchAlerts()
    const id = setInterval(fetchAlerts, 15_000)
    return () => clearInterval(id)
  }, [fetchAlerts])

  const silence = async (alertName: string) => {
    await fetch('/api/v1/monitoring/alerts/silence', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ alert_name: alertName, duration: silenceDuration, comment: 'Silenced via Orca dashboard' }),
    })
    fetchAlerts()
  }

  if (loading) return <div className="space-y-2">{Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-16 rounded-lg" />)}</div>
  if (error) return <ErrorCard message="Alertmanager unreachable — ensure monitoring services are running" />

  const active = alerts.filter((a: AlertmanagerAlert) => a.status.state === 'active')

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <h2 className="text-lg font-semibold">Active Alerts</h2>
        <Badge variant={active.length > 0 ? 'destructive' : 'default'}>{active.length} active</Badge>
        <div className="flex items-center gap-2 ml-auto">
          <Select value={silenceDuration} onValueChange={setSilenceDuration}>
            <SelectTrigger className="w-[100px]"><SelectValue /></SelectTrigger>
            <SelectContent>
              {SILENCE_DURATIONS.map((s: typeof SILENCE_DURATIONS[number]) => <SelectItem key={s.value} value={s.value}>{s.label}</SelectItem>)}
            </SelectContent>
          </Select>
        </div>
      </div>
      {active.length === 0 && (
        <Card><CardContent className="py-8 text-center text-muted-foreground">No active alerts</CardContent></Card>
      )}
      {active.map((a: AlertmanagerAlert) => {
        const severity = a.labels?.severity ?? 'info'
        return (
          <Card key={a.fingerprint}>
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Badge variant={severityVariant(severity)} size="sm">{severity}</Badge>
                  <CardTitle className="text-sm">{a.annotations?.summary ?? a.labels?.alertname ?? 'Unknown'}</CardTitle>
                </div>
                <Button variant="outline" size="sm" onClick={() => silence(a.labels?.alertname ?? a.fingerprint)}>
                  Silence {silenceDuration}
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-xs text-muted-foreground">{a.annotations?.description ?? 'No description'}</p>
              <p className="text-[10px] text-muted-foreground mt-1">Since: {new Date(a.startsAt).toLocaleString()}</p>
            </CardContent>
          </Card>
        )
      })}
    </div>
  )
}
