import { createFileRoute } from "@tanstack/react-router"

import Sources from "@/features/Sources"

export const Route = createFileRoute("/_auth/sources")({
  component: Sources,
})
