import { LoaderCircle } from "lucide-react"

import { cn } from "@/lib/utils"

export function Spinner({ className }: { className?: string }) {
  return <LoaderCircle className={cn("size-4 animate-spin text-muted-foreground", className)} />
}

// Полноэкранная заглушка загрузки страницы.
export function PageSpinner() {
  return (
    <div className="flex min-h-svh items-center justify-center">
      <Spinner className="size-8" />
    </div>
  )
}
