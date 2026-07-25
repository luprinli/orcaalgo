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

      const handleZoom = (zoomIn: boolean) => {
        const range = ts.getVisibleLogicalRange()
        if (!range) return
        const mid = (range.from + range.to) / 2
        const span = range.to - range.from
        const newSpan = zoomIn ? Math.max(span * 0.7, 10) : span / 0.7
        ts.setVisibleLogicalRange({ from: mid - newSpan / 2, to: mid + newSpan / 2 })
      }

      switch (e.key) {
        case '+':
        case '=':
          e.preventDefault()
          handleZoom(true)
          break
        case '-':
          e.preventDefault()
          handleZoom(false)
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
