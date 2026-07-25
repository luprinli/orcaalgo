import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { auth } from '../api/client'
import { Button } from '../components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card'
import { Input } from '../components/ui/input'

export default function RegisterPage({ onSwitchToLogin }: { onSwitchToLogin: () => void }) {
  const { t } = useTranslation()
  const [user, setUser] = useState('')
  const [pass, setPass] = useState('')
  const [err, setErr] = useState('')
  const [msg, setMsg] = useState('')
  const [loading, setLoading] = useState(false)

  const handleRegister = async () => {
    setLoading(true)
    setErr('')
    setMsg('')
    try {
      await auth.register({ username: user, password: pass })
      setMsg(t('auth:registerSuccess', 'Account created. You can now sign in.'))
    } catch (e) {
      setErr(e instanceof Error ? e.message : t('auth:registerFailed', 'Registration failed'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <Card className="w-[360px]">
        <CardHeader>
          <CardTitle className="text-center">{t('auth:register', 'Register')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2.5">
          <Input
            placeholder={t('auth:username', 'Username')}
            value={user}
            onChange={e => setUser(e.target.value)}
          />
          <Input
            placeholder={t('auth:password', 'Password')}
            type="password"
            value={pass}
            onChange={e => setPass(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && handleRegister()}
          />
          {err && <p role="alert" className="text-destructive text-xs">{err}</p>}
          {msg && <p className="text-xs">{msg}</p>}
          <Button className="w-full" onClick={handleRegister} disabled={loading}>
            {loading ? t('auth:registering', 'Creating account...') : t('auth:registerButton', 'Create Account')}
          </Button>
          <Button variant="outline" className="w-full" onClick={onSwitchToLogin}>
            {t('auth:backToLogin', 'Back to Login')}
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
