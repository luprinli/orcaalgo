import React from 'react'

interface ErrorBannerProps {
  error: Error | string
  onRetry?: () => void
  onDismiss?: () => void
}

export function ErrorBanner({ error, onRetry, onDismiss }: ErrorBannerProps) {
  const message = typeof error === 'string' ? error : error.message

  return (
    <div className="bg-destructive/10 border border-destructive/30 rounded-lg p-4 mb-4 flex items-start justify-between">
      <div className="flex items-start gap-3">
        <span className="text-destructive mt-0.5">⚠</span>
        <div>
          <p className="text-destructive font-medium text-sm">Error</p>
          <p className="text-destructive/80 text-sm mt-0.5">{message}</p>
        </div>
      </div>
      <div className="flex items-center gap-2 ml-4">
        {onRetry && (
          <button
            onClick={onRetry}
            className="text-xs text-destructive hover:text-destructive/80 underline"
          >
            Retry
          </button>
        )}
        {onDismiss && (
          <button
            onClick={onDismiss}
            className="text-xs text-muted-foreground hover:text-foreground"
          >
            Dismiss
          </button>
        )}
      </div>
    </div>
  )
}
