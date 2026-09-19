import type { ReactNode } from 'react'
import { BulkBar, DataTable } from '../ui'
import type { BulkAction, BulkProgress, Column, Selection } from '../ui'
import { PageBlock } from './PageBlock'
import { PageHeader } from './PageHeader'

export type ListPageProps<T> = {
  title: string
  description?: string
  // Keys the column-visibility persistence; defaults to the title so a screen
  // that never names one still gets its own key rather than sharing DataTable's.
  tableId?: string
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
  // Selection ships only with something to do: a screen that passes
  // `selection` without `bulkActions` is a defect, not a lighter feature.
  selection?: Selection<T>
  bulkActions?: BulkAction[]
  bulkProgress?: BulkProgress
}

export function ListPage<T>({
  title,
  description,
  tableId,
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
  selection,
  bulkActions,
  bulkProgress,
  ...table
}: ListPageProps<T>) {
  const filtered = filtersApplied === true
  // The bar and the table's own "N of M" must agree: count against the rows
  // actually on the page, not the raw selected-id set, so a row a reload
  // dropped (another agent archived it, say) is not counted as selected.
  const selectedCount = selection
    ? table.rows.filter((row) => selection.selected.has(table.rowKey(row))).length
    : 0
  return (
    <PageBlock>
      <PageHeader title={title} description={description} actions={primaryAction} />

      {banner}

      <div className="overflow-hidden rounded-lg border border-border bg-card">
        {note && (
          <p className="border-b border-border bg-muted px-4 py-2 text-caption text-muted-foreground">
            {note}
          </p>
        )}
        {selection && selectedCount > 0 && (
          <BulkBar
            count={selectedCount}
            total={table.rows.length}
            actions={bulkActions ?? []}
            progress={bulkProgress}
            onClear={() => selection.onChange(new Set())}
          />
        )}
        <DataTable
          toolbar={filters}
          {...table}
          selection={selection}
          tableId={tableId ?? title.toLowerCase().replace(/\s+/g, '-')}
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
