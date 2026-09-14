import { Link } from 'react-router-dom'
import { DataTable, PageHeader, PageStack, Status } from '../ui'
import type { Column } from '../ui'
import { listOrders, orderStatusLabels } from '../api/orders'
import type { Order } from '../api/orders'
import { formatDateTime, formatMoney } from '../api/format'
import { useResource } from '../api/useResource'
import { orderStatusTones } from '../app/statusTones'

const columns: Column<Order>[] = [
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
  { key: 'items', header: 'Items', numeric: true, cell: (order) => order.itemCount },
  { key: 'total', header: 'Total', numeric: true, cell: (order) => formatMoney(order.totalCents) },
]

export function Orders() {
  const orders = useResource('orders', listOrders)

  return (
    <PageStack>
      <PageHeader
        title="Orders"
        description="Everything placed through the shop, newest first."
      />
      <DataTable
        columns={columns}
        rows={orders.data ?? []}
        rowKey={(order) => order.id}
        status={orders.status}
        caption="Last 7 days"
        emptyTitle="No orders in this period"
        emptyDescription="Orders appear here as soon as the storefront accepts them."
        errorDescription={orders.error?.message}
        onRetry={orders.reload}
      />
    </PageStack>
  )
}
