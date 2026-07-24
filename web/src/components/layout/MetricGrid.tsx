import React from 'react'

interface MetricGridProps {
  columns?: 3 | 4 | 5
  children: React.ReactNode
}

export function MetricGrid({ columns = 3, children }: MetricGridProps) {
  const colClass =
    columns === 5
      ? 'grid-cols-2 sm:grid-cols-3 lg:grid-cols-5'
      : columns === 4
        ? 'grid-cols-2 sm:grid-cols-4'
        : 'grid-cols-2 sm:grid-cols-3'

  return <div className={`grid ${colClass} gap-4`}>{children}</div>
}
