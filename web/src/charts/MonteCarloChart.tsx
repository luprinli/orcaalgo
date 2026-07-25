import { useRef, useEffect, useMemo, useState, useCallback } from 'react'
import { type Time, type LineData } from 'lightweight-charts'
import { useChart, useLineSeries, useAreaSeries } from './useChart'
import { useChartKeyboard } from '../hooks/useChartKeyboard'
import { getChartColors } from './chartConfig'
import type { DailyReturn } from '../types/api'
import MonteCarloWorker from '../workers/monteCarlo.worker?worker'

export interface MCStats {
  numSimulations: number
  numDays: number
  avgPnlPct: number
  medianPnlPct: number
  p5PnlPct: number
  p10PnlPct: number
  avgMaxDDPct: number
  medianMaxDDPct: number
  p95MaxDDPct: number
  bustProbability: number
}

export interface MCResultData {
  p5: number[]
  p25: number[]
  p50: number[]
  p75: number[]
  p95: number[]
  allPnlPct: number[]
  allMaxDDPct: number[]
  stats: MCStats
}

interface MonteCarloChartProps {
  dailyReturns: DailyReturn[]
  simulations?: number
  forwardDays?: number
  seed?: number
  height?: number
  title?: string
  onMCResult?: (result: MCResultData) => void
}

const DAY_MS = 86400000

