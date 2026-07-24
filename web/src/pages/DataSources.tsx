import { useState } from 'react'
import { useTranslation } from 'react-i18next'

export default function DataSources() {
  const { t } = useTranslation()
  const [source, setSource] = useState<string>(() => localStorage.getItem('orca_datasource') || 'alpaca')

  const handleSelect = (s: string) => {
    setSource(s)
    localStorage.setItem('orca_datasource', s)
  }

  return <div>
    <h1 style={{margin:0}}>{t('sidebar:nav.dataSources', 'Data Sources')}</h1>
    <div className="card mt-4"><h2>{t('marketData:title', 'Market Data Provider')}</h2>
      <div className="flex gap-2 mt-2">
        {(['alpaca','stooq','mock'] as const).map(s=>
          <button key={s} className={`btn ${source===s?'btn-primary':'btn-outline'}`} onClick={()=>handleSelect(s)}>{s.charAt(0).toUpperCase()+s.slice(1)}</button>
        )}
      </div>
      <p className="text-muted mt-3">{t('dataSources:activeSource', 'Active source: {{source}}. Supports equities, forex, and crypto data feeds.', { source })}</p>
    </div>
  </div>
}
