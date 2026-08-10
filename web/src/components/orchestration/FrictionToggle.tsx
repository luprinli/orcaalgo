"use client"

import { ToggleGroup, ToggleGroupItem } from "../ui/toggle-group"
import { cn } from "../../lib/utils"
import { ShieldCheck, ShieldAlert } from "lucide-react"

type FrictionModel = "realistic" | "idealized"

interface FrictionToggleProps {
  model: FrictionModel
  onChange: (model: FrictionModel) => void
}

const MODEL_DESCRIPTIONS: Record<FrictionModel, { label: string; description: string; icon: React.ReactNode }> = {
  realistic: {
    label: "Realistic (E3)",
    description: "Per-asset-class spread-based cost model with liquidity tiers, volume-dependent slippage, and fee schedules.",
    icon: <ShieldCheck className="h-3.5 w-3.5" />,
  },
  idealized: {
    label: "Idealized (0.5bps)",
    description: "Pre-E3 simplified model with flat 0.5bps spread assumption. Ignores liquidity, volume impact, and asset-class fee differences.",
    icon: <ShieldAlert className="h-3.5 w-3.5" />,
  },
}

export function FrictionToggle({ model, onChange }: FrictionToggleProps) {
  const handleChange = (val: string) => {
    if (val === "realistic" || val === "idealized") {
      onChange(val)
    }
  }

  const active = MODEL_DESCRIPTIONS[model]

  return (
    <div className="space-y-3">
      <ToggleGroup
        type="single"
        value={model}
        onValueChange={handleChange}
        className="w-full"
      >
        <ToggleGroupItem value="realistic" size="lg" className="flex-1 gap-2">
          {MODEL_DESCRIPTIONS.realistic.icon}
          <span className="text-xs font-medium">Realistic (E3)</span>
        </ToggleGroupItem>
        <ToggleGroupItem value="idealized" size="lg" className="flex-1 gap-2">
          {MODEL_DESCRIPTIONS.idealized.icon}
          <span className="text-xs font-medium">Idealized (0.5bps)</span>
        </ToggleGroupItem>
      </ToggleGroup>

      <div
        className={cn(
          "rounded-lg border px-3 py-2.5 text-xs leading-relaxed transition-colors",
          model === "realistic"
            ? "border-amber-500/30 bg-amber-500/5 text-amber-700 dark:text-amber-300"
            : "border-emerald-500/30 bg-emerald-500/5 text-emerald-700 dark:text-emerald-300"
        )}
      >
        <div className="flex items-start gap-2">
          <div className="mt-0.5 shrink-0">{active.icon}</div>
          <div>
            <p className="font-medium">{active.label}</p>
            <p className="mt-0.5 text-muted-foreground">{active.description}</p>
            {model === "realistic" && (
              <p className="mt-1.5 pt-1.5 border-t border-amber-500/20 text-[10px] text-amber-600 dark:text-amber-400">
                Expected cost impact: spread costs vary by asset class (equities +0.8-2.5bps, FX +0.3-1.2bps, futures +0.5-1.8bps)
                with additional volume-dependent slippage. Net PnL estimates will be lower vs. idealized model.
              </p>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

export default FrictionToggle
