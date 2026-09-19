import type { ReactNode } from 'react'
import { DataTable } from '../ui'
import type { Column } from '../ui'
import { PageBlock } from './PageBlock'
import { PageHeader } from './PageHeader'

export type ListPageProps<T> = {
  title: string
  description?: string
  primaryAction?: ReactNode
  filters?: ReactNode
  note?: ReactNode
  // Sits between the header and the table: a reveal or a warning that must not
  // be scrolled out of view behind the list.
  banner?: ReactNode
  columns: Column<T>[]
  rows: T[]
  rowKey: (row: T) => string
  status: 'loading' | 'ready' | 'error'
  errorDescription?: string
  onRetry?: () => void
  // A list filtered down to nothing is a different screen from a list with
  // nothing in it, and it needs a way back rather than an invitation to create.
  filtersApplied?: boolean
  onClearFilters?: () => void
  emptyTitle: string
  emptyDescription?: string
  emptyAction?: ReactNode
  filteredEmptyTitle?: string
  sort?: string
  onSort?: (sort: string | undefined) => void
  rowHref?: (row: T) => string
  rowActions?: (row: T) => ReactNode
  pagination?: ReactNode
}

export function ListPage<T>({
  title,
  description,
  primaryAction,
  filters,
  note,
  banner,
  pagination,
  filtersApplied,
  onClearFilters,
  filteredEmptyTitle,
  emptyTitle,
  emptyDescription,
  emptyAction,
  ...table
}: ListPageProps<T>) {
  const filtered = filtersApplied === true
  return (
    <PageBlock>
      <PageHeader title={title} description={description} actions={primaryAction} />

      {banner}

      <div className="overflow-hidden rounded-lg border border-border bg-card">
        {filters && (
          <div className="flex flex-wrap items-end gap-3 border-b border-border px-4 py-3">
            {filters}
          </div>
        )}
        {note && (
          <p className="border-b border-border bg-muted px-4 py-2 text-caption text-muted-foreground">
            {note}
          </p>
        )}
        <DataTable
          {...table}
          emptyTitle={
            filtered ? (filteredEmptyTitle ?? `No ${title.toLowerCase()} match these filters`) : emptyTitle
          }
          emptyDescription={
            filtered
              ? 'Try a different search, or clear the filters to see everything.'
              : emptyDescription
          }
          emptyAction={filtered ? onClearFilters && <ClearFilters onClick={onClearFilters} /> : emptyAction}
        />
        {pagination && <div className="border-t border-border px-4 py-3">{pagination}</div>}
      </div>
    </PageBlock>
  )
}

function ClearFilters({ onClick }: { onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="inline-flex h-8 cursor-pointer items-center rounded-md border border-border bg-card px-3 text-label font-medium hover:bg-muted"
    >
      Clear filters
    </button>
  )
}
