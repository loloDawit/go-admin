import { Suspense, lazy } from 'react'
import { Link } from 'react-router-dom'
import { Skeleton } from '@/ui/shadcn/skeleton'
import { DataTable, StatusBadge } from '../ui'
import { PageBlock, PageHeader, SectionCard, StatCard } from '../patterns'
import type { Column } from '../ui'
import { getDashboard } from '../api/reports'
import type { RevenueDay } from '../api/reports'
import { orderStatusLabels } from '../api/orders'
import type { OrderStatus, OrderSummary } from '../api/orders'
import { formatDate, formatDateTime, formatMoney } from '../api/format'
import { useResource } from '../api/useResource'
import { orderStatusTones } from '../app/statusTones'

const orderColumns: Column<OrderSummary>[] = [
  {
    key: 'number',
    header: 'Order',
    width: '11rem',
    cell: (order) => <Link to={`/orders/${order.id}`}>{order.number}</Link>,
  },
  {
    key: 'placed',
    header: 'Placed',
    width: '11rem',
    cell: (order) => formatDateTime(order.placedAt),
  },
  {
    key: 'status',
    header: 'Status',
    width: '12rem',
    cell: (order) => (
      <StatusBadge tone={orderStatusTones[order.status]}>
        {orderStatusLabels[order.status]}
      </StatusBadge>
    ),
  },
  {
    key: 'total',
    header: 'Total',
    numeric: true,
    grow: true,
    cell: (order) => formatMoney(order.totalMinor, order.currency),
  },
]

const revenueColumns: Column<RevenueDay>[] = [
  { key: 'day', header: 'Day', grow: true, cell: (row) => formatDate(`${row.day}T00:00:00Z`) },
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

// recharts is roughly a third of the application's JavaScript and the chart
// appears on this screen alone, so every other route stops paying for it.
const RevenueChart = lazy(() =>
  import('../components/RevenueChart').then((m) => ({ default: m.RevenueChart })),
)

const WAITING: OrderStatus[] = ['pending', 'paid', 'packed']

export function Dashboard() {
  const report = useResource('dashboard', getDashboard)
  const counts = report.data?.counts ?? {}
  const waiting = WAITING.reduce((sum, status) => sum + (counts[status] ?? 0), 0)

  return (
    <PageBlock>
      <PageHeader title="Today" description="Where the shop stands this morning." />

      <div className="@container/main">
        <div className="grid gap-4 *:data-[slot=card]:bg-gradient-to-t *:data-[slot=card]:from-primary/5 *:data-[slot=card]:to-card *:data-[slot=card]:shadow-xs @xl/main:grid-cols-2 @5xl/main:grid-cols-3 dark:*:data-[slot=card]:bg-card">
        <StatCard
          label="Orders to handle"
          value={report.data ? waiting : undefined}
          context="Placed, paid or packed"
        />
        <StatCard
          label="Awaiting payment"
          value={report.data ? (counts.pending ?? 0) : undefined}
          context="Not yet paid"
        />
        <StatCard
          label="Ready to ship"
          value={report.data ? (counts.packed ?? 0) : undefined}
          context="Packed and waiting to go"
        />
        </div>
      </div>

      <Suspense fallback={<Skeleton className="h-[21rem] w-full rounded-lg" />}>
        <RevenueChart
          revenue={report.data?.revenue ?? []}
          status={report.status}
          errorDescription={report.error?.message}
        />
      </Suspense>

      <SectionCard title="Latest orders" bleed>
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
      </SectionCard>

      <SectionCard
        title="Revenue"
        description="Recognised when an order is paid, and reversed on the day it was recognised if it is refunded."
        bleed
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
      </SectionCard>
    </PageBlock>
  )
}
