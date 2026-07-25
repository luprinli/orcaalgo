import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import type { IChartApi } from 'lightweight-charts'
import { useChartKeyboard } from '../hooks/useChartKeyboard'

function createChartRef() {
  const applyOptions = vi.fn()
  const fitContent = vi.fn()
  const scrollPosition = vi.fn().mockReturnValue(500)
  const scrollToPosition = vi.fn()
  const options = vi.fn().mockReturnValue({ barSpacing: 20 })
  const getVisibleLogicalRange = vi.fn().mockReturnValue({ from: 100, to: 200 })
  const setVisibleLogicalRange = vi.fn()

  const timeScale = {
    applyOptions,
    fitContent,
    scrollPosition,
    scrollToPosition,
    options,
    getVisibleLogicalRange,
    setVisibleLogicalRange,
  }

  const chartRef: React.MutableRefObject<IChartApi | null> = {
    current: {
      timeScale: vi.fn().mockReturnValue(timeScale),
    } as unknown as IChartApi,
  }

  return { chartRef, mocks: { applyOptions, fitContent, scrollPosition, scrollToPosition, options, getVisibleLogicalRange, setVisibleLogicalRange } }
}

function fireKeyDown(key: string, opts: { ctrlKey?: boolean; metaKey?: boolean; target?: Pick<HTMLElement, 'tagName'> } = {}) {
  const event = new KeyboardEvent('keydown', {
    key,
    ctrlKey: opts.ctrlKey ?? false,
    metaKey: opts.metaKey ?? false,
    bubbles: true,
    cancelable: true,
  })
  if (opts.target) {
    Object.defineProperty(event, 'target', { value: opts.target, writable: false })
  }
  document.dispatchEvent(event)
}

describe('useChartKeyboard', () => {
  let addEventListenerSpy: ReturnType<typeof vi.spyOn>
  let removeEventListenerSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    addEventListenerSpy = vi.spyOn(document, 'addEventListener')
    removeEventListenerSpy = vi.spyOn(document, 'removeEventListener')
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('dispatches zoom-in on "+" key', () => {
    const { chartRef, mocks } = createChartRef()
    renderHook(() => useChartKeyboard(chartRef))

    fireKeyDown('+')

    expect(mocks.setVisibleLogicalRange).toHaveBeenCalled()
  })

  it('dispatches zoom-in on "=" key', () => {
    const { chartRef, mocks } = createChartRef()
    renderHook(() => useChartKeyboard(chartRef))

    fireKeyDown('=')

    expect(mocks.setVisibleLogicalRange).toHaveBeenCalled()
  })

  it('dispatches zoom-out on "-" key', () => {
    const { chartRef, mocks } = createChartRef()
    renderHook(() => useChartKeyboard(chartRef))

    fireKeyDown('-')

    expect(mocks.setVisibleLogicalRange).toHaveBeenCalled()
  })

  it('calls fitContent on "0" key', () => {
    const { chartRef, mocks } = createChartRef()
    renderHook(() => useChartKeyboard(chartRef))

    fireKeyDown('0')

    expect(mocks.fitContent).toHaveBeenCalled()
  })

  it('scrolls left on Ctrl+ArrowLeft', () => {
    const { chartRef, mocks } = createChartRef()
    mocks.scrollPosition.mockReturnValue(500)
    renderHook(() => useChartKeyboard(chartRef))

    fireKeyDown('ArrowLeft', { ctrlKey: true })

    expect(mocks.scrollToPosition).toHaveBeenCalledWith(expect.any(Number), false)
    const calledPosition = mocks.scrollToPosition.mock.calls[0][0] as number
    expect(calledPosition).toBeLessThan(500)
  })

  it('scrolls right on Ctrl+ArrowRight', () => {
    const { chartRef, mocks } = createChartRef()
    mocks.scrollPosition.mockReturnValue(500)
    renderHook(() => useChartKeyboard(chartRef))

    fireKeyDown('ArrowRight', { ctrlKey: true })

    expect(mocks.scrollToPosition).toHaveBeenCalledWith(expect.any(Number), false)
    const calledPosition = mocks.scrollToPosition.mock.calls[0][0] as number
    expect(calledPosition).toBeGreaterThan(500)
  })

  it('does not scroll on ArrowLeft without Ctrl', () => {
    const { chartRef, mocks } = createChartRef()
    renderHook(() => useChartKeyboard(chartRef))

    fireKeyDown('ArrowLeft', { ctrlKey: false })

    expect(mocks.scrollToPosition).not.toHaveBeenCalled()
  })

  it('does not scroll on ArrowRight without Ctrl', () => {
    const { chartRef, mocks } = createChartRef()
    renderHook(() => useChartKeyboard(chartRef))

    fireKeyDown('ArrowRight', { ctrlKey: false })

    expect(mocks.scrollToPosition).not.toHaveBeenCalled()
  })

  it('does not fire handlers when enabled=false', () => {
    const { chartRef, mocks } = createChartRef()
    renderHook(() => useChartKeyboard(chartRef, { enabled: false }))

    fireKeyDown('+')
    fireKeyDown('0')
    fireKeyDown('ArrowLeft', { ctrlKey: true })

    expect(mocks.applyOptions).not.toHaveBeenCalled()
    expect(mocks.fitContent).not.toHaveBeenCalled()
    expect(mocks.scrollToPosition).not.toHaveBeenCalled()
  })

  it('ignores events inside INPUT elements', () => {
    const { chartRef, mocks } = createChartRef()
    renderHook(() => useChartKeyboard(chartRef))

    fireKeyDown('+', { target: { tagName: 'INPUT' } })

    expect(mocks.applyOptions).not.toHaveBeenCalled()
  })

  it('ignores events inside TEXTAREA elements', () => {
    const { chartRef, mocks } = createChartRef()
    renderHook(() => useChartKeyboard(chartRef))

    fireKeyDown('0', { target: { tagName: 'TEXTAREA' } })

    expect(mocks.fitContent).not.toHaveBeenCalled()
  })

  it('ignores events inside SELECT elements', () => {
    const { chartRef, mocks } = createChartRef()
    renderHook(() => useChartKeyboard(chartRef))

    fireKeyDown('-', { target: { tagName: 'SELECT' } })

    expect(mocks.applyOptions).not.toHaveBeenCalled()
  })

  it('removes event listener on unmount', () => {
    const { chartRef } = createChartRef()
    const { unmount } = renderHook(() => useChartKeyboard(chartRef))

    expect(addEventListenerSpy).toHaveBeenCalledWith('keydown', expect.any(Function))

    const handler = addEventListenerSpy.mock.calls[0][1]
    unmount()

    expect(removeEventListenerSpy).toHaveBeenCalledWith('keydown', handler)
  })
})
