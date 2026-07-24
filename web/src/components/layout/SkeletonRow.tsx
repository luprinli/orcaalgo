import React from 'react'

interface SkeletonRowProps {
  rows?: number
  className?: string
}

export function SkeletonRow({ rows = 1, className = '' }: SkeletonRowProps) {
  return (
    <div className={`space-y-3 ${className}`}>
      {Array.from({ length: rows }).map((_, i) => (
        <div
          key={i}
          className="h-4 bg-slate-700/50 rounded animate-pulse"
          style={{ width: `${Math.max(40, 100 - i * 15)}%` }}
        />
      ))}
    </div>
  )
}
