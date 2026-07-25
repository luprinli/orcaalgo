import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useForm, Controller } from 'react-hook-form'
import toast from 'react-hot-toast'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import { orders } from '../api/client'
import { FormField } from '../components/FormField'
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '../components/ui/select'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/table'
import type { Order } from '../types/api'

const orderSchema = z.object({
  symbol: z.string().min(1, 'Symbol is required').max(20),
  side: z.enum(['BUY', 'SELL']),
  orderType: z.enum(['MARKET', 'LIMIT', 'STOP', 'STOP_LIMIT']),
  quantity: z.string().min(1, 'Quantity is required'),
  limitPrice: z.string().optional(),
  stopPrice: z.string().optional(),
})

type OrderFormData = z.infer<typeof orderSchema>

export default function ExecutionPage() {
  const { t } = useTranslation()
  const { register, handleSubmit, control, formState: { errors }, reset, watch } = useForm<OrderFormData>({
    resolver: zodResolver(orderSchema),
    defaultValues: { symbol: '', side: 'BUY', orderType: 'MARKET', quantity: '100', limitPrice: '', stopPrice: '' },
  })

  const [msg, setMsg] = useState('')
  const [orderList, setOrderList] = useState<Order[]>([])
  const [loading, setLoading] = useState(false)
  const orderType = watch('orderType')

  useEffect(() => {
    fetchOrders()
  }, [])

  const fetchOrders = async () => {
    try {
      const res = await orders.list()
      setOrderList(res.orders ?? [])
    } catch { /* ignore */ }
  }

  const place = handleSubmit(async (data) => {
    setLoading(true)
    setMsg('')
    try {
      const res = await orders.place({
        symbol: data.symbol,
        side: data.side as 'BUY' | 'SELL',
        type: data.orderType as 'MARKET' | 'LIMIT' | 'STOP' | 'STOP_LIMIT',
        quantity: parseFloat(data.quantity),
        limitPrice: data.limitPrice ? parseFloat(data.limitPrice) : undefined,
        stopPrice: data.stopPrice ? parseFloat(data.stopPrice) : undefined,
        timeInForce: 'DAY',
      })
      setMsg(t('execution:orderPlaced', 'Order placed: {{id}}', { id: res.state ?? res.order_id }))
      toast.success(t('execution:orderPlaced', 'Order placed: {{id}}', { id: res.order_id?.slice(0, 8) }))
      toast.success(`Order placed: ${data.side} ${data.quantity} ${data.symbol}`)
      reset()
      fetchOrders()
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : t('execution:orderFailed', 'Order failed')
      setMsg(errMsg)
      toast.error(`Order failed: ${errMsg}`)
    } finally {
      setLoading(false)
    }
  })

  const cancelOrder = async (orderId: string) => {
    try {
      await orders.cancel(orderId)
      fetchOrders()
    } catch {
      setMsg(t('execution:cancelFailed', 'Cancel failed'))
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold mb-0">{t('execution:title', 'Execution')}</h1>
        <Button variant="outline" onClick={fetchOrders}>{t('execution:refreshOrders', 'Refresh Orders')}</Button>
      </div>

      <div className="grid grid-cols-2 gap-4 mb-4">
        <Card>
          <CardHeader><CardTitle>{t('execution:placeOrder', 'Place Order')}</CardTitle></CardHeader>
          <CardContent>
            <form onSubmit={place} className="flex flex-col gap-2.5">
              <FormField label={t('execution:orderType', 'Order Type')} name="orderType" register={register} error={errors.orderType}>
                <Controller
                  name="orderType"
                  control={control}
                  render={({ field }) => (
                    <Select value={field.value} onValueChange={field.onChange}>
                      <SelectTrigger aria-label={t('execution:orderType', 'Order Type')}><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="MARKET">{t('execution:orderType_market', 'Market')}</SelectItem>
                        <SelectItem value="LIMIT">{t('execution:orderType_limit', 'Limit')}</SelectItem>
                        <SelectItem value="STOP">{t('execution:orderType_stop', 'Stop')}</SelectItem>
                        <SelectItem value="STOP_LIMIT">{t('execution:orderType_stopLimit', 'Stop Limit')}</SelectItem>
                      </SelectContent>
                    </Select>
                  )}
                />
              </FormField>
              <FormField label={t('execution:symbol', 'Symbol')} name="symbol" register={register} error={errors.symbol} placeholder={t('execution:placeholder.symbol', 'SPX500')} />
              <FormField label={t('execution:side', 'Side')} name="side" register={register} error={errors.side}>
                <Controller
                  name="side"
                  control={control}
                  render={({ field }) => (
                    <Select value={field.value} onValueChange={field.onChange}>
                      <SelectTrigger aria-label={t('execution:side', 'Side')}><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="BUY">{t('execution:side_buy', 'BUY')}</SelectItem>
                        <SelectItem value="SELL">{t('execution:side_sell', 'SELL')}</SelectItem>
                      </SelectContent>
                    </Select>
                  )}
                />
              </FormField>
              <FormField label={t('execution:quantity', 'Quantity')} name="quantity" register={register} error={errors.quantity} type="number" placeholder={t('execution:placeholder.quantity', '100')} />
              {(orderType === 'LIMIT' || orderType === 'STOP_LIMIT') && (
                <FormField label={t('execution:limitPrice', 'Limit Price')} name="limitPrice" register={register} error={errors.limitPrice} type="number" placeholder={t('execution:placeholder.limitPrice', '0.00')} />
              )}
              {(orderType === 'STOP' || orderType === 'STOP_LIMIT') && (
                <FormField label={t('execution:stopPrice', 'Stop Price')} name="stopPrice" register={register} error={errors.stopPrice} type="number" placeholder={t('execution:placeholder.limitPrice', '0.00')} />
              )}
              <Button type="submit" disabled={loading} className="justify-center">
                {loading ? t('execution:placing', 'Placing...') : t('execution:placeOrderButton', 'Place {{orderType}} Order', { orderType: t(`execution:orderType_${orderType.toLowerCase()}`, orderType) })}
              </Button>
              {msg && <CardDescription className="mt-0">{msg}</CardDescription>}
            </form>
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle>{t('execution:activeOrders', 'Active Orders')} ({orderList.length})</CardTitle></CardHeader>
          <CardContent>
            {orderList.length === 0 ? (
              <CardDescription>{t('execution:noActiveOrders', 'No active orders')}</CardDescription>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('execution:table.id', 'ID')}</TableHead>
                    <TableHead>{t('execution:table.symbol', 'Symbol')}</TableHead>
                    <TableHead>{t('execution:table.side', 'Side')}</TableHead>
                    <TableHead>{t('execution:table.type', 'Type')}</TableHead>
                    <TableHead>{t('execution:table.qty', 'Qty')}</TableHead>
                    <TableHead>{t('execution:table.filled', 'Filled')}</TableHead>
                    <TableHead>{t('execution:table.price', 'Price')}</TableHead>
                    <TableHead>{t('execution:table.status', 'Status')}</TableHead>
                    <TableHead>{t('execution:table.action', 'Action')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {orderList.map((o) => (
                    <TableRow key={o.order_id}>
                      <TableCell className="text-[11px] font-mono">{o.order_id?.slice(0, 8)}</TableCell>
                      <TableCell>{o.symbol}</TableCell>
                      <TableCell className={o.side === 'BUY' ? 'text-trading-success' : 'text-trading-danger'}>{o.side}</TableCell>
                      <TableCell>{o.order_type}</TableCell>
                      <TableCell>{o.quantity}</TableCell>
                      <TableCell>{o.filled_quantity}</TableCell>
                      <TableCell>{o.price != null ? `$${o.price}` : '--'}</TableCell>
                      <TableCell>{o.state}</TableCell>
                      <TableCell>
                        <Button variant="outline" size="sm" aria-label={t('execution:cancelOrderAria', 'Cancel order {{id}}', { id: o.order_id?.slice(0, 8) })} onClick={() => cancelOrder(o.order_id)}>
                          {t('execution:cancelOrder', 'Cancel')}
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
