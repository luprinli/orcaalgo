import { useTranslation } from 'react-i18next'

export default function CredentialManagement() {
  const { t } = useTranslation()
  return <div>
    <h1 style={{margin:0}}>{t('sidebar:nav.credentials', 'Credentials')}</h1>
    <div className="card mt-4"><p className="text-muted">{t('credentials:description', 'Manage API keys and broker credentials via environment variables or the admin panel.')}</p></div>
  </div>
}
