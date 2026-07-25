import React from 'react'

interface PageSectionProps {
  title?: string
  variant?: 'default' | 'error' | 'warning'
  children: React.ReactNode
}

export function PageSection({ title, variant = 'default', children }: PageSectionProps) {
  const borderClass =
    variant === 'error'
      ? 'border-l-4 border-red-500'
      : variant === 'warning'
        ? 'border-l-4 border-yellow-500'
        : ''

  return (
    <div className={`bg-card rounded-lg p-4 ${borderClass}`}>
      {title && (
        <h2 className="text-lg font-semibold text-white mb-3">{title}</h2>
      )}
      {children}
    </div>
  )
}
