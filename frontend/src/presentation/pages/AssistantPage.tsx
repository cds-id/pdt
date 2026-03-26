import { useState, useEffect, useRef, useCallback } from 'react'
import { useAppSelector } from '../../application/hooks/useAppSelector'
import {
  useListConversationsQuery,
  useDeleteConversationMutation,
} from '../../infrastructure/services/chat.service'
import { API_CONSTANTS } from '../../infrastructure/constants/api.constants'
import { PageHeader } from '../components/common/PageHeader'
import { ChatSidebar } from '../components/chat/ChatSidebar'
import { ChatMessage } from '../components/chat/ChatMessage'
import { ChatInput } from '../components/chat/ChatInput'
import { ToolStatus } from '../components/chat/ToolStatus'
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
  const [isStreaming, setIsStreaming] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const streamBufferRef = useRef('')

  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [])

  useEffect(() => {
    scrollToBottom()
  }, [messages, toolStatuses, scrollToBottom])

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

        case 'tool_status':
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

  const handleSend = (content: string) => {
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
  }

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
      <div className="flex flex-1 overflow-hidden border border-pdt-neutral-700 rounded-lg mx-4 mb-4">
        <ChatSidebar
          conversations={conversations}
          activeId={activeConversationId}
          onSelect={handleSelectConversation}
          onNew={handleNewConversation}
          onDelete={handleDeleteConversation}
        />
        <div className="flex-1 flex flex-col">
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            {messages.length === 0 && (
              <div className="flex items-center justify-center h-full text-pdt-neutral-500 text-sm">
                Start a conversation — ask about your commits, Jira cards, or reports.
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
            {toolStatuses.map((ts) => (
              <ToolStatus key={ts.tool} toolName={ts.tool} status={ts.status} />
            ))}
            <div ref={messagesEndRef} />
          </div>
          <ChatInput onSend={handleSend} disabled={isStreaming} />
        </div>
      </div>
    </div>
  )
}
