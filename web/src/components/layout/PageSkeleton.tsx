import React from 'react'

export function PageSkeleton() {
  return (
    <div className="p-6 space-y-6 animate-pulse">
      <div className="h-8 bg-muted rounded w-1/3" />
      <div className="grid grid-cols-3 gap-4">
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="h-20 bg-muted rounded" />
        ))}
      </div>
      <div className="h-64 bg-muted rounded" />
    </div>
  )
}
