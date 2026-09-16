import { Link, useParams } from 'react-router-dom'
import { Button, DataTable, DefinitionList, PageHeader, PageStack, Section, StateBlock, Status } from '../ui'
import type { Column } from '../ui'
import { getCustomer } from '../api/customers'
import { listOrders, orderStatusLabels } from '../api/orders'
import type { Order } from '../api/orders'
import { formatDate, formatDateTime, formatMoney } from '../api/format'
import { useResource } from '../api/useResource'
import { orderStatusTones } from '../app/statusTones'

const columns: Column<Order>[] = [
  {
    key: 'reference',
    header: 'Order',
    cell: (order) => <Link to={`/orders/${order.id}`}>{order.reference}</Link>,
  },
  { key: 'placed', header: 'Placed', cell: (order) => formatDateTime(order.placedAt) },
  {
    key: 'status',
    header: 'Status',
    cell: (order) => (
      <Status tone={orderStatusTones[order.status]}>{orderStatusLabels[order.status]}</Status>
    ),
  },
  { key: 'total', header: 'Total', numeric: true, cell: (order) => formatMoney(order.totalCents, 'USD') },
]

export function CustomerDetail() {
  const { customerId = '' } = useParams()
  const customer = useResource(`customer:${customerId}`, () => getCustomer(customerId))
  const orders = useResource('orders', listOrders)

  if (customer.status === 'loading') {
    return <StateBlock title="Loading customer" description="Fetching their order history." />
  }

  if (customer.status === 'error') {
    return (
      <StateBlock
        tone="error"
        title="This customer could not be loaded"
        description={customer.error?.message}
        action={
          <Button variant="secondary" onClick={customer.reload}>
            Try again
          </Button>
        }
      />
    )
  }

  if (!customer.data) {
    return (
      <StateBlock
        title="No such customer"
        description="The record may have been removed."
        action={<Link to="/customers">Back to customers</Link>}
      />
    )
  }

  const current = customer.data
  const theirOrders = (orders.data ?? []).filter((order) => order.customerEmail === current.email)

  return (
    <PageStack>
      <PageHeader
        breadcrumb={<Link to="/customers">Customers</Link>}
        title={current.name}
        description={current.email}
      />

      <DefinitionList
        items={[
          { term: 'Location', value: current.location },
          { term: 'Orders', value: String(current.orderCount) },
          { term: 'Lifetime value', value: formatMoney(current.lifetimeCents, 'USD') },
          { term: 'Last order', value: formatDate(current.lastOrderAt) },
        ]}
      />

      <Section title="Recent orders">
        <DataTable
          columns={columns}
          rows={theirOrders}
          rowKey={(order) => order.id}
          status={orders.status}
          emptyTitle="No orders in the current window"
          emptyDescription="Older orders are not loaded here yet."
          errorDescription={orders.error?.message}
          onRetry={orders.reload}
        />
      </Section>
    </PageStack>
  )
}
