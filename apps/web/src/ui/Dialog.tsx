import { useEffect, useRef } from 'react'
import type { ReactNode } from 'react'
import {
  Dialog as Base,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/ui/shadcn/dialog'

export function Dialog({
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
  // Radix returns focus to its own DialogTrigger. These dialogs are opened from
  // a button elsewhere in the page, so the element to come back to has to be
  // remembered here or focus is dropped on the body when the dialog closes.
  const opener = useRef<HTMLElement | null>(null)

  useEffect(() => {
    if (open) opener.current = document.activeElement as HTMLElement | null
  }, [open])

  return (
    <Base open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent
        className="sm:max-w-lg"
        onCloseAutoFocus={(event) => {
          if (!opener.current?.isConnected) return
          event.preventDefault()
          opener.current.focus()
        }}
      >
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          {description && <DialogDescription>{description}</DialogDescription>}
        </DialogHeader>
        {children && <div className="grid gap-4">{children}</div>}
        {footer && <DialogFooter>{footer}</DialogFooter>}
      </DialogContent>
    </Base>
  )
}
