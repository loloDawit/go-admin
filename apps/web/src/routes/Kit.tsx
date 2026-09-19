import { useState } from 'react'
import { Inbox, MoreHorizontal, SearchX } from 'lucide-react'
import { EmptyState, Money, StatusBadge } from '../ui'
import { StatCard } from '../patterns'
import type { StatusTone } from '../ui'
import { Button } from '@/ui/shadcn/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/ui/shadcn/card'
import { Input } from '@/ui/shadcn/input'
import { Textarea } from '@/ui/shadcn/textarea'
import { Label } from '@/ui/shadcn/label'
import { Checkbox } from '@/ui/shadcn/checkbox'
import { Skeleton } from '@/ui/shadcn/skeleton'
import { Separator } from '@/ui/shadcn/separator'
import { Alert, AlertDescription, AlertTitle } from '@/ui/shadcn/alert'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/ui/shadcn/select'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/ui/shadcn/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/ui/shadcn/dropdown-menu'

const TYPE_SCALE = [
  { name: 'display / 30px', className: 'text-display font-semibold' },
  { name: 'title / 24px', className: 'text-title font-semibold' },
  { name: 'section / 18px', className: 'text-section font-semibold' },
  { name: 'body / 14px', className: 'text-body' },
  { name: 'label / 13px', className: 'text-label font-medium' },
  { name: 'caption / 12px', className: 'text-caption text-muted-foreground' },
]

const TONES: { tone: StatusTone; label: string }[] = [
  { tone: 'neutral', label: 'Archived' },
  { tone: 'info', label: 'Paid' },
  { tone: 'success', label: 'Delivered' },
  { tone: 'warning', label: 'Pending' },
  { tone: 'danger', label: 'Refunded' },
]

const SURFACES = [
  { name: 'canvas', className: 'bg-canvas' },
  { name: 'card', className: 'bg-card' },
  { name: 'muted', className: 'bg-muted' },
  { name: 'accent', className: 'bg-accent' },
]

function Panel({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-section">{title}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-wrap items-start gap-3">{children}</CardContent>
    </Card>
  )
}

export function Kit() {
  const [checked, setChecked] = useState(true)

  return (
    <div className="flex max-w-[76rem] flex-col gap-5">
      <header>
        <h1 className="text-title font-semibold tracking-[-0.011em]">Component kit</h1>
        <p className="mt-1 text-muted-foreground">
          Every primitive and every state it can be in. A screen that needs something not
          shown here is a screen that needs a new primitive, not a one-off.
        </p>
      </header>

      <Panel title="Typography">
        <div className="flex w-full flex-col gap-3">
          {TYPE_SCALE.map((step) => (
            <div key={step.name} className="flex items-baseline gap-6">
              <span className="w-40 shrink-0 text-caption text-subtle-foreground">
                {step.name}
              </span>
              <span className={step.className}>Northgate Supply</span>
            </div>
          ))}
        </div>
      </Panel>

      <Panel title="Buttons">
        <Button>Save changes</Button>
        <Button variant="secondary">Cancel</Button>
        <Button variant="ghost">Dismiss</Button>
        <Button variant="destructive">Delete product</Button>
        <Button disabled>Disabled</Button>
        <Separator orientation="vertical" className="h-8" />
        <Button size="sm">Small</Button>
        <Button size="lg">Large</Button>
      </Panel>

      <Panel title="Status">
        {TONES.map((t) => (
          <StatusBadge key={t.tone} tone={t.tone}>
            {t.label}
          </StatusBadge>
        ))}
      </Panel>

      <Panel title="Surfaces">
        {SURFACES.map((s) => (
          <div key={s.name} className="flex flex-col items-center gap-1">
            <div className={`size-16 rounded-md border border-border ${s.className}`} />
            <span className="text-caption text-muted-foreground">{s.name}</span>
          </div>
        ))}
      </Panel>

      <Panel title="Form controls">
        <div className="grid w-full max-w-lg gap-4">
          <div className="grid gap-1.5">
            <Label htmlFor="kit-sku">SKU</Label>
            <Input id="kit-sku" placeholder="MUG-CLY-300" />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="kit-bad">Title</Label>
            <Input id="kit-bad" aria-invalid defaultValue="" aria-describedby="kit-bad-error" />
            <p id="kit-bad-error" className="text-caption text-destructive">
              Give the product a title.
            </p>
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="kit-status">Status</Label>
            <Select defaultValue="all">
              <SelectTrigger id="kit-status" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All statuses</SelectItem>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="draft">Draft</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="kit-notes">Description</Label>
            <Textarea id="kit-notes" rows={3} />
          </div>
          <div className="flex items-center gap-2">
            <Checkbox
              id="kit-check"
              checked={checked}
              onCheckedChange={(v) => setChecked(v === true)}
            />
            <Label htmlFor="kit-check">Show archived products</Label>
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="kit-disabled">Disabled</Label>
            <Input id="kit-disabled" disabled defaultValue="Cannot be changed" />
          </div>
        </div>
      </Panel>

      <Panel title="Stat cards">
        <div className="grid w-full gap-4 sm:grid-cols-2 xl:grid-cols-3">
          <StatCard label="Orders to handle" value={87} context="Placed, paid or packed" />
          <StatCard label="Revenue today" value="$12,480.00" context="Net of refunds" />
          <StatCard label="Still loading" value={undefined} context="No figure yet" />
        </div>
      </Panel>

      <Panel title="Money">
        <div className="flex flex-col items-end gap-1">
          <Money minor={1248000} currency="USD" className="text-display font-semibold" />
          <Money minor={4000} currency="USD" />
          <Money minor={129} currency="USD" />
        </div>
      </Panel>

      <Panel title="Feedback">
        <div className="grid w-full gap-3">
          <Alert>
            <AlertTitle>The projection is still catching up</AlertTitle>
            <AlertDescription>Revenue may lag a paid order by a few seconds.</AlertDescription>
          </Alert>
          <Alert variant="destructive">
            <AlertTitle>This product could not be saved</AlertTitle>
            <AlertDescription>Another product already uses this SKU.</AlertDescription>
          </Alert>
        </div>
      </Panel>

      <Panel title="Loading">
        <div className="grid w-full gap-2">
          <Skeleton className="h-4 w-48" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-2/3" />
        </div>
      </Panel>

      <Panel title="Empty states">
        <div className="grid w-full gap-4 lg:grid-cols-2">
          <div className="rounded-md border border-border">
            <EmptyState
              icon={Inbox}
              title="Nothing here yet"
              description="Products you add will appear in this list."
              action={<Button>Add product</Button>}
            />
          </div>
          <div className="rounded-md border border-border">
            <EmptyState
              icon={SearchX}
              title="No products match these filters"
              description="Try a different search, or clear the filters to see everything."
              action={<Button variant="secondary">Clear filters</Button>}
            />
          </div>
        </div>
      </Panel>

      <Panel title="Overlays">
        <Dialog>
          <DialogTrigger asChild>
            <Button variant="secondary">Open dialog</Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Cancel this order?</DialogTitle>
              <DialogDescription>
                The order moves to cancelled and the stock is released. This cannot be undone.
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <DialogClose asChild>
                <Button variant="secondary">Keep the order</Button>
              </DialogClose>
              <Button variant="destructive">Cancel order</Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" aria-label="Row actions">
              <MoreHorizontal />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem>Edit</DropdownMenuItem>
            <DropdownMenuItem>Duplicate</DropdownMenuItem>
            <DropdownMenuItem variant="destructive">Delete</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </Panel>
    </div>
  )
}
