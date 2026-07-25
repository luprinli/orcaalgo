import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { auth } from '../api/client'
import { Button } from '../components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card'
import { Input } from '../components/ui/input'

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

  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <Card className="w-[360px]">
        <CardHeader>
          <CardTitle className="text-center">{t('auth:forgotPassword', 'Reset Password')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2.5">
          <p className="text-sm text-muted-foreground">{t('auth:resetDescription', 'Enter your email to receive a reset link.')}</p>
          <Input
            placeholder={t('auth:email', 'Email')}
            value={email}
            onChange={e => setEmail(e.target.value)}
          />
          <Button className="w-full" onClick={handleSubmit} disabled={loading}>
            {loading ? t('auth:sending', 'Sending...') : t('auth:sendResetLink', 'Send Reset Link')}
          </Button>
          {msg && <p className="text-xs">{msg}</p>}
          {error && <p role="alert" className="text-destructive text-xs">{error}</p>}
          <Button variant="outline" className="w-full" onClick={onSwitchToLogin}>
            {t('auth:backToLogin', 'Back to Login')}
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
