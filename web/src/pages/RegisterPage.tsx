import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { auth } from '../api/client'

export default function RegisterPage({ onSwitchToLogin }: { onSwitchToLogin: () => void }) {
  const { t } = useTranslation()
  const [user, setUser] = useState('')
  const [pass, setPass] = useState('')
  const [err, setErr] = useState('')
  const [msg, setMsg] = useState('')

  const handleRegister = async () => {
    try {
      await auth.register({ username: user, password: pass })
      setMsg(t('auth:registerSuccess', 'Account created. You can now sign in.'))
    } catch (e) { setErr(e instanceof Error ? e.message : t('auth:registerFailed', 'Registration failed')) }
  }

  return <div className="auth-page"><div className="auth-card">
    <h1>{t('auth:register', 'Register')}</h1>
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      <input className="input" placeholder={t('auth:username', 'Username')} value={user} onChange={e => setUser(e.target.value)} />
      <input className="input" placeholder={t('auth:password', 'Password')} type="password" value={pass} onChange={e => setPass(e.target.value)} onKeyDown={e => e.key === 'Enter' && handleRegister()} />
      {err && <p style={{ color: 'var(--danger)', fontSize: 12, margin: 0 }}>{err}</p>}
      {msg && <p style={{ color: 'var(--success)', fontSize: 12, margin: 0 }}>{msg}</p>}
      <button className="btn btn-primary" onClick={handleRegister} style={{ justifyContent: 'center' }}>{t('auth:registerButton', 'Create Account')}</button>
      <button className="btn btn-outline" onClick={onSwitchToLogin} style={{ justifyContent: 'center' }}>{t('auth:backToLogin', 'Back to Login')}</button>
    </div>
  </div></div>
}
