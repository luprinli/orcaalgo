import { useState } from 'react'
import { useTranslation } from 'react-i18next'

export default function LoginPage({ onLogin }: { onLogin: (t: string) => void }) {
  const { t } = useTranslation()
  const [user, setUser] = useState('')
  const [pass, setPass] = useState('')
  const [err, setErr] = useState('')

  const login = async () => {
    try {
      const r = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: user, password: pass }),
      })
      const d = await r.json()
      if (d.access_token) {
        onLogin(JSON.stringify({ token: d.access_token, username: d.username, expires_at: Date.now() + 86400000 }))
      } else {
        setErr(t('auth:invalidCredentials', 'Invalid credentials'))
      }
    } catch {
      setErr(t('auth:loginFailed', 'Login failed'))
    }
  }

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h1>{t('sidebar:brandName', 'Orca Algo')}</h1>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <input
            className="input"
            placeholder={t('auth:username', 'Username')}
            aria-label={t('auth:username', 'Username')}
            value={user}
            onChange={e => setUser(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && login()}
          />
          <input
            className="input"
            placeholder={t('auth:password', 'Password')}
            type="password"
            aria-label={t('auth:password', 'Password')}
            value={pass}
            onChange={e => setPass(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && login()}
          />
          {err && <p role="alert" style={{ color: 'var(--danger)', fontSize: 12, margin: 0 }}>{err}</p>}
          <button className="btn btn-primary" onClick={login} aria-label={t('auth:loginButton', 'Sign in')} style={{ justifyContent: 'center' }}>
            {t('auth:loginButton', 'Sign In')}
          </button>
        </div>
      </div>
    </div>
  )
}
