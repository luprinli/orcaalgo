import { useEffect } from 'react'
import type { IChartApi } from 'lightweight-charts'

interface ChartKeyboardOptions {
  enabled?: boolean
  zoomStep?: number
  panStep?: number
}

export function useChartKeyboard(
  chartRef: React.MutableRefObject<IChartApi | null>,
  options: ChartKeyboardOptions = {},
) {
  const { enabled = true, zoomStep = 0.85, panStep = 200 } = options

  useEffect(() => {
    if (!enabled) return

    const handleKey = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.tagName === 'SELECT') return
      const chart = chartRef.current
      if (!chart) return
      const ts = chart.timeScale()
      const currentSpacing = ((ts.options() as Record<string, unknown>).barSpacing as number) ?? 10

      switch (e.key) {
        case '+':
        case '=':
          e.preventDefault()
          ts.applyOptions({ barSpacing: Math.max(5, currentSpacing * zoomStep) } as Record<string, unknown>)
          break
        case '-':
          e.preventDefault()
          ts.applyOptions({ barSpacing: Math.max(1, currentSpacing / zoomStep) } as Record<string, unknown>)
          break
        case '0':
          e.preventDefault()
          ts.fitContent()
          break
        case 'ArrowLeft':
          if (e.ctrlKey || e.metaKey) {
            e.preventDefault()
            ts.scrollToPosition(ts.scrollPosition() - panStep, false)
          }
          break
        case 'ArrowRight':
          if (e.ctrlKey || e.metaKey) {
            e.preventDefault()
            ts.scrollToPosition(ts.scrollPosition() + panStep, false)
          }
          break
      }
    }

    document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [chartRef, enabled, zoomStep, panStep])
}
