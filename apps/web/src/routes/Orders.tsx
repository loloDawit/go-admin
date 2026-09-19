import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import {
  ALL,
  DefinitionList,
  Pagination,
  RowActions,
  RowDrawer,
  StatusBadge,
  runBulk,
  reportBulk,
} from '../ui'
import { ListPage } from '../patterns'
import { Button } from '@/ui/shadcn/button'
import { DropdownMenuItem } from '@/ui/shadcn/dropdown-menu'
import { Tabs, TabsList, TabsTrigger } from '@/ui/shadcn/tabs'
import type { BulkAction, BulkProgress, Column } from '../ui'
import { listOrders, orderStatusLabels, setOrderStatus } from '../api/orders'
import type { OrderQuery, OrderStatus, OrderSummary } from '../api/orders'
import { formatDateTime, formatMoney } from '../api/format'
import { oneOf, positiveInt, toSearchParams } from '../api/listQuery'
import { useResource } from '../api/useResource'
import { orderStatusTones } from '../app/statusTones'

const columns: Column<OrderSummary>[] = [
  {
    key: 'number',
    header: 'Order',
    sortKey: 'number',
    width: '11rem',
    cell: (order) => <Link to={`/orders/${order.id}`}>{order.number}</Link>,
  },
  {
    key: 'customer',
    header: 'Customer',
    sortKey: 'customer',
    width: '18rem',
    cell: (order) => <Link to={`/customers/${order.customerId}`}>{order.customerName}</Link>,
  },
  {
    key: 'placed',
    header: 'Placed',
    sortKey: 'placed_at',
    width: '11rem',
    cell: (order) => formatDateTime(order.placedAt),
  },
  {
    key: 'status',
    header: 'Status',
    sortKey: 'status',
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
    sortKey: 'total_minor',
    grow: true,
    cell: (order) => formatMoney(order.totalMinor, order.currency),
  },
]

const ORDER_SORTS = ['number', 'customer', 'placed_at', 'status', 'total_minor'] as const

const STATUSES = Object.keys(orderStatusLabels) as OrderStatus[]

export function Orders() {
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()

  const query: OrderQuery = useMemo(() => {
    const raw = params.get('sort') ?? ''
    const descending = raw.startsWith('-')
    const key = oneOf(new URLSearchParams({ sort: raw.replace(/^-/, '') }), 'sort', ORDER_SORTS)
    return {
      status: oneOf(params, 'status', STATUSES),
      sort: key ? (descending ? `-${key}` : key) : undefined,
      page: positiveInt(params, 'page') ?? 1,
      pageSize: positiveInt(params, 'pageSize'),
    }
  }, [params])

  const paramsKey = params.toString()
  const orders = useResource(`orders:${paramsKey}`, () => listOrders(query))
  const page = orders.data

  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [bulkProgress, setBulkProgress] = useState<BulkProgress>()
  const [previewedId, setPreviewedId] = useState<string>()
  // Derived, not stored: switching tabs or pages while a preview is open drops
  // it naturally once its row is no longer on the page, rather than showing a
  // stale one.
  const previewed = (page?.items ?? []).find((order) => order.id === previewedId)

  // Selection is per page: a new page, status filter or sort is a different
  // set of rows, so a stale selection never survives to act on rows the
  // person can no longer see.
  useEffect(() => {
    setSelected(new Set())
  }, [paramsKey])

  const selectedOrders = (page?.items ?? []).filter((order) => selected.has(order.id))
  const canPack = selectedOrders.length > 0 && selectedOrders.every((order) => order.status === 'paid')

  async function runOrderBulk(verb: string, action: (order: OrderSummary) => Promise<unknown>) {
    const rows = selectedOrders
    setBulkProgress({ done: 0, total: rows.length })
    const outcome = await runBulk(rows, action, (done) => setBulkProgress({ done, total: rows.length }))
    setBulkProgress(undefined)
    reportBulk(outcome, verb, (order) => order.number)
    setSelected(new Set())
    orders.reload()
  }

  const bulkActions: BulkAction[] = [
    {
      label: 'Mark packed',
      disabled: !canPack,
      reason: !canPack ? 'Only paid orders can be marked packed' : undefined,
      onClick: () => runOrderBulk('packed', (order) => setOrderStatus(order.id, 'packed')),
    },
  ]

  function apply(next: OrderQuery) {
    setParams(
      toSearchParams({
        status: next.status,
        sort: next.sort,
        page: next.page === 1 ? undefined : next.page,
        pageSize: next.pageSize,
      }),
    )
  }

  return (
    <>
      <ListPage
        title="Orders"
        description="Everything placed through the shop, newest first."
        primaryAction={<Button onClick={() => navigate('/orders/new')}>New order</Button>}
        filters={
          <Tabs
            value={query.status ?? ALL}
            onValueChange={(status) =>
              apply({ page: 1, status: status === ALL ? undefined : (status as OrderStatus) })
            }
            className="max-w-full"
          >
            {/* Seven statuses outrun a narrow toolbar; scrolling the strip keeps
                every tab reachable instead of clipping it against the card edge. */}
            <div className="max-w-full overflow-x-auto">
              <TabsList aria-label="Status">
                <TabsTrigger value={ALL}>All</TabsTrigger>
                {STATUSES.map((status) => (
                  <TabsTrigger key={status} value={status}>
                    {orderStatusLabels[status]}
                  </TabsTrigger>
                ))}
              </TabsList>
            </div>
          </Tabs>
        }
        columns={columns}
        rows={page?.items ?? []}
        rowKey={(order) => order.id}
        rowHref={(order) => `/orders/${order.id}`}
        rowActions={(order) => (
          <RowActions label={`Actions for ${order.number}`}>
            <DropdownMenuItem onSelect={() => setPreviewedId(order.id)}>Quick look</DropdownMenuItem>
          </RowActions>
        )}
        status={orders.status}
        filtersApplied={query.status !== undefined}
        onClearFilters={() => apply({ page: 1 })}
        filteredEmptyTitle="No orders with this status"
        emptyTitle="No orders yet"
        emptyDescription="Take the first order and it appears here."
        emptyAction={<Button onClick={() => navigate('/orders/new')}>New order</Button>}
        errorDescription={orders.error?.message}
        onRetry={orders.reload}
        sort={query.sort}
        onSort={(next) => apply({ ...query, sort: next, page: 1 })}
        selection={{ selected, onChange: setSelected, label: (order) => order.number }}
        bulkActions={bulkActions}
        bulkProgress={bulkProgress}
        pagination={
          page && (
            <Pagination
              page={page.page}
              pageSize={page.pageSize}
              total={page.total}
              onChange={(next) => apply({ ...query, page: next })}
              onPageSizeChange={(size) => apply({ ...query, pageSize: size, page: 1 })}
            />
          )
        }
      />
      <RowDrawer
        open={previewed !== undefined}
        onClose={() => setPreviewedId(undefined)}
        title={previewed?.number ?? ''}
        description={previewed?.customerName}
        footer={
          previewed && (
            <Button asChild>
              <Link to={`/orders/${previewed.id}`}>Open order</Link>
            </Button>
          )
        }
      >
        {previewed && (
          <>
            <StatusBadge tone={orderStatusTones[previewed.status]}>
              {orderStatusLabels[previewed.status]}
            </StatusBadge>
            <DefinitionList
              items={[
                { term: 'Placed', value: formatDateTime(previewed.placedAt) },
                { term: 'Total', value: formatMoney(previewed.totalMinor, previewed.currency), lead: true },
              ]}
            />
          </>
        )}
      </RowDrawer>
    </>
  )
}
