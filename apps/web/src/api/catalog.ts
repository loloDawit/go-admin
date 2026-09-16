import { withScenario } from './client'
import type { components } from './generated/catalog'
import { api } from './http'

export type ProductStatus = components['schemas']['ProductStatus']
export type Product = components['schemas']['Product']
export type ProductImage = components['schemas']['Image']
export type CreateProductRequest = components['schemas']['CreateProductRequest']
export type UpdateProductRequest = components['schemas']['UpdateProductRequest']

export type ProductPage = {
  items: Product[]
  page: number
  pageSize: number
  total: number
}

export const productStatusLabels: Record<ProductStatus, string> = {
  draft: 'Draft',
  active: 'Active',
  archived: 'Archived',
}

// A draft product cannot be ordered. Orders refuses anything that is not
// active, so the distinction is operational rather than cosmetic.
export const productStatusHelp: Record<ProductStatus, string> = {
  draft: 'Not yet on sale. Activate it before it can be ordered.',
  active: 'On sale and available to order.',
  archived: 'Withdrawn from sale. Past orders keep what they bought.',
}

export type ProductQuery = {
  q?: string
  status?: ProductStatus
  sort?: string
  page?: number
  pageSize?: number
}

function queryString(query: ProductQuery): string {
  const params = new URLSearchParams()
  if (query.q) params.set('q', query.q)
  if (query.status) params.set('status', query.status)
  if (query.sort) params.set('sort', query.sort)
  if (query.page && query.page > 1) params.set('page', String(query.page))
  if (query.pageSize) params.set('pageSize', String(query.pageSize))
  const encoded = params.toString()
  return encoded ? `?${encoded}` : ''
}

const emptyPage: ProductPage = { items: [], page: 1, pageSize: 20, total: 0 }

// Search is its own endpoint with a required q: an optional q on the list would
// make an empty search indistinguishable from a full listing.
export function listProducts(query: ProductQuery = {}): Promise<ProductPage> {
  const path = query.q
    ? `/api/v1/products/search${queryString(query)}`
    : `/api/v1/products${queryString(query)}`
  return withScenario(() => api.get<ProductPage>(path), emptyPage)
}

export function getProduct(id: string): Promise<Product> {
  return withScenario(() => api.get<Product>(`/api/v1/products/${id}`))
}

export function createProduct(body: CreateProductRequest): Promise<Product> {
  return api.post<Product>('/api/v1/products', body)
}

export function updateProduct(id: string, body: UpdateProductRequest): Promise<Product> {
  return api.patch<Product>(`/api/v1/products/${id}`, body)
}

export function activateProduct(id: string): Promise<Product> {
  return api.post<Product>(`/api/v1/products/${id}/activate`)
}

export function archiveProduct(id: string): Promise<Product> {
  return api.post<Product>(`/api/v1/products/${id}/archive`)
}

// Presigned URLs expire, so this is called when a product is shown rather than
// cached alongside it.
export function listProductImages(id: string): Promise<ProductImage[]> {
  return withScenario(
    async () => (await api.get<{ images: ProductImage[] }>(`/api/v1/products/${id}/images`)).images,
    [],
  )
}

export function uploadProductImage(id: string, file: File): Promise<ProductImage> {
  const body = new FormData()
  body.append('file', file)
  return api.post<ProductImage>(`/api/v1/products/${id}/images`, body)
}

export function deleteProductImage(productId: string, imageId: string): Promise<void> {
  return api.delete<void>(`/api/v1/products/${productId}/images/${imageId}`)
}
