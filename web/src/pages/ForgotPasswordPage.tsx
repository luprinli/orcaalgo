import { useState } from 'react'
import { useTranslation } from 'react-i18next'

export default function ForgotPasswordPage({ onSwitchToLogin }: { onSwitchToLogin: () => void }) {
  const { t } = useTranslation()
  const [email, setEmail] = useState('')
  const [msg, setMsg] = useState('')
  return <div className="auth-page"><div className="auth-card">
    <h1>{t('auth:forgotPassword', 'Reset Password')}</h1>
    <p className="text-muted">{t('auth:resetDescription', 'Enter your email to receive a reset link.')}</p>
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      <input className="input" placeholder={t('auth:email', 'Email')} value={email} onChange={e => setEmail(e.target.value)} />
      <button className="btn btn-primary" onClick={() => setMsg(t('auth:resetLinkSent', 'If the email exists, a reset link has been sent.'))} style={{ justifyContent: 'center' }}>{t('auth:sendResetLink', 'Send Reset Link')}</button>
      {msg && <p style={{ color: 'var(--success)', fontSize: 12, margin: 0 }}>{msg}</p>}
      <button className="btn btn-outline" onClick={onSwitchToLogin} style={{ justifyContent: 'center' }}>{t('auth:backToLogin', 'Back to Login')}</button>
    </div>
  </div></div>
}
