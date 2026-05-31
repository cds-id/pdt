import { useEffect, useState } from 'react'
import {
  useListNumbersQuery,
  useListChatsQuery,
  useMarkChatReadMutation
} from '@/infrastructure/services/whatsapp.service'
import { useWhatsAppSocket } from '@/presentation/hooks/useWhatsAppSocket'
import { IWaChat } from '@/domain/whatsapp/interfaces/whatsapp.interface'
import { ChatList } from '@/presentation/components/whatsapp/inbox/ChatList'
import { ChatThread } from '@/presentation/components/whatsapp/inbox/ChatThread'
import { EmptyState } from '@/presentation/components/common'
import { MessageSquare } from 'lucide-react'

export function WhatsAppInboxPage() {
  const { data: numbers = [], isLoading: loadingNumbers } = useListNumbersQuery()
  const [numberId, setNumberId] = useState<number | null>(null)
  const [activeChat, setActiveChat] = useState<IWaChat | null>(null)
  const [search, setSearch] = useState('')

  // Auto-pick the first connected number.
  useEffect(() => {
    if (numberId == null && numbers.length > 0) {
      const connected = numbers.find((n) => n.status === 'connected') ?? numbers[0]
      setNumberId(connected.id)
    }
  }, [numbers, numberId])

  const { data: chats = [], isLoading: loadingChats } = useListChatsQuery(numberId as number, {
    skip: numberId == null
  })
  const [markRead] = useMarkChatReadMutation()

  // Realtime updates for the whole inbox.
  useWhatsAppSocket(numberId)

  const onSelect = (chat: IWaChat) => {
    setActiveChat(chat)
    if (chat.unread_count && numberId != null) {
      markRead({ listenerId: chat.id, numberId })
    }
  }

  if (!loadingNumbers && numbers.length === 0) {
    return (
      <div className="p-6">
        <EmptyState
          icon={MessageSquare}
          title="No WhatsApp number connected"
          description="Pair a WhatsApp number in Settings to use the inbox."
        />
      </div>
    )
  }

  return (
    <div className="flex h-[calc(100vh-4rem)] flex-col">
      {numbers.length > 1 && (
        <div className="flex items-center gap-2 border-b border-border bg-card px-4 py-2">
          <span className="text-sm text-muted-foreground">Number:</span>
          <select
            value={numberId ?? ''}
            onChange={(e) => {
              setNumberId(Number(e.target.value))
              setActiveChat(null)
            }}
            className="rounded-md border border-border bg-background px-2 py-1 text-sm"
          >
            {numbers.map((n) => (
              <option key={n.id} value={n.id}>
                {n.display_name || n.phone_number} ({n.status})
              </option>
            ))}
          </select>
        </div>
      )}

      <div className="flex min-h-0 flex-1">
        <div className="w-full max-w-sm shrink-0">
          <ChatList
            chats={chats}
            isLoading={loadingChats}
            activeId={activeChat?.id ?? null}
            search={search}
            onSearch={setSearch}
            onSelect={onSelect}
          />
        </div>
        <ChatThread numberId={numberId as number} chat={activeChat} />
      </div>
    </div>
  )
}
