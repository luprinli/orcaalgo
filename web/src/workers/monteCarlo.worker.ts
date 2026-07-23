import { mulberry32 } from '../lib/rng'

interface MCInput {
  returns: number[]
  simulations: number
  forwardDays: number
  seed?: number
}

interface MCPercentiles {
  p5: number[]
  p25: number[]
  p50: number[]
  p75: number[]
  p95: number[]
}

interface MCStats {
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

interface MCResult {
  p5: number[]
  p25: number[]
  p50: number[]
  p75: number[]
  p95: number[]
  allPnlPct: number[]
  allMaxDDPct: number[]
  stats: MCStats
}

function sampleWithReplacement(returns: number[], count: number, rng: () => number): number[] {
  const result: number[] = []
  for (let i = 0; i < count; i++) {
    const idx = Math.floor(rng() * returns.length)
    result.push(returns[idx])
  }
  return result
}

function computePercentiles(paths: number[][], steps: number): MCPercentiles {
  const p5: number[] = []
  const p25: number[] = []
  const p50: number[] = []
  const p75: number[] = []
  const p95: number[] = []

  for (let s = 0; s < steps; s++) {
    const values = paths.map((p) => p[s])
    values.sort((a, b) => a - b)
    const idx = (v: number) => Math.round((v / 100) * (values.length - 1))
    p5.push(values[idx(5)])
    p25.push(values[idx(25)])
    p50.push(values[idx(50)])
    p75.push(values[idx(75)])
    p95.push(values[idx(95)])
  }
  return { p5, p25, p50, p75, p95 }
}

function computeStats(allPnlPct: number[], allMaxDDPct: number[], simulations: number, forwardDays: number): MCStats {
  const stats: MCStats = {
    numSimulations: simulations,
    numDays: forwardDays,
    avgPnlPct: 0,
    medianPnlPct: 0,
    p5PnlPct: 0,
    p10PnlPct: 0,
    avgMaxDDPct: 0,
    medianMaxDDPct: 0,
    p95MaxDDPct: 0,
    bustProbability: 0,
  }

  if (allPnlPct.length > 0) {
    const sorted = [...allPnlPct].sort((a, b) => a - b)
    stats.avgPnlPct = allPnlPct.reduce((a, b) => a + b, 0) / allPnlPct.length
    stats.medianPnlPct = sorted[Math.floor(sorted.length / 2)]
    stats.p5PnlPct = sorted[Math.floor(sorted.length * 0.05)]
    stats.p10PnlPct = sorted[Math.floor(sorted.length * 0.1)]
    stats.bustProbability = sorted.filter((v) => v < 0).length / sorted.length
  }

  if (allMaxDDPct.length > 0) {
    const valid = allMaxDDPct.filter((dd) => dd < 100)
    if (valid.length > 0) {
      const sorted = [...valid].sort((a, b) => a - b)
      stats.avgMaxDDPct = valid.reduce((a, b) => a + b, 0) / valid.length
      stats.medianMaxDDPct = sorted[Math.floor(sorted.length / 2)]
      stats.p95MaxDDPct = sorted[Math.floor(sorted.length * 0.95)]
    }
  }

  return stats
}

self.onmessage = (e: MessageEvent<MCInput>) => {
  const { returns, simulations, forwardDays, seed } = e.data
  if (returns.length < 2) {
    const empty: MCResult = {
      p5: [], p25: [], p50: [], p75: [], p95: [],
      allPnlPct: [], allMaxDDPct: [],
      stats: computeStats([], [], simulations, forwardDays),
    }
    self.postMessage(empty)
    return
  }

  const rng = seed != null ? mulberry32(seed) : mulberry32(Date.now())
  const paths: number[][] = []
  const allPnlPct: number[] = []
  const allMaxDDPct: number[] = []
  const startEquity = 1

  for (let i = 0; i < simulations; i++) {
    const sampled = sampleWithReplacement(returns, forwardDays, rng)
    const path: number[] = []
    let equity = startEquity
    let peak = startEquity
    let maxDD = 0

    for (const r of sampled) {
      equity *= 1 + r
      path.push(equity)
      if (equity > peak) peak = equity
      const dd = peak > 0 ? ((peak - equity) / peak) * 100 : 0
      if (dd > maxDD) maxDD = dd
    }

    paths.push(path)

    const finalPnlPct = ((equity - startEquity) / startEquity) * 100
    allPnlPct.push(finalPnlPct)
    allMaxDDPct.push(maxDD)
  }

  const percentiles = computePercentiles(paths, forwardDays)
  const stats = computeStats(allPnlPct, allMaxDDPct, simulations, forwardDays)

  const result: MCResult = {
    ...percentiles,
    allPnlPct,
    allMaxDDPct,
    stats,
  }
  self.postMessage(result)
}
