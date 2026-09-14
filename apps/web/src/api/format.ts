const money = new Intl.NumberFormat('en-GB', { style: 'currency', currency: 'GBP' })
const dateTime = new Intl.DateTimeFormat('en-GB', {
  day: '2-digit',
  month: 'short',
  hour: '2-digit',
  minute: '2-digit',
})
const date = new Intl.DateTimeFormat('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })

export function formatMoney(cents: number): string {
  return money.format(cents / 100)
}

export function formatDateTime(iso: string): string {
  if (!iso.includes('T')) return iso
  return dateTime.format(new Date(iso))
}

export function formatDate(iso: string): string {
  if (!iso.includes('T')) return iso
  return date.format(new Date(iso))
}
