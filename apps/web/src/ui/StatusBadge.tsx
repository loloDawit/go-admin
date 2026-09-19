import type { StatusTone } from './tone'

const TONES: Record<StatusTone, string> = {
  neutral: 'bg-muted text-muted-foreground',
  info: 'bg-info-soft text-info',
  success: 'bg-success-soft text-success',
  warning: 'bg-warning-soft text-warning',
  danger: 'bg-destructive-soft text-destructive',
}

export function StatusBadge({ tone, children }: { tone: StatusTone; children: string }) {
  return (
    <span
      className={`inline-flex items-center rounded-sm px-1.5 py-0.5 text-caption font-medium whitespace-nowrap ${TONES[tone]}`}
    >
      {children}
    </span>
  )
}
