import { cn } from '../../lib/utils'

interface SoftHaltGaugeProps {
  dailyLossPct: number
  softHaltThreshold: number
  hardHaltThreshold: number
  isSoftHalted: boolean
  isHardHalted: boolean
}

export default function SoftHaltGauge({
  dailyLossPct,
  softHaltThreshold,
  hardHaltThreshold,
  isSoftHalted,
  isHardHalted,
}: SoftHaltGaugeProps) {
  const lossPct = Math.abs(dailyLossPct)
  const safeZone = softHaltThreshold * 0.7

  const barWidth = Math.min((lossPct / hardHaltThreshold) * 100, 100)

  let barColor = 'bg-green-500'
  if (lossPct >= hardHaltThreshold) {
    barColor = 'bg-red-500'
  } else if (lossPct >= softHaltThreshold) {
    barColor = 'bg-amber-500'
  } else if (lossPct >= safeZone) {
    barColor = 'bg-yellow-500'
  }

  let statusText = 'Normal'
  let statusColor = 'text-green-400'
  if (isHardHalted) {
    statusText = 'HARD HALTED'
    statusColor = 'text-red-400'
  } else if (isSoftHalted) {
    statusText = 'Soft Halt — Positions Reduced 50%'
    statusColor = 'text-amber-400'
  } else if (lossPct >= safeZone) {
    statusText = 'Caution'
    statusColor = 'text-yellow-400'
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between text-sm">
        <span>Daily Loss</span>
        <span className={cn('font-semibold tabular-nums', statusColor)}>
          {lossPct.toFixed(2)}%
        </span>
      </div>

      <div className="relative h-3 w-full rounded-full bg-secondary overflow-hidden">
        <div
          className={cn('h-full rounded-full transition-all duration-500', barColor)}
          style={{ width: `${barWidth}%` }}
        />
        {/* Soft halt marker */}
        <div
          className="absolute top-0 h-full w-0.5 bg-amber-400/60"
          style={{ left: `${(softHaltThreshold / hardHaltThreshold) * 100}%` }}
          title={`Soft Halt: ${softHaltThreshold}%`}
        />
        {/* Hard halt marker */}
        <div
          className="absolute top-0 h-full w-0.5 bg-red-400/60"
          style={{ left: '100%' }}
          title={`Hard Halt: ${hardHaltThreshold}%`}
        />
      </div>

      <div className="flex justify-between text-[11px] text-muted-foreground">
        <span>0%</span>
        <span>{softHaltThreshold}% Soft</span>
        <span>{hardHaltThreshold}% Hard</span>
      </div>

      <p className={cn('text-sm font-medium', statusColor)}>
        {statusText}
      </p>

      {isSoftHalted && !isHardHalted && (
        <p className="text-xs text-muted-foreground">
          All new positions reduced to 50% of normal size until daily loss recovers.
        </p>
      )}
      {isHardHalted && (
        <p className="text-xs text-red-400">
          Trading halted. Kill switch engaged. Contact admin to resume.
        </p>
      )}
    </div>
  )
}
