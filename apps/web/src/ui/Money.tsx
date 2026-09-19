import { formatMoney } from '../api/format'

export function Money({
  minor,
  currency,
  className = '',
}: {
  minor: number
  currency: string
  className?: string
}) {
  return <span className={`tabular-nums ${className}`}>{formatMoney(minor, currency)}</span>
}
