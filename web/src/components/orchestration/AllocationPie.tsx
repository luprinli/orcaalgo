"use client"

import { useRef, useEffect, useMemo } from "react"
import {
  Chart,
  DoughnutController,
  ArcElement,
  Tooltip,
  Legend,
} from "chart.js"
import { EmptyState } from "../EmptyState"
import { PieChart as PieChartIcon } from "lucide-react"

Chart.register(DoughnutController, ArcElement, Tooltip, Legend)

interface AllocationSlice {
  strategyId: string
  weight: number
  color?: string
}

interface AllocationPieProps {
  allocations: AllocationSlice[]
  title?: string
}

const DEFAULT_COLORS = [
  "#2962FF",
  "#3fb950",
  "#d29922",
  "#f85149",
  "#a371f7",
  "#79c0ff",
  "#56d4dd",
  "#ffa28b",
  "#8b949e",
  "#e3b341",
  "#f778ba",
  "#7ee787",
]

export function AllocationPie({ allocations, title }: AllocationPieProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const chartRef = useRef<Chart<"doughnut"> | null>(null)

  const sorted = useMemo(
    () => [...allocations].sort((a, b) => b.weight - a.weight),
    [allocations]
  )

  useEffect(() => {
    if (!canvasRef.current) return
    const ctx = canvasRef.current.getContext("2d")
    if (!ctx) return

    if (chartRef.current) {
      chartRef.current.destroy()
      chartRef.current = null
    }

    if (sorted.length === 0) return

    const labels = sorted.map((s) => s.strategyId)
    const data = sorted.map((s) => s.weight)
    const colors = sorted.map((s, i) => s.color ?? DEFAULT_COLORS[i % DEFAULT_COLORS.length])

    const totalWeight = data.reduce((sum, v) => sum + v, 0)

    chartRef.current = new Chart(ctx, {
      type: "doughnut",
      data: {
        labels,
        datasets: [
          {
            data,
            backgroundColor: colors,
            borderColor: "hsl(var(--background))",
            borderWidth: 2,
          },
        ],
      },
      options: {
        responsive: true,
        maintainAspectRatio: true,
        cutout: "55%",
        plugins: {
          legend: {
            position: "bottom" as const,
            labels: {
              boxWidth: 10,
              boxHeight: 10,
              padding: 12,
              font: { size: 11 },
              color: "hsl(var(--muted-foreground))",
              generateLabels(chart) {
                const ds = chart.data.datasets[0]
                const lbls = chart.data.labels as string[]
                return lbls.map((label, i) => ({
                  text: `${label} (${((ds.data[i] as number) / totalWeight * 100).toFixed(1)}%)`,
                  fillStyle: (ds.backgroundColor as string[])[i],
                  strokeStyle: (ds.backgroundColor as string[])[i],
                  lineWidth: 0,
                  hidden: false,
                  index: i,
                  fontColor: "hsl(var(--muted-foreground))",
                  borderRadius: 0,
                  pointStyle: undefined,
                  rotation: undefined,
                  textAlign: "left" as const,
                }))
              },
            },
          },
          tooltip: {
            callbacks: {
              title(tooltipItems) {
                return tooltipItems[0].label
              },
              label(tooltipItem) {
                const val = tooltipItem.raw as number
                const pct = totalWeight > 0 ? ((val / totalWeight) * 100).toFixed(2) : "0.00"
                return `Weight: ${val.toFixed(4)} (${pct}%)`
              },
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
  }, [sorted])

  if (allocations.length === 0) {
    return (
      <div className="rounded-xl bg-card ring-1 ring-foreground/10 p-6">
        {title && (
          <div className="mb-2">
            <h3 className="font-heading text-base font-medium leading-none">{title}</h3>
          </div>
        )}
        <EmptyState
          icon={<PieChartIcon className="h-8 w-8" />}
          title="No active strategies"
          description="Allocated capital will appear here once strategies are activated."
        />
      </div>
    )
  }

  return (
    <div className="rounded-xl bg-card ring-1 ring-foreground/10 p-4">
      {title && (
        <h3 className="font-heading text-base font-medium leading-none mb-3">{title}</h3>
      )}
      <div className="w-full max-w-[300px] mx-auto">
        <canvas ref={canvasRef} aria-label="Strategy allocation doughnut chart" />
      </div>
    </div>
  )
}

export default AllocationPie
