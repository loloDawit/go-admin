import type { CSSProperties, ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowDown, ArrowUp, ChevronsUpDown } from 'lucide-react'
import { Button } from '@/ui/shadcn/button'
import { Skeleton } from '@/ui/shadcn/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/ui/shadcn/table'
import { EmptyState } from './EmptyState'

export type Column<T> = {
  key: string
  header: string
  numeric?: boolean
  // The server's allowlisted column name. A column without one is not sortable.
  sortKey?: string
  // A column either sizes to its content via width, or absorbs the slack via
  // grow. Every table needs exactly one growing column; where none is declared
  // the first column without a width takes the role, which is the behaviour a
  // plain table already has.
  width?: string
  grow?: boolean
  // Table cells do not wrap, which is right for a date, a status or a figure
  // and wrong for prose. A cell holding text of unbounded length has to say so,
  // or it runs past the table rather than down it.
  wrap?: boolean
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
  // Where a row has a detail screen, the whole row opens it. The primary cell
  // still renders a real link, so the keyboard and middle-click keep working;
  // this only saves the mouse from a 14px target.
  rowHref?: (row: T) => string
  // Rendered in a trailing column the table owns, so every list puts its row
  // actions in the same place instead of inventing one.
  rowActions?: (row: T) => ReactNode
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

function growingKey<T>(columns: Column<T>[]): string | undefined {
  return (columns.find((c) => c.grow) ?? columns.find((c) => !c.width))?.key
}

function sizing<T>(column: Column<T>, growing: string | undefined): CSSProperties {
  if (column.key === growing) return { width: 'auto' }
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
  rowHref,
  rowActions,
}: DataTableProps<T>) {
  const navigate = useNavigate()
  const colCount = columns.length + (rowActions ? 1 : 0)
  const growing = growingKey(columns)

  return (
    <div>
      {caption && (
        <div className="text-caption text-subtle-foreground flex items-baseline justify-between gap-3 py-2">
          <span>{caption}</span>
          {status === 'ready' && <span>{rows.length} shown</span>}
        </div>
      )}
      <Table className="text-body">
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            {columns.map((column) => {
              const sortable = column.sortKey !== undefined && onSort !== undefined
              const active = sortable && activeKey(sort) === column.sortKey
              const direction = active
                ? sort?.startsWith('-')
                  ? 'descending'
                  : 'ascending'
                : 'none'
              return (
                <TableHead
                  key={column.key}
                  scope="col"
                  aria-sort={sortable ? direction : undefined}
                  style={sizing(column, growing)}
                  className={`bg-muted text-label text-muted-foreground h-9 px-3 ${
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
                        // Which columns sort has to be visible before the first
                        // click, not after it.
                        <ChevronsUpDown
                          className="size-3 opacity-35 group-hover:opacity-70"
                          aria-hidden
                        />
                      )}
                    </button>
                  ) : (
                    column.header
                  )}
                </TableHead>
              )
            })}
            {rowActions && (
              <TableHead className="bg-muted h-9 w-px px-3">
                <span className="sr-only">Actions</span>
              </TableHead>
            )}
          </TableRow>
        </TableHeader>

        <TableBody aria-busy={status === 'loading' || undefined}>
          {status === 'loading' &&
            Array.from({ length: SKELETON_ROWS }, (_, index) => (
              <TableRow key={index} className="hover:bg-transparent">
                {columns.map((column) => (
                  <TableCell key={column.key} className="px-3">
                    <Skeleton className="h-3 w-full max-w-44" />
                  </TableCell>
                ))}
                {rowActions && (
                  <TableCell className="px-3">
                    <Skeleton className="h-3 w-6" />
                  </TableCell>
                )}
              </TableRow>
            ))}

          {status === 'error' && (
            <TableRow className="hover:bg-transparent">
              <TableCell colSpan={colCount} className="p-0">
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
              </TableCell>
            </TableRow>
          )}

          {status === 'ready' && rows.length === 0 && (
            <TableRow className="hover:bg-transparent">
              <TableCell colSpan={colCount} className="p-0">
                <EmptyState
                  title={emptyTitle}
                  description={emptyDescription}
                  action={emptyAction}
                />
              </TableCell>
            </TableRow>
          )}

          {status === 'ready' &&
            rows.map((row) => {
              const href = rowHref?.(row)
              return (
                <TableRow
                  key={rowKey(row)}
                  className={href ? 'cursor-pointer' : undefined}
                  onClick={
                    href
                      ? (event) => {
                          // A click that already landed on a link, a button or
                          // a menu belongs to that control, not to the row.
                          if ((event.target as HTMLElement).closest('a,button,[role="menuitem"]')) {
                            return
                          }
                          navigate(href)
                        }
                      : undefined
                  }
                >
                  {columns.map((column) => (
                    <TableCell
                      key={column.key}
                      style={sizing(column, growing)}
                      className={`px-3 py-2 ${column.numeric ? 'text-right tabular-nums' : ''} ${
                        column.wrap ? 'break-words whitespace-normal' : ''
                      }`}
                    >
                      {column.cell(row)}
                    </TableCell>
                  ))}
                  {rowActions && (
                    <TableCell className="w-px px-3 text-right">{rowActions(row)}</TableCell>
                  )}
                </TableRow>
              )
            })}
        </TableBody>
      </Table>
    </div>
  )
}
