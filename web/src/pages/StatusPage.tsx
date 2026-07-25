import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import type { SystemHealth } from '../types/api'
import { system } from '../api/client'
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from '../components/ui/card'
import MetricCard from '../components/MetricCard'

export default function StatusPage() {
  const { t } = useTranslation()
  const [health, setHealth] = useState<SystemHealth | null>(null)
  useEffect(()=>{system.health().then(setHealth).catch(()=>{})},[])

  return <div>
    <h1 className="text-2xl font-bold mb-0">{t('status:title', 'System Status')}</h1>
    <Card className="mt-4">
      <CardHeader><CardTitle>{t('status:serviceHealth', 'Service Health')}</CardTitle></CardHeader>
      <CardContent>
        {health ? <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          {Object.entries(health).filter(([,v])=>typeof v!=='object').map(([k,v])=>(
            <MetricCard key={k} label={k.replace(/_/g,' ')} value={String(v)} />
          ))}
        </div> : <CardDescription>{t('status:loading', 'Loading health status...')}</CardDescription>}
      </CardContent>
    </Card>
  </div>
}
