import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { QRCodeSVG } from 'qrcode.react'

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
    <div>
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>{t('auth:twoFactorSetup', '2FA Setup')}</h1>
      </div>

      {msg && <p className="text-muted mb-2" style={{ color: 'var(--danger)' }}>{msg}</p>}

      <div className="card" style={{ maxWidth: 450 }}>
        {step === 'setup' && (
          <div>
            <h2>{t('auth:twoFactorEnable', 'Enable Two-Factor Authentication')}</h2>
            <p className="text-muted">
              {t('auth:twoFactorDescription', 'Two-factor authentication adds an extra layer of security to your account. You\'ll need an authenticator app like Google Authenticator or Authy.')}
            </p>
            <button className="btn btn-primary mt-3" onClick={handleSetup}>
              {t('auth:generateSecret', 'Generate Secret')}
            </button>
          </div>
        )}

        {step === 'confirm' && (
          <div>
            <h2>{t('auth:scanQrCode', 'Scan QR Code')}</h2>
            {uri && (
              <div className="card mb-3" style={{ textAlign: 'center', background: '#fff' }}>
                <QRCodeSVG value={uri} size={200} level="M" />
              </div>
            )}
            <div className="mb-3">
              <span className="text-muted">{t('auth:enterKeyManually', 'Or enter this key manually:')}</span>
              <div style={{ fontFamily: 'monospace', fontSize: 13, padding: 8, background: 'var(--bg-input)', borderRadius: 4, marginTop: 4, wordBreak: 'break-all' }}>
                {secret}
              </div>
            </div>
            <div>
              <label className="text-muted">{t('auth:verifyWithCode', 'Verify with 6-digit code')}</label>
              <div className="flex gap-2 mt-2">
                <input className="input" placeholder={t('risk:2faPlaceholder', '000000')} maxLength={6} value={code} onChange={e => setCode(e.target.value.replace(/\D/g, ''))} />
                <button className="btn btn-primary" disabled={code.length !== 6 || verifying} onClick={handleVerify}>{t('auth:verify', 'Verify')}</button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
