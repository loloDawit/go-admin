const digitsByCurrency = new Map<string, number>()

export function minorDigits(currency: string): number {
  let digits = digitsByCurrency.get(currency)
  if (digits === undefined) {
    digits =
      new Intl.NumberFormat('en-US', { style: 'currency', currency }).resolvedOptions()
        .maximumFractionDigits ?? 0
    digitsByCurrency.set(currency, digits)
  }
  return digits
}

// Money is an integer count of minor units. Parsing via parseFloat would make a
// price a binary float, where 1.15 * 100 is 114.99999999999999.
export function parseMoney(input: string, currency: string): number | undefined {
  const digits = minorDigits(currency)
  const pattern =
    digits === 0 ? /^(\d+)$/ : new RegExp(`^(\\d+)(?:\\.(\\d{1,${digits}}))?$`)
  const match = pattern.exec(input.trim())
  if (!match) return undefined
  const [, whole, fraction = ''] = match
  return Number(whole) * 10 ** digits + Number(fraction.padEnd(digits, '0') || '0')
}

export function moneyInputValue(minor: number, currency: string): string {
  const digits = minorDigits(currency)
  if (digits === 0) return String(minor)
  const scale = 10 ** digits
  return `${Math.floor(minor / scale)}.${String(minor % scale).padStart(digits, '0')}`
}
