import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { QRCodeSVG } from 'qrcode.react'
import { Button } from '../components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card'
import { Input } from '../components/ui/input'

export default function TwoFAPage() {
  const { t } = useTranslation()
  const [step, setStep] = useState<'setup' | 'confirm'>('setup')
  const [secret, setSecret] = useState('')
  const [uri, setUri] = useState('')
  const [code, setCode] = useState('')
  const [msg, setMsg] = useState('')
  const [verifying, setVerifying] = useState(false)

  const getToken = () => {
    try { return JSON.parse(localStorage.getItem('orca_auth') || '{}').token } catch { return null }
  }

  const handleSetup = async () => {
    try {
      const token = getToken()
      const res = await fetch('/api/v1/auth/2fa/setup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...(token ? { 'Authorization': `Bearer ${token}` } : {}) },
      })
      const data = await res.json()
      if (data.secret) {
        setSecret(data.secret)
        setUri(data.uri)
        setStep('confirm')
      } else {
        setMsg(data.error || t('auth:twoFaFailed', '2FA setup failed'))
      }
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t('auth:twoFaFailed', '2FA setup failed'))
    }
  }

  const handleVerify = async () => {
    if (code.length !== 6) return
    setVerifying(true)
    try {
      const token = getToken()
      const res = await fetch('/api/v1/auth/2fa/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...(token ? { 'Authorization': `Bearer ${token}` } : {}) },
        body: JSON.stringify({ code }),
      })
      const data = await res.json()
      if (data.verified) {
        setMsg(t('auth:twoFaEnabled', '2FA verified and enabled successfully!'))
        setStep('setup')
        setSecret('')
        setUri('')
        setCode('')
      } else {
        setMsg(data.error || t('auth:invalidCode', 'Invalid code. Please try again.'))
      }
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t('auth:verificationFailed', 'Verification failed'))
    } finally {
      setVerifying(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <Card className="w-[450px]">
        <CardHeader>
          <CardTitle>{t('auth:twoFactorSetup', '2FA Setup')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {msg && <p role="alert" className="text-destructive text-xs">{msg}</p>}

          {step === 'setup' && (
            <div className="space-y-3">
              <h2 className="text-lg font-semibold">{t('auth:twoFactorEnable', 'Enable Two-Factor Authentication')}</h2>
              <p className="text-sm text-muted-foreground">
                {t('auth:twoFactorDescription', 'Two-factor authentication adds an extra layer of security to your account. You\'ll need an authenticator app like Google Authenticator or Authy.')}
              </p>
              <Button onClick={handleSetup}>
                {t('auth:generateSecret', 'Generate Secret')}
              </Button>
            </div>
          )}

          {step === 'confirm' && (
            <div className="space-y-3">
              <h2 className="text-lg font-semibold">{t('auth:scanQrCode', 'Scan QR Code')}</h2>
              {uri && (
                <div className="flex justify-center bg-white rounded-lg p-4">
                  <QRCodeSVG value={uri} size={200} level="M" />
                </div>
              )}
              <div>
                <span className="text-sm text-muted-foreground">{t('auth:enterKeyManually', 'Or enter this key manually:')}</span>
                <div className="font-mono text-[13px] p-2 bg-muted rounded mt-1 break-all">
                  {secret}
                </div>
              </div>
              <div>
                <label className="text-sm text-muted-foreground">{t('auth:verifyWithCode', 'Verify with 6-digit code')}</label>
                <div className="flex gap-2 mt-2">
                  <Input
                    placeholder={t('risk:2faPlaceholder', '000000')}
                    maxLength={6}
                    value={code}
                    onChange={e => setCode(e.target.value.replace(/\D/g, ''))}
                  />
                  <Button disabled={code.length !== 6 || verifying} onClick={handleVerify}>
                    {t('auth:verify', 'Verify')}
                  </Button>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
