import { createContext, useContext, useState, useEffect, useCallback, type ReactNode, type Dispatch, type SetStateAction } from 'react'
import { useWebSocketMulti } from '../hooks/useWebSocket'
import { useIndicatorStore } from '../stores/indicatorStore'
import { restoreIndicators } from '../stores/indicatorStore'
import { useIndicatorCompute } from '../hooks/useIndicator'
import { useCandleAggregation } from '../hooks/useCandleAggregation'
import { useTimeframeStore } from '../stores/timeframeStore'
import { tradesToMarkers } from '../charts/MarkerManager'
import { indicators as indicatorsApi, live, candles as candlesApi } from '../api/client'
import type { Candle, IndicatorSpec, TradeSummary } from '../types/api'
import type { Time, SeriesMarker } from 'lightweight-charts'

// eslint-disable-next-line react-refresh/only-export-components
export const INDICATOR_IDS = ['sma', 'ema', 'rsi', 'macd', 'bbands', 'atr']

export interface TradingViewState {
  selectedSymbol: string
  setSelectedSymbol: (s: string) => void
  candles: Candle[]
  trades: TradeSummary[]
  chartMarkers: SeriesMarker<Time>[]
  specs: IndicatorSpec[]
  modalSpec: IndicatorSpec | null
  modalDefaults: Record<string, number | string>
  watchlistOpen: boolean
  setWatchlistOpen: Dispatch<SetStateAction<boolean>>
  aggregatedCandles: Candle[]
  timeframe: string
  setTimeframe: (tf: string) => void
  wsConnected: boolean
  addIndicator: (spec: IndicatorSpec, params: Record<string, number | string>) => void
  removeIndicator: (id: string) => void
  handleApplyIndicator: (params: Record<string, number | string>) => void
  openModal: (id: string) => void
  cancelModal: () => void
}

const TradingViewContext = createContext<TradingViewState | null>(null)

// eslint-disable-next-line react-refresh/only-export-components
export function useTradingView() {
  const ctx = useContext(TradingViewContext)
  if (!ctx) throw new Error('useTradingView must be used within TradingViewProvider')
  return ctx
}

interface TradingViewProviderProps {
  children: ReactNode
  initialSymbol?: string
  token: string | null
  wsChannels?: string[]
  onWSMessage?: (data: unknown, channel: string) => void
}

export function TradingViewProvider({
  children,
  initialSymbol = 'SPY',
  token: _token,
  wsChannels = ['risk', 'performance'],
  onWSMessage,
}: TradingViewProviderProps) {
  const [selectedSymbol, setSelectedSymbol] = useState(initialSymbol)
  const [candles, setCandles] = useState<Candle[]>([])
  const [trades, setTrades] = useState<TradeSummary[]>([])
  const [chartMarkers, setChartMarkers] = useState<SeriesMarker<Time>[]>([])
  const [specs, setSpecs] = useState<IndicatorSpec[]>([])
  const [modalSpec, setModalSpec] = useState<IndicatorSpec | null>(null)
  const [modalDefaults, setModalDefaults] = useState<Record<string, number | string>>({})
  const [watchlistOpen, setWatchlistOpen] = useState(false)

  const indicatorStore = useIndicatorStore()
  const { timeframe, setTimeframe } = useTimeframeStore()
  const { compute } = useIndicatorCompute()
  const aggregatedCandles = useCandleAggregation(candles, timeframe)

  const { connected: wsConnected } = useWebSocketMulti(wsChannels, {
    onMessage: (data, channel) => {
      onWSMessage?.(data, channel)
    },
    maxReconnects: 20,
    reconnectInterval: 2000,
  })

  useEffect(() => {
    indicatorsApi.list()
      .then(d => {
        const specs = d.indicators ?? []
        setSpecs(specs)
        if (!restoreIndicators(specs)) {
          useIndicatorStore.getState().clearAll()
        }
      })
      .catch(() => {})
  }, [])

  const fetchCandles = useCallback(async () => {
    try {
      const data = await candlesApi.get(selectedSymbol, '1W')
      setCandles(data.candles ?? [])
    } catch { /* ignore */ }
  }, [selectedSymbol])

  const fetchTrades = useCallback(async () => {
    try {
      const data = await live.trades(1, 50)
      const tradeList = data.trades as TradeSummary[]
      setTrades(tradeList)
      setChartMarkers(tradesToMarkers(tradeList))
    } catch { /* ignore */ }
  }, [])

  useEffect(() => {
    fetchCandles()
    fetchTrades()
    const interval = setInterval(() => {
      fetchCandles()
      fetchTrades()
    }, 60000)
    return () => clearInterval(interval)
  }, [fetchCandles, fetchTrades])

  useEffect(() => {
    if (aggregatedCandles.length === 0) return
    for (const indicator of indicatorStore.all()) {
      compute(indicator._id, aggregatedCandles)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [aggregatedCandles, indicatorStore.indicators.size])

  const addIndicator = useCallback((spec: IndicatorSpec, params: Record<string, number | string>) => {
    const _id = indicatorStore.addIndicator(spec, params)
    compute(_id, aggregatedCandles)
  }, [indicatorStore, compute, aggregatedCandles])

  const removeIndicator = useCallback((id: string) => {
    indicatorStore.removeIndicator(id)
  }, [indicatorStore])

  const handleApplyIndicator = useCallback((params: Record<string, number | string>) => {
    if (!modalSpec) return
    addIndicator(modalSpec, params)
    setModalSpec(null)
  }, [modalSpec, addIndicator])

  const openModal = useCallback((id: string) => {
    const spec = specs.find(s => s.id === id)
    if (!spec) return
    const defaults: Record<string, number | string> = {}
    for (const p of spec.parameters) {
      defaults[p.name] = p.default
    }
    setModalDefaults(defaults)
    setModalSpec(spec)
  }, [specs])

  const cancelModal = useCallback(() => {
    setModalSpec(null)
  }, [])

  const value: TradingViewState = {
    selectedSymbol,
    setSelectedSymbol: (s: string) => { indicatorStore.clearAll(); setSelectedSymbol(s) },
    candles,
    trades,
    chartMarkers,
    specs,
    modalSpec,
    modalDefaults,
    watchlistOpen,
    setWatchlistOpen,
    aggregatedCandles,
    timeframe,
    setTimeframe,
    wsConnected,
    addIndicator,
    removeIndicator,
    handleApplyIndicator,
    openModal,
    cancelModal,
  }

  return (
    <TradingViewContext.Provider value={value}>
      {children}
    </TradingViewContext.Provider>
  )
}
