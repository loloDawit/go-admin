import { ApiError } from '../interfaces/user';

const BASE_URL = process.env.REACT_APP_API_URL ?? 'http://localhost:8080';

export class ApiRequestError extends Error {
  constructor(public status: number, public code: string, message: string) {
    super(message);
  }
}

/** Turns a non-2xx response into a typed error. */
export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${BASE_URL}/api/v1${path}`, {
    ...init,
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...init.headers }
  });

  if (!response.ok) {
    let body: ApiError = { code: 'unknown', message: response.statusText };
    try {
      body = await response.json();
    } catch {
      // A non-JSON body (a proxy error page) keeps the default.
    }
    throw new ApiRequestError(response.status, body.code, body.message);
  }

  if (response.status === 204) {
    return undefined as unknown as T;
  }
  return response.json() as Promise<T>;
}
