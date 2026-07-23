import { useTranslation } from 'react-i18next'

export default function WebhookConfig() {
  const { t } = useTranslation()
  return <div>
    <h1 style={{margin:0}}>{t('sidebar:nav.webhooks', 'Webhooks')}</h1>
    <div className="card mt-4"><p className="text-muted">{t('webhooks:description', 'Configure webhook endpoints for external trade signals and notifications.')}</p></div>
  </div>
}
