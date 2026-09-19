import { withScenario } from './client'
import type { components } from './generated/orders'
import { api } from './http'

export type DashboardReport = components['schemas']['DashboardReport']
export type RevenueDay = components['schemas']['RevenueDay']

const empty: DashboardReport = { counts: {}, recent: [], revenue: [] }

let inflight: Promise<DashboardReport> | undefined

// The dashboard route and the sidebar badge both call this on the same load; sharing the in-flight promise sends one request, not two.
export function getDashboard(): Promise<DashboardReport> {
  inflight ??= withScenario(() => api.get<DashboardReport>('/api/v1/reports/dashboard'), empty).finally(
    () => {
      inflight = undefined
    },
  )
  return inflight
}
