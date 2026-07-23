import { useState } from 'react'
import { useTranslation } from 'react-i18next'

export default function ResetPasswordPage({ onSwitchToLogin }: { onSwitchToLogin: () => void }) {
  const { t } = useTranslation()
  const [pass, setPass] = useState('')
  const [msg, setMsg] = useState('')
  return <div className="auth-page"><div className="auth-card">
    <h1>{t('auth:setNewPassword', 'Set New Password')}</h1>
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      <input className="input" placeholder={t('auth:newPassword', 'New Password')} type="password" value={pass} onChange={e => setPass(e.target.value)} />
      <button className="btn btn-primary" onClick={() => setMsg(t('auth:passwordResetSuccess', 'Password has been reset.'))} style={{ justifyContent: 'center' }}>{t('auth:resetButton', 'Set Password')}</button>
      {msg && <p style={{ color: 'var(--success)', fontSize: 12, margin: 0 }}>{msg}</p>}
      <button className="btn btn-outline" onClick={onSwitchToLogin} style={{ justifyContent: 'center' }}>{t('auth:backToLogin', 'Back to Login')}</button>
    </div>
  </div></div>
}
