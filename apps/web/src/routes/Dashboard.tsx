import { Link } from 'react-router-dom'
import { Alert, DataTable, PageHeader, PageStack, Section, Status } from '../ui'
import type { Column } from '../ui'
import { listOrders, orderStatusLabels } from '../api/orders'
import type { Order } from '../api/orders'
import { listProducts } from '../api/catalog'
import type { Product } from '../api/catalog'
import { formatDateTime, formatMoney } from '../api/format'
import { useResource } from '../api/useResource'
import { orderStatusTones } from '../app/statusTones'
import styles from './Dashboard.module.css'

const orderColumns: Column<Order>[] = [
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
  { key: 'total', header: 'Total', numeric: true, cell: (order) => formatMoney(order.totalCents) },
]

const lowStockColumns: Column<Product>[] = [
  {
    key: 'name',
    header: 'Product',
    cell: (product) => <Link to={`/products/${product.id}`}>{product.name}</Link>,
  },
  { key: 'sku', header: 'SKU', cell: (product) => product.sku },
  {
    key: 'stock',
    header: 'In stock',
    numeric: true,
    cell: (product) => (product.stock === 0 ? 'Out of stock' : product.stock),
  },
]

export function Dashboard() {
  const orders = useResource('orders', listOrders)
  const products = useResource('products', listProducts)

  const openOrders = (orders.data ?? []).filter(
    (order) => order.status === 'pending' || order.status === 'paid' || order.status === 'packed',
  )
  const lowStock = (products.data ?? []).filter((product) => product.stock <= 3)
  const takings = (orders.data ?? [])
    .filter((order) => order.status !== 'cancelled' && order.status !== 'refunded')
    .reduce((total, order) => total + order.totalCents, 0)

  return (
    <PageStack>
      <PageHeader title="Today" description="Where the shop stands this morning." />

      <dl className={styles.figures}>
        <div className={styles.figure}>
          <dt className={styles.figureLabel}>Orders to handle</dt>
          <dd className={styles.figureValue}>{orders.status === 'ready' ? openOrders.length : '—'}</dd>
        </div>
        <div className={styles.figure}>
          <dt className={styles.figureLabel}>Taken, last 7 days</dt>
          <dd className={styles.figureValue}>
            {orders.status === 'ready' ? formatMoney(takings) : '—'}
          </dd>
        </div>
        <div className={styles.figure}>
          <dt className={styles.figureLabel}>Products low or out of stock</dt>
          <dd className={styles.figureValue}>
            {products.status === 'ready' ? lowStock.length : '—'}
          </dd>
        </div>
      </dl>

      {products.status === 'ready' && lowStock.length > 0 && (
        <Alert tone="warning" title={`${lowStock.length} products need restocking`}>
          The storefront keeps selling products that are out of stock. Update the counts or set
          them to draft.
        </Alert>
      )}

      <Section title="Orders waiting on you">
        <DataTable
          columns={orderColumns}
          rows={openOrders}
          rowKey={(order) => order.id}
          status={orders.status}
          emptyTitle="Nothing waiting"
          emptyDescription="Every order has been packed and dispatched."
          errorDescription={orders.error?.message}
          onRetry={orders.reload}
        />
      </Section>

      <Section title="Running low">
        <DataTable
          columns={lowStockColumns}
          rows={lowStock}
          rowKey={(product) => product.id}
          status={products.status}
          emptyTitle="Stock levels are healthy"
          emptyDescription="No product is down to its last few units."
          errorDescription={products.error?.message}
          onRetry={products.reload}
        />
      </Section>
    </PageStack>
  )
}
