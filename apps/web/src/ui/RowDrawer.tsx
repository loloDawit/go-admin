import { XIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import { Button } from '@/ui/shadcn/button'
import {
  Drawer as Base,
  DrawerClose,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from '@/ui/shadcn/drawer'

// A quick look at a row, not a replacement for its detail route: the caller's
// footer always carries a way through to the full screen for real work.
export function RowDrawer({
  open,
  title,
  description,
  children,
  footer,
  onClose,
}: {
  open: boolean
  title: string
  description?: string
  children?: ReactNode
  footer?: ReactNode
  onClose: () => void
}) {
  return (
    <Base open={open} onOpenChange={(next) => !next && onClose()} direction="right">
      <DrawerContent>
        <DrawerHeader className="flex-row items-start justify-between gap-2">
          <div className="min-w-0">
            <DrawerTitle>{title}</DrawerTitle>
            {description && <DrawerDescription>{description}</DrawerDescription>}
          </div>
          <DrawerClose asChild>
            <Button variant="ghost" size="icon-sm" aria-label="Close">
              <XIcon />
            </Button>
          </DrawerClose>
        </DrawerHeader>
        {children && <div className="flex-1 overflow-auto px-4">{children}</div>}
        {footer && <DrawerFooter className="flex-row justify-end gap-2">{footer}</DrawerFooter>}
      </DrawerContent>
    </Base>
  )
}
