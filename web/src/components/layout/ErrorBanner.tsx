import React from 'react'

interface ErrorBannerProps {
  error: Error | string
  onRetry?: () => void
  onDismiss?: () => void
}

export function ErrorBanner({ error, onRetry, onDismiss }: ErrorBannerProps) {
  const message = typeof error === 'string' ? error : error.message

  return (
    <div className="bg-red-400/10 border border-red-400/30 rounded-lg p-4 mb-4 flex items-start justify-between">
      <div className="flex items-start gap-3">
        <span className="text-red-400 mt-0.5">⚠</span>
        <div>
          <p className="text-red-400 font-medium text-sm">Error</p>
          <p className="text-red-300 text-sm mt-0.5">{message}</p>
        </div>
      </div>
      <div className="flex items-center gap-2 ml-4">
        {onRetry && (
          <button
            onClick={onRetry}
            className="text-xs text-red-400 hover:text-red-300 underline"
          >
            Retry
          </button>
        )}
        {onDismiss && (
          <button
            onClick={onDismiss}
            className="text-xs text-slate-400 hover:text-slate-300"
          >
            Dismiss
          </button>
        )}
      </div>
    </div>
  )
}
