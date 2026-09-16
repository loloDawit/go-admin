import { Link, useParams } from 'react-router-dom'
import {
  Button,
  DataTable,
  DefinitionList,
  PageHeader,
  PageStack,
  Section,
  StateBlock,
  Status,
} from '../ui'
import type { Column } from '../ui'
import { getCustomer, getLifetimeValue, listCustomerOrders } from '../api/customers'
import { orderStatusLabels } from '../api/orders'
import type { OrderSummary } from '../api/orders'
import { isApiError } from '../api/client'
import { formatDate, formatDateTime, formatMoney } from '../api/format'
import { useResource } from '../api/useResource'
import { orderStatusTones } from '../app/statusTones'

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

export function CustomerDetail() {
  const { customerId = '' } = useParams()
  const customer = useResource(`customer:${customerId}`, () => getCustomer(customerId))
  const orders = useResource(`customer-orders:${customerId}`, () => listCustomerOrders(customerId))
  const lifetime = useResource(`customer-ltv:${customerId}`, () => getLifetimeValue(customerId))

  if (customer.status === 'loading') {
    return <StateBlock title="Loading customer" description="Fetching their order history." />
  }

  if (customer.status === 'error' || !customer.data) {
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

  const current = customer.data

  return (
    <PageStack>
      <PageHeader
        breadcrumb={<Link to="/customers">Customers</Link>}
        title={current.name}
        description={current.email}
      />

      <DefinitionList
        items={[
          { term: 'Customer since', value: formatDate(current.createdAt) },
          { term: 'Orders', value: orders.data ? String(orders.data.total) : '—' },
          { term: 'Lifetime value', value: <LifetimeValue resource={lifetime} /> },
        ]}
      />

      <Section title="Orders">
        <DataTable
          columns={columns}
          rows={orders.data?.items ?? []}
          rowKey={(order) => order.id}
          status={orders.status}
          emptyTitle="No orders yet"
          emptyDescription="Their first order will appear here."
          errorDescription={orders.error?.message}
          onRetry={orders.reload}
        />
      </Section>
    </PageStack>
  )
}

// A customer whose orders span more than one currency has no single lifetime
// value; the service says so rather than summing figures that do not add up.
function LifetimeValue({
  resource,
}: {
  resource: ReturnType<typeof useResource<{ lifetimeValueMinor: number; currency: string }>>
}) {
  if (resource.status === 'loading') return <>—</>
  if (resource.status === 'error') {
    return (
      <>
        {isApiError(resource.error) && resource.error.code === 'mixed_currency_history'
          ? 'Orders in more than one currency'
          : 'Unavailable'}
      </>
    )
  }
  if (!resource.data) return <>—</>
  // No orders means no money taken in any currency, which the service reports as
  // an empty code rather than inventing one.
  if (!resource.data.currency) return <>No orders yet</>
  return <>{formatMoney(resource.data.lifetimeValueMinor, resource.data.currency)}</>
}
