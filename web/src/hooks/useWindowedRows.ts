import { useState, useCallback } from 'react'

export interface WindowedRows {
  start: number
  end: number
  topPad: number
  bottomPad: number
  onScroll: (e: React.UIEvent<HTMLDivElement>) => void
}

/**
 * useWindowedRows is a dependency-free virtualization helper for large tables:
 * it renders only the rows within (and just outside) the visible viewport,
 * padding above/below with spacer rows so scroll position and height stay exact.
 * Keeps the matrix results table smooth at thousands of combos (execution
 * framework plan §11.5) without pulling in react-window/@tanstack/virtual.
 *
 * Virtualization only kicks in above `threshold` rows; below that, everything
 * renders (start=0, end=total, no padding) to avoid spacer overhead on small sets.
 */
export function useWindowedRows(
  total: number,
  rowHeight: number,
  viewportHeight: number,
  overscan = 10,
  threshold = 100,
): WindowedRows {
  const [scrollTop, setScrollTop] = useState(0)

  const onScroll = useCallback((e: React.UIEvent<HTMLDivElement>) => {
    setScrollTop(e.currentTarget.scrollTop)
  }, [])

  if (total <= threshold || rowHeight <= 0) {
    return { start: 0, end: total, topPad: 0, bottomPad: 0, onScroll }
  }

  const start = Math.max(0, Math.floor(scrollTop / rowHeight) - overscan)
  const visibleCount = Math.ceil(viewportHeight / rowHeight) + overscan * 2
  const end = Math.min(total, start + visibleCount)
  const topPad = start * rowHeight
  const bottomPad = Math.max(0, (total - end) * rowHeight)

  return { start, end, topPad, bottomPad, onScroll }
}
