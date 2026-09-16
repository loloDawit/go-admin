import { Link } from 'react-router-dom'
import { DataTable, PageHeader, PageStack, Section, Status } from '../ui'
import type { Column } from '../ui'
import { listOrders, orderStatusLabels } from '../api/orders'
import type { OrderSummary } from '../api/orders'
import { formatDateTime, formatMoney } from '../api/format'
import { useResource } from '../api/useResource'
import { orderStatusTones } from '../app/statusTones'
import styles from './Dashboard.module.css'

const columns: Column<OrderSummary>[] = [
  {
    key: 'number',
    header: 'Order',
    cell: (order) => <Link to={`/orders/${order.id}`}>{order.number}</Link>,
  },
  { key: 'placed', header: 'Placed', cell: (order) => formatDateTime(order.placedAt) },
  {
    key: 'status',
    header: 'Status',
    cell: (order) => (
      <Status tone={orderStatusTones[order.status]}>{orderStatusLabels[order.status]}</Status>
    ),
  },
  {
    key: 'total',
    header: 'Total',
    numeric: true,
    cell: (order) => formatMoney(order.totalMinor, order.currency),
  },
]

// One request per status rather than a client-side filter over one page: a
// figure computed from the first page would silently undercount.
function useCount(status: OrderSummary['status']) {
  return useResource(`dashboard:${status}`, () => listOrders({ status, pageSize: 1 }))
}

export function Dashboard() {
  const pending = useCount('pending')
  const paid = useCount('paid')
  const packed = useCount('packed')
  const recent = useResource('dashboard:recent', () => listOrders({ pageSize: 10 }))

  const counted = [pending, paid, packed]
  const waiting = counted.every((c) => c.status === 'ready')
    ? counted.reduce((sum, c) => sum + (c.data?.total ?? 0), 0)
    : undefined

  return (
    <PageStack>
      <PageHeader title="Today" description="Where the shop stands this morning." />

      <dl className={styles.figures}>
        <Figure label="Orders to handle" value={waiting} />
        <Figure label="Awaiting payment" value={pending.data?.total} />
        <Figure label="Ready to ship" value={packed.data?.total} />
      </dl>

      <Section title="Latest orders">
        <DataTable
          columns={columns}
          rows={recent.data?.items ?? []}
          rowKey={(order) => order.id}
          status={recent.status}
          emptyTitle="No orders yet"
          emptyDescription="The first order placed appears here."
          errorDescription={recent.error?.message}
          onRetry={recent.reload}
        />
      </Section>
    </PageStack>
  )
}

function Figure({ label, value }: { label: string; value: number | undefined }) {
  return (
    <div className={styles.figure}>
      <dt className={styles.figureLabel}>{label}</dt>
      <dd className={styles.figureValue}>{value === undefined ? '—' : value}</dd>
    </div>
  )
}
