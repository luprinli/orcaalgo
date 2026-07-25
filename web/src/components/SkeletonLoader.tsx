interface SkeletonLoaderProps {
  lines?: number
  height?: number
  width?: string
  className?: string
}

export default function SkeletonLoader({ lines = 1, height = 16, width = '100%', className = '' }: SkeletonLoaderProps) {
  return (
    <div className={className} style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {Array.from({ length: lines }).map((_, i) => (
        <div
          key={i}
          style={{
            height,
            width: i === lines - 1 && lines > 1 ? '60%' : width,
            borderRadius: 4,
            background: 'var(--input)',
            animation: 'skeleton-pulse 1.5s ease-in-out infinite',
          }}
        />
      ))}
    </div>
  )
}

export function MetricSkeleton({ count = 3 }: { count?: number }) {
  return (
    <div className="metric-grid">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="rounded-lg bg-card ring-1 ring-foreground/10 p-3 flex flex-col gap-0.5">
          <div
            style={{
              height: 10, width: '60%', margin: '0 auto 8px',
              borderRadius: 3, background: 'var(--input)',
              animation: 'skeleton-pulse 1.5s ease-in-out infinite',
            }}
          />
          <div
            style={{
              height: 24, width: '40%', margin: '0 auto',
              borderRadius: 3, background: 'var(--input)',
              animation: 'skeleton-pulse 1.5s ease-in-out infinite',
            }}
          />
        </div>
      ))}
    </div>
  )
}

export function TableSkeleton({ rows = 5, cols = 4 }: { rows?: number; cols?: number }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      {Array.from({ length: rows }).map((_, r) => (
        <div key={r} style={{ display: 'flex', gap: 8 }}>
          {Array.from({ length: cols }).map((_, c) => (
            <div
              key={c}
              style={{
                flex: 1,
                height: 14,
                borderRadius: 3,
                background: 'var(--input)',
                animation: 'skeleton-pulse 1.5s ease-in-out infinite',
              }}
            />
          ))}
        </div>
      ))}
    </div>
  )
}
