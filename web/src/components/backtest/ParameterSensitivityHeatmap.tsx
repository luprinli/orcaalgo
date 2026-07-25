import { useMemo, useEffect, useRef } from 'react'

interface SensitivityEntry {
  strategy: string
  symbol: string
  timeframe: string
  sharpe: number
  params: Record<string, number>
}

interface ParameterSensitivityHeatmapProps {
  entries: SensitivityEntry[]
  height?: number
}

export function ParameterSensitivityHeatmap({ entries, height = 400 }: ParameterSensitivityHeatmapProps) {
  const containerRef = useRef<HTMLDivElement>(null)

  // Infer parameter names from entries
  const paramNames = useMemo(() => {
    if (entries.length === 0) return []
    return Object.keys(entries[0].params)
  }, [entries])

  const heatmapData = useMemo(() => {
    if (paramNames.length < 2) return null
    const p1 = paramNames[0]
    const p2 = paramNames[1]

    // Get unique values
    const xValues = [...new Set(entries.map(e => e.params[p1]))].sort((a, b) => a - b)
    const yValues = [...new Set(entries.map(e => e.params[p2]))].sort((a, b) => a - b)

    const zMatrix = yValues.map(y =>
      xValues.map(x => {
        const match = entries.find(e => e.params[p1] === x && e.params[p2] === y)
        return match ? match.sharpe : null
      })
    )

    return { xValues, yValues, zMatrix, p1, p2 }
  }, [entries, paramNames])

  useEffect(() => {
    if (!heatmapData || !containerRef.current) return

    const Plotly = (window as any).Plotly
    if (!Plotly) return

    const trace = {
      type: 'heatmap' as const,
      x: heatmapData.xValues.map(String),
      y: heatmapData.yValues.map(String),
      z: heatmapData.zMatrix,
      colorscale: [
        [0, '#ef4444'],
        [0.3, '#f59e0b'],
        [0.5, '#fbbf24'],
        [0.7, '#34d399'],
        [1, '#22c55e'],
      ],
      showscale: true,
      colorbar: { title: 'Sharpe' },
    }

    const layout = {
      title: `${heatmapData.p1} vs ${heatmapData.p2}`,
      xaxis: { title: heatmapData.p1 },
      yaxis: { title: heatmapData.p2 },
      margin: { t: 40, r: 20, b: 40, l: 50 },
      paper_bgcolor: 'transparent',
      plot_bgcolor: 'transparent',
      font: { color: '#94a3b8' },
      height,
    }

    Plotly.newPlot(containerRef.current, [trace], layout, {
      displayModeBar: false,
      responsive: true,
    })

    return () => { Plotly.purge(containerRef.current) }
  }, [heatmapData, height])

  if (!heatmapData) {
    return (
      <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4">
        <h2>Parameter Sensitivity</h2>
        <p className="text-muted text-sm mt-2">
          Run a parameter sweep with 2+ parameters to visualize interaction effects as a heatmap.
        </p>
      </div>
    )
  }

  return (
    <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4">
      <h2>Parameter Sensitivity — {heatmapData.p1} vs {heatmapData.p2}</h2>
      <div ref={containerRef} />
    </div>
  )
}
