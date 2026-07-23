export function createHistogramBins(
  values: number[],
  numBins: number = 20,
): { categories: string[]; data: number[] } {
  if (!values || values.length === 0) return { categories: [], data: [] }

  const numericValues = values.filter((v) => typeof v === 'number' && isFinite(v))
  if (numericValues.length === 0) return { categories: [], data: [] }

  const minVal = Math.min(...numericValues)
  let maxVal = Math.max(...numericValues)
  if (minVal === maxVal) {
    return {
      categories: [minVal.toFixed(2)],
      data: [numericValues.length],
    }
  }

  maxVal = maxVal + (maxVal - minVal) * 0.001
  const binSize = (maxVal - minVal) / numBins
  const categories: string[] = []
  const data: number[] = Array(numBins).fill(0)

  for (let i = 0; i < numBins; i++) {
    const lowerBound = minVal + i * binSize
    const upperBound = minVal + (i + 1) * binSize
    categories.push(`${lowerBound.toFixed(2)} to ${upperBound.toFixed(2)}`)
  }

  for (const value of numericValues) {
    let binIndex = Math.floor((value - minVal) / binSize)
    if (binIndex >= numBins) binIndex = numBins - 1
    if (binIndex < 0) binIndex = 0
    data[binIndex]++
  }

  return { categories, data }
}

interface PlotlyBarTrace {
  x: string[]
  y: number[]
  type: 'bar'
  marker?: { color?: string }
}

export function buildHistogramTraces(
  allPnlPct: number[],
  allMaxDDPct: number[],
  numBins: number = 20,
): { pnlTrace: PlotlyBarTrace; ddTrace: PlotlyBarTrace } {
  const pnlHist = createHistogramBins(allPnlPct, numBins)
  const ddHist = createHistogramBins(allMaxDDPct, numBins)

  return {
    pnlTrace: {
      x: pnlHist.categories,
      y: pnlHist.data,
      type: 'bar',
    },
    ddTrace: {
      x: ddHist.categories,
      y: ddHist.data,
      type: 'bar',
    },
  }
}
