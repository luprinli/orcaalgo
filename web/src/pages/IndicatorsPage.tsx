import { useState, useEffect, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import {
  createChart, ColorType, CandlestickSeries, HistogramSeries, LineSeries,
  type IChartApi, type Time,
} from 'lightweight-charts'
import { useIndicatorStore } from '../stores/indicatorStore'
import { useIndicatorCompute } from '../hooks/useIndicator'
import { useCandleAggregation } from '../hooks/useCandleAggregation'
import { useTimeframeStore } from '../stores/timeframeStore'
import TimeframeChips from '../components/TimeframeChips'
import { symbols as symbolsApi, indicators as indicatorsApi, candles as candlesApi } from '../api/client'
import type { Candle, IndicatorSpec } from '../types/api'

const INDICATOR_IDS = ['sma', 'ema', 'rsi', 'macd', 'bbands', 'atr']
const DEFAULT_PARAMS: Record<string, Record<string, number | string>> = {
  sma: { period: 20, source: 'close' },
  ema: { period: 20, source: 'close' },
  rsi: { period: 14, source: 'close' },
  macd: { fast: 12, slow: 26, signal: 9, source: 'close' },
  bbands: { period: 20, std_dev: 2, source: 'close' },
  atr: { period: 14 },
}

export default function IndicatorsPage() {
  const { t } = useTranslation()
  const [symbols, setSymbols] = useState<Array<{ ticker: string; exchange: string; asset_type: string; id: number; is_active: boolean }>>([])
  const [selectedSymbol, setSelectedSymbol] = useState('')
  const [range, setRange] = useState('1M')
  const [candles, setCandles] = useState<Candle[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [specs, setSpecs] = useState<IndicatorSpec[]>([])
  const [isFullscreen, setIsFullscreen] = useState(false)

  const chartContainerRef = useRef<HTMLDivElement | null>(null)
  const fullscreenContainerRef = useRef<HTMLDivElement | null>(null)
  const chartRef = useRef<IChartApi | null>(null)
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const candlestickSeriesRef = useRef<any>(null)
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const volumeSeriesRef = useRef<any>(null)
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const indicatorSeriesRef = useRef<Map<string, any>>(new Map())

  const store = useIndicatorStore()
  const { timeframe } = useTimeframeStore()
  const { compute } = useIndicatorCompute()
  const aggregatedCandles = useCandleAggregation(candles, timeframe)

  const toggleFullscreen = useCallback(() => {
    if (!fullscreenContainerRef.current) return
    if (document.fullscreenElement) {
      document.exitFullscreen().catch(() => {})
    } else {
      fullscreenContainerRef.current.requestFullscreen().catch(() => {})
    }
  }, [])

  useEffect(() => {
    const handleFullscreenChange = () => {
      setIsFullscreen(!!document.fullscreenElement)
    }
    document.addEventListener('fullscreenchange', handleFullscreenChange)
    return () => document.removeEventListener('fullscreenchange', handleFullscreenChange)
  }, [])

  useEffect(() => {
    symbolsApi.list()
      .then(d => {
        const all = (d.symbols ?? []) as unknown as Array<{ ticker: string; exchange: string; asset_type: string; id: number; is_active: boolean }>
        setSymbols(all)
        if (!selectedSymbol && all.length > 0) {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const stoq = all.find((s: any) => s.exchange === 'STOOQ' && s.is_active)
          setSelectedSymbol(stoq ? stoq.ticker : all[0].ticker)
        }
      })
      .catch(() => {})
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    indicatorsApi.list()
      .then(d => {
        const list = d.indicators ?? []
        setSpecs(list)
      })
      .catch(() => {})
  }, [])

  useEffect(() => {
    if (!chartContainerRef.current) return
    const chart = createChart(chartContainerRef.current, {
      height: 500,
      layout: {
        background: { type: ColorType.Solid, color: '#1a1a2e' },
        textColor: '#d1d4dc',
      },
      grid: {
        vertLines: { color: '#2a2a3e' },
        horzLines: { color: '#2a2a3e' },
      },
      crosshair: {
        vertLine: { color: '#758696', width: 1, style: 2, labelBackgroundColor: '#758696' },
        horzLine: { color: '#758696', width: 1, style: 2, labelBackgroundColor: '#758696' },
      },
      timeScale: {
        borderColor: '#2a2a3e',
        timeVisible: true,
        secondsVisible: false,
        rightOffset: 5,
      },
      rightPriceScale: {
        borderColor: '#2a2a3e',
      },
      leftPriceScale: {
        borderColor: '#2a2a3e',
      },
    })

    chartRef.current = chart

    const cs = chart.addSeries(CandlestickSeries, {
      upColor: '#26a69a', downColor: '#ef5350',
      borderUpColor: '#26a69a', borderDownColor: '#ef5350',
      wickUpColor: '#26a69a', wickDownColor: '#ef5350',
    })
    candlestickSeriesRef.current = cs

    const vs = chart.addSeries(HistogramSeries, {
      color: '#26a69a',
      priceFormat: { type: 'volume' },
      priceScaleId: 'volume',
    })
    chart.priceScale('volume').applyOptions({
      scaleMargins: { top: 0.8, bottom: 0 },
    })
    volumeSeriesRef.current = vs

    const handleResize = () => {
      if (chartContainerRef.current) {
        chart.applyOptions({ width: chartContainerRef.current.clientWidth })
      }
    }
    window.addEventListener('resize', handleResize)

    return () => {
      window.removeEventListener('resize', handleResize)
      chart.remove()
      chartRef.current = null
    }
  }, [])

  const renderChartData = useCallback((data: Candle[]) => {
    if (!data.length) return
    const cs = candlestickSeriesRef.current
    const vs = volumeSeriesRef.current
    if (!cs || !vs) return

    const candleData = data.map((c) => ({
      time: (new Date(c.time).getTime() / 1000) as Time,
      open: c.open, high: c.high, low: c.low, close: c.close,
    }))
    const volumeData = data.map((c) => ({
      time: (new Date(c.time).getTime() / 1000) as Time,
      value: c.volume,
      color: c.close >= c.open ? 'rgba(38,166,154,0.3)' : 'rgba(239,83,80,0.3)',
    }))

    cs.setData(candleData)
    vs.setData(volumeData)
    chartRef.current?.timeScale().fitContent()
  }, [])

  const renderIndicatorOnChart = useCallback((indicator: ReturnType<typeof store.all>[0]) => {
    const chart = chartRef.current
    if (!chart || !indicator.result) return

    const spec = indicator.spec
    const result = indicator.result
    const paneIndex = indicator.paneIndex

    for (const output of spec.outputs) {
      const seriesKey = `${indicator._id}_${output.name}`
      const plotOpts = output.plotOptions ?? { color: '#ffffff', lineWidth: 2 }

      const existing = indicatorSeriesRef.current.get(seriesKey)

      if (existing) {
        const points = result.data
          .filter(p => p.values[output.name] !== undefined && isFinite(p.values[output.name]))
          .map(p => ({ time: p.time as Time, value: p.values[output.name] }))
        existing.setData(points)
        continue
      }

      const seriesOptions: Record<string, unknown> = {
        color: plotOpts.color ?? '#ffffff',
        lineWidth: plotOpts.lineWidth ?? 2,
        lastValueVisible: true,
        priceLineVisible: false,
      }

      if (plotOpts.precision !== undefined) {
        seriesOptions.priceFormat = { type: 'price', precision: plotOpts.precision, minMove: plotOpts.minMove ?? 0.01 }
      }

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      let series: any
      if (output.type === 'histogram') {
        series = chart.addSeries(HistogramSeries, { color: plotOpts.color ?? '#ffffff' }, paneIndex)
        const points = result.data
          .filter(p => p.values[output.name] !== undefined && isFinite(p.values[output.name]))
          .map(p => ({ time: p.time as Time, value: p.values[output.name] }))
        series.setData(points)
      } else {
        series = chart.addSeries(LineSeries, seriesOptions, paneIndex)
        const points = result.data
          .filter(p => p.values[output.name] !== undefined && isFinite(p.values[output.name]))
          .map(p => ({ time: p.time as Time, value: p.values[output.name] }))
        series.setData(points)
      }

      indicatorSeriesRef.current.set(seriesKey, series)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const fetchCandles = useCallback(async () => {
    if (!selectedSymbol) return
    setLoading(true)
    setError(null)
    try {
      const d = await candlesApi.get(selectedSymbol, range)
      setCandles(d.candles ?? [])
    } catch (err) {
      setError(err instanceof Error ? err.message : t('indicators:failedToLoad', 'Failed to load candles'))
    } finally {
      setLoading(false)
    }
  }, [selectedSymbol, range])

  useEffect(() => {
    fetchCandles()
  }, [fetchCandles])

  useEffect(() => {
    if (aggregatedCandles.length === 0) return
    renderChartData(aggregatedCandles)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [aggregatedCandles])

  useEffect(() => {
    if (aggregatedCandles.length === 0) return
    for (const indicator of store.all()) {
      compute(indicator._id, aggregatedCandles)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [aggregatedCandles, store.indicators.size])

  useEffect(() => {
    for (const indicator of store.all()) {
      if (indicator.result) {
        renderIndicatorOnChart(indicator)
      }
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [store.indicators])

  const addIndicator = (id: string) => {
    const spec = specs.find(s => s.id === id)
    if (!spec) return
    const params = DEFAULT_PARAMS[id] ?? {}
    const _id = store.addIndicator(spec, params)
    compute(_id, aggregatedCandles)
  }

  const removeIndicator = (id: string) => {
    const indicator = store.getById(id)
    if (!indicator) return

    for (const output of indicator.spec.outputs) {
      const seriesKey = `${id}_${output.name}`
      const series = indicatorSeriesRef.current.get(seriesKey)
      if (series) {
        chartRef.current?.removeSeries(series)
        indicatorSeriesRef.current.delete(seriesKey)
      }
    }
    store.removeIndicator(id)
  }

  const activeIds = store.all().map(i => i._id)
  const activeSpecIds = store.all().map(i => i.spec.id)
  const latestValues: Record<string, number> = {}
  for (const i of store.all()) {
    if (i.result?.data.length) {
      const last = i.result.data[i.result.data.length - 1]
      for (const [k, v] of Object.entries(last.values)) {
        latestValues[`${i.spec.id}_${k}`] = v
      }
    }
  }

  return (
    <div>
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>{t('indicators:title', 'Indicators')}</h1>
        <div className="flex gap-2">
          <select className="input" style={{ width: 100 }} value={range} onChange={e => setRange(e.target.value)}>
            <option value="1D">{t('indicators:range.1D', '1 Day')}</option><option value="1W">{t('indicators:range.1W', '1 Week')}</option><option value="1M">{t('indicators:range.1M', '1 Month')}</option><option value="3M">{t('indicators:range.3M', '3 Months')}</option><option value="1Y">{t('indicators:range.1Y', '1 Year')}</option>
          </select>
          <TimeframeChips variant="toolbar" />
          <button className="btn btn-primary" onClick={fetchCandles} disabled={loading}>
            {loading ? t('indicators:loading', 'Loading...') : t('indicators:refresh', 'Refresh')}
          </button>
        </div>
      </div>

      {error && (
        <div className="card mb-4" style={{ background: 'rgba(218,54,51,.1)', border: '1px solid var(--danger)' }}>
          <span style={{ color: 'var(--danger)' }}>{error}</span>
        </div>
      )}

      <div className="card mb-4">
        <h2 className="mb-3">{t('indicators:addIndicator', 'Add Indicator')}</h2>
        <div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
          {INDICATOR_IDS.map(id => {
            const spec = specs.find(s => s.id === id)
            const isActive = activeSpecIds.includes(id)
            return (
              <button
                key={id}
                className={`btn ${isActive ? 'btn-primary' : 'btn-outline'}`}
                onClick={() => isActive ? null : addIndicator(id)}
                disabled={isActive}
                title={spec?.description ?? ''}
              >
                {spec?.name ?? id.toUpperCase()}
              </button>
            )
          })}
        </div>
      </div>

      {activeIds.length > 0 && (
        <div className="card mb-4">
          <h2 className="mb-3">{t('indicators:activeOverlays', 'Active Overlays ({{n}})', { n: activeIds.length })}</h2>
          <div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
            {store.all().map(i => (
              <div key={i._id} className="flex gap-2" style={{ alignItems: 'center', padding: '4px 8px', background: 'var(--bg-hover)', borderRadius: 6 }}>
                {i.spec.outputs.map(o => (
                  <span key={o.name} style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 12 }}>
                    <span style={{ display: 'inline-block', width: 10, height: 10, borderRadius: '50%', background: o.plotOptions?.color ?? '#fff' }} />
                    {o.name}
                  </span>
                ))}
                <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>{i.spec.name}</span>
                <button className="btn btn-outline" style={{ fontSize: 10, padding: '1px 6px' }} onClick={() => removeIndicator(i._id)}>
                  {t('indicators:remove', 'Remove')}
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      <div ref={fullscreenContainerRef} style={{ width: '100%', borderRadius: 8, overflow: 'hidden', position: 'relative', background: isFullscreen ? 'var(--chart-bg)' : undefined }}>
        <button
          onClick={toggleFullscreen}
          title={isFullscreen ? t('indicators:exitFullscreen', 'Exit fullscreen') : t('indicators:fullscreen', 'Fullscreen')}
          style={{
            position: 'absolute', top: 8, right: 8, zIndex: 10,
            background: 'transparent', border: 'none', color: '#d1d4dc',
            opacity: 0.7, cursor: 'pointer', fontSize: 18, padding: '2px 4px',
            lineHeight: 1, fontFamily: 'sans-serif',
          }}
          onMouseEnter={e => (e.currentTarget.style.opacity = '1')}
          onMouseLeave={e => (e.currentTarget.style.opacity = '0.7')}
        >
          {isFullscreen ? '\u2715' : '\u26F6'}
        </button>
        <div ref={chartContainerRef} style={{ width: '100%' }} />
      </div>

      <div className="card mt-4">
        <div className="flex-between mb-3">
          <h2 style={{ margin: 0 }}>{t('indicators:symbols', 'Symbols ({{n}})', { n: symbols.length })}</h2>
        </div>
        <div style={{ maxHeight: 300, overflowY: 'auto' }}>
          <table className="data-table">
            <thead><tr><th>{t('indicators:table.ticker', 'Ticker')}</th><th>{t('indicators:table.exchange', 'Exchange')}</th><th>{t('indicators:table.type', 'Type')}</th><th>{t('indicators:table.active', 'Active')}</th></tr></thead>
            <tbody>
              {symbols.map(s => (
                <tr key={s.id}
                  style={{ cursor: 'pointer', background: s.ticker === selectedSymbol ? 'var(--bg-hover)' : undefined }}
                  onClick={() => {
                    store.clearAll()
                    indicatorSeriesRef.current.forEach(series => chartRef.current?.removeSeries(series))
                    indicatorSeriesRef.current.clear()
                    setSelectedSymbol(s.ticker)
                  }}>
                  <td><strong>{s.ticker}</strong></td>
                  <td style={{ color: s.exchange === 'STOOQ' ? 'var(--success)' : 'var(--text-secondary)', fontSize: 12 }}>
                    {s.exchange}
                  </td>
                  <td>{s.asset_type}</td>
                  <td><span className={`badge ${s.is_active ? 'badge-ok' : 'badge-err'}`}>{s.is_active ? t('common:active', 'Active') : t('common:inactive', 'Inactive')}</span></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
