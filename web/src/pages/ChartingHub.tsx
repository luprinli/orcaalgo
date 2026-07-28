import { useState, useEffect, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import {
  createChart, ColorType, CandlestickSeries, HistogramSeries, LineSeries,
  type IChartApi, type Time,
} from 'lightweight-charts'
import { candles, indicators as indicatorsApi } from '../api/client'
import { useWebSocket } from '../hooks/useWebSocket'
import { useIndicatorStore } from '../stores/indicatorStore'
import { useIndicatorCompute } from '../hooks/useIndicator'
import { useCandleAggregation } from '../hooks/useCandleAggregation'
import { useTimeframeStore } from '../stores/timeframeStore'
import CandlesChart from '../charts/CandlesChart'
import TimeframeChips from '../components/TimeframeChips'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../components/ui/tabs'
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Badge } from '../components/ui/badge'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '../components/ui/select'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/table'
import type { Candle, IndicatorSpec } from '../types/api'
import type { WSTickData } from '../types/ws'

const RANGES = [
  { label: '1D', value: '1D' },
  { label: '1W', value: '1W' },
  { label: '1M', value: '1M' },
  { label: '3M', value: '3M' },
  { label: '1Y', value: '1Y' },
  { label: 'ALL', value: 'ALL' },
]

const INDICATOR_IDS = ['sma', 'ema', 'rsi', 'macd', 'bbands', 'atr']
const DEFAULT_PARAMS: Record<string, Record<string, number | string>> = {
  sma: { period: 20, source: 'close' },
  ema: { period: 20, source: 'close' },
  rsi: { period: 14, source: 'close' },
  macd: { fast: 12, slow: 26, signal: 9, source: 'close' },
  bbands: { period: 20, std_dev: 2, source: 'close' },
  atr: { period: 14 },
}

