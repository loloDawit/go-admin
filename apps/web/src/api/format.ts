const dateTime = new Intl.DateTimeFormat('en-GB', {
  day: '2-digit',
  month: 'short',
  hour: '2-digit',
  minute: '2-digit',
})
const date = new Intl.DateTimeFormat('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })

const moneyFormats = new Map<string, Intl.NumberFormat>()

// Intl.NumberFormat throws a RangeError on a code it does not recognise, which
// unmounts the tree above it. A formatter must not be able to take down a page.
function isCurrencyCode(currency: string): boolean {
  return /^[A-Za-z]{3}$/.test(currency)
}

function moneyFormat(currency: string): Intl.NumberFormat {
  let format = moneyFormats.get(currency)
  if (!format) {
    format = new Intl.NumberFormat('en-US', { style: 'currency', currency })
    moneyFormats.set(currency, format)
  }
  return format
}

// Money is an integer count of the currency's smallest unit; dividing would make
// it a float. formatToParts lets the fraction be assembled as digits.
export function formatMoney(minor: number, currency: string): string {
  if (!isCurrencyCode(currency)) return `${minor} ${currency}`.trim()

  const format = moneyFormat(currency)
  const digits = format.resolvedOptions().maximumFractionDigits ?? 0
  const scale = 10 ** digits
  const negative = minor < 0
  const absolute = Math.abs(minor)
  const whole = Math.floor(absolute / scale)
  const fraction = absolute % scale

  const rendered = format.format(whole + fraction / scale)
  return negative && !rendered.startsWith('-') ? `-${rendered}` : rendered
}

export function formatDateTime(iso: string): string {
  if (!iso.includes('T')) return iso
  return dateTime.format(new Date(iso))
}

export function formatDate(iso: string): string {
  if (!iso.includes('T')) return iso
  return date.format(new Date(iso))
}
