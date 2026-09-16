import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  Alert,
  Button,
  PageHeader,
  PageStack,
  Section,
  StateBlock,
  TextField,
  TextareaField,
} from '../ui'
import { createProduct, getProduct, updateProduct } from '../api/catalog'
import type { Product } from '../api/catalog'
import { isApiError } from '../api/client'
import { moneyInputValue, parseMoney } from '../api/money'
import { useResource } from '../api/useResource'
import styles from './ProductForm.module.css'

const DEFAULT_CURRENCY = 'USD'

type Errors = { sku?: string; title?: string; price?: string }

function Fields({ product }: { product?: Product }) {
  const navigate = useNavigate()
  const currency = product?.currency ?? DEFAULT_CURRENCY
  const [sku, setSku] = useState(product?.sku ?? '')
  const [title, setTitle] = useState(product?.title ?? '')
  const [description, setDescription] = useState(product?.description ?? '')
  const [price, setPrice] = useState(product ? moneyInputValue(product.priceMinor, currency) : '')
  const [errors, setErrors] = useState<Errors>({})
  const [failure, setFailure] = useState<string>()
  const [saving, setSaving] = useState(false)

  async function submit() {
    const priceMinor = parseMoney(price, currency)
    const found: Errors = {}
    if (!product && sku.trim() === '') found.sku = 'Give the product a SKU.'
    if (title.trim() === '') found.title = 'Give the product a title.'
    if (price.trim() !== '' && priceMinor === undefined) {
      found.price = `Enter an amount in ${currency}, such as 22.00.`
    }
    setErrors(found)
    if (Object.keys(found).length > 0) return

    setSaving(true)
    setFailure(undefined)
    try {
      const saved = product
        ? await updateProduct(product.id, { title, description, priceMinor })
        : await createProduct({ sku, title, description, priceMinor, currency })
      navigate(`/products/${saved.id}`)
    } catch (cause) {
      if (isApiError(cause) && cause.code === 'sku_taken') {
        setErrors({ sku: 'Another product already uses this SKU.' })
      } else {
        setFailure(isApiError(cause) ? cause.message : 'The product could not be saved.')
      }
      setSaving(false)
    }
  }

  return (
    <form
      className={styles.form}
      onSubmit={(event) => {
        event.preventDefault()
        void submit()
      }}
    >
      {failure && <Alert tone="danger" title={failure} />}

      <Section title="Details">
        <div className={styles.grid}>
          {product ? (
            <TextField label="SKU" value={product.sku} readOnly help="A SKU cannot be changed." />
          ) : (
            <TextField
              label="SKU"
              value={sku}
              error={errors.sku}
              onChange={(event) => setSku(event.target.value)}
              placeholder="MUG-CLY-300"
            />
          )}
          <TextField
            label="Title"
            value={title}
            error={errors.title}
            onChange={(event) => setTitle(event.target.value)}
            placeholder="Clay mug, 300 ml"
          />
          <TextField
            label={`Price (${currency})`}
            inputMode="decimal"
            value={price}
            error={errors.price}
            optional
            onChange={(event) => setPrice(event.target.value)}
            placeholder="22.00"
          />
        </div>
        <TextareaField
          label="Description"
          optional
          value={description}
          onChange={(event) => setDescription(event.target.value)}
        />
      </Section>

      <div className={styles.actions}>
        <Button type="submit" variant="primary" loading={saving}>
          {product ? 'Save changes' : 'Create product'}
        </Button>
        <Button variant="ghost" onClick={() => navigate(product ? `/products/${product.id}` : '/products')}>
          Cancel
        </Button>
      </div>
    </form>
  )
}

export function ProductForm() {
  const { productId } = useParams()

  return (
    <PageStack>
      <PageHeader
        breadcrumb={<Link to="/products">Products</Link>}
        title={productId ? 'Edit product' : 'New product'}
        description={
          productId
            ? undefined
            : 'A new product starts as a draft. Activate it from its page once it is ready to sell.'
        }
      />
      {productId ? <EditFields productId={productId} /> : <Fields />}
    </PageStack>
  )
}

function EditFields({ productId }: { productId: string }) {
  const product = useResource(`product:${productId}`, () => getProduct(productId))

  if (product.status === 'loading') {
    return <StateBlock title="Loading product" description="Fetching the catalog entry." />
  }
  if (product.status === 'error' || !product.data) {
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
  return <Fields key={product.data.id} product={product.data} />
}
