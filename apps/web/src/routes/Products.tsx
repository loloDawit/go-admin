import { useMemo } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { FilterSelect, Pagination, StatusBadge } from '../ui'
import { ListPage } from '../patterns'
import { Button } from '@/ui/shadcn/button'
import { Input } from '@/ui/shadcn/input'
import { Label } from '@/ui/shadcn/label'
import type { Column } from '../ui'
import { listProducts, productStatusLabels } from '../api/catalog'
import type { Product, ProductQuery, ProductStatus } from '../api/catalog'
import { formatDate, formatMoney } from '../api/format'
import { oneOf, positiveInt, text, toSearchParams } from '../api/listQuery'
import { useResource } from '../api/useResource'
import { productStatusTones } from '../app/statusTones'

const columns: Column<Product>[] = [
  {
    key: 'title',
    header: 'Product',
    sortKey: 'title',
    grow: true,
    cell: (product) => <Link to={`/products/${product.id}`}>{product.title}</Link>,
  },
  { key: 'sku', header: 'SKU', sortKey: 'sku', width: '11rem', cell: (product) => product.sku },
  {
    key: 'status',
    header: 'Status',
    width: '8rem',
    cell: (product) => (
      <StatusBadge tone={productStatusTones[product.status]}>
        {productStatusLabels[product.status]}
      </StatusBadge>
    ),
  },
  {
    key: 'price',
    header: 'Price',
    numeric: true,
    sortKey: 'price',
    width: '8rem',
    cell: (product) => formatMoney(product.priceMinor, product.currency),
  },
  // Added, not Updated: catalog allowlists created_at, so updated_at would be the
  // one column here that cannot be sorted.
  {
    key: 'added',
    header: 'Added',
    sortKey: 'created_at',
    width: '9rem',
    cell: (product) => formatDate(product.createdAt),
  },
]

const PRODUCT_SORTS = ['title', 'sku', 'price', 'created_at'] as const

const STATUS_FILTERS: ProductStatus[] = ['draft', 'active', 'archived']

export function Products() {
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()

  const query: ProductQuery = useMemo(() => {
    const raw = params.get('sort') ?? ''
    const descending = raw.startsWith('-')
    const key = oneOf(new URLSearchParams({ sort: raw.replace(/^-/, '') }), 'sort', PRODUCT_SORTS)
    const q = text(params, 'q')
    return {
      q,
      status: oneOf(params, 'status', STATUS_FILTERS),
      // The search endpoint ranks by relevance and takes no sort, so a sort
      // alongside a search term would be a control that silently does nothing.
      sort: q || !key ? undefined : descending ? `-${key}` : key,
      page: positiveInt(params, 'page') ?? 1,
    }
  }, [params])

  const products = useResource(`products:${params.toString()}`, () => listProducts(query))
  const page = products.data
  const filtered = Boolean(query.q) || Boolean(query.status)

  function apply(next: ProductQuery) {
    setParams(
      toSearchParams({
        q: next.q,
        status: next.status,
        sort: next.q ? undefined : next.sort,
        page: next.page === 1 ? undefined : next.page,
      }),
    )
  }

  return (
    <ListPage
      title="Products"
      description="The catalog orders are placed against."
      primaryAction={<Button onClick={() => navigate('/products/new')}>Add product</Button>}
      filters={
        <form
          key={query.q ?? ''}
          className="flex flex-wrap items-end gap-3"
          onSubmit={(event) => {
            event.preventDefault()
            const entered = new FormData(event.currentTarget).get('q')
            apply({ ...query, q: String(entered ?? '').trim() || undefined, page: 1 })
          }}
        >
          <div className="grid gap-1.5">
            <Label htmlFor="product-search" className="text-caption text-muted-foreground">
              Search
            </Label>
            <Input
              id="product-search"
              type="search"
              name="q"
              defaultValue={query.q ?? ''}
              placeholder="Title or description"
              className="h-8 w-64"
            />
          </div>
          <FilterSelect
            label="Status"
            value={query.status}
            allLabel="Draft and active"
            options={STATUS_FILTERS.map((status) => ({
              value: status,
              label: productStatusLabels[status],
            }))}
            onChange={(status) =>
              apply({ ...query, status: status as ProductStatus | undefined, page: 1 })
            }
          />
          <Button type="submit" variant="secondary" size="sm">
            Search
          </Button>
        </form>
      }
      note={
        query.q
          ? 'Search results are ranked by how well they match, so they are not sorted by column.'
          : undefined
      }
      columns={columns}
      rows={page?.items ?? []}
      rowKey={(product) => product.id}
      status={products.status}
      filtersApplied={filtered}
      onClearFilters={() => apply({ page: 1 })}
      filteredEmptyTitle="No products match"
      emptyTitle="The catalog is empty"
      emptyDescription="Add the first product. It starts as a draft and can be activated once it is ready to sell."
      emptyAction={<Button onClick={() => navigate('/products/new')}>Add product</Button>}
      errorDescription={products.error?.message}
      onRetry={products.reload}
      sort={query.sort}
      onSort={query.q ? undefined : (next) => apply({ ...query, sort: next, page: 1 })}
      pagination={
        page && (
          <Pagination
            page={page.page}
            pageSize={page.pageSize}
            total={page.total}
            onChange={(next) => apply({ ...query, page: next })}
          />
        )
      }
    />
  )
}
