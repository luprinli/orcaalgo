import { useState } from 'react'

export default function DataSources() {
  const [source, setSource] = useState<'alpaca'|'stooq'|'mock'>('alpaca')
  return <div>
    <h1 style={{margin:0}}>Data Sources</h1>
    <div className="card mt-4"><h2>Market Data Provider</h2>
      <div className="flex gap-2 mt-2">
        {(['alpaca','stooq','mock'] as const).map(s=>
          <button key={s} className={`btn ${source===s?'btn-primary':'btn-outline'}`} onClick={()=>setSource(s)}>{s.charAt(0).toUpperCase()+s.slice(1)}</button>
        )}
      </div>
      <p className="text-muted mt-3">Active source: {source}. Supports equities, forex, and crypto data feeds.</p>
    </div>
  </div>
}
