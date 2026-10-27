import { useSyncExternalStore } from "react"
import {
  dismissToast,
  getToasts,
  subscribeToasts,
  type Toast,
  type ToastKind,
} from "@/lib/toastStore"
import { cn } from "@/lib/utils"

const kindClasses: Record<ToastKind, string> = {
  error: "border-destructive/40 bg-destructive/15 text-destructive-foreground",
  success: "border-primary/40 bg-primary/15 text-foreground",
  info: "border-border bg-popover text-popover-foreground",
}

// Фиксированный контейнер тостов (правый нижний угол); монтируется в __root.
export function Toaster() {
  const toasts = useSyncExternalStore(subscribeToasts, getToasts)

  return (
    <div className="pointer-events-none fixed right-4 bottom-4 z-50 flex w-80 flex-col gap-2">
      {toasts.map((t: Toast) => (
        <button
          key={t.id}
          type="button"
          onClick={() => dismissToast(t.id)}
          className={cn(
            "pointer-events-auto rounded-md border px-4 py-3 text-left text-sm shadow-lg",
            kindClasses[t.kind],
          )}
        >
          {t.message}
        </button>
      ))}
    </div>
  )
}
