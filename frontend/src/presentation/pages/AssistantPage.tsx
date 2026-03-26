import { useState, useEffect, useRef, useCallback } from 'react'
import { ArrowLeft, Bot, Copy, Check, Loader2, CheckCircle, Menu, X } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useAppSelector } from '../../application/hooks/useAppSelector'
import {
  useListConversationsQuery,
  useDeleteConversationMutation,
} from '../../infrastructure/services/chat.service'
import { API_CONSTANTS } from '../../infrastructure/constants/api.constants'
import { ChatSidebar } from '../components/chat/ChatSidebar'
import {
  Conversation,
  ConversationContent,
  ConversationEmptyState,
  ConversationScrollButton,
} from '../../components/ai-elements/conversation'
import {
  Message,
  MessageContent,
  MessageResponse,
  MessageActions,
  MessageAction,
} from '../../components/ai-elements/message'
import {
  PromptInput,
  PromptInputTextarea,
  PromptInputFooter,
  PromptInputSubmit,
} from '../../components/ai-elements/prompt-input'
import {
  Reasoning,
  ReasoningTrigger,
  ReasoningContent,
} from '../../components/ai-elements/reasoning'
import {
  ChainOfThought,
  ChainOfThoughtHeader,
  ChainOfThoughtContent,
  ChainOfThoughtStep,
} from '../../components/ai-elements/chain-of-thought'
import { Suggestions, Suggestion } from '../../components/ai-elements/suggestion'
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

const toolLabels: Record<string, string> = {
  search_commits: 'Searching commits',
  get_commit_detail: 'Getting commit details',
  get_commit_changes: 'Fetching code changes',
  analyze_card_changes: 'Analyzing card changes',
  list_repos: 'Listing repositories',
  get_repo_stats: 'Getting repo statistics',
  get_sprints: 'Fetching sprints',
  get_cards: 'Fetching Jira cards',
  get_card_detail: 'Getting card details',
  search_cards: 'Searching cards',
  link_commit_to_card: 'Linking commit to card',
  generate_daily_report: 'Generating daily report',
  generate_monthly_report: 'Generating monthly report',
  list_reports: 'Listing reports',
  get_report: 'Getting report',
  preview_template: 'Previewing template',
  search_comments: 'Searching comments',
  get_card_comments: 'Fetching card comments',
  find_person_statements: 'Finding statements',
  get_comment_timeline: 'Building timeline',
  detect_quality_issues: 'Detecting quality issues',
  check_requirement_coverage: 'Checking requirement coverage',
}

const STARTER_SUGGESTIONS = [
  'Show my commits from today',
  'Summarize my Jira cards',
  'Generate a daily report',
  'What did I work on this week?',
]

const FOLLOWUP_SUGGESTIONS = [
  'Tell me more',
  'Show related commits',
  'Generate a report',
]

