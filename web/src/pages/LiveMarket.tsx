import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useWebSocket } from '../hooks/useWebSocket'

export default function LiveMarket() {
  const { t } = useTranslation()
  const [symbol, setSymbol] = useState('SPY')
  const [ticks, setTicks] = useState<{ time: string; bid: number; ask: number; last: number; volume: number }[]>([])
  useWebSocket('ticks', { onMessage: (d: unknown) => {
    const t = d as { symbol?: string; bid?: number; ask?: number; last?: number; volume?: number }
    setTicks(prev => [...prev.slice(-99), { time: new Date().toISOString(), bid: t.bid ?? 0, ask: t.ask ?? 0, last: t.last ?? 0, volume: t.volume ?? 0 }])
  }})

  return <div>
    <div className="flex-between mb-4"><h1 style={{margin:0}}>{t('sidebar:nav.liveMarket', 'Live Market')}</h1>
      <select className="input orca-input" style={{width:150}} value={symbol} onChange={e=>setSymbol(e.target.value)}>
        {['SPY','QQQ','AAPL','MSFT','TSLA','NVDA','BTCUSD'].map(s=><option key={s} value={s}>{s}</option>)}
      </select>
    </div>
    <div className="grid-3 mb-4">
      <div className="metric-card"><div className="metric-label">{t('common:symbol', 'Symbol')}</div><div className="metric-value">{symbol}</div></div>
      <div className="metric-card"><div className="metric-label">{t('liveMarket:ticks', 'Ticks')}</div><div className="metric-value">{ticks.length}</div></div>
      <div className="metric-card"><div className="metric-label">{t('liveMarket:last', 'Last')}</div><div className="metric-value">{ticks.length>0?`$${ticks[ticks.length-1].last.toFixed(2)}`:t('common:noData', '--')}</div></div>
    </div>
    <div className="card">
      <h2>{t('marketData:recentTicks', 'Recent Ticks')}</h2>
      {ticks.length===0?<p className="text-muted">{t('liveMarket:waitingForData', 'Waiting for market data...')}</p>:(
        <table className="data-table"><thead><tr><th>{t('common:time', 'Time')}</th><th>{t('liveMarket:bid', 'Bid')}</th><th>{t('liveMarket:ask', 'Ask')}</th><th>{t('liveMarket:last', 'Last')}</th><th>{t('common:volume', 'Volume')}</th></tr></thead>
          <tbody>{ticks.slice(-10).map((t,i)=><tr key={i}><td style={{fontSize:11}}>{new Date(t.time).toLocaleTimeString()}</td><td>${t.bid.toFixed(2)}</td><td>${t.ask.toFixed(2)}</td><td>${t.last.toFixed(2)}</td><td>{t.volume}</td></tr>)}</tbody>
        </table>
      )}
    </div>
  </div>
}
