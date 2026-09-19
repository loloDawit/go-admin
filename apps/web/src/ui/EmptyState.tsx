import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'

export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  // h3 suits an empty state inside a table or a section. A page that is
  // nothing but an empty state — a 404, a 403 — owns the page heading.
  as: Heading = 'h3',
}: {
  icon?: LucideIcon
  title: string
  description?: string
  action?: ReactNode
  as?: 'h1' | 'h2' | 'h3'
}) {
  return (
    <div className="flex flex-col items-center gap-2 px-6 py-14 text-center">
      {Icon && <Icon className="size-6 text-subtle-foreground" strokeWidth={1.5} aria-hidden />}
      <Heading className="text-section font-semibold">{title}</Heading>
      {description && <p className="max-w-[48ch] text-muted-foreground">{description}</p>}
      {action && <div className="mt-2">{action}</div>}
    </div>
  )
}
