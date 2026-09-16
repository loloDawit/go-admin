export type { ApiError } from './http'
export { isApiError } from './http'

type Scenario = 'ready' | 'empty' | 'error' | 'slow'

const SLOW_LATENCY_MS = 4000

function currentScenario(): Scenario {
  const value = new URLSearchParams(window.location.search).get('mock')
  return value === 'empty' || value === 'error' || value === 'slow' ? value : 'ready'
}

// withScenario overrides a real call so the loading, empty and error states stay
// reachable without breaking a service. ?mock=error and ?mock=empty never reach
// the network; ?mock=slow does, after a delay.
export async function withScenario<T>(load: () => Promise<T>, emptyValue?: T): Promise<T> {
  const scenario = currentScenario()

  if (scenario === 'error') {
    throw { code: 'internal', message: 'The service did not respond. Try again in a moment.' }
  }
  if (scenario === 'empty' && emptyValue !== undefined) return emptyValue
  if (scenario === 'slow') {
    await new Promise((resolve) => setTimeout(resolve, SLOW_LATENCY_MS))
  }
  return load()
}
