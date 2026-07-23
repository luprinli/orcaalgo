import { useTranslation } from 'react-i18next'

interface ErrorCardProps {
  message: string
  onRetry?: () => void
}

export default function ErrorCard({ message, onRetry }: ErrorCardProps) {
  const { t } = useTranslation()
  return (
    <div className="card mb-4" style={{ background: 'rgba(218,54,51,.1)', border: '1px solid var(--danger)' }}>
      <div className="flex-between">
        <span style={{ color: 'var(--danger)' }}>{message}</span>
        {onRetry && (
          <button className="btn btn-outline" style={{ fontSize: 12, padding: '2px 10px' }} onClick={onRetry}>
            {t('components:errorCard.retry', 'Retry')}
          </button>
        )}
      </div>
    </div>
  )
}
