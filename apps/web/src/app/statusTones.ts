import type { ProductStatus } from '../api/catalog'
import type { OrderStatus } from '../api/orders'
import type { StatusTone } from '../ui'

export const orderStatusTones: Record<OrderStatus, StatusTone> = {
  pending: 'warning',
  paid: 'info',
  packed: 'info',
  shipped: 'info',
  delivered: 'success',
  cancelled: 'neutral',
  refunded: 'danger',
}

export const productStatusTones: Record<ProductStatus, StatusTone> = {
  active: 'success',
  draft: 'warning',
  archived: 'neutral',
}

export const staffActiveTones: Record<'active' | 'inactive', StatusTone> = {
  active: 'success',
  inactive: 'neutral',
}
