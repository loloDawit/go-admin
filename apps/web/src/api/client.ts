export type ApiError = {
  code: string
  message: string
}

export type Scenario = 'ready' | 'empty' | 'error' | 'slow'

export function scenarioFromSearch(search: string): Scenario {
  const value = new URLSearchParams(search).get('mock')
  return value === 'empty' || value === 'error' || value === 'slow' ? value : 'ready'
}

const LATENCY_MS = 220
const SLOW_LATENCY_MS = 4000

export async function respond<T>(scenario: Scenario, data: T, emptyValue?: T): Promise<T> {
  await new Promise((resolve) =>
    setTimeout(resolve, scenario === 'slow' ? SLOW_LATENCY_MS : LATENCY_MS),
  )
  if (scenario === 'error') {
    const failure: ApiError = {
      code: 'service_unavailable',
      message: 'The service did not respond. Try again in a moment.',
    }
    throw failure
  }
  if (scenario === 'empty' && emptyValue !== undefined) return emptyValue
  return data
}

export function isApiError(value: unknown): value is ApiError {
  return typeof value === 'object' && value !== null && 'code' in value && 'message' in value
}
