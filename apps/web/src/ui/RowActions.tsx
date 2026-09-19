import { MoreHorizontal } from 'lucide-react'
import type { ReactNode } from 'react'
import { Button } from '@/ui/shadcn/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/ui/shadcn/dropdown-menu'

// Every list puts its row actions behind the same trigger in the same place.
// Inline buttons on one screen and nothing on the next is what makes a table
// feel assembled per page rather than defined once.
export function RowActions({ label, children }: { label: string; children: ReactNode }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon-sm" aria-label={label}>
          <MoreHorizontal />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">{children}</DropdownMenuContent>
    </DropdownMenu>
  )
}
