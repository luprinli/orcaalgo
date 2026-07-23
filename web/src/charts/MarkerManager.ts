import type { Time, SeriesMarker } from 'lightweight-charts'

export interface TradeMarker {
  time: number
  side: 'BUY' | 'SELL'
  price: number
  label?: string
}

function formatTime(iso: string): number {
  return Math.floor(new Date(iso).getTime() / 1000)
}

export function tradesToMarkers(trades: Array<{
  entry_time?: string
  exit_time?: string
  side?: string
  entry_price?: number
  exit_price?: number
  symbol?: string
}>): SeriesMarker<Time>[] {
  const markers: SeriesMarker<Time>[] = []

  for (const trade of trades) {
    if (trade.entry_time && trade.entry_price) {
      const side = trade.side?.toUpperCase()
      markers.push({
        time: formatTime(trade.entry_time) as Time,
        position: side === 'SELL' ? 'belowBar' : 'aboveBar',
        color: side === 'SELL' ? '#ef5350' : '#26a69a',
        shape: side === 'SELL' ? 'arrowDown' : 'arrowUp',
        text: side === 'SELL' ? 'S' : 'B',
        size: 2,
      })
    }

    if (trade.exit_time && trade.exit_price) {
      markers.push({
        time: formatTime(trade.exit_time) as Time,
        position: 'inBar',
        color: '#758696',
        shape: 'circle',
        text: 'X',
        size: 1,
      })
    }
  }

  return markers.sort((a, b) => (a.time as number) - (b.time as number))
}

export function ordersToMarkers(orders: Array<{
  created_at?: string
  side?: string
  price?: number | null
  symbol?: string
  state?: string
}>): SeriesMarker<Time>[] {
  return orders
    .filter((o) => o.created_at)
    .map((o) => {
      const side = o.side?.toUpperCase()
      return {
        time: formatTime(o.created_at!) as Time,
        position: 'inBar' as const,
        color: o.state === 'filled' ? '#4CAF50' : '#FF9800',
        shape: 'circle' as const,
        text: side === 'SELL' ? 'SELL' : 'BUY',
        size: 1,
      }
    })
    .sort((a, b) => (a.time as number) - (b.time as number))
}
