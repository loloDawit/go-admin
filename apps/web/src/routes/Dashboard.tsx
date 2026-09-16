import { Link } from 'react-router-dom'
import { DataTable, PageHeader, PageStack, Section, Status } from '../ui'
import type { Column } from '../ui'
import { listOrders, orderStatusLabels } from '../api/orders'
import type { Order } from '../api/orders'
import { formatDateTime, formatMoney } from '../api/format'
import { useResource } from '../api/useResource'
import { orderStatusTones } from '../app/statusTones'
import styles from './Dashboard.module.css'

const orderColumns: Column<Order>[] = [
  {
    key: 'reference',
    header: 'Order',
    cell: (order) => <Link to={`/orders/${order.id}`}>{order.reference}</Link>,
  },
  { key: 'placed', header: 'Placed', cell: (order) => formatDateTime(order.placedAt) },
  { key: 'customer', header: 'Customer', cell: (order) => order.customerName },
  {
    key: 'status',
    header: 'Status',
    cell: (order) => (
      <Status tone={orderStatusTones[order.status]}>{orderStatusLabels[order.status]}</Status>
    ),
  },
  { key: 'total', header: 'Total', numeric: true, cell: (order) => formatMoney(order.totalCents, 'USD') },
]

export function Dashboard() {
  const orders = useResource('orders', listOrders)

  const openOrders = (orders.data ?? []).filter(
    (order) => order.status === 'pending' || order.status === 'paid' || order.status === 'packed',
  )
  const takings = (orders.data ?? [])
    .filter((order) => order.status !== 'cancelled' && order.status !== 'refunded')
    .reduce((total, order) => total + order.totalCents, 0)

  return (
    <PageStack>
      <PageHeader title="Today" description="Where the shop stands this morning." />

      <dl className={styles.figures}>
        <div className={styles.figure}>
          <dt className={styles.figureLabel}>Orders to handle</dt>
          <dd className={styles.figureValue}>{orders.status === 'ready' ? openOrders.length : '—'}</dd>
        </div>
        <div className={styles.figure}>
          <dt className={styles.figureLabel}>Taken, last 7 days</dt>
          <dd className={styles.figureValue}>
            {orders.status === 'ready' ? formatMoney(takings, 'USD') : '—'}
          </dd>
        </div>
      </dl>

      <Section title="Orders waiting on you">
        <DataTable
          columns={orderColumns}
          rows={openOrders}
          rowKey={(order) => order.id}
          status={orders.status}
          emptyTitle="Nothing waiting"
          emptyDescription="Every order has been packed and dispatched."
          errorDescription={orders.error?.message}
          onRetry={orders.reload}
        />
      </Section>
    </PageStack>
  )
}
