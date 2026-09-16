import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  Button,
  DataTable,
  PageHeader,
  PageStack,
  Pagination,
  SelectField,
  Status,
  TextField,
} from '../ui'
import type { Column } from '../ui'
import { listProducts, productStatusLabels } from '../api/catalog'
import type { Product, ProductQuery, ProductStatus } from '../api/catalog'
import { formatDate, formatMoney } from '../api/format'
import { useResource } from '../api/useResource'
import { productStatusTones } from '../app/statusTones'
import styles from './Products.module.css'

const columns: Column<Product>[] = [
  {
    key: 'title',
    header: 'Product',
    cell: (product) => <Link to={`/products/${product.id}`}>{product.title}</Link>,
  },
  { key: 'sku', header: 'SKU', cell: (product) => product.sku },
  {
    key: 'status',
    header: 'Status',
    cell: (product) => (
      <Status tone={productStatusTones[product.status]}>
        {productStatusLabels[product.status]}
      </Status>
    ),
  },
  {
    key: 'price',
    header: 'Price',
    numeric: true,
    cell: (product) => formatMoney(product.priceMinor, product.currency),
  },
  { key: 'updated', header: 'Updated', cell: (product) => formatDate(product.updatedAt) },
]

const STATUS_FILTERS: ProductStatus[] = ['draft', 'active', 'archived']

export function Products() {
  const navigate = useNavigate()
  const [term, setTerm] = useState('')
  const [query, setQuery] = useState<ProductQuery>({ page: 1 })
  const products = useResource(`products:${JSON.stringify(query)}`, () => listProducts(query))

  const page = products.data
  const filtered = Boolean(query.q) || Boolean(query.status)

  return (
    <PageStack>
      <PageHeader
        title="Products"
        description="The catalog orders are placed against."
        actions={
          <Button variant="primary" onClick={() => navigate('/products/new')}>
            Add product
          </Button>
        }
      />

      <form
        className={styles.filters}
        onSubmit={(event) => {
          event.preventDefault()
          setQuery((current) => ({ ...current, q: term.trim() || undefined, page: 1 }))
        }}
      >
        <div className={styles.search}>
          <TextField
            label="Search"
            type="search"
            value={term}
            placeholder="Title or description"
            onChange={(event) => setTerm(event.target.value)}
          />
        </div>
        <div className={styles.status}>
          <SelectField
            label="Status"
            value={query.status ?? ''}
            onChange={(event) =>
              setQuery((current) => ({
                ...current,
                status: (event.target.value || undefined) as ProductStatus | undefined,
                page: 1,
              }))
            }
          >
            <option value="">Draft and active</option>
            {STATUS_FILTERS.map((status) => (
              <option key={status} value={status}>
                {productStatusLabels[status]}
              </option>
            ))}
          </SelectField>
        </div>
        <Button type="submit">Search</Button>
      </form>

      <DataTable
        columns={columns}
        rows={page?.items ?? []}
        rowKey={(product) => product.id}
        status={products.status}
        emptyTitle={filtered ? 'No products match' : 'The catalog is empty'}
        emptyDescription={
          filtered
            ? 'Try a different search term, or clear the status filter.'
            : 'Add the first product. It starts as a draft and can be activated once it is ready to sell.'
        }
        emptyAction={
          filtered ? undefined : (
            <Button variant="primary" onClick={() => navigate('/products/new')}>
              Add product
            </Button>
          )
        }
        errorDescription={products.error?.message}
        onRetry={products.reload}
      />

      {page && (
        <Pagination
          page={page.page}
          pageSize={page.pageSize}
          total={page.total}
          onChange={(next) => setQuery((current) => ({ ...current, page: next }))}
        />
      )}
    </PageStack>
  )
}
