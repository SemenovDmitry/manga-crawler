import * as React from "react"

import { cn } from "@/lib/utils"

function Dialog({
  open,
  onOpenChange,
  children,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  children: React.ReactNode
}) {
  // Нативный <dialog>: Esc закрывает, фокус-ловушка и модальность из коробки.
  const ref = React.useRef<HTMLDialogElement>(null)

  React.useEffect(() => {
    const el = ref.current
    if (!el) return
    if (open && !el.open) {
      el.showModal()
    } else if (!open && el.open) {
      el.close()
    }
  }, [open])

  return (
    <dialog
      ref={ref}
      onCancel={(e) => {
        e.preventDefault()
        onOpenChange(false)
      }}
      className={cn(
        "m-auto w-full max-w-md rounded-lg border bg-popover p-0 text-popover-foreground shadow-xl backdrop:bg-black/50",
        "open:animate-in open:fade-in-0 open:zoom-in-95 closed:animate-out closed:fade-out-0 closed:zoom-out-95",
      )}
    >
      {children}
    </dialog>
  )
}

function DialogHeader({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn("space-y-1.5 p-6 pb-3", className)} {...props} />
}

function DialogTitle({ className, ...props }: React.ComponentProps<"h2">) {
  return <h2 className={cn("text-lg font-semibold leading-none", className)} {...props} />
}

function DialogDescription({ className, ...props }: React.ComponentProps<"p">) {
  return <p className={cn("text-sm text-muted-foreground", className)} {...props} />
}

function DialogFooter({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn("flex justify-end gap-2 p-6 pt-3", className)} {...props} />
}

export { Dialog, DialogDescription, DialogFooter, DialogHeader, DialogTitle }
