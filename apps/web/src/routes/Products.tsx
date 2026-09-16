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
  TextField,
} from '../ui'
import type { Column } from '../ui'
import { listProducts, productStatusLabels } from '../api/catalog'
import type { Product, ProductQuery, ProductStatus } from '../api/catalog'
import { formatDate, formatMoney } from '../api/format'
import { oneOf, positiveInt, text, toSearchParams } from '../api/listQuery'
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
  const [params, setParams] = useSearchParams()

  const query: ProductQuery = useMemo(
    () => ({
      q: text(params, 'q'),
      status: oneOf(params, 'status', STATUS_FILTERS),
      page: positiveInt(params, 'page') ?? 1,
    }),
    [params],
  )

  const products = useResource(`products:${params.toString()}`, () => listProducts(query))
  const page = products.data
  const filtered = Boolean(query.q) || Boolean(query.status)

  function apply(next: ProductQuery) {
    setParams(
      toSearchParams({
        q: next.q,
        status: next.status,
        page: next.page === 1 ? undefined : next.page,
      }),
    )
  }

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
        key={query.q ?? ''}
        className={styles.filters}
        onSubmit={(event) => {
          event.preventDefault()
          const entered = new FormData(event.currentTarget).get('q')
          apply({ ...query, q: String(entered ?? '').trim() || undefined, page: 1 })
        }}
      >
        <div className={styles.search}>
          <TextField
            label="Search"
            type="search"
            name="q"
            defaultValue={query.q ?? ''}
            placeholder="Title or description"
          />
        </div>
        <div className={styles.status}>
          <SelectField
            label="Status"
            value={query.status ?? ''}
            onChange={(event) =>
              apply({
                ...query,
                status: (event.target.value || undefined) as ProductStatus | undefined,
                page: 1,
              })
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
          onChange={(next) => apply({ ...query, page: next })}
        />
      )}
    </PageStack>
  )
}
