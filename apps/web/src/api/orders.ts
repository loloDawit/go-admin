import { respond } from './client'

export type OrderStatus = 'pending' | 'paid' | 'packed' | 'shipped' | 'cancelled' | 'refunded'

export type OrderLine = {
  sku: string
  name: string
  quantity: number
  unitPriceCents: number
}

export type OrderEvent = {
  at: string
  actor: string
  summary: string
}

export type Order = {
  id: string
  reference: string
  customerName: string
  customerEmail: string
  placedAt: string
  status: OrderStatus
  totalCents: number
  itemCount: number
  shippingAddress: string
  lines: OrderLine[]
  history: OrderEvent[]
}

const ORDERS: Order[] = [
  {
    id: 'o-5107',
    reference: 'NG-5107',
    customerName: 'Rita Alvarez',
    customerEmail: 'rita.alvarez@example.com',
    placedAt: '2026-09-12T07:41:00Z',
    status: 'pending',
    totalCents: 8600,
    itemCount: 3,
    shippingAddress: '14 Brookside Lane, Leeds LS6 2AH',
    lines: [
      { sku: 'MUG-CLY-300', name: 'Clay mug, 300 ml', quantity: 2, unitPriceCents: 2200 },
      { sku: 'TWL-LIN-NAT', name: 'Linen tea towel, natural', quantity: 1, unitPriceCents: 1400 },
    ],
    history: [{ at: '2026-09-12T07:41:00Z', actor: 'Storefront', summary: 'Order placed' }],
  },
  {
    id: 'o-5106',
    reference: 'NG-5106',
    customerName: 'Tom Whitfield',
    customerEmail: 'tom.whitfield@example.com',
    placedAt: '2026-09-11T16:02:00Z',
    status: 'paid',
    totalCents: 6400,
    itemCount: 1,
    shippingAddress: '3 Cobden Street, Manchester M15 5RX',
    lines: [
      { sku: 'KTL-STN-500', name: 'Stoneware kettle, 0.5 L', quantity: 1, unitPriceCents: 6400 },
    ],
    history: [
      { at: '2026-09-11T16:02:00Z', actor: 'Storefront', summary: 'Order placed' },
      { at: '2026-09-11T16:03:00Z', actor: 'Payments', summary: 'Payment captured' },
    ],
  },
  {
    id: 'o-5105',
    reference: 'NG-5105',
    customerName: 'Priya Nandi',
    customerEmail: 'priya.nandi@example.com',
    placedAt: '2026-09-11T09:22:00Z',
    status: 'shipped',
    totalCents: 12100,
    itemCount: 4,
    shippingAddress: '88 Fern Road, Bristol BS6 5JT',
    lines: [
      { sku: 'BRD-OAK-LRG', name: 'Oak serving board, large', quantity: 1, unitPriceCents: 8900 },
      {
        sku: 'CDL-BEE-SET',
        name: 'Beeswax candles, set of six',
        quantity: 1,
        unitPriceCents: 3200,
      },
    ],
    history: [
      { at: '2026-09-11T09:22:00Z', actor: 'Storefront', summary: 'Order placed' },
      { at: '2026-09-11T09:23:00Z', actor: 'Payments', summary: 'Payment captured' },
      { at: '2026-09-11T11:40:00Z', actor: 'Dana Okafor', summary: 'Packed for dispatch' },
      { at: '2026-09-11T15:10:00Z', actor: 'Dana Okafor', summary: 'Handed to courier' },
    ],
  },
  {
    id: 'o-5104',
    reference: 'NG-5104',
    customerName: 'Joel Brenner',
    customerEmail: 'joel.brenner@example.com',
    placedAt: '2026-09-10T18:55:00Z',
    status: 'cancelled',
    totalCents: 5600,
    itemCount: 1,
    shippingAddress: '22 Castle View, Edinburgh EH3 9DR',
    lines: [{ sku: 'APR-CAN-BLK', name: 'Canvas apron, black', quantity: 1, unitPriceCents: 5600 }],
    history: [
      { at: '2026-09-10T18:55:00Z', actor: 'Storefront', summary: 'Order placed' },
      { at: '2026-09-10T19:30:00Z', actor: 'Mara Lindqvist', summary: 'Cancelled at customer request' },
    ],
  },
  {
    id: 'o-5103',
    reference: 'NG-5103',
    customerName: 'Aisha Rahman',
    customerEmail: 'aisha.rahman@example.com',
    placedAt: '2026-09-10T12:14:00Z',
    status: 'packed',
    totalCents: 4400,
    itemCount: 2,
    shippingAddress: '5 Harbour Way, Cardiff CF10 4AN',
    lines: [{ sku: 'MUG-CLY-300', name: 'Clay mug, 300 ml', quantity: 2, unitPriceCents: 2200 }],
    history: [
      { at: '2026-09-10T12:14:00Z', actor: 'Storefront', summary: 'Order placed' },
      { at: '2026-09-10T12:15:00Z', actor: 'Payments', summary: 'Payment captured' },
      { at: '2026-09-10T14:02:00Z', actor: 'Dana Okafor', summary: 'Packed for dispatch' },
    ],
  },
]

export function listOrders(): Promise<Order[]> {
  return respond(ORDERS, [])
}

export function getOrder(id: string): Promise<Order | undefined> {
  return respond(ORDERS.find((order) => order.id === id))
}

export const orderStatusLabels: Record<OrderStatus, string> = {
  pending: 'Awaiting payment',
  paid: 'Paid',
  packed: 'Packed',
  shipped: 'Shipped',
  cancelled: 'Cancelled',
  refunded: 'Refunded',
}
