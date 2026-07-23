import { useState, useEffect } from 'react'
import { useForm } from 'react-hook-form'
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
      setMsg(`Order placed: ${res.state ?? res.order_id}`)
      showToast('success', `Order placed: ${res.order_id?.slice(0, 8)}`)
      reset()
      fetchOrders()
    } catch (err) {
      setMsg(err instanceof Error ? err.message : 'Order failed')
    } finally {
      setLoading(false)
    }
  })

  const cancelOrder = async (orderId: string) => {
    try {
      await orders.cancel(orderId)
      fetchOrders()
    } catch {
      setMsg('Cancel failed')
    }
  }

  return (
    <div>
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>Execution</h1>
        <button className="btn btn-outline" onClick={fetchOrders}>Refresh Orders</button>
      </div>

      <div className="grid-2 mb-4">
        <div className="card">
          <h2>Place Order</h2>
          <form onSubmit={place} style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            <FormField label="Order Type" name="orderType" register={register} error={errors.orderType}>
              <select className="input" aria-label="Order Type" {...register('orderType')}>
                <option value="MARKET">Market</option>
                <option value="LIMIT">Limit</option>
                <option value="STOP">Stop</option>
                <option value="STOP_LIMIT">Stop Limit</option>
              </select>
            </FormField>
            <FormField label="Symbol" name="symbol" register={register} error={errors.symbol} placeholder="SPX500" />
            <FormField label="Side" name="side" register={register} error={errors.side}>
              <select className="input" aria-label="Side" {...register('side')}>
                <option value="BUY">BUY</option>
                <option value="SELL">SELL</option>
              </select>
            </FormField>
            <FormField label="Quantity" name="quantity" register={register} error={errors.quantity} type="number" placeholder="100" />
            {(orderType === 'LIMIT' || orderType === 'STOP_LIMIT') && (
              <FormField label="Limit Price" name="limitPrice" register={register} error={errors.limitPrice} type="number" placeholder="0.00" />
            )}
            {(orderType === 'STOP' || orderType === 'STOP_LIMIT') && (
              <FormField label="Stop Price" name="stopPrice" register={register} error={errors.stopPrice} type="number" placeholder="0.00" />
            )}
            <button className="btn btn-primary" type="submit" disabled={loading} style={{ justifyContent: 'center' }}>
              {loading ? 'Placing...' : `Place ${orderType} Order`}
            </button>
            {msg && <p className="text-muted" style={{ margin: 0, fontSize: 13 }}>{msg}</p>}
          </form>
        </div>

        <div className="card">
          <h2>Active Orders ({orderList.length})</h2>
          {orderList.length === 0 ? (
            <p className="text-muted">No active orders</p>
          ) : (
            <div style={{ overflowX: 'auto' }}>
              <table className="data-table">
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>Symbol</th>
                    <th>Side</th>
                    <th>Type</th>
                    <th>Qty</th>
                    <th>Filled</th>
                    <th>Price</th>
                    <th>Status</th>
                    <th>Action</th>
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
                        <button className="btn btn-outline" aria-label={`Cancel order ${o.order_id?.slice(0, 8)}`} style={{ fontSize: 11, padding: '2px 8px' }} onClick={() => cancelOrder(o.order_id)}>
                          Cancel
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
