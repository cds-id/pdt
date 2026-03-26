# Assistant Page Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the AI Assistant page into a full-screen industrial dark chat experience with global dark mode fix.

**Architecture:** Move the assistant page out of the dashboard layout into its own top-level route. Add `class="dark"` to `<html>` to fix CSS variable theming globally. Restructure the assistant page layout with a slim top bar, always-visible conversation sidebar, dark message cards, and docked input bar.

**Tech Stack:** React, TypeScript, Tailwind CSS, shadcn/ui, React Router DOM

---

### Task 1: Fix Global Dark Mode

**Files:**
- Modify: `frontend/index.html:2`

- [ ] **Step 1: Add dark class to html element**

In `frontend/index.html`, change line 2:

```html
<!-- Before -->
<html lang="en">

<!-- After -->
<html lang="en" class="dark">
```

- [ ] **Step 2: Verify the fix**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build --mode development 2>&1 | tail -5`
Expected: Build succeeds without errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/index.html
git commit -m "fix: enable dark mode globally by adding class to html element"
```

---

### Task 2: Move Assistant Route Out of Dashboard Layout

**Files:**
- Modify: `frontend/src/presentation/routes/index.tsx:41`
- Modify: `frontend/src/config/navigation.ts:43`

- [ ] **Step 1: Move assistant route to top-level**

In `frontend/src/presentation/routes/index.tsx`, remove the assistant route from the DashboardLayout children and add it as a top-level route:

```tsx
// Remove this line from DashboardLayout children:
{ path: 'dashboard/assistant', element: <AssistantPage /> }

// Add this as a new top-level route (after the DashboardLayout block, before the catch-all):
{
  path: 'assistant',
  element: <AssistantPage />
},
```

- [ ] **Step 2: Update navigation link**

In `frontend/src/config/navigation.ts`, change line 43:

```typescript
// Before
items: [{ title: 'AI Assistant', href: '/dashboard/assistant', icon: Bot }]

// After
items: [{ title: 'AI Assistant', href: '/assistant', icon: Bot }]
```

- [ ] **Step 3: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build --mode development 2>&1 | tail -5`
Expected: Build succeeds.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/presentation/routes/index.tsx frontend/src/config/navigation.ts
git commit -m "refactor: move assistant page to top-level route outside dashboard layout"
```

---

### Task 3: Redesign AssistantPage Layout

**Files:**
- Modify: `frontend/src/presentation/pages/AssistantPage.tsx`

- [ ] **Step 1: Rewrite the AssistantPage component render**

Replace the entire return block in `AssistantPage.tsx` (lines 220-270) with the new full-screen layout. Also update the imports to add `ArrowLeft` from lucide-react and `useNavigate` from react-router-dom, and remove the `PageHeader` import:

Add to imports at the top of the file:

```tsx
import { Send, ArrowLeft } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
```

Remove the `PageHeader` import:

```tsx
// Remove this line:
import { PageHeader } from '../components/common/PageHeader'
```

Add inside the component function (after the existing state declarations, before the useEffect):

```tsx
const navigate = useNavigate()
```

Replace the return statement (lines 220-270) with:

```tsx
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
      <Chat className="flex-1 bg-[#242428]">
        <ChatMessages>
          <div className="flex flex-col max-w-3xl mx-auto w-full px-4">
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
            <ChatToolbarButton
              onClick={handleSubmit}
              disabled={isStreaming || !inputValue.trim()}
              className="bg-pdt-accent text-pdt-primary hover:bg-pdt-accent-hover"
            >
              <Send />
            </ChatToolbarButton>
          </ChatToolbarAddon>
        </ChatToolbar>
      </Chat>
    </div>
  </div>
)
```

- [ ] **Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build --mode development 2>&1 | tail -5`
Expected: Build succeeds.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/presentation/pages/AssistantPage.tsx
git commit -m "feat: redesign assistant page with full-screen industrial layout"
```

---

### Task 4: Restyle ChatSidebar

**Files:**
- Modify: `frontend/src/presentation/components/chat/ChatSidebar.tsx`

- [ ] **Step 1: Rewrite the ChatSidebar component**

Replace the entire content of `ChatSidebar.tsx` with:

