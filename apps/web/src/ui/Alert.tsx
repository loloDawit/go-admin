import type { ReactNode } from 'react'
import { CircleAlert, CircleCheck, Info, TriangleAlert } from 'lucide-react'

export type AlertTone = 'info' | 'success' | 'warning' | 'danger'

const TONES = {
  info: { className: 'border-info/25 bg-info-soft text-info', icon: Info },
  success: { className: 'border-success/25 bg-success-soft text-success', icon: CircleCheck },
  warning: { className: 'border-warning/25 bg-warning-soft text-warning', icon: TriangleAlert },
  danger: {
    className: 'border-destructive/25 bg-destructive-soft text-destructive',
    icon: CircleAlert,
  },
} as const

export function Alert({
  tone = 'info',
  title,
  children,
  actions,
}: {
  tone?: AlertTone
  title: string
  children?: ReactNode
  actions?: ReactNode
}) {
  const { className, icon: Icon } = TONES[tone]
  return (
    <div
      className={`flex gap-2.5 rounded-md border p-3 ${className}`}
      role={tone === 'danger' ? 'alert' : 'status'}
    >
      <Icon className="mt-0.5 size-4 shrink-0" aria-hidden />
      <div className="min-w-0 flex-1">
        <p className="font-medium">{title}</p>
        {children && <div className="mt-1 text-foreground/80">{children}</div>}
        {actions && <div className="mt-2 flex flex-wrap gap-2">{actions}</div>}
      </div>
    </div>
  )
}
