import { useState, useEffect, useCallback } from 'react'
import { candles } from '../api/client'
import { useWebSocket } from '../hooks/useWebSocket'
import CandlesChart from '../charts/CandlesChart'
import CVDChart from '../charts/CVDChart'
import type { Candle } from '../types/api'
import type { WSTickData, WSCVDData, WSDivergenceData } from '../types/ws'

const RANGES = [
  { label: '1D', value: '1D' },
  { label: '1W', value: '1W' },
  { label: '1M', value: '1M' },
  { label: '3M', value: '3M' },
  { label: '1Y', value: '1Y' },
  { label: 'ALL', value: 'ALL' },
]

export default function MarketDataPage() {
  const [symbol, setSymbol] = useState('SPY')
  const [range, setRange] = useState('1D')
  const [candleData, setCandleData] = useState<Candle[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [ticks, setTicks] = useState<WSTickData[]>([])
  const [cvdHistory, setCvdHistory] = useState<Array<{ time: string; delta: number; buy_volume: number; sell_volume: number }>>([])
  const [latestCvd, setLatestCvd] = useState<WSCVDData | null>(null)
  const [divergence, setDivergence] = useState<WSDivergenceData | null>(null)

  const { connected } = useWebSocket({
    channels: ['ticks', 'cvd', 'divergence'],
    onMessage: (data, channel) => {
      switch (channel) {
        case 'ticks': {
          const tick = data as WSTickData
          setTicks((prev) => [tick, ...prev].slice(0, 100))
          break
        }
        case 'cvd': {
          const cvdData = data as WSCVDData
          setLatestCvd(cvdData)
          setCvdHistory((prev) => {
            const bar = cvdData.bar
            const existing = prev.findIndex((b) => b.time === bar.time)
            if (existing >= 0) {
              const next = [...prev]
              next[existing] = { time: bar.time, delta: bar.delta, buy_volume: bar.buy_volume, sell_volume: bar.sell_volume }
              return next
            }
            return [...prev, { time: bar.time, delta: bar.delta, buy_volume: bar.buy_volume, sell_volume: bar.sell_volume }].slice(-200)
          })
          break
        }
        case 'divergence':
          setDivergence(data as WSDivergenceData)
          break
      }
    },
  })

  const fetchCandles = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await candles.get(symbol, range)
      setCandleData(res.candles ?? [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load candles')
    } finally {
      setLoading(false)
    }
  }, [symbol, range])

  useEffect(() => {
    fetchCandles()
  }, [fetchCandles])

  return (
    <div>
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>Market Data</h1>
        <div className="flex gap-2">
          <span className={`badge ${connected ? 'badge-ok' : 'badge-err'}`}>
            {connected ? 'Live' : 'Offline'}
          </span>
          <input
            className="input"
            style={{ width: 120 }}
            placeholder="Symbol"
            value={symbol}
            onChange={(e) => setSymbol(e.target.value.toUpperCase())}
            onKeyDown={(e) => e.key === 'Enter' && fetchCandles()}
          />
          <select className="input" style={{ width: 80 }} value={range} onChange={(e) => setRange(e.target.value)}>
            {RANGES.map((r) => (
              <option key={r.value} value={r.value}>{r.label}</option>
            ))}
          </select>
          <button className="btn btn-primary" onClick={fetchCandles} disabled={loading}>
            {loading ? 'Loading...' : 'Load'}
          </button>
        </div>
      </div>

      {error && (
        <div className="card mb-4" style={{ background: 'rgba(218,54,51,.1)', border: '1px solid var(--danger)' }}>
          <span style={{ color: 'var(--danger)' }}>{error}</span>
        </div>
      )}

      {candleData.length > 0 && (
        <CandlesChart data={candleData} height={400} title={`${symbol} — ${range}`} />
      )}

      <div className="grid-2 mt-4">
        <div className="card">
          <h2>Recent Ticks</h2>
          {ticks.length === 0 ? (
            <p className="text-muted">Waiting for tick data...</p>
          ) : (
            <div style={{ overflowX: 'auto', maxHeight: 300, overflowY: 'auto' }}>
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Symbol</th>
                    <th>Price</th>
                    <th>Volume</th>
                    <th>Side</th>
                    <th>Time</th>
                  </tr>
                </thead>
                <tbody>
                  {ticks.slice(0, 50).map((t, i) => (
                    <tr key={i}>
                      <td>{t.symbol}</td>
                      <td>${t.price?.toFixed(2)}</td>
                      <td>{t.volume}</td>
                      <td style={{ color: t.side === 'BUY' ? 'var(--success)' : 'var(--danger)' }}>{t.side}</td>
                      <td style={{ fontSize: 11 }}>{t.time ? new Date(t.time).toLocaleTimeString() : '--'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        <div>
          {cvdHistory.length > 0 && (
            <CVDChart data={cvdHistory} height={200} title="CVD" />
          )}

          {latestCvd && (
            <div className="text-muted mt-2" style={{ fontSize: 11 }}>
              {latestCvd.bar.time ? new Date(latestCvd.bar.time).toLocaleString() : ''} &middot;
              O:{latestCvd.bar.open?.toFixed(2)} H:{latestCvd.bar.high?.toFixed(2)} L:{latestCvd.bar.low?.toFixed(2)} C:{latestCvd.bar.close?.toFixed(2)}
            </div>
          )}

          {divergence && (
            <div className="card" style={{ border: '1px solid var(--warn)' }}>
              <h2>Divergence Alert</h2>
              <div style={{ padding: 8, borderRadius: 6, background: 'rgba(210,153,34,.1)' }}>
                <div><strong>Type:</strong> {divergence.type}</div>
                <div><strong>Confidence:</strong> {(divergence.confidence * 100).toFixed(0)}%</div>
                <div className="text-muted">{divergence.time ? new Date(divergence.time).toLocaleString() : ''}</div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
