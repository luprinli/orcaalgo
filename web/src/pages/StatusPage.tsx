import { useState, useEffect } from 'react'
import { system } from '../api/client'

export default function StatusPage() {
  const [health, setHealth] = useState<Record<string, unknown> | null>(null)
  useEffect(()=>{system.health().then(setHealth).catch(()=>{})},[])

  return <div>
    <h1 style={{margin:0}}>System Status</h1>
    <div className="card mt-4">
      <h2>Service Health</h2>
      {health ? <div className="metric-grid">
        {Object.entries(health).filter(([,v])=>typeof v!=='object').map(([k,v])=>(
          <div key={k} className="metric-card"><div className="metric-label">{k.replace(/_/g,' ')}</div><div className="metric-value" style={{fontSize:16}}>{String(v)}</div></div>
        ))}
      </div> : <p className="text-muted">Loading health status...</p>}
    </div>
  </div>
}
