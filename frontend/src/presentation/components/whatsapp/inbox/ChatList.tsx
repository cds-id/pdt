import { useMemo } from 'react'
import { Search } from 'lucide-react'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { IWaChat } from '@/domain/whatsapp/interfaces/whatsapp.interface'
import { cn } from '@/lib/utils'

interface ChatListProps {
  chats: IWaChat[]
  isLoading: boolean
  activeId: number | null
  search: string
  onSearch: (v: string) => void
  onSelect: (chat: IWaChat) => void
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

function formatTime(iso?: string) {
  if (!iso) return ''
  const d = new Date(iso)
  const now = new Date()
  const sameDay = d.toDateString() === now.toDateString()
  if (sameDay) return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  return d.toLocaleDateString([], { day: '2-digit', month: 'short' })
}

export function ChatList({
  chats,
  isLoading,
  activeId,
  search,
  onSearch,
  onSelect
}: ChatListProps) {
  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return chats
    return chats.filter(
      (c) =>
        c.name.toLowerCase().includes(q) ||
        (c.last_message_preview ?? '').toLowerCase().includes(q)
    )
  }, [chats, search])

  return (
    <div className="flex h-full w-full flex-col border-r border-border bg-card">
      <div className="border-b border-border p-3">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={search}
            onChange={(e) => onSearch(e.target.value)}
            placeholder="Search or start new chat"
            className="pl-9"
          />
        </div>
      </div>

      <ScrollArea className="flex-1">
        {isLoading ? (
          <div className="space-y-3 p-3">
            {Array.from({ length: 8 }).map((_, i) => (
              <div key={i} className="flex items-center gap-3">
                <Skeleton className="h-12 w-12 rounded-full" />
                <div className="flex-1 space-y-2">
                  <Skeleton className="h-3 w-1/2" />
                  <Skeleton className="h-3 w-3/4" />
                </div>
              </div>
            ))}
          </div>
        ) : filtered.length === 0 ? (
          <div className="p-6 text-center text-sm text-muted-foreground">
            No conversations yet.
          </div>
        ) : (
          filtered.map((chat) => (
            <button
              key={chat.id}
              onClick={() => onSelect(chat)}
              className={cn(
                'flex w-full items-center gap-3 border-b border-border/50 px-3 py-3 text-left transition-colors hover:bg-accent',
                activeId === chat.id && 'bg-accent'
              )}
            >
              <Avatar className="h-12 w-12 shrink-0">
                <AvatarFallback
                  className={cn(
                    chat.type === 'group'
                      ? 'bg-emerald-100 text-emerald-700'
                      : 'bg-sky-100 text-sky-700'
                  )}
                >
                  {initials(chat.name)}
                </AvatarFallback>
              </Avatar>
              <div className="min-w-0 flex-1">
                <div className="flex items-center justify-between gap-2">
                  <span className="truncate font-medium">{chat.name}</span>
                  <span className="shrink-0 text-xs text-muted-foreground">
                    {formatTime(chat.last_message_at)}
                  </span>
                </div>
                <div className="flex items-center justify-between gap-2">
                  <span className="truncate text-sm text-muted-foreground">
                    {chat.last_message_preview || 'No messages'}
                  </span>
                  {!!chat.unread_count && chat.unread_count > 0 && (
                    <Badge className="h-5 min-w-5 shrink-0 justify-center rounded-full bg-emerald-500 px-1.5 text-xs text-white hover:bg-emerald-500">
                      {chat.unread_count}
                    </Badge>
                  )}
                </div>
              </div>
            </button>
          ))
        )}
      </ScrollArea>
    </div>
  )
}
