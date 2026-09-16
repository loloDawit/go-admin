// List state lives in the URL. Parsing is total: a value the client did not
// generate is dropped rather than forwarded to a service that would refuse it.

export function oneOf<T extends string>(
  params: URLSearchParams,
  key: string,
  allowed: readonly T[],
): T | undefined {
  const value = params.get(key)
  return value !== null && (allowed as readonly string[]).includes(value) ? (value as T) : undefined
}

export function positiveInt(params: URLSearchParams, key: string): number | undefined {
  const value = params.get(key)
  if (value === null || !/^\d+$/.test(value)) return undefined
  const parsed = Number(value)
  return parsed > 0 ? parsed : undefined
}

export function text(params: URLSearchParams, key: string): string | undefined {
  return params.get(key)?.trim() || undefined
}

// An undefined entry is omitted, so a default is absent from the URL rather
// than spelled out: /products and /products?page=1 must not be two URLs.
export function toSearchParams(
  entries: Record<string, string | number | undefined>,
): URLSearchParams {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(entries)) {
    if (value !== undefined && value !== '') params.set(key, String(value))
  }
  return params
}
