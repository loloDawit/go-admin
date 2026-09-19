import { Loader2 } from 'lucide-react'
import type { ButtonHTMLAttributes } from 'react'
import { Button as Base } from '@/ui/shadcn/button'

type Variant = 'primary' | 'secondary' | 'danger' | 'dangerSolid' | 'ghost'
type Size = 'sm' | 'md' | 'lg'

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: Variant
  size?: Size
  loading?: boolean
  block?: boolean
}

// This project's vocabulary, mapped onto the component library's. Screens say
// primary and danger; shadcn says default and destructive.
// danger is quiet where it opens a confirmation; dangerSolid is the filled
// button inside that confirmation. A screen has one filled button, and on a
// screen that can both advance and cancel an order, it is the advance.
const VARIANTS = {
  primary: 'default',
  secondary: 'secondary',
  danger: 'destructiveSoft',
  dangerSolid: 'destructive',
  ghost: 'ghost',
} as const

const SIZES = { sm: 'sm', md: 'default', lg: 'lg' } as const

export function Button({
  variant = 'secondary',
  size = 'md',
  loading = false,
  block = false,
  disabled,
  children,
  className,
  type = 'button',
  ...rest
}: ButtonProps) {
  return (
    <Base
      type={type}
      variant={VARIANTS[variant]}
      size={SIZES[size]}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={`${block ? 'w-full' : ''} ${className ?? ''}`.trim() || undefined}
      {...rest}
    >
      {loading && <Loader2 className="animate-spin" aria-hidden />}
      {children}
    </Base>
  )
}
