import { createFileRoute } from "@tanstack/react-router"

import ChatUsers from "@/features/ChatUsers"

export const Route = createFileRoute("/_auth/chat-users")({
  component: ChatUsers,
})