```tsx
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
    <div className="w-64 border-r border-border flex flex-col h-full bg-[#1B1B1E] shrink-0">
      <div className="p-3">
        <button
          onClick={onNew}
          className="w-full flex items-center gap-2 px-3 py-2 rounded-lg border border-border text-sm text-muted-foreground hover:bg-[#2d2d32] hover:text-foreground transition-colors"
        >
          <Plus className="w-4 h-4" />
          New Conversation
        </button>
      </div>
      <div className="flex-1 overflow-y-auto px-2 scrollbar-none">
        {conversations.map((conv) => (
          <div
            key={conv.id}
            className={`group flex items-center gap-2 px-3 py-2 rounded-lg cursor-pointer mb-1 text-sm transition-colors ${
              activeId === conv.id
                ? 'bg-pdt-accent/10 text-pdt-accent'
                : 'text-muted-foreground hover:bg-[#2d2d32] hover:text-foreground'
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
              className="opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-red-400 transition-all"
            >
              <Trash2 className="w-3 h-3" />
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build --mode development 2>&1 | tail -5`
Expected: Build succeeds.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/presentation/components/chat/ChatSidebar.tsx
git commit -m "feat: restyle chat sidebar with dark industrial theme"
```

---

### Task 5: Restyle ChatMessage as Dark Cards

**Files:**
- Modify: `frontend/src/presentation/components/chat/ChatMessage.tsx`

- [ ] **Step 1: Rewrite ChatMessage with card styling**

Replace the entire content of `ChatMessage.tsx` with:

```tsx
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
    <ChatEvent className="py-2">
      <ChatEventAddon>
        <ChatEventAvatar
          fallback={role === 'assistant' ? <Bot className="size-4" /> : <User className="size-4" />}
          className={role === 'assistant' ? 'bg-pdt-accent/20 text-pdt-accent' : 'bg-[#2d2d32] text-pdt-neutral'}
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
          <div className="rounded-lg border border-border bg-[#2d2d32] p-4">
            <div className="prose prose-invert prose-sm max-w-none text-foreground
              prose-p:my-1 prose-p:text-foreground prose-ul:my-1 prose-ol:my-1 prose-li:my-0.5 prose-li:text-foreground
              prose-headings:text-foreground prose-headings:mt-3 prose-headings:mb-1
              prose-code:text-pdt-accent prose-code:bg-[#1B1B1E] prose-code:px-1 prose-code:py-0.5 prose-code:rounded
              prose-pre:bg-[#1B1B1E] prose-pre:border prose-pre:border-border
              prose-strong:text-foreground
              prose-a:text-pdt-accent
              prose-td:text-foreground prose-th:text-foreground">
              <ReactMarkdown>{content}</ReactMarkdown>
            </div>
            {isStreaming && (
              <span className="inline-block w-2 h-4 bg-pdt-accent animate-pulse ml-0.5 mt-1" />
            )}
          </div>
        </ChatEventContent>
      </ChatEventBody>
    </ChatEvent>
  )
}
```

- [ ] **Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build --mode development 2>&1 | tail -5`
Expected: Build succeeds.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/presentation/components/chat/ChatMessage.tsx
git commit -m "feat: restyle chat messages as dark cards with border"
```

---

### Task 6: Update ChatToolbar Styling for Docked Input

**Files:**
- Modify: `frontend/src/components/chat/chat-toolbar.tsx:61-81`

- [ ] **Step 1: Update ChatToolbar background**

In `frontend/src/components/chat/chat-toolbar.tsx`, update the ChatToolbar component's outer div className (line 68):

```tsx
// Before
className={cn("sticky bottom-0 p-2 pt-0 bg-background", className)}

// After
className={cn("sticky bottom-0 p-3 pt-0 bg-[#1B1B1E] border-t border-border", className)}
```

- [ ] **Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite build --mode development 2>&1 | tail -5`
Expected: Build succeeds.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/chat/chat-toolbar.tsx
git commit -m "feat: restyle chat toolbar as docked dark input bar"
```

---

### Task 7: Visual Verification

- [ ] **Step 1: Start the dev server**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx vite --host 2>&1 | head -10`

- [ ] **Step 2: Manual verification checklist**

Open the app in a browser and verify:
1. Dashboard pages (Jira, Commits, Reports) — shadcn buttons now render correctly (not invisible black)
2. Navigate to AI Assistant via sidebar — goes to `/assistant`
3. Assistant page is full-screen (no dashboard sidebar/navbar)
4. Top bar shows "Back to Dashboard" button and "PDT Assistant" title
5. Clicking "Back to Dashboard" navigates to `/dashboard/home`
6. Conversation sidebar is visible on the left with dark background
7. Chat area has slightly lighter dark background
8. Messages display in dark cards with borders
9. Input bar is docked at bottom with dark background
10. Send button is golden accent colored
11. Chat functionality works: send message, receive streaming response, switch conversations

- [ ] **Step 3: Commit any final tweaks if needed**
