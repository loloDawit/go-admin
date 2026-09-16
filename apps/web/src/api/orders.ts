import { withScenario } from './client'
import type { components } from './generated/orders'
import { api } from './http'

export type OrderStatus = components['schemas']['OrderStatus']
export type Order = components['schemas']['Order']
export type OrderSummary = components['schemas']['OrderSummary']
export type OrderItem = components['schemas']['OrderItem']
export type OrderEvent = components['schemas']['OrderEvent']
export type OrderPage = components['schemas']['OrderPage']

export const orderStatusLabels: Record<OrderStatus, string> = {
  pending: 'Awaiting payment',
  paid: 'Paid',
  packed: 'Packed',
  shipped: 'Shipped',
  delivered: 'Delivered',
  cancelled: 'Cancelled',
  refunded: 'Refunded',
}

// The service's transition table, mirrored so a screen can offer only the
// moves that will be accepted. The service remains the authority; this decides
// which buttons exist, never whether a transition is legal.
const forward: Record<OrderStatus, OrderStatus | undefined> = {
  pending: 'paid',
  paid: 'packed',
  packed: 'shipped',
  shipped: 'delivered',
  delivered: undefined,
  cancelled: undefined,
  refunded: undefined,
}

const cancellable: OrderStatus[] = ['pending', 'paid', 'packed']
const refundable: OrderStatus[] = ['paid', 'packed', 'shipped', 'delivered']

export function nextStatus(status: OrderStatus): OrderStatus | undefined {
  return forward[status]
}

export function canCancel(status: OrderStatus): boolean {
  return cancellable.includes(status)
}

export function canRefund(status: OrderStatus): boolean {
  return refundable.includes(status)
}

export type OrderQuery = {
  status?: OrderStatus
  customerId?: string
  sort?: string
  page?: number
  pageSize?: number
}

function queryString(query: OrderQuery): string {
  const params = new URLSearchParams()
  if (query.status) params.set('status', query.status)
  if (query.customerId) params.set('customerId', query.customerId)
  if (query.sort) params.set('sort', query.sort)
  if (query.page && query.page > 1) params.set('page', String(query.page))
  if (query.pageSize) params.set('pageSize', String(query.pageSize))
  const encoded = params.toString()
  return encoded ? `?${encoded}` : ''
}

const emptyPage: OrderPage = { items: [], page: 1, pageSize: 20, total: 0 }

export function listOrders(query: OrderQuery = {}): Promise<OrderPage> {
  return withScenario(() => api.get<OrderPage>(`/api/v1/orders${queryString(query)}`), emptyPage)
}

export function getOrder(id: string): Promise<Order> {
  return withScenario(() => api.get<Order>(`/api/v1/orders/${id}`))
}

export function listOrderEvents(id: string): Promise<OrderEvent[]> {
  return withScenario(
    async () => (await api.get<{ events: OrderEvent[] }>(`/api/v1/orders/${id}/events`)).events,
    [],
  )
}

export function createOrder(customerId: string, items: { productId: string; quantity: number }[]) {
  return api.post<Order>('/api/v1/orders', { customerId, items })
}

export function setOrderStatus(id: string, status: OrderStatus): Promise<Order> {
  return api.post<Order>(`/api/v1/orders/${id}/status`, { status })
}

export function cancelOrder(id: string, reason: string): Promise<Order> {
  return api.post<Order>(`/api/v1/orders/${id}/cancel`, { reason })
}

export function refundOrder(id: string, reason: string): Promise<Order> {
  return api.post<Order>(`/api/v1/orders/${id}/refund`, { reason })
}
