import { useState, useEffect, useRef, useCallback } from 'react'
import { Send } from 'lucide-react'
import { useAppSelector } from '../../application/hooks/useAppSelector'
import {
  useListConversationsQuery,
  useDeleteConversationMutation,
} from '../../infrastructure/services/chat.service'
import { API_CONSTANTS } from '../../infrastructure/constants/api.constants'
import { PageHeader } from '../components/common/PageHeader'
import { ChatSidebar } from '../components/chat/ChatSidebar'
import { ChatMessage } from '../components/chat/ChatMessage'
import { ThinkingIndicator } from '../components/chat/ThinkingIndicator'
import { ToolStatus } from '../components/chat/ToolStatus'
import { Chat } from '../../components/chat/chat'
import { ChatMessages } from '../../components/chat/chat-messages'
import {
  ChatToolbar,
  ChatToolbarTextarea,
  ChatToolbarAddon,
  ChatToolbarButton,
} from '../../components/chat/chat-toolbar'
import type { IWSResponse } from '../../domain/chat/interfaces/chat.interface'

interface DisplayMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  isStreaming?: boolean
}

interface ToolStatusItem {
  tool: string
  status: 'executing' | 'completed'
}

export function AssistantPage() {
  const token = useAppSelector((state) => state.auth.token)
  const { data: conversations = [], refetch } = useListConversationsQuery()
  const [deleteConversation] = useDeleteConversationMutation()

  const [activeConversationId, setActiveConversationId] = useState<string | undefined>()
  const [messages, setMessages] = useState<DisplayMessage[]>([])
  const [toolStatuses, setToolStatuses] = useState<ToolStatusItem[]>([])
  const [isThinking, setIsThinking] = useState(false)
  const [thinkingMessage, setThinkingMessage] = useState('')
  const [isStreaming, setIsStreaming] = useState(false)
  const [inputValue, setInputValue] = useState('')
  const wsRef = useRef<WebSocket | null>(null)
  const streamBufferRef = useRef('')

  // Connect WebSocket
  useEffect(() => {
    if (!token) return

    const wsUrl = `${API_CONSTANTS.BASE_URL.replace('http', 'ws')}${API_CONSTANTS.API_PREFIX}${API_CONSTANTS.CHAT.WS}?token=${token}`
    const ws = new WebSocket(wsUrl)

    ws.onopen = () => {
      console.log('[ws] connected')
    }

    ws.onmessage = (event) => {
      const data: IWSResponse = JSON.parse(event.data)

      switch (data.type) {
        case 'stream':
          setIsThinking(false)
          if (data.conversation_id && !activeConversationId) {
            setActiveConversationId(data.conversation_id)
          }
          streamBufferRef.current += data.content || ''
          setMessages((prev) => {
            const last = prev[prev.length - 1]
            if (last && last.role === 'assistant' && last.isStreaming) {
              return [
                ...prev.slice(0, -1),
                { ...last, content: streamBufferRef.current },
              ]
            }
            return [
              ...prev,
              {
                id: crypto.randomUUID(),
                role: 'assistant',
                content: streamBufferRef.current,
                isStreaming: true,
              },
            ]
          })
          break

        case 'thinking':
          setIsThinking(true)
          setThinkingMessage(data.content || 'Thinking...')
          break

        case 'tool_status':
          setIsThinking(false)
          if (data.tool && data.status) {
            setToolStatuses((prev) => {
              const existing = prev.findIndex((t) => t.tool === data.tool)
              if (existing >= 0) {
                const updated = [...prev]
                updated[existing] = { tool: data.tool!, status: data.status! }
                return updated
              }
              return [...prev, { tool: data.tool!, status: data.status! }]
            })
          }
          break

        case 'done':
          setMessages((prev) => {
            const last = prev[prev.length - 1]
            if (last && last.isStreaming) {
              return [...prev.slice(0, -1), { ...last, isStreaming: false }]
            }
            return prev
          })
          setToolStatuses([])
          setIsThinking(false)
          setIsStreaming(false)
          streamBufferRef.current = ''
          refetch()
          break

        case 'error':
          setIsStreaming(false)
          setToolStatuses([])
          streamBufferRef.current = ''
          setMessages((prev) => [
            ...prev,
            {
              id: crypto.randomUUID(),
              role: 'assistant',
              content: `Error: ${data.content}`,
            },
          ])
          break
      }
    }

    ws.onclose = () => {
      console.log('[ws] disconnected')
    }

    wsRef.current = ws

    return () => {
      ws.close()
    }
  }, [token])

  const handleSend = useCallback((content: string) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return

    setMessages((prev) => [
      ...prev,
      { id: crypto.randomUUID(), role: 'user', content },
    ])
    setIsStreaming(true)
    streamBufferRef.current = ''

    wsRef.current.send(
      JSON.stringify({
        type: 'message',
        content,
        conversation_id: activeConversationId,
      })
    )
  }, [activeConversationId])

  const handleSubmit = useCallback(() => {
    const trimmed = inputValue.trim()
    if (!trimmed || isStreaming) return
    handleSend(trimmed)
    setInputValue('')
  }, [inputValue, isStreaming, handleSend])

  const handleNewConversation = () => {
    setActiveConversationId(undefined)
    setMessages([])
    setToolStatuses([])
  }

  const handleSelectConversation = async (id: string) => {
    setActiveConversationId(id)
    setToolStatuses([])

    // Load messages from API
    try {
      const resp = await fetch(
        `${API_CONSTANTS.BASE_URL}${API_CONSTANTS.API_PREFIX}/conversations/${id}`,
        { headers: { Authorization: `Bearer ${token}` } }
      )
      const data = await resp.json()
      if (data.messages) {
        setMessages(
          data.messages
            .filter((m: { role: string }) => m.role === 'user' || m.role === 'assistant')
            .map((m: { id: string; role: 'user' | 'assistant'; content: string }) => ({
              id: m.id,
              role: m.role,
              content: m.content,
            }))
        )
      }
    } catch (err) {
      console.error('Failed to load conversation:', err)
    }
  }

  const handleDeleteConversation = async (id: string) => {
    await deleteConversation(id).unwrap()
    if (activeConversationId === id) {
      handleNewConversation()
    }
  }

  return (
    <div className="min-w-0 flex flex-col h-[calc(100vh-4rem)]">
      <PageHeader title="AI Assistant" description="Chat with your development data" />
      <div className="flex flex-1 overflow-hidden border border-border rounded-lg mx-4 mb-4">
        <ChatSidebar
          conversations={conversations}
          activeId={activeConversationId}
          onSelect={handleSelectConversation}
          onNew={handleNewConversation}
          onDelete={handleDeleteConversation}
        />
        <Chat className="flex-1">
          <ChatMessages>
            <div className="flex flex-col">
              {messages.length === 0 && (
                <div className="flex items-center justify-center h-full min-h-[200px] text-muted-foreground text-sm">
                  Start a conversation...
                </div>
              )}
              {messages.map((msg) => (
                <ChatMessage
                  key={msg.id}
                  role={msg.role}
                  content={msg.content}
                  isStreaming={msg.isStreaming}
                />
              ))}
              {isThinking && <ThinkingIndicator message={thinkingMessage} />}
              {toolStatuses.map((ts) => (
                <ToolStatus key={ts.tool} toolName={ts.tool} status={ts.status} />
              ))}
            </div>
          </ChatMessages>
          <ChatToolbar>
            <ChatToolbarTextarea
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onSubmit={handleSubmit}
              placeholder="Ask about your commits, Jira cards, or reports..."
              disabled={isStreaming}
            />
            <ChatToolbarAddon align="inline-end">
              <ChatToolbarButton onClick={handleSubmit} disabled={isStreaming || !inputValue.trim()}>
                <Send />
              </ChatToolbarButton>
            </ChatToolbarAddon>
          </ChatToolbar>
        </Chat>
      </div>
    </div>
  )
}
