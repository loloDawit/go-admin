import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  Alert,
  Button,
  DataTable,
  DefinitionList,
  Dialog,
  PageHeader,
  PageStack,
  Section,
  SelectField,
  StateBlock,
  Status,
  TextareaField,
} from '../ui'
import type { Column } from '../ui'
import { getOrder, orderStatusLabels } from '../api/orders'
import type { OrderLine, OrderStatus } from '../api/orders'
import { formatDateTime, formatMoney } from '../api/format'
import { useResource } from '../api/useResource'
import { useScenario } from '../app/useScenario'
import { orderStatusTones } from '../app/statusTones'
import styles from './OrderDetail.module.css'

const lineColumns: Column<OrderLine>[] = [
  { key: 'name', header: 'Item', cell: (line) => line.name },
  { key: 'sku', header: 'SKU', cell: (line) => line.sku },
  { key: 'quantity', header: 'Qty', numeric: true, cell: (line) => line.quantity },
  {
    key: 'unit',
    header: 'Unit price',
    numeric: true,
    cell: (line) => formatMoney(line.unitPriceCents),
  },
  {
    key: 'total',
    header: 'Line total',
    numeric: true,
    cell: (line) => formatMoney(line.unitPriceCents * line.quantity),
  },
]

const NEXT_STATUSES: OrderStatus[] = ['paid', 'packed', 'shipped', 'cancelled', 'refunded']

export function OrderDetail() {
  const scenario = useScenario()
  const { orderId = '' } = useParams()
  const order = useResource(() => getOrder(scenario, orderId), [scenario, orderId])
  const [advanceOpen, setAdvanceOpen] = useState(false)
  const [saved, setSaved] = useState(false)

  if (order.status === 'loading') {
    return <StateBlock title="Loading order" description="Fetching the order and its history." />
  }

  if (order.status === 'error') {
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

  if (!order.data) {
    return (
      <StateBlock
        title="No such order"
        description="The reference may be mistyped, or the order was removed."
        action={<Link to="/orders">Back to orders</Link>}
      />
    )
  }

  const current = order.data

  return (
    <PageStack>
      <PageHeader
        breadcrumb={<Link to="/orders">Orders</Link>}
        title={current.reference}
        description={`Placed ${formatDateTime(current.placedAt)} by ${current.customerName}`}
        actions={
          <>
            <Button variant="secondary">Print packing slip</Button>
            <Button variant="primary" onClick={() => setAdvanceOpen(true)}>
              Update status
            </Button>
          </>
        }
      />

      {saved && (
        <Alert tone="success" title="Status updated">
          The change is local until the orders service is connected.
        </Alert>
      )}

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
          { term: 'Total', value: formatMoney(current.totalCents) },
          { term: 'Customer', value: current.customerEmail },
          { term: 'Ship to', value: current.shippingAddress },
        ]}
      />

      <Section title="Items">
        <DataTable
          columns={lineColumns}
          rows={current.lines}
          rowKey={(line) => line.sku}
          emptyTitle="No items on this order"
        />
      </Section>

      <Section title="History">
        <ol className={styles.timeline}>
          {current.history.map((event) => (
            <li key={event.at} className={styles.event}>
              <span className={styles.eventTime}>{formatDateTime(event.at)}</span>
              <span className={styles.eventSummary}>{event.summary}</span>
              <span className={styles.eventActor}>{event.actor}</span>
            </li>
          ))}
        </ol>
      </Section>

      <Dialog
        open={advanceOpen}
        title="Update order status"
        description={`${current.reference} is currently ${orderStatusLabels[current.status].toLowerCase()}.`}
        onClose={() => setAdvanceOpen(false)}
        footer={
          <>
            <Button variant="ghost" onClick={() => setAdvanceOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              onClick={() => {
                setAdvanceOpen(false)
                setSaved(true)
              }}
            >
              Save status
            </Button>
          </>
        }
      >
        <SelectField label="New status" defaultValue="packed">
          {NEXT_STATUSES.map((status) => (
            <option key={status} value={status}>
              {orderStatusLabels[status]}
            </option>
          ))}
        </SelectField>
        <TextareaField
          label="Note for the history"
          optional
          placeholder="Courier collected at 15:10"
        />
      </Dialog>
    </PageStack>
  )
}
