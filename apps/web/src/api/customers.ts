import { withScenario } from './client'
import type { components } from './generated/orders'
import { api } from './http'

export type Customer = components['schemas']['Customer']
export type CustomerPage = components['schemas']['CustomerPage']
export type LifetimeValue = components['schemas']['LifetimeValue']

const emptyPage: CustomerPage = { items: [], page: 1, pageSize: 20, total: 0 }

export function listCustomers(page = 1): Promise<CustomerPage> {
  const query = page > 1 ? `?page=${page}` : ''
  return withScenario(() => api.get<CustomerPage>(`/api/v1/customers${query}`), emptyPage)
}

export function getCustomer(id: string): Promise<Customer> {
  return withScenario(() => api.get<Customer>(`/api/v1/customers/${id}`))
}

export function createCustomer(email: string, name: string): Promise<Customer> {
  return api.post<Customer>('/api/v1/customers', { email, name })
}

// Refused with mixed_currency_history when the customer's orders span more
// than one currency, which is a real answer and not a failure to show.
export function getLifetimeValue(id: string): Promise<LifetimeValue> {
  return api.get<LifetimeValue>(`/api/v1/customers/${id}/lifetime-value`)
}

export function listCustomerOrders(id: string, page = 1) {
  const query = page > 1 ? `?page=${page}` : ''
  return withScenario(
    () => api.get<components['schemas']['OrderHistoryPage']>(`/api/v1/customers/${id}/orders${query}`),
    { items: [], page: 1, pageSize: 20, total: 0 },
  )
}
