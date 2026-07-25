import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { auth } from '../api/client'
import { Button } from '../components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card'
import { Input } from '../components/ui/input'

export default function ResetPasswordPage({ onSwitchToLogin }: { onSwitchToLogin: () => void }) {
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') ?? ''
  const [pass, setPass] = useState('')
  const [loading, setLoading] = useState(false)
  const [msg, setMsg] = useState('')
  const [error, setError] = useState('')

  const handleSubmit = async () => {
    if (!pass.trim() || !token) return
    setLoading(true)
    setError('')
    setMsg('')
    try {
      await auth.resetPassword(token, pass)
      setMsg(t('auth:passwordResetSuccess', 'Password has been reset.'))
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
          <CardTitle className="text-center">{t('auth:setNewPassword', 'Set New Password')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2.5">
          <Input
            placeholder={t('auth:newPassword', 'New Password')}
            type="password"
            value={pass}
            onChange={e => setPass(e.target.value)}
          />
          <Button className="w-full" onClick={handleSubmit} disabled={loading || !token}>
            {loading ? t('auth:resetting', 'Resetting...') : t('auth:resetButton', 'Set Password')}
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
