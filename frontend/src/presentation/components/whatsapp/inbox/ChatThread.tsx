import { useEffect, useRef } from 'react'
import { MessageSquare } from 'lucide-react'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { IWaChat } from '@/domain/whatsapp/interfaces/whatsapp.interface'
import { useGetChatMessagesQuery } from '@/infrastructure/services/whatsapp.service'
import { MessageBubble } from './MessageBubble'
import { Composer } from './Composer'

interface ChatThreadProps {
  numberId: number
  chat: IWaChat | null
}

function initials(name: string) {
  return name
    .split(' ')
    .map((p) => p[0])
    .filter(Boolean)
    .slice(0, 2)
    .join('')
    .toUpperCase()
}

export function ChatThread({ numberId, chat }: ChatThreadProps) {
  const bottomRef = useRef<HTMLDivElement | null>(null)

  const { data, isLoading, isFetching } = useGetChatMessagesQuery(
    chat ? { listenerId: chat.id, page: 1, pageSize: 50 } : ({} as never),
    { skip: !chat }
  )

  const messages = data?.data ?? []

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'auto' })
  }, [messages.length, chat?.id])

  if (!chat) {
    return (
      <div className="flex h-full flex-1 flex-col items-center justify-center bg-muted/30 text-center">
        <MessageSquare className="mb-4 h-16 w-16 text-muted-foreground/40" />
        <p className="text-lg font-medium text-muted-foreground">WhatsApp Inbox</p>
        <p className="text-sm text-muted-foreground">Select a conversation to start messaging.</p>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-1 flex-col bg-muted/30">
      {/* header */}
      <div className="flex items-center gap-3 border-b border-border bg-card px-4 py-3">
        <Avatar className="h-10 w-10">
          <AvatarFallback
            className={
              chat.type === 'group' ? 'bg-emerald-100 text-emerald-700' : 'bg-sky-100 text-sky-700'
            }
          >
            {initials(chat.name)}
          </AvatarFallback>
        </Avatar>
        <div className="min-w-0">
          <div className="truncate font-medium">{chat.name}</div>
          <div className="truncate text-xs text-muted-foreground">
            {chat.type === 'group' ? 'Group' : chat.jid.split('@')[0]}
          </div>
        </div>
      </div>

      {/* messages */}
      <div className="flex-1 overflow-y-auto py-3">
        {isLoading ? (
          <div className="space-y-3 px-4">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className={`h-12 ${i % 2 ? 'ml-auto w-1/2' : 'w-1/2'} rounded-lg`} />
            ))}
          </div>
        ) : messages.length === 0 ? (
          <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
            No messages yet. Say hello 👋
          </div>
        ) : (
          <>
            {data && data.total > messages.length && (
              <div className="mb-2 flex justify-center">
                <Button variant="ghost" size="sm" disabled={isFetching}>
                  {isFetching ? 'Loading…' : `${data.total - messages.length} earlier messages`}
                </Button>
              </div>
            )}
            {messages.map((m) => (
              <MessageBubble key={m.id} msg={m} />
            ))}
            <div ref={bottomRef} />
          </>
        )}
      </div>

      <Composer numberId={numberId} chatJid={chat.jid} />
    </div>
  )
}