export function AssistantPage() {
  const token = useAppSelector((state) => state.auth.token)
  const { data: conversations = [], refetch } = useListConversationsQuery()
  const [deleteConversation] = useDeleteConversationMutation()

  const [activeConversationId, setActiveConversationId] = useState<string | undefined>()
  const [messages, setMessages] = useState<DisplayMessage[]>([])
  const [toolStatuses, setToolStatuses] = useState<ToolStatusItem[]>([])
  const [isThinking, setIsThinking] = useState(false)
  const [thinkingMessage, setThinkingMessage] = useState('')
  const [hasThought, setHasThought] = useState(false)
  const [isStreaming, setIsStreaming] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const streamBufferRef = useRef('')
  const navigate = useNavigate()

  const [copiedId, setCopiedId] = useState<string | null>(null)
  const [sidebarOpen, setSidebarOpen] = useState(false)

  const handleCopy = useCallback((content: string, id: string) => {
    navigator.clipboard.writeText(content)
    setCopiedId(id)
    setTimeout(() => setCopiedId(null), 2000)
  }, [])

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
          setHasThought(true)
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
          setHasThought(false)
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

  const handlePromptSubmit = useCallback(
    (message: { text: string; files: unknown[] }) => {
      const trimmed = message.text.trim()
      if (!trimmed || isStreaming) return
      handleSend(trimmed)
    },
    [isStreaming, handleSend]
  )

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

  const lastMessage = messages[messages.length - 1]
  const showFollowups = !isStreaming && !isThinking && messages.length > 0 && lastMessage?.role === 'assistant' && !lastMessage?.isStreaming

  return (
    <div className="h-screen flex flex-col bg-[#1B1B1E]">
      {/* Top Bar */}
      <div className="h-12 flex items-center justify-between px-4 border-b border-border bg-[#1B1B1E] shrink-0">
        <div className="flex items-center gap-2">
          <button
            onClick={() => setSidebarOpen(!sidebarOpen)}
            className="md:hidden flex items-center justify-center size-8 text-muted-foreground hover:text-foreground transition-colors"
          >
            {sidebarOpen ? <X className="size-4" /> : <Menu className="size-4" />}
          </button>
          <button
            onClick={() => navigate('/dashboard/home')}
            className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
          >
            <ArrowLeft className="size-4" />
            <span className="hidden sm:inline">Back to Dashboard</span>
          </button>
        </div>
        <span className="text-sm font-medium text-foreground hidden sm:inline">PDT Assistant</span>
        <div className="hidden sm:block w-[140px]" />
      </div>

      {/* Main Content */}
      <div className="flex flex-1 overflow-hidden relative">
        <ChatSidebar
          conversations={conversations}
          activeId={activeConversationId}
          isOpen={sidebarOpen}
          onSelect={(id) => { handleSelectConversation(id); setSidebarOpen(false) }}
          onNew={() => { handleNewConversation(); setSidebarOpen(false) }}
          onDelete={handleDeleteConversation}
        />
        <div className="flex-1 flex flex-col bg-[#242428] min-h-0 min-w-0">
          <Conversation className="flex-1 min-h-0 overflow-x-hidden">
            <ConversationContent className="max-w-3xl mx-auto w-full px-3 sm:px-4 overflow-x-hidden">
              {messages.length === 0 ? (
                <ConversationEmptyState>
                  <div className="flex flex-col items-center gap-3">
                    <div className="flex items-center gap-3">
                      <Bot className="size-8 text-muted-foreground" />
                    </div>
                    <div className="space-y-1 text-center">
                      <h3 className="font-medium text-sm">PDT Assistant</h3>
                      <p className="text-muted-foreground text-sm">Ask about your commits, Jira cards, or reports</p>
                    </div>
                    <div className="mt-4 flex flex-wrap justify-center gap-2">
                      {STARTER_SUGGESTIONS.map((s) => (
                        <Suggestion key={s} suggestion={s} onClick={handleSend} />
                      ))}
                    </div>
                  </div>
                </ConversationEmptyState>
              ) : (
                <>
                  {messages.map((msg) => (
                    <Message key={msg.id} from={msg.role}>
                      <MessageContent>
                        {msg.role === 'assistant' ? (
                          <MessageResponse isAnimating={msg.isStreaming}>
                            {msg.content}
                          </MessageResponse>
                        ) : (
                          msg.content
                        )}
                      </MessageContent>
                      {msg.role === 'assistant' && !msg.isStreaming && (
                        <MessageActions>
                          <MessageAction
                            tooltip="Copy"
                            onClick={() => handleCopy(msg.content, msg.id)}
                          >
                            {copiedId === msg.id ? (
                              <Check className="size-3.5" />
                            ) : (
                              <Copy className="size-3.5" />
                            )}
                          </MessageAction>
                        </MessageActions>
                      )}
                    </Message>
                  ))}

                  {(isThinking || hasThought) && (
                    <Reasoning isStreaming={isThinking}>
                      <ReasoningTrigger />
                      <ReasoningContent>{thinkingMessage || 'Thinking...'}</ReasoningContent>
                    </Reasoning>
                  )}

                  {toolStatuses.length > 0 && (
                    <ChainOfThought defaultOpen>
                      <ChainOfThoughtHeader>Tool execution</ChainOfThoughtHeader>
                      <ChainOfThoughtContent>
                        {toolStatuses.map((ts) => (
                          <ChainOfThoughtStep
                            key={ts.tool}
                            icon={ts.status === 'executing' ? Loader2 : CheckCircle}
                            label={toolLabels[ts.tool] || ts.tool}
                            status={ts.status === 'executing' ? 'active' : 'complete'}
                          />
                        ))}
                      </ChainOfThoughtContent>
                    </ChainOfThought>
                  )}

                  {showFollowups && (
                    <div className="flex flex-wrap gap-2">
                      {FOLLOWUP_SUGGESTIONS.map((s) => (
                        <Suggestion key={s} suggestion={s} onClick={handleSend} />
                      ))}
                    </div>
                  )}
                </>
              )}
            </ConversationContent>
            <ConversationScrollButton />
          </Conversation>

          <div className="shrink-0 p-3 bg-[#1B1B1E] border-t border-border">
            <PromptInput
              onSubmit={handlePromptSubmit}
            >
              <PromptInputTextarea
                placeholder="Ask about your commits, Jira cards, or reports..."
              />
              <PromptInputFooter>
                <div />
                <PromptInputSubmit
                  status={isStreaming ? 'streaming' : undefined}
                  className="bg-pdt-accent text-pdt-primary hover:bg-pdt-accent-hover"
                />
              </PromptInputFooter>
            </PromptInput>
          </div>
        </div>
      </div>
    </div>
  )
}
