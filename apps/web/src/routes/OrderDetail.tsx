import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { toast } from 'sonner'
import {
  Alert,
  Button,
  DataTable,
  DefinitionList,
  Dialog,
  PageHeader,
  PageStack,
  Section,
  StateBlock,
  Status,
  TextareaField,
} from '../ui'
import type { Column } from '../ui'
import {
  canCancel,
  canRefund,
  cancelOrder,
  getOrder,
  listOrderEvents,
  nextStatus,
  orderStatusLabels,
  refundOrder,
  setOrderStatus,
} from '../api/orders'
import type { OrderItem } from '../api/orders'
import { getCustomer } from '../api/customers'
import { isApiError } from '../api/client'
import { formatDateTime, formatMoney } from '../api/format'
import { useResource } from '../api/useResource'
import { orderStatusTones } from '../app/statusTones'

const itemColumns: Column<OrderItem>[] = [
  { key: 'title', header: 'Item', cell: (item) => item.titleSnapshot },
  { key: 'quantity', header: 'Qty', numeric: true, cell: (item) => item.quantity },
  {
    key: 'unit',
    header: 'Unit price',
    numeric: true,
    cell: (item) => formatMoney(item.unitPriceMinor, item.currency),
  },
  {
    key: 'total',
    header: 'Line total',
    numeric: true,
    cell: (item) => formatMoney(item.lineTotalMinor, item.currency),
  },
]

type Reasoned = 'cancel' | 'refund'

export function OrderDetail() {
  const { orderId = '' } = useParams()
  const order = useResource(`order:${orderId}`, () => getOrder(orderId))
  const events = useResource(`order-events:${orderId}`, () => listOrderEvents(orderId))
  const [asking, setAsking] = useState<Reasoned>()
  const [reason, setReason] = useState('')
  const [busy, setBusy] = useState(false)
  const [failure, setFailure] = useState<string>()

  // The confirmation names what happened in the same words the button used.
  async function run(action: () => Promise<unknown>, done: string) {
    setBusy(true)
    setFailure(undefined)
    try {
      await action()
      order.reload()
      events.reload()
      toast.success(done)
    } catch (cause) {
      setFailure(isApiError(cause) ? cause.message : 'The order could not be updated.')
    } finally {
      setBusy(false)
    }
  }

  if (order.status === 'loading') {
    return <StateBlock title="Loading order" description="Fetching the order and its history." />
  }

  if (order.status === 'error' || !order.data) {
    return (
      <StateBlock
        tone="error"
        title="This order could not be loaded"
        description={order.error?.message}
        action={
          <Button variant="secondary" onClick={order.reload}>
            Try again
          </Button>
        }
      />
    )
  }

  const current = order.data
  const advance = nextStatus(current.status)

  return (
    <PageStack>
      <PageHeader
        breadcrumb={<Link to="/orders">Orders</Link>}
        title={current.number}
        description={`Placed ${formatDateTime(current.placedAt)}`}
        actions={
          <>
            {canRefund(current.status) && (
              <Button onClick={() => setAsking('refund')}>Refund</Button>
            )}
            {canCancel(current.status) && (
              <Button variant="danger" onClick={() => setAsking('cancel')}>
                Cancel order
              </Button>
            )}
            {advance && (
              <Button
                variant="primary"
                loading={busy}
                onClick={() =>
                  void run(
                    () => setOrderStatus(current.id, advance),
                    `Order ${orderStatusLabels[advance].toLowerCase()}`,
                  )
                }
              >
                Mark {orderStatusLabels[advance].toLowerCase()}
              </Button>
            )}
          </>
        }
      />

      {failure && <Alert tone="danger" title={failure} />}

      <DefinitionList
        items={[
          {
            term: 'Status',
            value: (
              <Status tone={orderStatusTones[current.status]}>
                {orderStatusLabels[current.status]}
              </Status>
            ),
          },
          {
            term: 'Total',
            value: formatMoney(current.totalMinor, current.currency),
            lead: true,
          },
          { term: 'Customer', value: <CustomerName id={current.customerId} /> },
        ]}
      />

      <Section
        title="Items"
        description="Titles and prices are what was bought; a later catalog change does not alter them."
      >
        <DataTable
          columns={itemColumns}
          rows={current.items}
          rowKey={(item) => item.productId}
          emptyTitle="No items on this order"
        />
      </Section>

      <Section title="History">
        {events.status === 'error' && (
          <StateBlock
            tone="error"
            title="The history could not be loaded"
            description={events.error?.message}
            action={
              <Button variant="secondary" onClick={events.reload}>
                Try again
              </Button>
            }
          />
        )}
        {events.status === 'ready' && (
          <ol className="m-0 list-none p-0">
            {events.data?.map((event) => (
              <li key={event.id} className="grid gap-3 border-b border-border py-2 last:border-b-0 max-sm:grid-cols-1 max-sm:gap-1 sm:grid-cols-[10rem_1fr_auto]">
                <span className="text-muted-foreground tabular-nums">{formatDateTime(event.at)}</span>
                <span className="text-foreground">
                  {event.fromStatus
                    ? `${orderStatusLabels[event.fromStatus]} → ${orderStatusLabels[event.toStatus]}`
                    : 'Order placed'}
                  {event.reason && ` — ${event.reason}`}
                </span>
                <span className="text-caption text-subtle-foreground sm:text-right">Staff #{event.actorId}</span>
              </li>
            ))}
          </ol>
        )}
      </Section>

      <Dialog
        open={asking !== undefined}
        title={asking === 'refund' ? 'Refund this order?' : 'Cancel this order?'}
        description={
          asking === 'refund'
            ? 'A refunded order does not move again.'
            : 'A cancelled order does not move again.'
        }
        onClose={() => setAsking(undefined)}
        footer={
          <>
            <Button onClick={() => setAsking(undefined)}>Back</Button>
            <Button
              variant="dangerSolid"
              loading={busy}
              onClick={() => {
                const action = asking
                setAsking(undefined)
                if (!action) return
                void run(
                  () =>
                    action === 'refund'
                      ? refundOrder(current.id, reason)
                      : cancelOrder(current.id, reason),
                  action === 'refund' ? 'Order refunded' : 'Order cancelled',
                ).then(() => setReason(''))
              }}
            >
              {asking === 'refund' ? 'Refund' : 'Cancel order'}
            </Button>
          </>
        }
      >
        <TextareaField
          label="Reason"
          optional
          value={reason}
          onChange={(event) => setReason(event.target.value)}
          placeholder="Customer asked for it before dispatch"
        />
      </Dialog>
    </PageStack>
  )
}

// The order carries only a customer id; the name lives in another endpoint and
// a failure here must not take the order down with it.
function CustomerName({ id }: { id: string }) {
  const customer = useResource(`customer:${id}`, () => getCustomer(id))
  if (!customer.data) return <Link to={`/customers/${id}`}>Customer {id}</Link>
  return <Link to={`/customers/${id}`}>{customer.data.name}</Link>
}
