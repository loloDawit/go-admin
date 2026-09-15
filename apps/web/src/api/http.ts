import type { components } from './generated/identity'

export type ApiErrorCode = components['schemas']['Error']['code'] | 'network' | 'unknown'

export type ApiError = {
  code: ApiErrorCode
  message: string
}

export function isApiError(value: unknown): value is ApiError {
  return typeof value === 'object' && value !== null && 'code' in value && 'message' in value
}

type Listener = () => void

let unauthorizedListener: Listener | undefined
let passwordChangeRequiredListener: Listener | undefined

// Set once by the auth provider. A module-level singleton, not a React context value: every
// request() call anywhere in the tree must reach it, including calls made outside a component.
export function onUnauthorized(listener: Listener): void {
  unauthorizedListener = listener
}

export function onPasswordChangeRequired(listener: Listener): void {
  passwordChangeRequiredListener = listener
}

type RequestOptions = {
  method?: 'GET' | 'POST' | 'PATCH' | 'DELETE'
  body?: unknown
  // Login legitimately answers 401 for wrong credentials; that is not a dead session.
  suppressUnauthorized?: boolean
}

const NETWORK_ERROR: ApiError = {
  code: 'network',
  message: 'Could not reach the server. Check your connection and try again.',
}

const UNKNOWN_ERROR: ApiError = {
  code: 'unknown',
  message: 'Something went wrong.',
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  let response: Response
  try {
    response = await fetch(path, {
      method: options.method ?? 'GET',
      credentials: 'include',
      headers: options.body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
      body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
    })
  } catch {
    throw NETWORK_ERROR
  }

  if (response.status === 401 && !options.suppressUnauthorized) {
    unauthorizedListener?.()
  }

  if (response.status === 204) return undefined as T

  const payload: unknown = await response.json().catch(() => undefined)

  if (!response.ok) {
    const failure = isApiError(payload) ? payload : UNKNOWN_ERROR
    if (failure.code === 'password_change_required') passwordChangeRequiredListener?.()
    throw failure
  }

  return payload as T
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown, options?: Omit<RequestOptions, 'method' | 'body'>) =>
    request<T>(path, { ...options, method: 'POST', body }),
  patch: <T>(path: string, body?: unknown) => request<T>(path, { method: 'PATCH', body }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
}
