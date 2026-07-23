interface ErrorCardProps {
  message: string
  onRetry?: () => void
}

export default function ErrorCard({ message, onRetry }: ErrorCardProps) {
  return (
    <div className="card mb-4" style={{ background: 'rgba(218,54,51,.1)', border: '1px solid var(--danger)' }}>
      <div className="flex-between">
        <span style={{ color: 'var(--danger)' }}>{message}</span>
        {onRetry && (
          <button className="btn btn-outline" style={{ fontSize: 12, padding: '2px 10px' }} onClick={onRetry}>
            Retry
          </button>
        )}
      </div>
    </div>
  )
}
