import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '../components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '../components/ui/card'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'

interface LoginPageProps {
  onLogin: (t: string) => void
  onForgotPassword?: () => void
  onRegister?: () => void
}

export default function LoginPage({ onLogin, onForgotPassword, onRegister }: LoginPageProps) {
  const { t } = useTranslation()
  const [user, setUser] = useState('')
  const [pass, setPass] = useState('')
  const [err, setErr] = useState('')
  const [loading, setLoading] = useState(false)
  const [showPass, setShowPass] = useState(false)

  const isEmpty = user.trim().length === 0 || pass.length === 0

  const login = async () => {
    if (isEmpty) {
      setErr(t('auth:fillAllFields', 'Please enter your username and password'))
      return
    }
    setLoading(true)
    setErr('')
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
        setErr(d.message || t('auth:invalidCredentials', 'Invalid username or password'))
      }
    } catch {
      setErr(t('auth:loginFailed', 'Unable to connect. Check your network and try again.'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <Card className="w-[380px]">
        <CardHeader className="space-y-1">
          <CardTitle className="text-center text-xl">{t('sidebar:brandName', 'Orca Algo')}</CardTitle>
          <CardDescription className="text-center">{t('auth:signInDescription', 'Sign in to your trading account')}</CardDescription>
        </CardHeader>

        <CardContent className="space-y-4">
          {/* Username */}
          <div className="space-y-2">
            <Label htmlFor="username">{t('auth:username', 'Username')}</Label>
            <Input
              id="username"
              placeholder="admin"
              value={user}
              onChange={e => { setUser(e.target.value); setErr('') }}
              disabled={loading}
              autoComplete="username"
              autoFocus
            />
          </div>

          {/* Password with visibility toggle */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label htmlFor="password">{t('auth:password', 'Password')}</Label>
              {onForgotPassword && (
                <button
                  type="button"
                  className="text-xs text-primary hover:underline"
                  onClick={onForgotPassword}
                  tabIndex={-1}
                >
                  {t('auth:forgotPassword', 'Forgot password?')}
                </button>
              )}
            </div>
            <div className="relative">
              <Input
                id="password"
                type={showPass ? 'text' : 'password'}
                value={pass}
                onChange={e => { setPass(e.target.value); setErr('') }}
                onKeyDown={e => e.key === 'Enter' && login()}
                disabled={loading}
                autoComplete="current-password"
                className="pr-10"
              />
              <button
                type="button"
                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground text-xs"
                onClick={() => setShowPass(!showPass)}
                tabIndex={-1}
                aria-label={showPass ? t('auth:hidePass', 'Hide password') : t('auth:showPass', 'Show password')}
              >
                {showPass ? 'Hide' : 'Show'}
              </button>
            </div>
          </div>

          {/* Validation error */}
          {err && (
            <div role="alert" className="rounded-md bg-destructive/10 border border-destructive/20 px-3 py-2">
              <p className="text-destructive text-xs">{err}</p>
            </div>
          )}

          {/* Submit */}
          <Button className="w-full" onClick={login} disabled={loading}>
            {loading ? t('auth:signingIn', 'Signing in...') : t('auth:loginButton', 'Sign In')}
          </Button>
        </CardContent>

        {/* Footer links */}
        <CardFooter className="flex flex-col gap-2">
          {onRegister && (
            <Button variant="outline" className="w-full" onClick={onRegister}>
              {t('auth:createAccount', 'Create an account')}
            </Button>
          )}
        </CardFooter>
      </Card>
    </div>
  )
}
