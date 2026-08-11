"use client"

import { useRef, useEffect, useMemo, useState } from "react"
import {
  Chart,
  DoughnutController,
  ArcElement,
  Tooltip,
  Legend,
} from "chart.js"
import { CardContent } from "../ui/card"
import { Slider } from "../ui/slider"
import type { AllocationEntry } from "../../types/api"

Chart.register(DoughnutController, ArcElement, Tooltip, Legend)

const COLORS = ["#60a5fa", "#34d399", "#fbbf24", "#f87171", "#a78bfa", "#fb923c", "#4ade80", "#f472b6"]

interface AllocationPieProps {
  allocations: { strategyId: string; weight: number }[]
  title?: string
  history?: AllocationEntry[]
}

export function AllocationPie({ allocations, title, history }: AllocationPieProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const chartRef = useRef<Chart | null>(null)
  const [timelineIndex, setTimelineIndex] = useState(0)

  const hasHistory = history && history.length > 0
  const timelineSteps = hasHistory
    ? Array.from(new Set(history!.map(h => h.bar_time))).sort()
    : []

  const currentAllocations = useMemo(() => {
    if (!hasHistory || timelineSteps.length === 0) return allocations
    const selectedTime = timelineSteps[Math.min(timelineIndex, timelineSteps.length - 1)]
    const entries = history!.filter(h => h.bar_time === selectedTime)
    const byStrategy: Record<string, number> = {}
    let total = 0
    for (const e of entries) {
      byStrategy[e.strategy_id] = (byStrategy[e.strategy_id] || 0) + e.allocated_capital
      total += e.allocated_capital
    }
    if (total <= 0) return allocations
    return Object.entries(byStrategy).map(([id, capital]) => ({
      strategyId: id,
      weight: capital / total,
    }))
  }, [history, hasHistory, timelineIndex, timelineSteps, allocations])

  useEffect(() => {
    if (chartRef.current) {
      chartRef.current.destroy()
      chartRef.current = null
    }
    if (!canvasRef.current || currentAllocations.length === 0) return

    const ctx = canvasRef.current.getContext("2d")
    if (!ctx) return

    chartRef.current = new Chart(ctx, {
      type: "doughnut",
      data: {
        labels: currentAllocations.map(a => a.strategyId),
        datasets: [{
          data: currentAllocations.map(a => a.weight),
          backgroundColor: currentAllocations.map((_, i) => COLORS[i % COLORS.length]),
          borderWidth: 1,
          borderColor: "var(--background)",
        }],
      },
      options: {
        responsive: true,
        maintainAspectRatio: true,
        plugins: {
          legend: {
            position: "bottom",
            labels: { boxWidth: 10, padding: 8, font: { size: 10 } },
          },
          tooltip: {
            callbacks: {
              label: (ctx) => `${ctx.label}: ${(ctx.raw as number * 100).toFixed(1)}%`,
            },
          },
        },
      },
    })

    return () => {
      if (chartRef.current) {
        chartRef.current.destroy()
        chartRef.current = null
      }
    }
  }, [currentAllocations])

  if (currentAllocations.length === 0) {
    return (
      <CardContent className="p-6 text-center">
        <p className="text-sm text-muted-foreground">No active strategies</p>
      </CardContent>
    )
  }

  return (
    <CardContent>
      <canvas ref={canvasRef} />
      {title && <p className="text-center text-xs text-muted-foreground mt-2">{title}</p>}
      {hasHistory && timelineSteps.length > 0 && (
        <div className="mt-3 px-2">
          <div className="flex justify-between text-[10px] text-muted-foreground mb-1">
            <span>{new Date(timelineSteps[0]).toLocaleDateString()}</span>
            <span>{new Date(timelineSteps[timelineIndex] || timelineSteps[0]).toLocaleDateString()}</span>
            <span>{new Date(timelineSteps[timelineSteps.length - 1]).toLocaleDateString()}</span>
          </div>
          <Slider
            min={0}
            max={Math.max(0, timelineSteps.length - 1)}
            step={1}
            value={[Math.min(timelineIndex, Math.max(0, timelineSteps.length - 1))]}
            onValueChange={([v]) => setTimelineIndex(v ?? 0)}
          />
        </div>
      )}
    </CardContent>
  )
}
