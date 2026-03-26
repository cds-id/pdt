import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Bot, User } from 'lucide-react'
import {
  ChatEvent,
  ChatEventAddon,
  ChatEventAvatar,
  ChatEventBody,
  ChatEventContent,
  ChatEventTitle,
  ChatEventTime,
} from '@/components/chat/chat-event'

interface ChatMessageProps {
  role: 'user' | 'assistant'
  content: string
  isStreaming?: boolean
  timestamp?: Date
}

export function ChatMessage({ role, content, isStreaming, timestamp }: ChatMessageProps) {
  return (
    <ChatEvent className="py-2 hover:bg-pdt-neutral-800/30 rounded-lg">
      <ChatEventAddon>
        <ChatEventAvatar
          fallback={role === 'assistant' ? <Bot className="size-4" /> : <User className="size-4" />}
          className={role === 'assistant' ? 'bg-pdt-accent/20 text-pdt-accent' : 'bg-pdt-neutral-700 text-pdt-neutral-300'}
        />
      </ChatEventAddon>
      <ChatEventBody>
        <ChatEventTitle>
          <span className="font-medium text-pdt-neutral-200">
            {role === 'assistant' ? 'PDT Assistant' : 'You'}
          </span>
          {timestamp && (
            <ChatEventTime timestamp={timestamp} format="time" />
          )}
        </ChatEventTitle>
        <ChatEventContent>
          <div className="prose prose-invert prose-sm max-w-none
            prose-p:my-1 prose-ul:my-1 prose-ol:my-1 prose-li:my-0.5
            prose-headings:text-pdt-neutral-100 prose-headings:mt-3 prose-headings:mb-1
            prose-code:text-pdt-accent prose-code:bg-pdt-neutral-800 prose-code:px-1 prose-code:py-0.5 prose-code:rounded
            prose-pre:bg-pdt-neutral-800 prose-pre:border prose-pre:border-pdt-neutral-700
            prose-strong:text-pdt-neutral-100
            prose-a:text-pdt-accent">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
          </div>
          {isStreaming && (
            <span className="inline-block w-2 h-4 bg-pdt-accent animate-pulse ml-0.5" />
          )}
        </ChatEventContent>
      </ChatEventBody>
    </ChatEvent>
  )
}
