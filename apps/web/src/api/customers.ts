import { respond } from './client'
import type { Scenario } from './client'

export type Customer = {
  id: string
  name: string
  email: string
  orderCount: number
  lifetimeCents: number
  lastOrderAt: string
  location: string
}

const CUSTOMERS: Customer[] = [
  {
    id: 'c-1',
    name: 'Rita Alvarez',
    email: 'rita.alvarez@example.com',
    orderCount: 7,
    lifetimeCents: 41200,
    lastOrderAt: '2026-09-12T07:41:00Z',
    location: 'Leeds',
  },
  {
    id: 'c-2',
    name: 'Tom Whitfield',
    email: 'tom.whitfield@example.com',
    orderCount: 2,
    lifetimeCents: 9800,
    lastOrderAt: '2026-09-11T16:02:00Z',
    location: 'Manchester',
  },
  {
    id: 'c-3',
    name: 'Priya Nandi',
    email: 'priya.nandi@example.com',
    orderCount: 12,
    lifetimeCents: 88400,
    lastOrderAt: '2026-09-11T09:22:00Z',
    location: 'Bristol',
  },
  {
    id: 'c-4',
    name: 'Aisha Rahman',
    email: 'aisha.rahman@example.com',
    orderCount: 1,
    lifetimeCents: 4400,
    lastOrderAt: '2026-09-10T12:14:00Z',
    location: 'Cardiff',
  },
]

export function listCustomers(scenario: Scenario): Promise<Customer[]> {
  return respond(scenario, CUSTOMERS, [])
}

export function getCustomer(scenario: Scenario, id: string): Promise<Customer | undefined> {
  return respond(
    scenario,
    CUSTOMERS.find((customer) => customer.id === id),
    undefined,
  )
}
