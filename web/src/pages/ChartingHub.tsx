import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { candles, indicators as indicatorsApi, symbols as symbolsApi } from '../api/client'
import { useWebSocket } from '../hooks/useWebSocket'
import LiveMonitorChart from '../charts/LiveMonitorChart'
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from '../components/ui/card'
import { Badge } from '../components/ui/badge'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/table'
import type { Candle, IndicatorSpec } from '../types/api'
import type { WSTickData } from '../types/ws'

export default function ChartingHub() {
  const { t } = useTranslation()
  const [symbol, setSymbol] = useState('SPY')
  const [range, setRange] = useState('1D')
  const [candleData, setCandleData] = useState<Candle[]>([])
  const [indicatorSpecs, setIndicatorSpecs] = useState<IndicatorSpec[]>([])
  const [symbolList, setSymbolList] = useState<string[]>(['SPY', 'AAPL', 'MSFT', 'GOOGL', 'TSLA', 'AMZN'])
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

  useEffect(() => { fetchCandles() }, [fetchCandles])

  useEffect(() => {
    indicatorsApi.list()
      .then(d => setIndicatorSpecs(d.indicators ?? []))
      .catch(() => {})
    symbolsApi.list()
      .then(s => { if (s?.length) setSymbolList((s as {ticker: string}[]).map(x => x.ticker)) })
      .catch(() => {})
  }, [])

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold mb-0">{t('chartingHub:title', 'Charting Hub')}</h1>
        <Badge variant={connected ? 'default' : 'destructive'}>
          {connected ? t('chartingHub:live', 'Live') : t('chartingHub:offline', 'Offline')}
        </Badge>
      </div>

      <LiveMonitorChart
        candles={candleData}
        symbol={symbol}
        range={range}
        symbols={symbolList}
        onSymbolChange={setSymbol}
        onRangeChange={setRange}
        onLoad={fetchCandles}
        indicatorSpecs={indicatorSpecs}
        loading={loading}
        error={error}
        height={500}
      />

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
    </div>
  )
}
