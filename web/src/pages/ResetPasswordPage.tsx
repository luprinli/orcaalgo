import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { auth } from '../api/client'

export default function ResetPasswordPage({ onSwitchToLogin }: { onSwitchToLogin: () => void }) {
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') ?? ''
  const [pass, setPass] = useState('')
  const [loading, setLoading] = useState(false)
  const [msg, setMsg] = useState('')
  const [error, setError] = useState('')

  const handleSubmit = async () => {
    if (!pass.trim() || !token) return
    setLoading(true)
    setError('')
    setMsg('')
    try {
      await auth.resetPassword(token, pass)
      setMsg(t('auth:passwordResetSuccess', 'Password has been reset.'))
    } catch (e) {
      setError(e instanceof Error ? e.message : t('auth:genericError', 'An error occurred.'))
    } finally {
      setLoading(false)
    }
  }

  return <div className="auth-page"><div className="auth-card">
    <h1>{t('auth:setNewPassword', 'Set New Password')}</h1>
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      <input className="input" placeholder={t('auth:newPassword', 'New Password')} type="password" value={pass} onChange={e => setPass(e.target.value)} />
      <button className="btn btn-primary" onClick={handleSubmit} disabled={loading || !token} style={{ justifyContent: 'center' }}>{loading ? t('auth:resetting', 'Resetting...') : t('auth:resetButton', 'Set Password')}</button>
      {msg && <p style={{ color: 'var(--success)', fontSize: 12, margin: 0 }}>{msg}</p>}
      {error && <p style={{ color: 'var(--danger)', fontSize: 12, margin: 0 }}>{error}</p>}
      <button className="btn btn-outline" onClick={onSwitchToLogin} style={{ justifyContent: 'center' }}>{t('auth:backToLogin', 'Back to Login')}</button>
    </div>
  </div></div>
}
