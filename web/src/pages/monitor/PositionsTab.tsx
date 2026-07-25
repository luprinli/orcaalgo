import { useTranslation } from 'react-i18next'
import { Card, CardContent, CardHeader, CardTitle } from '../../components/ui/card'
import { Badge } from '../../components/ui/badge'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../components/ui/table'
import MetricCard from '../../components/MetricCard'
import EquityCurveChart from '../../charts/EquityCurveChart'
import type { EquityPoint, Position, Order, TradeSummary } from '../../types/api'
import type { OverviewComputed } from './OverviewTab'

interface PositionsTabProps {
  positions: Position[]
  orders: Order[]
  liveTrades: TradeSummary[]
  equity: EquityPoint[]
  computed: OverviewComputed
}

export default function PositionsTab({ positions, orders, liveTrades, equity, computed }: PositionsTabProps) {
  const { t } = useTranslation()
  const format = (v: number | null | undefined, d = 2) => v != null ? v.toFixed(d) : '--'

  const c = computed

  return (
    <div>
      {/* Metric Cards */}
      <div className="grid grid-cols-3 gap-2 mb-4">
        <MetricCard label="Open Positions" value={positions.length} format="number" />
        <MetricCard label="Active Orders" value={orders.length} format="number" />
        <MetricCard label="Today's Trades" value={liveTrades.length} format="number" />
        <MetricCard label="Win Rate" value={computed.winRate} format="percent_raw" />
        <MetricCard label="Equity" value={computed.equityVal} format="currency" />
        <MetricCard label="Daily P&L" value={computed.dailyPnl} format="percent_raw" color="auto" />
      </div>

      {/* Equity Curve */}
      {equity.length > 0 && (
        <EquityCurveChart
          data={equity}
          height={260}
          title={t('liveTrading:liveEquityCurve', 'Live Equity Curve')}
          color="#3fb950"
        />
      )}

      {/* Positions + Orders tables */}
      <div className="grid grid-cols-2 gap-4 mt-4">
        {positions.length > 0 ? (
          <Card>
            <CardHeader><CardTitle>{t('liveTrading:positions', 'Positions')} ({positions.length})</CardTitle></CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('liveTrading:table.symbol', 'Symbol')}</TableHead>
                    <TableHead>{t('liveTrading:table.side', 'Side')}</TableHead>
                    <TableHead>{t('liveTrading:table.qty', 'Qty')}</TableHead>
                    <TableHead>{t('liveTrading:table.entry', 'Entry')}</TableHead>
                    <TableHead>{t('liveTrading:table.pnl', 'P&L')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {positions.map((p, i) => (
                    <TableRow key={i}>
                      <TableCell className="font-semibold">{p.symbol}</TableCell>
                      <TableCell>{p.side}</TableCell>
                      <TableCell>{p.quantity}</TableCell>
                      <TableCell>${format(p.average_entry_price)}</TableCell>
                      <TableCell className={(p.unrealized_pnl ?? 0) >= 0 ? 'text-trading-success' : 'text-destructive'}>
                        ${format(p.unrealized_pnl)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardContent className="pt-6 text-muted-foreground">
              {t('liveTrading:noPositions', 'No open positions')}
            </CardContent>
          </Card>
        )}

        {orders.length > 0 ? (
          <Card>
            <CardHeader><CardTitle>{t('liveTrading:activeOrders', 'Active Orders')} ({orders.length})</CardTitle></CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('liveTrading:table.symbol', 'Symbol')}</TableHead>
                    <TableHead>{t('liveTrading:table.side', 'Side')}</TableHead>
                    <TableHead>{t('liveTrading:table.type', 'Type')}</TableHead>
                    <TableHead>{t('liveTrading:table.qty', 'Qty')}</TableHead>
                    <TableHead>{t('liveTrading:table.price', 'Price')}</TableHead>
                    <TableHead>{t('liveTrading:table.state', 'State')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {orders.map((o, i) => (
                    <TableRow key={o.order_id ?? i}>
                      <TableCell className="font-semibold">{o.symbol}</TableCell>
                      <TableCell>{o.side}</TableCell>
                      <TableCell>{o.order_type}</TableCell>
                      <TableCell>{o.quantity}</TableCell>
                      <TableCell>{o.price != null ? `$${format(o.price)}` : '\u2014'}</TableCell>
                      <TableCell>
                        <Badge variant={o.state === 'filled' ? 'success' : o.state === 'cancelled' ? 'destructive' : 'warning'}>
                          {o.state}
                        </Badge>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardContent className="pt-6 text-muted-foreground">
              {t('liveTrading:noOrders', 'No active orders')}
            </CardContent>
          </Card>
        )}
      </div>

      {/* Recent Trades */}
      {liveTrades.length > 0 && (
        <Card className="mt-4">
          <CardHeader><CardTitle>{t('liveTrading:recentTrades', 'Recent Trades')} ({liveTrades.length})</CardTitle></CardHeader>
          <CardContent>
            <div className="max-h-[400px] overflow-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('liveTrading:table.symbol', 'Symbol')}</TableHead>
                    <TableHead>{t('liveTrading:table.side', 'Side')}</TableHead>
                    <TableHead>{t('liveTrading:table.qty', 'Qty')}</TableHead>
                    <TableHead>{t('liveTrading:table.entry', 'Entry')}</TableHead>
                    <TableHead>{t('liveTrading:table.exit', 'Exit')}</TableHead>
                    <TableHead>{t('liveTrading:table.pnl', 'P&L')}</TableHead>
                    <TableHead>{t('liveTrading:table.date', 'Date')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {liveTrades.map((tr, i) => (
                    <TableRow key={tr.id ?? i}>
                      <TableCell className="font-semibold">{tr.symbol}</TableCell>
                      <TableCell>{tr.side}</TableCell>
                      <TableCell>{tr.quantity}</TableCell>
                      <TableCell>${format(tr.entry_price)}</TableCell>
                      <TableCell>${format(tr.exit_price)}</TableCell>
                      <TableCell className={(tr.pnl ?? 0) >= 0 ? 'text-trading-success' : 'text-destructive'}>
                        ${format(tr.pnl)}
                      </TableCell>
                      <TableCell className="text-xs">{tr.entry_time ? new Date(tr.entry_time).toLocaleDateString() : '\u2014'}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