export default function MonteCarloChart({
  dailyReturns,
  simulations = 500,
  forwardDays = 252,
  seed,
  height = 300,
  title,
  onMCResult,
}: MonteCarloChartProps) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const workerRef = useRef<Worker | null>(null)
  const onMCResultRef = useRef(onMCResult)
  onMCResultRef.current = onMCResult
  const [simulationData, setSimulationData] = useState<MCResultData | null>(null)
  const [crosshairData, setCrosshairData] = useState<{
    time: string; p5: number; p25: number; p50: number; p75: number; p95: number
  } | null>(null)

  useEffect(() => {
    workerRef.current = new MonteCarloWorker()
    return () => {
      workerRef.current?.terminate()
      workerRef.current = null
    }
  }, [])

  useEffect(() => {
    if (dailyReturns.length < 2) {
      setSimulationData(null)
      return
    }
    const returns = dailyReturns.map(r => r.return_pct / 100)
    workerRef.current?.postMessage({ returns, simulations, forwardDays, seed })
  }, [dailyReturns, simulations, forwardDays])

  useEffect(() => {
    const worker = workerRef.current
    if (!worker) return
    const handler = (e: MessageEvent<MCResultData>) => {
      setSimulationData(e.data)
      onMCResultRef.current?.(e.data)
    }
    worker.addEventListener('message', handler)
    return () => worker.removeEventListener('message', handler)
  }, [])

  const today = useMemo(() => Date.now(), [])

  const chartRef = useChart(containerRef, {
    height,
    rightPriceScaleMargins: { top: 0.05, bottom: 0.05 },
  })

  const colors = getChartColors()

  const { setData: setMedian } = useLineSeries(chartRef, colors.line, {
    lineWidth: 2,
    priceFormat: { type: 'price', precision: 4, minMove: 0.0001 },
  })

  const { setData: setBand75 } = useAreaSeries(chartRef, {
    lineColor: 'rgba(41, 98, 255, 0.3)',
    topColor: 'rgba(41, 98, 255, 0.15)',
    bottomColor: 'rgba(41, 98, 255, 0.02)',
    lineWidth: 1,
    priceFormat: { type: 'price', precision: 4, minMove: 0.0001 },
  })

  const { setData: setBand95 } = useAreaSeries(chartRef, {
    lineColor: 'rgba(41, 98, 255, 0.15)',
    topColor: 'rgba(41, 98, 255, 0.06)',
    bottomColor: 'rgba(41, 98, 255, 0.01)',
    lineWidth: 1,
    priceFormat: { type: 'price', precision: 4, minMove: 0.0001 },
  })

  const stepMs = useMemo(() => forwardDays > 0 ? (DAY_MS * 252 / forwardDays) : DAY_MS, [forwardDays])
  const baseTime = useMemo(() => Math.floor(today / 1000), [today])

  const simTimeMap = useMemo(() => {
    if (!simulationData) return null
    const map = new Map<number, { p5: number; p25: number; p50: number; p75: number; p95: number }>()
    for (let i = 0; i < simulationData.p50.length; i++) {
      const ts = baseTime + i * stepMs
      map.set(ts, {
        p5: simulationData.p5[i] ?? 0,
        p25: simulationData.p25[i] ?? 0,
        p50: simulationData.p50[i] ?? 0,
        p75: simulationData.p75[i] ?? 0,
        p95: simulationData.p95[i] ?? 0,
      })
    }
    return map
  }, [simulationData, baseTime, stepMs])

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleCrosshair = useCallback((param: any) => {
    if (!param.time || !simTimeMap) {
      setCrosshairData(null)
      return
    }
    const ts = typeof param.time === 'number' ? param.time : 0
    const data = simTimeMap.get(ts)
    if (data) {
      setCrosshairData({
        time: new Date(ts * 1000).toLocaleString(),
        ...data,
      })
    }
  }, [simTimeMap])

  useEffect(() => {
    if (!chartRef.current) return
    const chart = chartRef.current
    chart.subscribeCrosshairMove(handleCrosshair)
    return () => {
      chart.unsubscribeCrosshairMove(handleCrosshair)
      setCrosshairData(null)
    }
  }, [chartRef, handleCrosshair])

  useEffect(() => {
    if (!simulationData) return

    const toLineData = (values: number[]): LineData[] =>
      values.map((v, i) => ({
        time: (baseTime + i * stepMs) as Time,
        value: v,
      }))

    setMedian(toLineData(simulationData.p50))
    setBand75(toLineData(simulationData.p75))
    setBand95(toLineData(simulationData.p95))
  }, [simulationData, baseTime, stepMs, setMedian, setBand75, setBand95])

  useChartKeyboard(chartRef)

  if (!simulationData) {
    return (
      <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4">
        {title && <div className="flex items-center justify-between border-b border-border pb-2 mb-3"><h3>{title}</h3></div>}
        <p className="text-muted">
          Insufficient daily returns data ({dailyReturns.length} days). Need at least 2 days for Monte Carlo simulation.
        </p>
      </div>
    )
  }

  return (
    <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-4" style={{ position: 'relative' }}>
      {title && (
        <div className="flex items-center justify-between border-b border-border pb-2 mb-3">
          <h3>{title}</h3>
          <span className="text-muted" style={{ fontSize: 11 }}>
            {simulations} simulations, {forwardDays} days forward, sampling from {dailyReturns.length} daily returns
          </span>
        </div>
      )}
      <div ref={containerRef} role="img" aria-label="Monte Carlo simulation chart" />
      {crosshairData && (
        <div style={{
          position: 'absolute', top: 4, left: 4, zIndex: 50,
          background: 'var(--chart-tooltip-bg)',
          border: '1px solid var(--border)',
          borderRadius: 6, padding: '8px 10px',
          fontSize: 11, fontFamily: 'monospace',
          color: 'var(--muted-foreground)',
          pointerEvents: 'none',
          boxShadow: '0 2px 8px rgba(0,0,0,0.3)',
        }}>
          <div style={{ color: 'var(--foreground)', fontWeight: 600, marginBottom: 4, fontSize: 10 }}>
            {crosshairData.time}
          </div>
          <div>P50 <span style={{ color: 'var(--chart-line)', fontWeight: 600 }}>{crosshairData.p50.toFixed(4)}</span></div>
          <div>P25 <span style={{ color: 'var(--foreground)' }}>{crosshairData.p25.toFixed(4)}</span> P75 <span style={{ color: 'var(--foreground)' }}>{crosshairData.p75.toFixed(4)}</span></div>
          <div>P5 <span style={{ color: 'var(--foreground)' }}>{crosshairData.p5.toFixed(4)}</span> P95 <span style={{ color: 'var(--foreground)' }}>{crosshairData.p95.toFixed(4)}</span></div>
        </div>
      )}
      <div className="flex gap-3 mt-2" style={{ fontSize: 10, color: 'var(--muted-foreground)' }}>
        <span>95% band (5th–95th percentile)</span>
        <span>75% band (25th–75th percentile)</span>
        <span style={{ color: 'var(--chart-line)' }}>Median</span>
      </div>
    </div>
  )
}
