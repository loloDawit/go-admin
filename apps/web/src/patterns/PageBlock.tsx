import type { ReactNode } from 'react'

// The frame fills the viewport; the page block sizes to what it holds, with a
// floor so a sparse screen still reads as a page rather than a fragment. A cap
// on the frame is what made the application look shrunk on a wide monitor.
export function PageBlock({ children }: { children: ReactNode }) {
  return (
    <div className="flex w-fit min-w-[min(100%,52rem)] max-w-full flex-col gap-5">{children}</div>
  )
}
