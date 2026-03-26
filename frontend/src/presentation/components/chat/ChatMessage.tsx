import { User, Bot } from 'lucide-react'

interface ChatMessageProps {
  role: 'user' | 'assistant'
  content: string
  isStreaming?: boolean
}

export function ChatMessage({ role, content, isStreaming }: ChatMessageProps) {
  return (
    <div className={`flex gap-3 ${role === 'user' ? 'justify-end' : 'justify-start'}`}>
      {role === 'assistant' && (
        <div className="flex-shrink-0 w-8 h-8 rounded-full bg-pdt-accent/20 flex items-center justify-center">
          <Bot className="w-4 h-4 text-pdt-accent" />
        </div>
      )}
      <div
        className={`max-w-[80%] rounded-lg px-4 py-2 ${
          role === 'user'
            ? 'bg-pdt-accent text-pdt-neutral-900'
            : 'bg-pdt-neutral-800 text-pdt-neutral-100'
        }`}
      >
        <div className="whitespace-pre-wrap text-sm">{content}</div>
        {isStreaming && (
          <span className="inline-block w-2 h-4 bg-current animate-pulse ml-0.5" />
        )}
      </div>
      {role === 'user' && (
        <div className="flex-shrink-0 w-8 h-8 rounded-full bg-pdt-neutral-700 flex items-center justify-center">
          <User className="w-4 h-4 text-pdt-neutral-300" />
        </div>
      )}
    </div>
  )
}
