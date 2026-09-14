import type { ProductStatus } from '../api/catalog'
import type { OrderStatus } from '../api/orders'
import type { StaffStatus } from '../api/identity'
import type { StatusTone } from '../ui'

export const orderStatusTones: Record<OrderStatus, StatusTone> = {
  pending: 'warning',
  paid: 'info',
  packed: 'info',
  shipped: 'success',
  cancelled: 'neutral',
  refunded: 'danger',
}

export const productStatusTones: Record<ProductStatus, StatusTone> = {
  active: 'success',
  draft: 'warning',
  discontinued: 'neutral',
}

export const staffStatusTones: Record<StaffStatus, StatusTone> = {
  active: 'success',
  invited: 'info',
  suspended: 'danger',
}
