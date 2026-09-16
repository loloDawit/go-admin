import { useMemo } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import {
  Button,
  DataTable,
  PageHeader,
  PageStack,
  Pagination,
  SelectField,
  Status,
} from '../ui'
import type { Column } from '../ui'
import { listOrders, orderStatusLabels } from '../api/orders'
import type { OrderQuery, OrderStatus, OrderSummary } from '../api/orders'
import { formatDateTime, formatMoney } from '../api/format'
import { oneOf, positiveInt, toSearchParams } from '../api/listQuery'
import { useResource } from '../api/useResource'
import { orderStatusTones } from '../app/statusTones'
import styles from './Orders.module.css'

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

const STATUSES = Object.keys(orderStatusLabels) as OrderStatus[]

export function Orders() {
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()

  const query: OrderQuery = useMemo(
    () => ({
      status: oneOf(params, 'status', STATUSES),
      page: positiveInt(params, 'page') ?? 1,
    }),
    [params],
  )

  const orders = useResource(`orders:${params.toString()}`, () => listOrders(query))
  const page = orders.data

  function apply(next: OrderQuery) {
    setParams(
      toSearchParams({
        status: next.status,
        page: next.page === 1 ? undefined : next.page,
      }),
    )
  }

  return (
    <PageStack>
      <PageHeader
        title="Orders"
        description="Everything placed through the shop, newest first."
        actions={
          <Button variant="primary" onClick={() => navigate('/orders/new')}>
            New order
          </Button>
        }
      />

      <div className={styles.filters}>
        <div className={styles.status}>
          <SelectField
            label="Status"
            value={query.status ?? ''}
            onChange={(event) =>
              apply({
                page: 1,
                status: (event.target.value || undefined) as OrderStatus | undefined,
              })
            }
          >
            <option value="">All statuses</option>
            {STATUSES.map((status) => (
              <option key={status} value={status}>
                {orderStatusLabels[status]}
              </option>
            ))}
          </SelectField>
        </div>
      </div>

      <DataTable
        columns={columns}
        rows={page?.items ?? []}
        rowKey={(order) => order.id}
        status={orders.status}
        emptyTitle={query.status ? 'No orders with this status' : 'No orders yet'}
        emptyDescription={
          query.status
            ? 'Clear the filter to see the rest.'
            : 'Take the first order and it appears here.'
        }
        emptyAction={
          query.status ? undefined : (
            <Button variant="primary" onClick={() => navigate('/orders/new')}>
              New order
            </Button>
          )
        }
        errorDescription={orders.error?.message}
        onRetry={orders.reload}
      />

      {page && (
        <Pagination
          page={page.page}
          pageSize={page.pageSize}
          total={page.total}
          onChange={(next) => apply({ ...query, page: next })}
        />
      )}
    </PageStack>
  )
}
