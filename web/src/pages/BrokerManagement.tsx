import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import toast from 'react-hot-toast'
import { brokers } from '../api/client'

interface BrokerInfo {
  id: string
  label: string
  status?: string
  connected?: boolean
  last_seen?: string
}

export default function BrokerManagement() {
  const { t } = useTranslation()
  const [brokerList, setBrokerList] = useState<BrokerInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchBrokers = async () => {
    try {
      setLoading(true)
      setError(null)
      const data = await brokers.list()
      const items = ((data as any)?.brokers ?? []) as BrokerInfo[]
      setBrokerList(items)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load brokers')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchBrokers() }, [])

  return (
    <div>
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>{t('sidebar:nav.brokers', 'Broker Management')}</h1>
        <button className="btn btn-outline" onClick={fetchBrokers} disabled={loading}>
          {loading ? 'Loading...' : 'Refresh'}
        </button>
      </div>

      {error && (
        <div className="card mb-4" style={{ borderLeft: '4px solid var(--danger)' }}>
          <p className="text-muted" style={{ color: 'var(--danger-text)' }}>{error}</p>
        </div>
      )}

      {loading && !brokerList.length ? (
        <div className="card"><p className="text-muted">Loading brokers...</p></div>
      ) : brokerList.length === 0 ? (
        <div className="card">
          <h2>{t('brokers:available', 'Available Brokers')}</h2>
          <p className="text-muted">No broker adapters configured. Supported: Alpaca, IBKR, Paper Trading.</p>
        </div>
      ) : (
        <div className="card">
          <h2>{t('brokers:available', 'Available Brokers')}</h2>
          <div style={{ overflowX: 'auto' }}>
            <table>
              <thead>
                <tr>
                  <th>{t('brokers:id', 'ID')}</th>
                  <th>{t('brokers:name', 'Name')}</th>
                  <th>{t('brokers:status', 'Status')}</th>
                  <th>{t('brokers:lastSeen', 'Last Seen')}</th>
                </tr>
              </thead>
              <tbody>
                {brokerList.map(b => (
                  <tr key={b.id}>
                    <td className="font-mono text-xs">{b.id}</td>
                    <td className="font-medium text-white">{b.label}</td>
                    <td>
                      <span className={`badge ${b.connected ? 'badge-ok' : b.status === 'error' ? 'badge-err' : 'badge-warn'}`}>
                        {b.connected ? 'Connected' : b.status || 'Unknown'}
                      </span>
                    </td>
                    <td className="text-muted text-xs">{b.last_seen ? new Date(b.last_seen).toLocaleString() : '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <div className="card mt-4">
        <h2>{t('brokers:support', 'Supported Brokers')}</h2>
        <div className="grid-3 mt-3">
          {[
            { name: 'Paper Trading', desc: 'Simulated fills with configurable latency and slippage', icon: '◧' },
            { name: 'Alpaca Markets', desc: 'Commission-free US equities via Alpaca API', icon: '◧' },
            { name: 'Interactive Brokers', desc: 'Global multi-asset via IBKR TWS/Gateway', icon: '◧' },
          ].map(b => (
            <div key={b.name} className="card" style={{ background: 'var(--bg-secondary)', textAlign: 'center' }}>
              <div style={{ fontSize: 24, marginBottom: 8 }}>{b.icon}</div>
              <div className="font-medium text-white">{b.name}</div>
              <div className="text-muted text-xs mt-1">{b.desc}</div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
