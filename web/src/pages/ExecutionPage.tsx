import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useForm } from 'react-hook-form'
import toast from 'react-hot-toast'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import { orders } from '../api/client'
import { FormField } from '../components/FormField'
import { showToast } from '../stores/toastStore'
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
  const { register, handleSubmit, formState: { errors }, reset, watch } = useForm<OrderFormData>({
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
      showToast('success', t('execution:orderPlaced', 'Order placed: {{id}}', { id: res.order_id?.slice(0, 8) }))
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
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>{t('execution:title', 'Execution')}</h1>
        <button className="btn btn-outline" onClick={fetchOrders}>{t('execution:refreshOrders', 'Refresh Orders')}</button>
      </div>

      <div className="grid-2 mb-4">
        <div className="card">
          <h2>{t('execution:placeOrder', 'Place Order')}</h2>
          <form onSubmit={place} style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            <FormField label={t('execution:orderType', 'Order Type')} name="orderType" register={register} error={errors.orderType}>
              <select className="input" aria-label={t('execution:orderType', 'Order Type')} {...register('orderType')}>
                <option value="MARKET">{t('execution:orderType_market', 'Market')}</option>
                <option value="LIMIT">{t('execution:orderType_limit', 'Limit')}</option>
                <option value="STOP">{t('execution:orderType_stop', 'Stop')}</option>
                <option value="STOP_LIMIT">{t('execution:orderType_stopLimit', 'Stop Limit')}</option>
              </select>
            </FormField>
            <FormField label={t('execution:symbol', 'Symbol')} name="symbol" register={register} error={errors.symbol} placeholder={t('execution:placeholder.symbol', 'SPX500')} />
            <FormField label={t('execution:side', 'Side')} name="side" register={register} error={errors.side}>
              <select className="input" aria-label={t('execution:side', 'Side')} {...register('side')}>
                <option value="BUY">{t('execution:side_buy', 'BUY')}</option>
                <option value="SELL">{t('execution:side_sell', 'SELL')}</option>
              </select>
            </FormField>
            <FormField label={t('execution:quantity', 'Quantity')} name="quantity" register={register} error={errors.quantity} type="number" placeholder={t('execution:placeholder.quantity', '100')} />
            {(orderType === 'LIMIT' || orderType === 'STOP_LIMIT') && (
              <FormField label={t('execution:limitPrice', 'Limit Price')} name="limitPrice" register={register} error={errors.limitPrice} type="number" placeholder={t('execution:placeholder.limitPrice', '0.00')} />
            )}
            {(orderType === 'STOP' || orderType === 'STOP_LIMIT') && (
              <FormField label={t('execution:stopPrice', 'Stop Price')} name="stopPrice" register={register} error={errors.stopPrice} type="number" placeholder={t('execution:placeholder.limitPrice', '0.00')} />
            )}
            <button className="btn btn-primary" type="submit" disabled={loading} style={{ justifyContent: 'center' }}>
              {loading ? t('execution:placing', 'Placing...') : t('execution:placeOrderButton', 'Place {{orderType}} Order', { orderType: t(`execution:orderType_${orderType.toLowerCase()}`, orderType) })}
            </button>
            {msg && <p className="text-muted" style={{ margin: 0, fontSize: 13 }}>{msg}</p>}
          </form>
        </div>

        <div className="card">
          <h2>{t('execution:activeOrders', 'Active Orders')} ({orderList.length})</h2>
          {orderList.length === 0 ? (
            <p className="text-muted">{t('execution:noActiveOrders', 'No active orders')}</p>
          ) : (
            <div style={{ overflowX: 'auto' }}>
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{t('execution:table.id', 'ID')}</th>
                    <th>{t('execution:table.symbol', 'Symbol')}</th>
                    <th>{t('execution:table.side', 'Side')}</th>
                    <th>{t('execution:table.type', 'Type')}</th>
                    <th>{t('execution:table.qty', 'Qty')}</th>
                    <th>{t('execution:table.filled', 'Filled')}</th>
                    <th>{t('execution:table.price', 'Price')}</th>
                    <th>{t('execution:table.status', 'Status')}</th>
                    <th>{t('execution:table.action', 'Action')}</th>
                  </tr>
                </thead>
                <tbody>
                  {orderList.map((o) => (
                    <tr key={o.order_id}>
                      <td style={{ fontSize: 11, fontFamily: 'monospace' }}>{o.order_id?.slice(0, 8)}</td>
                      <td>{o.symbol}</td>
                      <td style={{ color: o.side === 'BUY' ? 'var(--success)' : 'var(--danger)' }}>{o.side}</td>
                      <td>{o.order_type}</td>
                      <td>{o.quantity}</td>
                      <td>{o.filled_quantity}</td>
                      <td>{o.price != null ? `$${o.price}` : '--'}</td>
                      <td>{o.state}</td>
                      <td>
                        <button className="btn btn-outline" aria-label={t('execution:cancelOrderAria', 'Cancel order {{id}}', { id: o.order_id?.slice(0, 8) })} style={{ fontSize: 11, padding: '2px 8px' }} onClick={() => cancelOrder(o.order_id)}>
                          {t('execution:cancelOrder', 'Cancel')}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
