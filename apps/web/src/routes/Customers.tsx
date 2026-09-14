import { Link } from 'react-router-dom'
import { DataTable, PageHeader, PageStack } from '../ui'
import type { Column } from '../ui'
import { listCustomers } from '../api/customers'
import type { Customer } from '../api/customers'
import { formatDate, formatMoney } from '../api/format'
import { useResource } from '../api/useResource'
import { useScenario } from '../app/useScenario'

const columns: Column<Customer>[] = [
  {
    key: 'name',
    header: 'Customer',
    cell: (customer) => <Link to={`/customers/${customer.id}`}>{customer.name}</Link>,
  },
  { key: 'email', header: 'Email', cell: (customer) => customer.email },
  { key: 'location', header: 'Location', cell: (customer) => customer.location },
  { key: 'orders', header: 'Orders', numeric: true, cell: (customer) => customer.orderCount },
  {
    key: 'lifetime',
    header: 'Lifetime value',
    numeric: true,
    cell: (customer) => formatMoney(customer.lifetimeCents),
  },
  { key: 'last', header: 'Last order', cell: (customer) => formatDate(customer.lastOrderAt) },
]

export function Customers() {
  const scenario = useScenario()
  const customers = useResource(() => listCustomers(scenario), [scenario])

  return (
    <PageStack>
      <PageHeader title="Customers" description="Everyone who has ordered from the shop." />
      <DataTable
        columns={columns}
        rows={customers.data ?? []}
        rowKey={(customer) => customer.id}
        status={customers.status}
        emptyTitle="No customers yet"
        emptyDescription="A customer record is created with their first order."
        errorDescription={customers.error?.message}
        onRetry={customers.reload}
      />
    </PageStack>
  )
}
