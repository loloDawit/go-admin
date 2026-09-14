import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  Button,
  DefinitionList,
  PageHeader,
  PageStack,
  Section,
  StateBlock,
  Status,
} from '../ui'
import { getProduct, productStatusLabels } from '../api/catalog'
import { formatDate, formatMoney } from '../api/format'
import { useResource } from '../api/useResource'
import { useScenario } from '../app/useScenario'
import { productStatusTones } from '../app/statusTones'

export function ProductDetail() {
  const scenario = useScenario()
  const navigate = useNavigate()
  const { productId = '' } = useParams()
  const product = useResource(() => getProduct(scenario, productId), [scenario, productId])

  if (product.status === 'loading') {
    return <StateBlock title="Loading product" description="Fetching the catalog entry." />
  }

  if (product.status === 'error') {
    return (
      <StateBlock
        tone="error"
        title="This product could not be loaded"
        description={product.error?.message}
        action={
          <Button variant="secondary" onClick={product.reload}>
            Try again
          </Button>
        }
      />
    )
  }

  if (!product.data) {
    return (
      <StateBlock
        title="No such product"
        description="It may have been removed from the catalog."
        action={<Link to="/products">Back to products</Link>}
      />
    )
  }

  const current = product.data

  return (
    <PageStack>
      <PageHeader
        breadcrumb={<Link to="/products">Products</Link>}
        title={current.name}
        description={current.description}
        actions={
          <Button variant="primary" onClick={() => navigate(`/products/${current.id}/edit`)}>
            Edit product
          </Button>
        }
      />

      <DefinitionList
        items={[
          { term: 'SKU', value: current.sku },
          { term: 'Category', value: current.category },
          {
            term: 'Status',
            value: (
              <Status tone={productStatusTones[current.status]}>
                {productStatusLabels[current.status]}
              </Status>
            ),
          },
          { term: 'Price', value: formatMoney(current.priceCents) },
          {
            term: 'In stock',
            value: current.stock === 0 ? 'Out of stock' : String(current.stock),
          },
          { term: 'Last updated', value: formatDate(current.updatedAt) },
        ]}
      />

      <Section title="Stock movements" description="Available once the catalog service reports them.">
        <StateBlock
          title="No movements recorded"
          description="Adjustments made in this back office will be listed here."
        />
      </Section>
    </PageStack>
  )
}
