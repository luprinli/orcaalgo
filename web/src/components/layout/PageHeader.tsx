import React from 'react'

interface PageHeaderProps {
  title: string
  subtitle?: string
  badge?: { text: string; variant: 'ok' | 'err' | 'warn' }
  actions?: React.ReactNode
}

export function PageHeader({ title, subtitle, badge, actions }: PageHeaderProps) {
  return (
    <div className="flex items-center justify-between mb-6">
      <div>
        <h1 className="text-2xl font-bold text-white">{title}</h1>
        {subtitle && (
          <p className="text-sm text-slate-400 mt-1">{subtitle}</p>
        )}
      </div>
      <div className="flex items-center gap-3">
        {badge && (
          <span
            className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
              badge.variant === 'ok'
                ? 'bg-green-400/10 text-green-400'
                : badge.variant === 'err'
                  ? 'bg-red-400/10 text-red-400'
                  : 'bg-yellow-400/10 text-yellow-400'
            }`}
          >
            {badge.text}
          </span>
        )}
        {actions}
      </div>
    </div>
  )
}
