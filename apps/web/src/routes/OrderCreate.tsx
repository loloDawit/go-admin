import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  Alert,
  Button,
  DataTable,
  PageHeader,
  PageStack,
  Section,
  StateBlock,
  TextField,
} from '../ui'
import type { Column } from '../ui'
import { listProducts } from '../api/catalog'
import type { Product } from '../api/catalog'
import { getCustomerByEmail } from '../api/customers'
import type { Customer } from '../api/customers'
import { createOrder } from '../api/orders'
import { isApiError } from '../api/client'
import { formatMoney } from '../api/format'
import { useResource } from '../api/useResource'
import styles from './OrderCreate.module.css'

type Line = {
  productId: string
  title: string
  unitPriceMinor: number
  currency: string
  quantity: number
}

export function OrderCreate() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [customer, setCustomer] = useState<Customer>()
  const [lookupError, setLookupError] = useState<string>()
  const [term, setTerm] = useState('')
  const [search, setSearch] = useState<string>()
  const [lines, setLines] = useState<Line[]>([])
  const [failure, setFailure] = useState<string>()
  const [placing, setPlacing] = useState(false)

  // Only active products can be ordered; the service refuses anything else, so
  // offering draft and archived ones here would only produce a failed order.
  const results = useResource(`order-search:${search ?? ''}`, () =>
    search ? listProducts({ q: search, status: 'active' }) : Promise.resolve(undefined),
  )

  function add(product: Product) {
    setLines((current) => {
      const existing = current.find((line) => line.productId === product.id)
      if (existing) {
        return current.map((line) =>
          line.productId === product.id ? { ...line, quantity: line.quantity + 1 } : line,
        )
      }
      return [
        ...current,
        {
          productId: product.id,
          title: product.title,
          unitPriceMinor: product.priceMinor,
          currency: product.currency,
          quantity: 1,
        },
      ]
    })
  }

  const currency = lines[0]?.currency
  const mixedCurrency = lines.some((line) => line.currency !== currency)
  const total = lines.reduce((sum, line) => sum + line.unitPriceMinor * line.quantity, 0)

  const resultColumns: Column<Product>[] = [
    { key: 'title', header: 'Product', cell: (product) => product.title },
    { key: 'sku', header: 'SKU', cell: (product) => product.sku },
    {
      key: 'price',
      header: 'Price',
      numeric: true,
      cell: (product) => formatMoney(product.priceMinor, product.currency),
    },
    {
      key: 'add',
      header: '',
      cell: (product) => (
        <Button size="sm" onClick={() => add(product)}>
          Add
        </Button>
      ),
    },
  ]

  async function findCustomer(value: string) {
    setLookupError(undefined)
    try {
      setCustomer(await getCustomerByEmail(value.trim()))
    } catch (cause) {
      setCustomer(undefined)
      setLookupError(
        isApiError(cause) && cause.code === 'not_found'
          ? 'No customer with that email. Add them on the Customers screen first.'
          : 'The customer could not be looked up.',
      )
    }
  }

  async function place() {
    if (!customer) return
    setPlacing(true)
    setFailure(undefined)
    try {
      const order = await createOrder(
        customer.id,
        lines.map((line) => ({ productId: line.productId, quantity: line.quantity })),
      )
      navigate(`/orders/${order.id}`)
    } catch (cause) {
      setFailure(isApiError(cause) ? cause.message : 'The order could not be placed.')
      setPlacing(false)
    }
  }

  return (
    <PageStack>
      <PageHeader
        breadcrumb={<Link to="/orders">Orders</Link>}
        title="New order"
        description="Prices are taken at the moment the order is placed."
      />

      {failure && <Alert tone="danger" title={failure} />}
      {mixedCurrency && (
        <Alert tone="warning" title="Every line must be in one currency">
          Remove the lines that do not match, or place them as a separate order.
        </Alert>
      )}

      <div className={styles.layout}>
        <Section title="Add products">
          <form
            className={styles.search}
            onSubmit={(event) => {
              event.preventDefault()
              setSearch(term.trim() || undefined)
            }}
          >
            <div className={styles.searchField}>
              <TextField
                label="Search the catalog"
                type="search"
                value={term}
                placeholder="Title or description"
                onChange={(event) => setTerm(event.target.value)}
              />
            </div>
            <Button type="submit">Search</Button>
          </form>

          {search === undefined ? (
            <StateBlock
              title="Search for a product"
              description="Only active products can be ordered."
            />
          ) : (
            <DataTable
              columns={resultColumns}
              rows={results.data?.items ?? []}
              rowKey={(product) => product.id}
              status={results.status}
              emptyTitle="Nothing active matches"
              emptyDescription="A draft product has to be activated before it can be sold."
              errorDescription={results.error?.message}
              onRetry={results.reload}
            />
          )}
        </Section>

        <Section title="This order">
          <form
            onSubmit={(event) => {
              event.preventDefault()
              void findCustomer(email)
            }}
          >
            <TextField
              label="Customer email"
              type="email"
              value={email}
              error={lookupError}
              help={customer ? `Ordering for ${customer.name}` : undefined}
              onChange={(event) => setEmail(event.target.value)}
            />
          </form>

          {lines.length === 0 ? (
            <StateBlock title="No lines yet" description="Add a product to start the order." />
          ) : (
            <>
              <ul className={styles.lines}>
                {lines.map((line) => (
                  <li key={line.productId} className={styles.line}>
                    <span className={styles.lineTitle}>{line.title}</span>
                    <div className={styles.lineControls}>
                    <input
                      className={styles.quantity}
                      type="number"
                      min={1}
                      aria-label={`Quantity of ${line.title}`}
                      value={line.quantity}
                      onChange={(event) => {
                        const quantity = Math.max(1, Number(event.target.value) || 1)
                        setLines((current) =>
                          current.map((l) =>
                            l.productId === line.productId ? { ...l, quantity } : l,
                          ),
                        )
                      }}
                    />
                    <span className={styles.lineTotal}>
                      {formatMoney(line.unitPriceMinor * line.quantity, line.currency)}
                    </span>
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() =>
                        setLines((current) =>
                          current.filter((l) => l.productId !== line.productId),
                        )
                      }
                    >
                      Remove
                    </Button>
                    </div>
                  </li>
                ))}
              </ul>
              {!mixedCurrency && currency && (
                <p className={styles.total}>
                  <span>Total</span>
                  <span>{formatMoney(total, currency)}</span>
                </p>
              )}
            </>
          )}

          <Button
            variant="primary"
            block
            loading={placing}
            disabled={lines.length === 0 || !customer || mixedCurrency}
            onClick={() => void place()}
          >
            Place order
          </Button>
        </Section>
      </div>
    </PageStack>
  )
}
