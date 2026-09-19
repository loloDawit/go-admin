import { useRef, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { toast } from 'sonner'
import {
  Alert,
  Button,
  DefinitionList,
  Dialog,
  PageHeader,
  PageStack,
  Section,
  StateBlock,
  Status,
} from '../ui'
import {
  activateProduct,
  archiveProduct,
  deleteProductImage,
  getProduct,
  listProductImages,
  productStatusHelp,
  productStatusLabels,
  uploadProductImage,
} from '../api/catalog'
import type { ProductImage } from '../api/catalog'
import { isApiError } from '../api/client'
import { formatDate, formatMoney } from '../api/format'
import { useResource } from '../api/useResource'
import { productStatusTones } from '../app/statusTones'

export function ProductDetail() {
  const navigate = useNavigate()
  const { productId = '' } = useParams()
  const product = useResource(`product:${productId}`, () => getProduct(productId))
  const images = useResource(`images:${productId}`, () => listProductImages(productId))
  const fileInput = useRef<HTMLInputElement>(null)
  const [busy, setBusy] = useState(false)
  const [failure, setFailure] = useState<string>()
  const [confirmArchive, setConfirmArchive] = useState(false)
  const retried = useRef(new Set<string>())

  // The confirmation names what happened in the same words the button used.
  async function run(action: () => Promise<unknown>, done: string) {
    setBusy(true)
    setFailure(undefined)
    try {
      await action()
      product.reload()
      images.reload()
      toast.success(done)
    } catch (cause) {
      setFailure(isApiError(cause) ? cause.message : 'The change could not be saved.')
    } finally {
      setBusy(false)
    }
  }

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

  const current = product.data
  const editable = current.status !== 'archived'

  // A presigned URL expires, so a page left open eventually serves broken images.
  // The first failure per image refetches the list rather than showing a gap.
  function refreshOnce(image: ProductImage) {
    if (retried.current.has(image.id)) return
    retried.current.add(image.id)
    images.reload()
  }

  return (
    <PageStack>
      <PageHeader
        breadcrumb={<Link to="/products">Products</Link>}
        title={current.title}
        description={current.description || undefined}
        actions={
          <>
            {editable && (
              <Button onClick={() => navigate(`/products/${current.id}/edit`)}>Edit</Button>
            )}
            {current.status === 'draft' && (
              <Button variant="primary" loading={busy} onClick={() => run(() => activateProduct(current.id), 'Product activated')}>
                Activate
              </Button>
            )}
            {current.status === 'active' && (
              <Button variant="danger" onClick={() => setConfirmArchive(true)}>
                Archive
              </Button>
            )}
          </>
        }
      />

      {failure && <Alert tone="danger" title={failure} />}

      <DefinitionList
        items={[
          { term: 'SKU', value: current.sku },
          {
            term: 'Status',
            value: (
              <div className="flex flex-col items-start gap-1">
                <Status tone={productStatusTones[current.status]}>
                  {productStatusLabels[current.status]}
                </Status>
                <p className="text-caption text-muted-foreground">{productStatusHelp[current.status]}</p>
              </div>
            ),
          },
          { term: 'Price', value: formatMoney(current.priceMinor, current.currency) },
          { term: 'Added', value: formatDate(current.createdAt) },
          { term: 'Last updated', value: formatDate(current.updatedAt) },
        ]}
      />

      <Section
        title="Images"
        actions={
          editable && (
            <>
              <input
                ref={fileInput}
                type="file"
                accept="image/png,image/jpeg,image/webp"
                hidden
                onChange={(event) => {
                  const file = event.target.files?.[0]
                  event.target.value = ''
                  if (file) void run(() => uploadProductImage(current.id, file), 'Image added')
                }}
              />
              <Button loading={busy} onClick={() => fileInput.current?.click()}>
                Upload image
              </Button>
            </>
          )
        }
      >
        {images.status === 'error' && (
          <StateBlock
            tone="error"
            title="The images could not be loaded"
            description={images.error?.message}
            action={
              <Button variant="secondary" onClick={images.reload}>
                Try again
              </Button>
            }
          />
        )}
        {images.status === 'ready' && images.data?.length === 0 && (
          <StateBlock title="No images yet" description="Upload one to show the product." />
        )}
        {images.status === 'ready' && (images.data?.length ?? 0) > 0 && (
          <ul className="grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(10rem,1fr))]">
            {images.data?.map((image) => (
              <li key={image.id} className="flex flex-col gap-2 rounded-md border border-border p-2">
                <img
                  className="aspect-square w-full rounded-sm bg-muted object-cover"
                  src={image.url}
                  alt={image.alt}
                  onError={() => refreshOnce(image)}
                />
                {editable && (
                  <Button
                    size="sm"
                    variant="ghost"
                    loading={busy}
                    onClick={() => void run(() => deleteProductImage(current.id, image.id), 'Image removed')}
                  >
                    Remove
                  </Button>
                )}
              </li>
            ))}
          </ul>
        )}
      </Section>

      <Dialog
        open={confirmArchive}
        title="Archive this product?"
        description="It stops being orderable and disappears from the default list. There is no way to bring it back."
        onClose={() => setConfirmArchive(false)}
        footer={
          <>
            <Button onClick={() => setConfirmArchive(false)}>Cancel</Button>
            <Button
              variant="dangerSolid"
              loading={busy}
              onClick={() => {
                setConfirmArchive(false)
                void run(() => archiveProduct(current.id), 'Product archived')
              }}
            >
              Archive
            </Button>
          </>
        }
      />
    </PageStack>
  )
}
