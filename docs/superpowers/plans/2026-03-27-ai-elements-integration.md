# AI-Elements Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace custom chat components with ai-elements primitives for better markdown rendering, reasoning display, tool chain-of-thought, and suggestion buttons.

**Architecture:** Rewrite the render section of `AssistantPage.tsx` to use `Conversation`, `Message`, `PromptInput`, `Reasoning`, `ChainOfThought`, and `Suggestion` components from `components/ai-elements/`. All state/WebSocket logic stays unchanged. Delete the replaced custom components.

**Tech Stack:** React, TypeScript, ai-elements (shadcn registry), Streamdown, use-stick-to-bottom, Shiki

---

### Task 1: Rewrite AssistantPage with ai-elements components

**Files:**
- Modify: `frontend/src/presentation/pages/AssistantPage.tsx`

This is the main task — replace all imports and the entire render section.

- [ ] **Step 1: Replace imports**

In `frontend/src/presentation/pages/AssistantPage.tsx`, replace the import section (lines 1-22) with:

```tsx
import { useState, useEffect, useRef, useCallback } from 'react'
import { Send, ArrowLeft, Bot, Copy, Check, Loader2, CheckCircle } from 'lucide-react'
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
```

- [ ] **Step 2: Add tool labels map, suggestions constants, and copy handler**

After the `ToolStatusItem` interface (around line 33), add:

```tsx
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
```

- [ ] **Step 3: Add copy message helper inside the component**

Inside the `AssistantPage` function, after the `navigate` declaration, add:

```tsx
const [copiedId, setCopiedId] = useState<string | null>(null)

const handleCopy = useCallback((content: string, id: string) => {
  navigator.clipboard.writeText(content)
  setCopiedId(id)
  setTimeout(() => setCopiedId(null), 2000)
}, [])
```

- [ ] **Step 4: Update handleSubmit to work with PromptInput**

Replace the existing `handleSubmit` callback (around line 174-178) with:

```tsx
const handlePromptSubmit = useCallback(
  (message: { text: string; files: unknown[] }) => {
    const trimmed = message.text.trim()
    if (!trimmed || isStreaming) return
    handleSend(trimmed)
  },
  [isStreaming, handleSend]
)
```

Also remove `const [inputValue, setInputValue] = useState('')` (line 47) — PromptInput manages its own input state.

- [ ] **Step 5: Replace the entire return block**

Replace the return statement (lines 221-289) with:

```tsx
const lastMessage = messages[messages.length - 1]
const showFollowups = !isStreaming && !isThinking && messages.length > 0 && lastMessage?.role === 'assistant' && !lastMessage?.isStreaming

return (
  <div className="h-screen flex flex-col bg-[#1B1B1E]">
    {/* Top Bar */}
    <div className="h-12 flex items-center justify-between px-4 border-b border-border bg-[#1B1B1E] shrink-0">
      <button
        onClick={() => navigate('/dashboard/home')}
        className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
      >
        <ArrowLeft className="size-4" />
        Back to Dashboard
      </button>
      <span className="text-sm font-medium text-foreground">PDT Assistant</span>
      <div className="w-[140px]" />
    </div>

    {/* Main Content */}
    <div className="flex flex-1 overflow-hidden">
      <ChatSidebar
        conversations={conversations}
        activeId={activeConversationId}
        onSelect={handleSelectConversation}
        onNew={handleNewConversation}
        onDelete={handleDeleteConversation}
      />
      <div className="flex-1 flex flex-col bg-[#242428]">
        <Conversation className="flex-1">
          <ConversationContent className="max-w-3xl mx-auto w-full">
            {messages.length === 0 ? (
              <ConversationEmptyState
                icon={<Bot className="size-8" />}
                title="PDT Assistant"
                description="Ask about your commits, Jira cards, or reports"
              >
                <div className="mt-4 flex flex-col items-center gap-3">
                  <div className="flex items-center gap-3">
                    <Bot className="size-8 text-muted-foreground" />
                  </div>
                  <div className="space-y-1 text-center">
                    <h3 className="font-medium text-sm">PDT Assistant</h3>
                    <p className="text-muted-foreground text-sm">Ask about your commits, Jira cards, or reports</p>
                  </div>
                  <Suggestions className="mt-4">
                    {STARTER_SUGGESTIONS.map((s) => (
                      <Suggestion key={s} suggestion={s} onClick={handleSend} />
                    ))}
                  </Suggestions>
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

                {isThinking && (
                  <Reasoning isStreaming={isThinking}>
                    <ReasoningTrigger />
                    <ReasoningContent>{thinkingMessage}</ReasoningContent>
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
                  <Suggestions>
                    {FOLLOWUP_SUGGESTIONS.map((s) => (
                      <Suggestion key={s} suggestion={s} onClick={handleSend} />
                    ))}
                  </Suggestions>
                )}
              </>
            )}
          </ConversationContent>
          <ConversationScrollButton />
        </Conversation>

        <div className="p-3 pt-0 bg-[#1B1B1E] border-t border-border">
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
```

