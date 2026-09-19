import type { ReactNode } from 'react'

export function SectionCard({
  title,
  description,
  actions,
  bleed,
  children,
}: {
  title?: string
  description?: string
  actions?: ReactNode
  // A table draws its own edges, so it sits flush against the card rather than
  // inside its padding.
  bleed?: boolean
  children: ReactNode
}) {
  return (
    <section className="overflow-hidden rounded-lg border border-border bg-card">
      {title && (
        <header className="flex flex-wrap items-start justify-between gap-3 border-b border-border px-4 py-3">
          <div className="min-w-0">
            <h2 className="text-section font-semibold">{title}</h2>
            {description && (
              <p className="mt-0.5 max-w-[80ch] text-caption text-muted-foreground">
                {description}
              </p>
            )}
          </div>
          {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
        </header>
      )}
      <div className={bleed ? '' : 'p-4'}>{children}</div>
    </section>
  )
}
