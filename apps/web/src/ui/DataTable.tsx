import type { CSSProperties, ReactNode } from 'react'
import { ArrowDown, ArrowUp, ChevronsUpDown } from 'lucide-react'
import { Button } from '@/ui/shadcn/button'
import { Skeleton } from '@/ui/shadcn/skeleton'
import { EmptyState } from './EmptyState'

export type Column<T> = {
  key: string
  header: string
  numeric?: boolean
  // The server's allowlisted column name. A column without one is not sortable.
  sortKey?: string
  // Width sizes the column to its content. Exactly one column per table should
  // grow instead: without one, the browser hands the slack to the first column,
  // which is how an order number came to occupy 1400px at a workstation width.
  width?: string
  grow?: boolean
  cell: (row: T) => ReactNode
}

export type DataTableProps<T> = {
  columns: Column<T>[]
  rows: T[]
  rowKey: (row: T) => string
  status?: 'ready' | 'loading' | 'error'
  caption?: string
  emptyTitle?: string
  emptyDescription?: string
  emptyAction?: ReactNode
  errorTitle?: string
  errorDescription?: string
  onRetry?: () => void
  sort?: string
  onSort?: (sort: string | undefined) => void
}

const SKELETON_ROWS = 6

function activeKey(sort: string | undefined): string | undefined {
  return sort ? sort.replace(/^-/, '') : undefined
}

// Ascending, then descending, then off: a third click has to be able to undo the
// sort, or the default ordering becomes unreachable.
function cycle(sort: string | undefined, key: string): string | undefined {
  if (activeKey(sort) !== key) return key
  return sort?.startsWith('-') ? undefined : `-${key}`
}

function sizing<T>(column: Column<T>): CSSProperties {
  if (column.grow) return { width: 'auto' }
  if (column.width) return { width: column.width, minWidth: column.width }
  return { width: '1px' }
}

export function DataTable<T>({
  columns,
  rows,
  rowKey,
  status = 'ready',
  caption,
  emptyTitle = 'Nothing here yet',
  emptyDescription,
  emptyAction,
  errorTitle = 'This list could not be loaded',
  errorDescription = 'The service did not respond. Try again in a moment.',
  onRetry,
  sort,
  onSort,
}: DataTableProps<T>) {
  const colCount = columns.length
  // Without a growing column the table sizes to its content rather than
  // stretching: a fixed width on every column plus w-full just hands the slack
  // back out proportionally.
  const grows = columns.some((column) => column.grow)

  return (
    <div>
      {caption && (
        <div className="text-caption text-subtle-foreground flex items-baseline justify-between gap-3 py-2">
          <span>{caption}</span>
          {status === 'ready' && <span>{rows.length} shown</span>}
        </div>
      )}
      <div className="overflow-x-auto">
        <table className={`${grows ? 'w-full' : 'w-auto'} text-body border-collapse`}>
          <thead>
            <tr>
              {columns.map((column) => {
                const sortable = column.sortKey !== undefined && onSort !== undefined
                const active = sortable && activeKey(sort) === column.sortKey
                const direction = active
                  ? sort?.startsWith('-')
                    ? 'descending'
                    : 'ascending'
                  : 'none'
                return (
                  <th
                    key={column.key}
                    scope="col"
                    aria-sort={sortable ? direction : undefined}
                    style={sizing(column)}
                    className={`border-border bg-muted text-label text-muted-foreground border-b px-3 py-2 font-medium whitespace-nowrap ${
                      column.numeric ? 'text-right' : 'text-left'
                    }`}
                  >
                    {sortable ? (
                      <button
                        type="button"
                        // The header's own typography, not the Button primitive's:
                        // this is a column affordance, not an action.
                        className={`group hover:text-foreground inline-flex cursor-pointer items-center gap-1 ${
                          column.numeric ? 'flex-row-reverse' : ''
                        }`}
                        onClick={() => onSort(cycle(sort, column.sortKey as string))}
                      >
                        {column.header}
                        {active ? (
                          direction === 'ascending' ? (
                            <ArrowUp className="size-3" aria-hidden />
                          ) : (
                            <ArrowDown className="size-3" aria-hidden />
                          )
                        ) : (
                          // Which columns sort has to be visible before the
                          // first click, not after it.
                          <ChevronsUpDown
                            className="size-3 opacity-35 group-hover:opacity-70"
                            aria-hidden
                          />
                        )}
                      </button>
                    ) : (
                      column.header
                    )}
                  </th>
                )
              })}
            </tr>
          </thead>
          <tbody aria-busy={status === 'loading' || undefined}>
            {status === 'loading' &&
              Array.from({ length: SKELETON_ROWS }, (_, index) => (
                <tr key={index}>
                  {columns.map((column) => (
                    <td key={column.key} className="border-border border-b px-3 py-2">
                      <Skeleton className="h-3 w-full max-w-44" />
                    </td>
                  ))}
                </tr>
              ))}

            {status === 'error' && (
              <tr>
                <td colSpan={colCount} className="p-0">
                  {/* A failed list has to be announced, not just drawn. */}
                  <div role="alert">
                    <EmptyState
                      title={errorTitle}
                      description={errorDescription}
                      action={
                        onRetry && (
                          <Button variant="secondary" onClick={onRetry}>
                            Try again
                          </Button>
                        )
                      }
                    />
                  </div>
                </td>
              </tr>
            )}

            {status === 'ready' && rows.length === 0 && (
              <tr>
                <td colSpan={colCount} className="p-0">
                  <EmptyState
                    title={emptyTitle}
                    description={emptyDescription}
                    action={emptyAction}
                  />
                </td>
              </tr>
            )}

            {status === 'ready' &&
              rows.map((row) => (
                <tr key={rowKey(row)} className="hover:bg-muted [&:last-child>td]:border-b-0">
                  {columns.map((column) => (
                    <td
                      key={column.key}
                      style={sizing(column)}
                      className={`border-border border-b px-3 py-2 whitespace-nowrap ${
                        column.numeric ? 'text-right tabular-nums' : ''
                      }`}
                    >
                      {column.cell(row)}
                    </td>
                  ))}
                </tr>
              ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
