import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import type { SystemHealth } from '../types/api'
import { system } from '../api/client'

export default function StatusPage() {
  const { t } = useTranslation()
  const [health, setHealth] = useState<SystemHealth | null>(null)
  useEffect(()=>{system.health().then(setHealth).catch(()=>{})},[])

  return <div>
    <h1 style={{margin:0}}>{t('status:title', 'System Status')}</h1>
    <div className="card mt-4">
      <h2>{t('status:serviceHealth', 'Service Health')}</h2>
      {health ? <div className="metric-grid">
        {Object.entries(health).filter(([,v])=>typeof v!=='object').map(([k,v])=>(
          <div key={k} className="metric-card"><div className="metric-label">{k.replace(/_/g,' ')}</div><div className="metric-value" style={{fontSize:16}}>{String(v)}</div></div>
        ))}
      </div> : <p className="text-muted">{t('status:loading', 'Loading health status...')}</p>}
    </div>
  </div>
}
