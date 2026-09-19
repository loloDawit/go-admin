import { useState } from 'react'
import { Settings2 } from 'lucide-react'
import { Button } from '@/ui/shadcn/button'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/ui/shadcn/dropdown-menu'

export type HideableColumn = { key: string; header: string }

const STORAGE_PREFIX = 'columns:'

function load(tableId: string): Set<string> {
  try {
    const raw = localStorage.getItem(STORAGE_PREFIX + tableId)
    return new Set(raw ? (JSON.parse(raw) as string[]) : [])
  } catch {
    return new Set()
  }
}

function save(tableId: string, hidden: Set<string>): void {
  try {
    localStorage.setItem(STORAGE_PREFIX + tableId, JSON.stringify([...hidden]))
  } catch {
    // Storage can be unavailable — private browsing, quota — so visibility
    // just does not survive a reload rather than breaking the table.
  }
}

// One table per screen, so the key is the screen's own id: a column hidden on
// Products must not also hide it on Orders.
export function useColumnVisibility(tableId: string | undefined, columns: HideableColumn[]) {
  const [hidden, setHidden] = useState<Set<string>>(() => (tableId ? load(tableId) : new Set()))

  function toggle(key: string) {
    if (!tableId) return
    setHidden((previous) => {
      const visible = columns.length - previous.size
      if (!previous.has(key) && visible <= 1) return previous
      const next = new Set(previous)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      save(tableId, next)
      return next
    })
  }

  return { hidden, toggle }
}

export function ColumnVisibilityMenu({
  columns,
  hidden,
  onToggle,
}: {
  columns: HideableColumn[]
  hidden: Set<string>
  onToggle: (key: string) => void
}) {
  const visibleCount = columns.length - hidden.size
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm">
          <Settings2 />
          Columns
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuLabel>Toggle columns</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {columns.map((column) => {
          const visible = !hidden.has(column.key)
          return (
            <DropdownMenuCheckboxItem
              key={column.key}
              checked={visible}
              disabled={visible && visibleCount <= 1}
              onSelect={(event) => event.preventDefault()}
              onCheckedChange={() => onToggle(column.key)}
            >
              {column.header}
            </DropdownMenuCheckboxItem>
          )
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
