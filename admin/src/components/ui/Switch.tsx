import type * as React from "react"

import { cn } from "@/lib/utils"

// Switch на нативном чекбоксе: доступность бесплатно, стили — Tailwind.
function Switch({ className, ...props }: React.ComponentProps<"input">) {
  return (
    <label className="inline-flex cursor-pointer items-center">
      <input type="checkbox" className="peer sr-only" {...props} />
      <span
        className={cn(
          "h-5 w-9 rounded-full border border-input bg-muted transition-colors",
          "peer-checked:bg-primary peer-focus-visible:ring-2 peer-focus-visible:ring-ring",
          "relative after:absolute after:top-0.5 after:left-0.5 after:h-3.5 after:w-3.5 after:rounded-full after:bg-foreground after:transition-transform peer-checked:after:translate-x-4 peer-checked:after:bg-primary-foreground peer-disabled:cursor-not-allowed peer-disabled:opacity-50",
          className,
        )}
      />
    </label>
  )
}

export { Switch }
