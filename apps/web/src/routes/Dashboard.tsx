import { Link } from 'react-router-dom'
import { DataTable, PageHeader, PageStack, Section, Status } from '../ui'
import type { Column } from '../ui'
import { getDashboard } from '../api/reports'
import type { RevenueDay } from '../api/reports'
import { orderStatusLabels } from '../api/orders'
import type { OrderStatus, OrderSummary } from '../api/orders'
import { formatDate, formatDateTime, formatMoney } from '../api/format'
import { useResource } from '../api/useResource'
import { orderStatusTones } from '../app/statusTones'
import styles from './Dashboard.module.css'

const orderColumns: Column<OrderSummary>[] = [
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

const revenueColumns: Column<RevenueDay>[] = [
  { key: 'day', header: 'Day', cell: (row) => formatDate(`${row.day}T00:00:00Z`) },
  { key: 'currency', header: 'Currency', cell: (row) => row.currency },
  { key: 'placed', header: 'Placed', numeric: true, cell: (row) => row.placedCount },
  { key: 'paid', header: 'Paid', numeric: true, cell: (row) => row.paidCount },
  {
    key: 'refunded',
    header: 'Refunded',
    numeric: true,
    cell: (row) => formatMoney(row.refundedMinor, row.currency),
  },
  {
    key: 'net',
    header: 'Net',
    numeric: true,
    cell: (row) => formatMoney(row.netMinor, row.currency),
  },
]

const WAITING: OrderStatus[] = ['pending', 'paid', 'packed']

export function Dashboard() {
  const report = useResource('dashboard', getDashboard)
  const counts = report.data?.counts ?? {}
  const waiting = WAITING.reduce((sum, status) => sum + (counts[status] ?? 0), 0)

  return (
    <PageStack>
      <PageHeader title="Today" description="Where the shop stands this morning." />

      <dl className={styles.figures}>
        <Figure label="Orders to handle" value={report.data ? waiting : undefined} />
        <Figure label="Awaiting payment" value={report.data ? (counts.pending ?? 0) : undefined} />
        <Figure label="Ready to ship" value={report.data ? (counts.packed ?? 0) : undefined} />
      </dl>

      <Section title="Latest orders">
        <DataTable
          columns={orderColumns}
          rows={report.data?.recent ?? []}
          rowKey={(order) => order.id}
          status={report.status}
          emptyTitle="No orders yet"
          emptyDescription="The first order placed appears here."
          errorDescription={report.error?.message}
          onRetry={report.reload}
        />
      </Section>

      <Section
        title="Revenue"
        description="Recognised when an order is paid, and reversed on the day it was recognised if it is refunded."
      >
        <DataTable
          columns={revenueColumns}
          rows={report.data?.revenue ?? []}
          rowKey={(row) => `${row.day}:${row.currency}`}
          status={report.status}
          emptyTitle="No revenue recorded yet"
          emptyDescription="A day appears here once an order placed that day is paid."
          errorDescription={report.error?.message}
          onRetry={report.reload}
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
