import { createFileRoute, redirect } from "@tanstack/react-router"

// / — редирект на /manga (дашборд спекой не предусмотрен).
export const Route = createFileRoute("/")({
  beforeLoad: () => {
    throw redirect({ to: "/manga" })
  },
})
