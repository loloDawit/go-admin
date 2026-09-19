import { ChevronsLeft, ChevronsRight } from 'lucide-react'
import { Button } from '@/ui/shadcn/button'
import { Label } from '@/ui/shadcn/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/ui/shadcn/select'

const DEFAULT_PAGE_SIZE_OPTIONS = [10, 20, 30, 50]

export function Pagination({
  page,
  pageSize,
  total,
  onChange,
  pageSizeOptions = DEFAULT_PAGE_SIZE_OPTIONS,
  onPageSizeChange,
}: {
  page: number
  pageSize: number
  total: number
  onChange: (page: number) => void
  pageSizeOptions?: number[]
  // Rows-per-page appears only where a caller can act on it: a control the
  // server ignores is worse than none.
  onPageSizeChange?: (pageSize: number) => void
}) {
  const pages = Math.max(1, Math.ceil(total / pageSize))
  if (total === 0) return null

  const first = (page - 1) * pageSize + 1
  const last = Math.min(page * pageSize, total)

  return (
    <div className="flex flex-wrap items-center justify-between gap-4">
      <p className="text-caption text-muted-foreground tabular-nums">
        {first}–{last} of {total}
      </p>
      <div className="flex flex-wrap items-center gap-6">
        {onPageSizeChange && (
          <div className="flex items-center gap-2">
            <Label htmlFor="rows-per-page" className="text-caption text-muted-foreground">
              Rows per page
            </Label>
            <Select
              value={String(pageSize)}
              onValueChange={(next) => onPageSizeChange(Number(next))}
            >
              <SelectTrigger id="rows-per-page" size="sm" className="w-[4.5rem]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {pageSizeOptions.map((size) => (
                  <SelectItem key={size} value={String(size)}>
                    {size}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
        <p className="text-caption font-medium tabular-nums">
          Page {page} of {pages}
        </p>
        <div className="flex items-center gap-1">
          <Button
            variant="outline"
            size="icon-sm"
            aria-label="First page"
            disabled={page <= 1}
            onClick={() => onChange(1)}
          >
            <ChevronsLeft />
          </Button>
          <Button variant="secondary" size="sm" disabled={page <= 1} onClick={() => onChange(page - 1)}>
            Previous
          </Button>
          <Button variant="secondary" size="sm" disabled={page >= pages} onClick={() => onChange(page + 1)}>
            Next
          </Button>
          <Button
            variant="outline"
            size="icon-sm"
            aria-label="Last page"
            disabled={page >= pages}
            onClick={() => onChange(pages)}
          >
            <ChevronsRight />
          </Button>
        </div>
      </div>
    </div>
  )
}
