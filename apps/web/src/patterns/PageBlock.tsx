import type { ReactNode } from 'react'

// Content fills the frame. The cap only stops a line of text running the whole
// width of an ultrawide display; at ordinary workstation widths it never binds.
export function PageBlock({ children }: { children: ReactNode }) {
  return <div className="flex w-full max-w-[120rem] flex-col gap-5">{children}</div>
}
