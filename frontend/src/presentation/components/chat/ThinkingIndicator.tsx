import { Brain } from 'lucide-react'
import { ChatEvent, ChatEventAddon, ChatEventBody, ChatEventContent } from '../../../components/chat/chat-event'

interface ThinkingIndicatorProps {
  message: string
}

export function ThinkingIndicator({ message }: ThinkingIndicatorProps) {
  return (
    <ChatEvent className="py-2">
      <ChatEventAddon>
        <div className="size-8 rounded-full bg-pdt-accent/10 flex items-center justify-center">
          <Brain className="size-4 text-pdt-accent animate-pulse" />
        </div>
      </ChatEventAddon>
      <ChatEventBody>
        <ChatEventContent>
          <div className="flex items-center gap-3 text-sm text-muted-foreground">
            <div className="flex gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-pdt-accent animate-bounce" style={{ animationDelay: '0ms' }} />
              <span className="w-1.5 h-1.5 rounded-full bg-pdt-accent animate-bounce" style={{ animationDelay: '150ms' }} />
              <span className="w-1.5 h-1.5 rounded-full bg-pdt-accent animate-bounce" style={{ animationDelay: '300ms' }} />
            </div>
            <span>{message}</span>
          </div>
        </ChatEventContent>
      </ChatEventBody>
    </ChatEvent>
  )
}
