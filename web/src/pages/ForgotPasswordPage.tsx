import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { auth } from '../api/client'

export default function ForgotPasswordPage({ onSwitchToLogin }: { onSwitchToLogin: () => void }) {
  const { t } = useTranslation()
  const [email, setEmail] = useState('')
  const [loading, setLoading] = useState(false)
  const [msg, setMsg] = useState('')
  const [error, setError] = useState('')

  const handleSubmit = async () => {
    if (!email.trim()) return
    setLoading(true)
    setError('')
    setMsg('')
    try {
      await auth.forgotPassword(email.trim())
      setMsg(t('auth:resetLinkSent', 'If the email exists, a reset link has been sent.'))
    } catch (e) {
      setError(e instanceof Error ? e.message : t('auth:genericError', 'An error occurred.'))
    } finally {
      setLoading(false)
    }
  }

  return <div className="auth-page"><div className="auth-card">
    <h1>{t('auth:forgotPassword', 'Reset Password')}</h1>
    <p className="text-muted">{t('auth:resetDescription', 'Enter your email to receive a reset link.')}</p>
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      <input className="input" placeholder={t('auth:email', 'Email')} value={email} onChange={e => setEmail(e.target.value)} />
      <button className="btn btn-primary" onClick={handleSubmit} disabled={loading} style={{ justifyContent: 'center' }}>{loading ? t('auth:sending', 'Sending...') : t('auth:sendResetLink', 'Send Reset Link')}</button>
      {msg && <p style={{ color: 'var(--success)', fontSize: 12, margin: 0 }}>{msg}</p>}
      {error && <p style={{ color: 'var(--danger)', fontSize: 12, margin: 0 }}>{error}</p>}
      <button className="btn btn-outline" onClick={onSwitchToLogin} style={{ justifyContent: 'center' }}>{t('auth:backToLogin', 'Back to Login')}</button>
    </div>
  </div></div>
}
