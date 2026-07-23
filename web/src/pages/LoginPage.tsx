import { useState } from 'react'

export default function LoginPage({ onLogin }: { onLogin: (t: string) => void }) {
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
        onLogin(d.access_token)
      } else {
        setErr('Invalid credentials')
      }
    } catch {
      setErr('Login failed')
    }
  }

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h1>Orca Algo</h1>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <input
            className="input"
            placeholder="Username"
            aria-label="Username"
            value={user}
            onChange={e => setUser(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && login()}
          />
          <input
            className="input"
            placeholder="Password"
            type="password"
            aria-label="Password"
            value={pass}
            onChange={e => setPass(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && login()}
          />
          {err && <p role="alert" style={{ color: 'var(--danger)', fontSize: 12, margin: 0 }}>{err}</p>}
          <button className="btn btn-primary" onClick={login} aria-label="Sign in" style={{ justifyContent: 'center' }}>
            Sign In
          </button>
        </div>
      </div>
    </div>
  )
}
