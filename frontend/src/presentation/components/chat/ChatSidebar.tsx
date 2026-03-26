import { Plus, MessageSquare, Trash2 } from 'lucide-react'
import type { IConversation } from '../../../domain/chat/interfaces/chat.interface'

interface ChatSidebarProps {
  conversations: IConversation[]
  activeId?: string
  onSelect: (id: string) => void
  onNew: () => void
  onDelete: (id: string) => void
}

export function ChatSidebar({ conversations, activeId, onSelect, onNew, onDelete }: ChatSidebarProps) {
  return (
    <div className="w-64 border-r border-pdt-neutral-700 flex flex-col h-full">
      <div className="p-3">
        <button
          onClick={onNew}
          className="w-full flex items-center gap-2 px-3 py-2 rounded-lg border border-pdt-neutral-600 text-sm text-pdt-neutral-300 hover:bg-pdt-neutral-800"
        >
          <Plus className="w-4 h-4" />
          New Conversation
        </button>
      </div>
      <div className="flex-1 overflow-y-auto px-2">
        {conversations.map((conv) => (
          <div
            key={conv.id}
            className={`group flex items-center gap-2 px-3 py-2 rounded-lg cursor-pointer mb-1 text-sm ${
              activeId === conv.id
                ? 'bg-pdt-accent/20 text-pdt-accent'
                : 'text-pdt-neutral-400 hover:bg-pdt-neutral-800'
            }`}
            onClick={() => onSelect(conv.id)}
          >
            <MessageSquare className="w-4 h-4 flex-shrink-0" />
            <span className="truncate flex-1">{conv.title}</span>
            <button
              onClick={(e) => {
                e.stopPropagation()
                onDelete(conv.id)
              }}
              className="opacity-0 group-hover:opacity-100 text-pdt-neutral-500 hover:text-red-400"
            >
              <Trash2 className="w-3 h-3" />
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
