import type { ReactNode } from 'react'

export function PageHeader({
  breadcrumb,
  title,
  description,
  actions,
}: {
  breadcrumb?: ReactNode
  title: string
  description?: string
  actions?: ReactNode
}) {
  return (
    <header className="flex flex-wrap items-start justify-between gap-4">
      <div className="min-w-0">
        {breadcrumb && <div className="mb-1 text-caption text-muted-foreground">{breadcrumb}</div>}
        <h1 className="text-title font-semibold tracking-[-0.011em]">{title}</h1>
        {description && (
          <p className="mt-1 max-w-[70ch] text-muted-foreground">{description}</p>
        )}
      </div>
      {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
    </header>
  )
}
