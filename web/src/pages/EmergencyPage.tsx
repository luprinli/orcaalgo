import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useLiveRiskData } from '../hooks/useLiveRiskData'
import { risk } from '../api/client'

export default function EmergencyPage() {
  const { t } = useTranslation()
  const { riskData, isHalted } = useLiveRiskData()
  const [code, setCode] = useState('')
  const [confirming, setConfirming] = useState(false)
  const [action, setAction] = useState<'stop' | 'resume' | null>(null)
  const [msg, setMsg] = useState('')
  const [loading, setLoading] = useState(false)

  const handleAction = async () => {
    if (code.length !== 6) return
    setLoading(true)
    setMsg('')
    try {
      if (action === 'stop') {
        await risk.emergencyStop(code)
        setMsg('✅ Emergency stop activated. All positions are being closed.')
      } else if (action === 'resume') {
        await risk.emergencyResume(code)
        setMsg('✅ Trading resumed.')
      }
      setConfirming(false)
      setCode('')
      setAction(null)
    } catch (err) {
      setMsg(`❌ Failed: ${err instanceof Error ? err.message : 'Unknown error'}`)
    } finally {
      setLoading(false)
    }
  }

  const formatCurrency = (n?: number) => {
    if (n == null) return '—'
    if (Math.abs(n) >= 1e6) return '$' + (n / 1e6).toFixed(2) + 'M'
    if (Math.abs(n) >= 1e3) return '$' + (n / 1e3).toFixed(1) + 'K'
    return '$' + n.toFixed(2)
  }

  return (
    <div className="min-h-screen bg-gray-900 text-white p-4" style={{ fontFamily: '-apple-system, BlinkMacSystemFont, sans-serif' }}>
      <div className="max-w-md mx-auto">
        <h1 className="text-lg font-bold mb-1">{t('emergency.title', 'Emergency Access')}</h1>
        <p className="text-xs text-slate-400 mb-4">{t('emergency.subtitle', 'Quick kill-switch access on mobile')}</p>

        {/* Live Status */}
        {riskData && (
          <div className="space-y-2 mb-6">
            <div className="flex justify-between text-sm">
              <span className="text-slate-400">Balance</span>
              <span className="font-medium">{formatCurrency(riskData.balance)}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-slate-400">Daily PnL</span>
              <span className={`font-medium ${(riskData.daily_pnl_pct ?? 0) >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                {(riskData.daily_pnl_pct ?? 0) >= 0 ? '+' : ''}{(riskData.daily_pnl_pct ?? 0).toFixed(2)}%
              </span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-slate-400">Drawdown Used</span>
              <span className="font-medium text-yellow-400">{(riskData.drawdown_used ?? 0).toFixed(1)}%</span>
            </div>
            <div className="mt-2">
              <div className="h-2 bg-slate-700 rounded-full overflow-hidden">
                <div className="h-full rounded-full transition-all" style={{
                  width: `${Math.min((riskData.drawdown_used ?? 0), 100)}%`,
                  backgroundColor: (riskData.drawdown_used ?? 0) > 70 ? '#ef4444' : '#f59e0b',
                }} />
              </div>
              <div className="flex justify-between text-xs text-slate-500 mt-1">
                <span>0%</span>
                <span>{(riskData.max_dd_pct ?? 20)}% limit</span>
              </div>
            </div>
          </div>
        )}

        {/* Status */}
        {isHalted ? (
          <div className="bg-red-900/40 border border-red-700 rounded-lg p-4 mb-4 text-center">
            <div className="text-red-400 font-bold text-lg mb-1">⚠️ TRADING HALTED</div>
            <p className="text-red-300 text-xs">{riskData?.reason || 'Emergency stop was activated'}</p>
          </div>
        ) : (
          <div className="bg-green-900/20 border border-green-700/50 rounded-lg p-3 mb-4 text-center">
            <span className="text-green-400 text-sm font-medium">● Trading Active</span>
          </div>
        )}

        {/* Message */}
        {msg && (
          <div className={`rounded-lg p-3 mb-4 text-sm ${msg.includes('✅') ? 'bg-green-900/30 border border-green-700 text-green-300' : 'bg-red-900/30 border border-red-700 text-red-300'}`}>
            {msg}
          </div>
        )}

        {/* Confirmation Dialog */}
        {confirming ? (
          <div className="border border-red-700 rounded-lg p-4">
            <p className="text-sm mb-3 text-slate-300">
              {action === 'stop' ? 'Enter 2FA code to confirm EMERGENCY STOP:' : 'Enter 2FA code to confirm resume:'}
            </p>
            <input
              type="text" inputMode="numeric" maxLength={6} autoFocus
              value={code} onChange={e => setCode(e.target.value.replace(/\D/g, ''))}
              className="w-full bg-gray-800 border border-gray-600 rounded-lg p-3 text-center text-2xl tracking-widest font-mono text-white outline-none focus:border-red-500"
              placeholder="000000"
            />
            <button
              className="w-full bg-red-700 hover:bg-red-800 disabled:bg-gray-700 text-white font-bold py-3 rounded-lg mt-3 transition-colors"
              disabled={code.length !== 6 || loading}
              onClick={handleAction}
            >
              {loading ? 'Processing...' : action === 'stop' ? 'Confirm Emergency Stop' : 'Confirm Resume'}
            </button>
            <button
              className="w-full bg-transparent border border-gray-600 text-slate-400 py-2 rounded-lg mt-2 text-sm"
              onClick={() => { setConfirming(false); setCode(''); setAction(null); setMsg('') }}
            >
              Cancel
            </button>
          </div>
        ) : (
          <div className="space-y-2">
            {!isHalted && (
              <button
                className="w-full bg-red-600 hover:bg-red-700 text-white font-bold py-4 rounded-lg transition-colors"
                onClick={() => { setConfirming(true); setAction('stop'); setMsg('') }}
              >
                🛑 Emergency Stop All Trading
              </button>
            )}
            {isHalted && (
              <button
                className="w-full bg-green-700 hover:bg-green-800 text-white font-bold py-4 rounded-lg transition-colors"
                onClick={() => { setConfirming(true); setAction('resume'); setMsg('') }}
              >
                ▶️ Resume Trading
              </button>
            )}
            <a
              href="/"
              className="block w-full text-center bg-gray-800 hover:bg-gray-700 text-slate-300 py-3 rounded-lg text-sm transition-colors"
            >
              ← Back to Dashboard
            </a>
          </div>
        )}
      </div>
    </div>
  )
}
