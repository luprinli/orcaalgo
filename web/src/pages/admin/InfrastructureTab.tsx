import { useState, useEffect } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '../../components/ui/card'
import { Skeleton } from '../../components/ui/skeleton'
import { Button } from '../../components/ui/button'
import { settings } from '../../api/client'

export default function InfrastructureTab() {
  const [grafanaUrl, setGrafanaUrl] = useState('http://localhost:3000')
  const [loaded, setLoaded] = useState(false)
  const [error, setError] = useState(false)

  useEffect(() => {
    settings.get().then((res) => {
      const raw = res as Record<string, unknown>
      const url = (raw.grafana_url ?? raw.settings as Record<string, unknown> | null) as string | undefined
      setGrafanaUrl((url ?? 'http://localhost:3000').replace(/\/+$/, ''))
    }).catch(() => {
      setGrafanaUrl('http://localhost:3000')
    }).finally(() => setLoaded(true))
  }, [])

  const dashboards = [
    { uid: 'orca-broker', title: 'Broker Health' },
    { uid: 'orca-risk', title: 'Risk Status' },
    { uid: 'orca-backtest', title: 'Backtest Execution' },
  ]

  if (!loaded) return <div className="space-y-4"><Skeleton className="h-[500px] w-full rounded-lg" /></div>

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Infrastructure Dashboards</h2>
        <a href={grafanaUrl} target="_blank" rel="noopener noreferrer">
          <Button variant="outline" size="sm">Open Grafana</Button>
        </a>
      </div>
      {dashboards.map(d => (
        <Card key={d.uid}>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm">{d.title}</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {error ? (
              <div className="flex items-center justify-center h-[400px] text-muted-foreground text-sm">
                Grafana dashboard unavailable — ensure Grafana is running on {grafanaUrl}
              </div>
            ) : (
              <iframe
                src={`${grafanaUrl}/d/${d.uid}?kiosk&theme=dark&refresh=10s`}
                className="w-full h-[400px] border-0 rounded-b-lg"
                title={d.title}
                onError={() => setError(true)}
              />
            )}
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
