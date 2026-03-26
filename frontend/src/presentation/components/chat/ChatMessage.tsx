import ReactMarkdown from 'react-markdown'
import { Bot, User } from 'lucide-react'
import {
  ChatEvent,
  ChatEventAddon,
  ChatEventAvatar,
  ChatEventBody,
  ChatEventContent,
  ChatEventTitle,
  ChatEventTime,
} from '../../../components/chat/chat-event'

interface ChatMessageProps {
  role: 'user' | 'assistant'
  content: string
  isStreaming?: boolean
  timestamp?: Date
}

export function ChatMessage({ role, content, isStreaming, timestamp }: ChatMessageProps) {
  return (
    <ChatEvent className="py-2 hover:bg-muted/30 rounded-lg">
      <ChatEventAddon>
        <ChatEventAvatar
          fallback={role === 'assistant' ? <Bot className="size-4" /> : <User className="size-4" />}
          className={role === 'assistant' ? 'bg-pdt-accent/20 text-pdt-accent' : 'bg-pdt-primary-light text-pdt-neutral'}
        />
      </ChatEventAddon>
      <ChatEventBody>
        <ChatEventTitle>
          <span className="font-medium text-foreground">
            {role === 'assistant' ? 'PDT Assistant' : 'You'}
          </span>
          {timestamp && (
            <ChatEventTime timestamp={timestamp} format="time" />
          )}
        </ChatEventTitle>
        <ChatEventContent>
          <div className="prose prose-invert prose-sm max-w-none text-foreground
            prose-p:my-1 prose-p:text-foreground prose-ul:my-1 prose-ol:my-1 prose-li:my-0.5 prose-li:text-foreground
            prose-headings:text-foreground prose-headings:mt-3 prose-headings:mb-1
            prose-code:text-pdt-accent prose-code:bg-pdt-primary-light prose-code:px-1 prose-code:py-0.5 prose-code:rounded
            prose-pre:bg-pdt-primary-light prose-pre:border prose-pre:border-border
            prose-strong:text-foreground
            prose-a:text-pdt-accent
            prose-td:text-foreground prose-th:text-foreground">
            <ReactMarkdown>{content}</ReactMarkdown>
          </div>
          {isStreaming && (
            <span className="inline-block w-2 h-4 bg-pdt-accent animate-pulse ml-0.5" />
          )}
        </ChatEventContent>
      </ChatEventBody>
    </ChatEvent>
  )
}
