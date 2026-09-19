import type { ReactNode } from 'react'

export function PageHeader({
  title,
  description,
  breadcrumb,
  actions,
}: {
  title: string
  description?: string
  breadcrumb?: ReactNode
  actions?: ReactNode
}) {
  return (
    <header className="flex flex-wrap items-start justify-between gap-4">
      <div className="min-w-0">
        {breadcrumb && <div className="mb-1 text-caption text-muted-foreground">{breadcrumb}</div>}
        <h1 className="text-title font-semibold tracking-[-0.011em]">{title}</h1>
        {description && <p className="mt-1 max-w-[70ch] text-muted-foreground">{description}</p>}
      </div>
      {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
    </header>
  )
}

export function PageStack({ children }: { children: ReactNode }) {
  return <div className="flex w-full max-w-[120rem] flex-col gap-5">{children}</div>
}

export function Section({
  title,
  description,
  actions,
  children,
}: {
  title?: string
  description?: string
  actions?: ReactNode
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
      <div className="flex flex-col gap-4 p-4">{children}</div>
    </section>
  )
}

// The value is what someone came to read, so it is the heavier of the pair. The
// label sitting above it at caption size is what makes the value scannable.
export function DefinitionList({
  items,
}: {
  items: { term: string; value: ReactNode; lead?: boolean }[]
}) {
  return (
    <dl className="grid gap-4 rounded-lg border border-border bg-card p-4 sm:grid-cols-2 xl:grid-cols-4">
      {items.map((item) => (
        <div key={item.term} className="min-w-0">
          <dt className="text-caption text-muted-foreground">{item.term}</dt>
          <dd className={item.lead ? 'mt-0.5 text-section font-semibold tabular-nums' : 'mt-0.5 font-medium'}>
            {item.value}
          </dd>
        </div>
      ))}
    </dl>
  )
}
