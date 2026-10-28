import { createFileRoute } from "@tanstack/react-router"

import { UserSubscription } from "@/features/ChatUsers"

export const Route = createFileRoute("/_auth/chat-users/$chatId")({
  component: UserSubscriptionPage,
})

function UserSubscriptionPage() {
  const { chatId } = Route.useParams()
  return <UserSubscription chatId={Number(chatId)} />
}
