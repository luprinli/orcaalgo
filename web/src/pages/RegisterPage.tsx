import { useState } from 'react'
import { auth } from '../api/client'

export default function RegisterPage({ onSwitchToLogin }: { onSwitchToLogin: () => void }) {
  const [user, setUser] = useState('')
  const [pass, setPass] = useState('')
  const [err, setErr] = useState('')
  const [msg, setMsg] = useState('')

  const handleRegister = async () => {
    try {
      await auth.register({ username: user, password: pass })
      setMsg('Account created. You can now sign in.')
    } catch (e) { setErr(e instanceof Error ? e.message : 'Registration failed') }
  }

  return <div className="auth-page"><div className="auth-card">
    <h1>Register</h1>
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      <input className="input" placeholder="Username" value={user} onChange={e => setUser(e.target.value)} />
      <input className="input" placeholder="Password" type="password" value={pass} onChange={e => setPass(e.target.value)} onKeyDown={e => e.key === 'Enter' && handleRegister()} />
      {err && <p style={{ color: 'var(--danger)', fontSize: 12, margin: 0 }}>{err}</p>}
      {msg && <p style={{ color: 'var(--success)', fontSize: 12, margin: 0 }}>{msg}</p>}
      <button className="btn btn-primary" onClick={handleRegister} style={{ justifyContent: 'center' }}>Create Account</button>
      <button className="btn btn-outline" onClick={onSwitchToLogin} style={{ justifyContent: 'center' }}>Back to Login</button>
    </div>
  </div></div>
}
