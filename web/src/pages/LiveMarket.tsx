import { useState } from 'react'
import { useWebSocket } from '../hooks/useWebSocket'

export default function LiveMarket() {
  const [symbol, setSymbol] = useState('SPY')
  const [ticks, setTicks] = useState<{ time: string; bid: number; ask: number; last: number; volume: number }[]>([])
  useWebSocket('ticks', { onMessage: (d: unknown) => {
    const t = d as { symbol?: string; bid?: number; ask?: number; last?: number; volume?: number }
    setTicks(prev => [...prev.slice(-99), { time: new Date().toISOString(), bid: t.bid ?? 0, ask: t.ask ?? 0, last: t.last ?? 0, volume: t.volume ?? 0 }])
  }})

  return <div>
    <div className="flex-between mb-4"><h1 style={{margin:0}}>Live Market</h1>
      <select className="input orca-input" style={{width:150}} value={symbol} onChange={e=>setSymbol(e.target.value)}>
        {['SPY','QQQ','AAPL','MSFT','TSLA','NVDA','BTCUSD'].map(s=><option key={s} value={s}>{s}</option>)}
      </select>
    </div>
    <div className="grid-3 mb-4">
      <div className="metric-card"><div className="metric-label">Symbol</div><div className="metric-value">{symbol}</div></div>
      <div className="metric-card"><div className="metric-label">Ticks</div><div className="metric-value">{ticks.length}</div></div>
      <div className="metric-card"><div className="metric-label">Last</div><div className="metric-value">{ticks.length>0?`$${ticks[ticks.length-1].last.toFixed(2)}`:'--'}</div></div>
    </div>
    <div className="card">
      <h2>Recent Ticks</h2>
      {ticks.length===0?<p className="text-muted">Waiting for market data...</p>:(
        <table className="data-table"><thead><tr><th>Time</th><th>Bid</th><th>Ask</th><th>Last</th><th>Volume</th></tr></thead>
          <tbody>{ticks.slice(-10).map((t,i)=><tr key={i}><td style={{fontSize:11}}>{new Date(t.time).toLocaleTimeString()}</td><td>${t.bid.toFixed(2)}</td><td>${t.ask.toFixed(2)}</td><td>${t.last.toFixed(2)}</td><td>{t.volume}</td></tr>)}</tbody>
        </table>
      )}
    </div>
  </div>
}
