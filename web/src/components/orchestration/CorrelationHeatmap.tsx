"use client"

import { useMemo } from "react"
import { cn } from "../../lib/utils"

interface CorrelationHeatmapProps {
  correlationMatrix: Record<string, Record<string, number>>
  strategyIds: string[]
  threshold?: number
  onPairClick?: (a: string, b: string) => void
}

function correlationColor(value: number): string {
  if (value <= 0.2) return "bg-emerald-500/80"
  if (value <= 0.4) return "bg-emerald-400/70"
  if (value <= 0.5) return "bg-yellow-400/70"
  if (value <= 0.6) return "bg-orange-400/70"
  if (value <= 0.8) return "bg-red-400/70"
  return "bg-red-600/80"
}

function correlationTextColor(value: number): string {
  if (value <= 0.4) return "text-emerald-900 dark:text-emerald-100"
  if (value <= 0.6) return "text-yellow-900 dark:text-yellow-100"
  return "text-red-900 dark:text-red-100"
}

const LEGEND_STOPS = [
  { label: "0.0", className: "bg-emerald-500/80" },
  { label: "0.2", className: "bg-emerald-400/70" },
  { label: "0.4", className: "bg-yellow-400/70" },
  { label: "0.5", className: "bg-orange-400/70" },
  { label: "0.8", className: "bg-red-400/70" },
  { label: "1.0", className: "bg-red-600/80" },
]

export function CorrelationHeatmap({
  correlationMatrix,
  strategyIds,
  threshold = 0.6,
  onPairClick,
}: CorrelationHeatmapProps) {
  const size = strategyIds.length

  const maxCorrelation = useMemo(() => {
    let max = 0
    for (let r = 0; r < size; r++) {
      for (let c = r + 1; c < size; c++) {
        const val = correlationMatrix[strategyIds[r]]?.[strategyIds[c]]
        if (val !== undefined && val > max) max = val
      }
    }
    return max
  }, [correlationMatrix, strategyIds, size])

  if (size === 0) {
    return (
      <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
        No strategies to display
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <div className="overflow-x-auto">
        <div
          className="grid gap-px rounded-lg bg-border/40 p-px"
          style={{
            gridTemplateColumns: `auto repeat(${size}, minmax(48px, 1fr))`,
            minWidth: size * 52 + 72,
          }}
        >
          <div className="flex items-center justify-end px-1.5" />

          {strategyIds.map((id) => (
            <div
              key={`col-${id}`}
              className="flex items-center justify-center px-1 py-1.5 text-[10px] font-medium text-muted-foreground truncate"
              title={id}
            >
              {id}
            </div>
          ))}

          {strategyIds.map((rowId, ri) => (
            <div key={`row-${rowId}`} className="contents">
              <div
                className="flex items-center justify-end px-1.5 py-1 text-[10px] font-medium text-muted-foreground truncate"
                title={rowId}
              >
                {rowId}
              </div>

              {strategyIds.map((colId, ci) => {
                const isDiagonal = ri === ci
                const value = correlationMatrix[rowId]?.[colId] ?? 0
                const isAboveThreshold = !isDiagonal && value >= threshold
                const isClickable = !isDiagonal && onPairClick

                if (isDiagonal) {
                  return (
                    <div
                      key={`${rowId}-${colId}`}
                      className="flex items-center justify-center px-1 py-2 text-[10px] font-semibold bg-muted/60 text-foreground truncate"
                      title={rowId}
                    >
                      {rowId}
                    </div>
                  )
                }

                return (
                  <div
                    key={`${rowId}-${colId}`}
                    role={isClickable ? "button" : undefined}
                    tabIndex={isClickable ? 0 : undefined}
                    className={cn(
                      "flex items-center justify-center px-1 py-2 text-[10px] font-mono font-medium relative transition-opacity",
                      correlationColor(value),
                      correlationTextColor(value),
                      isAboveThreshold && "ring-2 ring-red-500/80 ring-inset",
                      isClickable && "cursor-pointer hover:opacity-80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    )}
                    title={`${rowId} × ${colId}: ${value.toFixed(3)}`}
                    onClick={() => {
                      if (isClickable && onPairClick) onPairClick(rowId, colId)
                    }}
                    onKeyDown={(e) => {
                      if (isClickable && onPairClick && (e.key === "Enter" || e.key === " ")) {
                        e.preventDefault()
                        onPairClick(rowId, colId)
                      }
                    }}
                  >
                    {value.toFixed(2)}
                  </div>
                )
              })}
            </div>
          ))}
        </div>
      </div>

      <div className="flex items-center gap-3 px-1 flex-wrap">
        <span className="text-[10px] text-muted-foreground">Correlation</span>
        <div className="flex items-center gap-0.5">
          {LEGEND_STOPS.map((stop) => (
            <div
              key={stop.label}
              className={cn("h-3 w-7 rounded-sm", stop.className)}
              title={stop.label}
            />
          ))}
        </div>
        {maxCorrelation > 0 && (
          <span className="text-[10px] text-muted-foreground ml-auto">
            Max: <span className="font-mono font-semibold text-foreground">{maxCorrelation.toFixed(3)}</span>
            {threshold < 1 && (
              <span className="ml-2">
                Threshold: <span className="font-mono font-semibold text-foreground">{threshold.toFixed(1)}</span>
              </span>
            )}
          </span>
        )}
      </div>
    </div>
  )
}

export default CorrelationHeatmap