- [ ] **Step 6: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build --mode development 2>&1 | tail -10`
Expected: Build succeeds.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/presentation/pages/AssistantPage.tsx
git commit -m "feat: integrate ai-elements components into assistant page"
```

---

### Task 2: Delete replaced custom components

**Files:**
- Delete: `frontend/src/presentation/components/chat/ChatMessage.tsx`
- Delete: `frontend/src/presentation/components/chat/ThinkingIndicator.tsx`
- Delete: `frontend/src/presentation/components/chat/ToolStatus.tsx`

- [ ] **Step 1: Delete the files**

```bash
rm frontend/src/presentation/components/chat/ChatMessage.tsx
rm frontend/src/presentation/components/chat/ThinkingIndicator.tsx
rm frontend/src/presentation/components/chat/ToolStatus.tsx
```

- [ ] **Step 2: Verify no other files import these**

Run: `cd /home/nst/GolandProjects/pdt && grep -r "ChatMessage\|ThinkingIndicator\|ToolStatus" frontend/src --include="*.tsx" --include="*.ts" -l`
Expected: No results (all imports were removed in Task 1).

- [ ] **Step 3: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build --mode development 2>&1 | tail -5`
Expected: Build succeeds.

- [ ] **Step 4: Commit**

```bash
git add -A frontend/src/presentation/components/chat/ChatMessage.tsx frontend/src/presentation/components/chat/ThinkingIndicator.tsx frontend/src/presentation/components/chat/ToolStatus.tsx
git commit -m "chore: remove replaced custom chat components"
```

---

### Task 3: Install missing dependencies and verify

**Files:**
- Modify: `frontend/package.json` (if needed)

- [ ] **Step 1: Check for missing dependencies**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build --mode development 2>&1 | grep -i "error\|cannot find\|not found" | head -10`

If there are missing module errors (e.g., `@streamdown/*`, `use-stick-to-bottom`), install them:

```bash
bun add <missing-package>
```

The ai-elements install should have added these, but verify.

- [ ] **Step 2: Full build verification**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build --mode development 2>&1 | tail -5`
Expected: `✓ built in X.XXs`

- [ ] **Step 3: Commit if any dependencies were added**

```bash
git add frontend/package.json frontend/bun.lock
git commit -m "chore: add missing dependencies for ai-elements"
```

---

### Task 4: Visual Verification

- [ ] **Step 1: Start the dev server**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite --host 2>&1 | head -10`

- [ ] **Step 2: Manual verification checklist**

Open the app in a browser and verify:
1. Navigate to `/assistant`
2. Empty state shows PDT Assistant icon, title, description, and 4 starter suggestion buttons
3. Click a suggestion — it sends as a message
4. Message renders with `MessageResponse` (Streamdown markdown)
5. Code blocks in responses have Shiki syntax highlighting
6. Thinking state shows collapsible `Reasoning` with shimmer "Thinking..." text
7. Reasoning auto-opens during thinking, auto-closes after
8. Tool execution shows `ChainOfThought` with steps (active spinner, completed check)
9. After assistant response completes, 3 follow-up suggestion buttons appear
10. Copy button on assistant messages works
11. Scroll-to-bottom button appears when scrolled up
12. Input area has PromptInput with golden accent submit button
13. Enter submits, Shift+Enter adds newline
14. Sidebar still works (new conversation, select, delete)

- [ ] **Step 3: Commit any final tweaks if needed**
