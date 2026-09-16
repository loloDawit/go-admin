import type { ReactNode } from 'react'
import { Button } from './Button'
import { StateBlock } from './StateBlock'
import styles from './DataTable.module.css'

export type Column<T> = {
  key: string
  header: string
  numeric?: boolean
  // The server's allowlisted column name. A column without one is not sortable.
  sortKey?: string
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

  return (
    <div>
      {caption && (
        <div className={styles.caption}>
          <span>{caption}</span>
          {status === 'ready' && <span>{rows.length} shown</span>}
        </div>
      )}
      <div className={styles.scroll}>
        <table className={styles.table}>
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
                    className={column.numeric ? styles.numeric : undefined}
                  >
                    {sortable ? (
                      <button
                        type="button"
                        className={styles.sort}
                        onClick={() => onSort(cycle(sort, column.sortKey as string))}
                      >
                        {column.header}
                        <span
                          aria-hidden="true"
                          className={active ? styles.sortMark : styles.sortHint}
                        >
                          {active ? (direction === 'ascending' ? '↑' : '↓') : '↕'}
                        </span>
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
                    <td key={column.key}>
                      <span className={styles.skeleton} />
                    </td>
                  ))}
                </tr>
              ))}

            {status === 'error' && (
              <tr>
                <td colSpan={colCount} className={styles.stateCell}>
                  <StateBlock
                    tone="error"
                    centered
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
                </td>
              </tr>
            )}

            {status === 'ready' && rows.length === 0 && (
              <tr>
                <td colSpan={colCount} className={styles.stateCell}>
                  <StateBlock
                    centered
                    title={emptyTitle}
                    description={emptyDescription}
                    action={emptyAction}
                  />
                </td>
              </tr>
            )}

            {status === 'ready' &&
              rows.map((row) => (
                <tr key={rowKey(row)} className={styles.row}>
                  {columns.map((column) => (
                    <td key={column.key} className={column.numeric ? styles.numeric : undefined}>
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
