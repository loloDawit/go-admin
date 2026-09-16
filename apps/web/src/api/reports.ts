import { withScenario } from './client'
import type { components } from './generated/orders'
import { api } from './http'

export type DashboardReport = components['schemas']['DashboardReport']
export type RevenueDay = components['schemas']['RevenueDay']

const empty: DashboardReport = { counts: {}, recent: [], revenue: [] }

export function getDashboard(): Promise<DashboardReport> {
  return withScenario(() => api.get<DashboardReport>('/api/v1/reports/dashboard'), empty)
}
