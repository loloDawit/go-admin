import { CircleCheck, CircleDashed, CircleX, LoaderCircle, TriangleAlert } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { Badge } from '@/ui/shadcn/badge'
import type { StatusTone } from './tone'

// The block carries the colour in the icon and leaves the chip outlined: a row
// of filled colour blocks competes with the data it is describing.
const TONES: Record<StatusTone, { icon: LucideIcon; className: string }> = {
  neutral: { icon: CircleDashed, className: '' },
  info: { icon: LoaderCircle, className: 'text-blue-500 dark:text-blue-400' },
  success: { icon: CircleCheck, className: 'fill-green-500 dark:fill-green-400 text-background' },
  warning: { icon: TriangleAlert, className: 'text-amber-500 dark:text-amber-400' },
  danger: { icon: CircleX, className: 'text-destructive' },
}

export function StatusBadge({ tone, children }: { tone: StatusTone; children: string }) {
  const { icon: Icon, className } = TONES[tone]
  return (
    <Badge variant="outline" className="text-muted-foreground px-1.5">
      <Icon className={className} aria-hidden />
      {children}
    </Badge>
  )
}