export default function ChartingHub() {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState('candles')
  const [symbol, setSymbol] = useState('SPY')
  const [range, setRange] = useState('1D')
  const [candleData, setCandleData] = useState<Candle[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const [ticks, setTicks] = useState<WSTickData[]>([])
  const { connected } = useWebSocket({
    channels: ['ticks'],
    onMessage: (data, channel) => {
      if (channel === 'ticks') {
        const tick = data as WSTickData
        setTicks((prev) => [tick, ...prev].slice(0, 100))
      }
    },
  })

  const fetchCandles = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await candles.get(symbol, range)
      setCandleData(res.candles ?? [])
    } catch (err) {
      setError(err instanceof Error ? err.message : t('chartingHub:failedToLoad', 'Failed to load candles'))
    } finally {
      setLoading(false)
    }
  }, [symbol, range, t])

  useEffect(() => {
    fetchCandles()
  }, [fetchCandles])

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold mb-0">{t('chartingHub:title', 'Charting Hub')}</h1>
        <div className="flex gap-2 items-center">
          <Badge variant={connected ? 'default' : 'destructive'}>
            {connected ? t('chartingHub:live', 'Live') : t('chartingHub:offline', 'Offline')}
          </Badge>
          <Input
            className="w-[120px]"
            placeholder={t('chartingHub:symbol', 'Symbol')}
            value={symbol}
            onChange={(e) => setSymbol(e.target.value.toUpperCase())}
            onKeyDown={(e) => e.key === 'Enter' && fetchCandles()}
          />
          <Select value={range} onValueChange={setRange}>
            <SelectTrigger className="w-[80px]"><SelectValue /></SelectTrigger>
            <SelectContent>
              {RANGES.map((r) => (
                <SelectItem key={r.value} value={r.value}>{r.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button onClick={fetchCandles} disabled={loading}>
            {loading ? t('chartingHub:loading', 'Loading...') : t('chartingHub:load', 'Load')}
          </Button>
        </div>
      </div>

      {error && (
        <Card className="mb-4 border-destructive border-l-4">
          <CardContent className="text-destructive text-sm pt-4">{error}</CardContent>
        </Card>
      )}

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList variant="line">
          <TabsTrigger value="candles" variant="line">{t('chartingHub:candles', 'Candles')}</TabsTrigger>
          <TabsTrigger value="indicators" variant="line">{t('chartingHub:indicators', 'Indicators')}</TabsTrigger>
        </TabsList>

        <TabsContent value="candles">
          {candleData.length > 0 && (
            <CandlesChart data={candleData} height={400} title={`${symbol} — ${range}`} />
          )}
          <Card className="mt-4">
            <CardHeader><CardTitle>{t('chartingHub:recentTicks', 'Recent Ticks')}</CardTitle></CardHeader>
            <CardContent>
              {ticks.length === 0 ? (
                <CardDescription>{t('chartingHub:waitingForTicks', 'Waiting for tick data...')}</CardDescription>
              ) : (
                <div className="overflow-auto max-h-[300px]">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('chartingHub:table.symbol', 'Symbol')}</TableHead>
                        <TableHead>{t('chartingHub:table.price', 'Price')}</TableHead>
                        <TableHead>{t('chartingHub:table.volume', 'Volume')}</TableHead>
                        <TableHead>{t('chartingHub:table.side', 'Side')}</TableHead>
                        <TableHead>{t('chartingHub:table.time', 'Time')}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {ticks.slice(0, 50).map((t, i) => (
                        <TableRow key={i}>
                          <TableCell>{t.symbol}</TableCell>
                          <TableCell>${t.price?.toFixed(2)}</TableCell>
                          <TableCell>{t.volume}</TableCell>
                          <TableCell className={t.side === 'BUY' ? 'text-trading-success' : 'text-trading-danger'}>{t.side}</TableCell>
                          <TableCell className="text-[11px]">{t.time ? new Date(t.time).toLocaleTimeString() : '--'}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="indicators">
          <IndicatorsPanel
            symbol={symbol}
            candleData={candleData}
            activeTab={activeTab}
          />
        </TabsContent>
      </Tabs>
    </div>
  )
}

function IndicatorsPanel({
  symbol,
  candleData,
  activeTab,
}: {
  symbol: string
  candleData: Candle[]
  activeTab: string
}) {
  const { t } = useTranslation()
  const [specs, setSpecs] = useState<IndicatorSpec[]>([])
  const [isFullscreen, setIsFullscreen] = useState(false)

  const chartContainerRef = useRef<HTMLDivElement | null>(null)
  const fullscreenContainerRef = useRef<HTMLDivElement | null>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const candlestickSeriesRef = useRef<any>(null)
  const volumeSeriesRef = useRef<any>(null)
  const indicatorSeriesRef = useRef<Map<string, any>>(new Map())

  const store = useIndicatorStore()
  const { timeframe } = useTimeframeStore()
  const { compute } = useIndicatorCompute()
  const aggregatedCandles = useCandleAggregation(candleData, timeframe)

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
    indicatorsApi.list()
      .then(d => { setSpecs(d.indicators ?? []) })
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
      rightPriceScale: { borderColor: '#2a2a3e' },
      leftPriceScale: { borderColor: '#2a2a3e' },
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
      if (chartContainerRef.current && chartRef.current) {
        const { clientWidth, clientHeight } = chartContainerRef.current
        chartRef.current.resize(clientWidth, clientHeight)
      }
    }
    window.addEventListener('resize', handleResize)

    return () => {
      window.removeEventListener('resize', handleResize)
      chart.remove()
      chartRef.current = null
    }
  }, [])

  useEffect(() => {
    if (activeTab === 'indicators') {
      const id = requestAnimationFrame(() => {
        if (chartContainerRef.current && chartRef.current) {
          const { clientWidth, clientHeight } = chartContainerRef.current
          chartRef.current.resize(clientWidth, clientHeight)
        }
      })
      return () => cancelAnimationFrame(id)
    }
  }, [activeTab])

  const renderChartData = useCallback((data: Candle[]) => {
    if (!data.length) return
    const cs = candlestickSeriesRef.current
    const vs = volumeSeriesRef.current
    if (!cs || !vs) return

    const candleDataArr = data.map((c) => ({
      time: (new Date(c.time).getTime() / 1000) as Time,
      open: c.open, high: c.high, low: c.low, close: c.close,
    }))
    const volumeDataArr = data.map((c) => ({
      time: (new Date(c.time).getTime() / 1000) as Time,
      value: c.volume,
      color: c.close >= c.open ? 'rgba(38,166,154,0.3)' : 'rgba(239,83,80,0.3)',
    }))
    cs.setData(candleDataArr)
    vs.setData(volumeDataArr)
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

      let series: any
      const points = result.data
        .filter(p => p.values[output.name] !== undefined && isFinite(p.values[output.name]))
        .map(p => ({ time: p.time as Time, value: p.values[output.name] }))

      if (output.type === 'histogram') {
        series = chart.addSeries(HistogramSeries, { color: plotOpts.color ?? '#ffffff' }, paneIndex)
      } else {
        series = chart.addSeries(LineSeries, seriesOptions, paneIndex)
      }
      series.setData(points)
      indicatorSeriesRef.current.set(seriesKey, series)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

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

  const activeSpecIds = store.all().map(i => i.spec.id)

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <div />
        <div className="flex gap-2 items-center">
          <TimeframeChips variant="toolbar" />
        </div>
      </div>

      <Card className="mb-4">
        <CardHeader>
          <CardTitle>{t('chartingHub:addIndicator', 'Add Indicator')}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex gap-2 flex-wrap">
            {INDICATOR_IDS.map(id => {
              const spec = specs.find(s => s.id === id)
              const isActive = activeSpecIds.includes(id)
              return (
                <Button
                  key={id}
                  variant={isActive ? 'default' : 'outline'}
                  onClick={() => isActive ? null : addIndicator(id)}
                  disabled={isActive}
                  title={spec?.description ?? ''}
                >
                  {spec?.name ?? id.toUpperCase()}
                </Button>
              )
            })}
          </div>
        </CardContent>
      </Card>

      {store.all().length > 0 && (
        <Card className="mb-4">
          <CardHeader>
            <CardTitle>{t('chartingHub:activeOverlays', 'Active Overlays ({{n}})', { n: store.all().length })}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex gap-2 flex-wrap">
              {store.all().map(i => (
                <div key={i._id} className="flex gap-2 items-center px-2 py-1 rounded-md bg-muted">
                  {i.spec.outputs.map(o => (
                    <span key={o.name} className="flex items-center gap-1 text-xs">
                      <span className="inline-block w-2.5 h-2.5 rounded-full" style={{ background: o.plotOptions?.color ?? '#fff' }} />
                      {o.name}
                    </span>
                  ))}
                  <span className="text-xs text-muted-foreground">{i.spec.name}</span>
                  <Button variant="outline" size="sm" className="text-[10px] px-1.5 py-0 h-5" onClick={() => removeIndicator(i._id)}>
                    {t('chartingHub:remove', 'Remove')}
                  </Button>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      <div ref={fullscreenContainerRef} className="w-full rounded-lg overflow-hidden relative" style={{ background: isFullscreen ? '#1a1a2e' : undefined }}>
        <button
          onClick={toggleFullscreen}
          title={isFullscreen ? t('chartingHub:exitFullscreen', 'Exit fullscreen') : t('chartingHub:fullscreen', 'Fullscreen')}
          className="absolute top-2 right-2 z-10 bg-transparent border-0 text-[#d1d4dc] opacity-70 cursor-pointer text-lg px-1 py-0.5 leading-none font-sans hover:opacity-100"
        >
          {isFullscreen ? '\u2715' : '\u26F6'}
        </button>
        <div ref={chartContainerRef} className="w-full" />
      </div>
    </div>
  )
}
