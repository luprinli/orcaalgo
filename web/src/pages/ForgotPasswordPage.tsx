import { useState } from 'react'

export default function ForgotPasswordPage({ onSwitchToLogin }: { onSwitchToLogin: () => void }) {
  const [email, setEmail] = useState('')
  const [msg, setMsg] = useState('')
  return <div className="auth-page"><div className="auth-card">
    <h1>Reset Password</h1>
    <p className="text-muted">Enter your email to receive a reset link.</p>
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      <input className="input" placeholder="Email" value={email} onChange={e => setEmail(e.target.value)} />
      <button className="btn btn-primary" onClick={() => setMsg('If the email exists, a reset link has been sent.')} style={{ justifyContent: 'center' }}>Send Reset Link</button>
      {msg && <p style={{ color: 'var(--success)', fontSize: 12, margin: 0 }}>{msg}</p>}
      <button className="btn btn-outline" onClick={onSwitchToLogin} style={{ justifyContent: 'center' }}>Back to Login</button>
    </div>
  </div></div>
}
