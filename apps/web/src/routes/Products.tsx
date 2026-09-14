import { Link, useNavigate } from 'react-router-dom'
import { Button, DataTable, PageHeader, PageStack, Status } from '../ui'
import type { Column } from '../ui'
import { listProducts, productStatusLabels } from '../api/catalog'
import type { Product } from '../api/catalog'
import { formatDate, formatMoney } from '../api/format'
import { useResource } from '../api/useResource'
import { useScenario } from '../app/useScenario'
import { productStatusTones } from '../app/statusTones'

const columns: Column<Product>[] = [
  {
    key: 'name',
    header: 'Product',
    cell: (product) => <Link to={`/products/${product.id}`}>{product.name}</Link>,
  },
  { key: 'sku', header: 'SKU', cell: (product) => product.sku },
  { key: 'category', header: 'Category', cell: (product) => product.category },
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
    key: 'stock',
    header: 'In stock',
    numeric: true,
    cell: (product) => (product.stock === 0 ? 'Out of stock' : product.stock),
  },
  {
    key: 'price',
    header: 'Price',
    numeric: true,
    cell: (product) => formatMoney(product.priceCents),
  },
  { key: 'updated', header: 'Updated', cell: (product) => formatDate(product.updatedAt) },
]

export function Products() {
  const scenario = useScenario()
  const navigate = useNavigate()
  const products = useResource(() => listProducts(scenario), [scenario])

  return (
    <PageStack>
      <PageHeader
        title="Products"
        description="The catalog the storefront sells from."
        actions={
          <Button variant="primary" onClick={() => navigate('/products/new')}>
            Add product
          </Button>
        }
      />
      <DataTable
        columns={columns}
        rows={products.data ?? []}
        rowKey={(product) => product.id}
        status={products.status}
        caption="All products"
        emptyTitle="The catalog is empty"
        emptyDescription="Add the first product and it becomes available to the storefront."
        emptyAction={
          <Button variant="primary" onClick={() => navigate('/products/new')}>
            Add product
          </Button>
        }
        errorDescription={products.error?.message}
        onRetry={products.reload}
      />
    </PageStack>
  )
}
