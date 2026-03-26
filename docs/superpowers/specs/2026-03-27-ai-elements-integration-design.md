# AI-Elements Integration — Assistant Page

## Overview

Replace custom chat components in the assistant page with ai-elements primitives from the `@ai-elements/all` shadcn registry. This provides built-in markdown rendering with Shiki syntax highlighting, sticky-to-bottom scrolling, collapsible reasoning/thinking display, tool execution chain-of-thought, and suggestion buttons.

## Current State

The assistant page was just redesigned with a full-screen industrial dark layout. It uses custom chat components (`ChatMessage`, `ThinkingIndicator`, `ToolStatus`) and shadcn chat primitives (`Chat`, `ChatMessages`, `ChatToolbar`). These work but lack polish — no syntax highlighting, no collapsible reasoning, basic tool status display, no suggestions.

## Design Goals

- Replace custom chat components with ai-elements equivalents
- Keep all existing WebSocket/state logic untouched
- Add suggestion buttons (empty state + after responses)
- Better markdown rendering (Shiki code highlighting, math, mermaid support)
- Collapsible reasoning/thinking with auto-open/close behavior
- Tool execution displayed as chain-of-thought steps

## Component Mapping

| Current Component | Replaced By | Source |
|-------------------|------------|--------|
| `Chat` + `ChatMessages` | `Conversation` + `ConversationContent` + `ConversationScrollButton` | `components/ai-elements/conversation.tsx` |
| Empty state div | `ConversationEmptyState` | `components/ai-elements/conversation.tsx` |
| `ChatMessage` | `Message` + `MessageContent` + `MessageResponse` | `components/ai-elements/message.tsx` |
| `ChatToolbar` + `ChatToolbarTextarea` + `ChatToolbarButton` | `PromptInput` (text-only, form-based) | `components/ai-elements/prompt-input.tsx` |
| `ThinkingIndicator` | `Reasoning` + `ReasoningTrigger` + `ReasoningContent` | `components/ai-elements/reasoning.tsx` |
| `ToolStatus` | `ChainOfThought` + `ChainOfThoughtHeader` + `ChainOfThoughtContent` + `ChainOfThoughtStep` | `components/ai-elements/chain-of-thought.tsx` |
| (new) | `Suggestions` + `Suggestion` | `components/ai-elements/suggestion.tsx` |
| (new) | `MessageActions` + `MessageAction` | `components/ai-elements/message.tsx` |

## Data Flow

### Messages
- `Message` expects `from: "user" | "assistant"` — matches existing `DisplayMessage.role`
- `MessageResponse` accepts markdown string as children + `isAnimating` for streaming
- Replaces custom `ReactMarkdown` + prose classes with `Streamdown` (code, math, mermaid, cjk plugins built-in)

### Thinking/Reasoning
- `Reasoning` takes `isStreaming` prop — maps from `isThinking` state
- `ReasoningContent` takes string children — maps from `thinkingMessage`
- Auto-opens when streaming starts, auto-closes 1s after streaming ends

### Tool Execution
- `ChainOfThought` wraps all tool steps
- Each `toolStatus` item maps to a `ChainOfThoughtStep` with:
  - `label`: tool display name (from existing `toolLabels` map)
  - `status`: `"active"` for executing, `"complete"` for completed
  - `icon`: `Loader2` for executing, `CheckCircle` for completed
- Chain is collapsible with `ChainOfThoughtHeader`

### Prompt Input
- `PromptInput` is a `<form>` with `onSubmit(message)` callback
- We use it text-only (no attachments)
- Send button uses `PromptInputAction` or a custom button inside the input group
- Maps to existing `handleSend` logic

### Suggestions
- **Empty state:** Static array of starter prompts:
  - "Show my commits from today"
  - "Summarize my Jira cards"
  - "Generate a daily report"
  - "What did I work on this week?"
- **After responses:** Static follow-up suggestions after last assistant message (only when not streaming):
  - "Tell me more"
  - "Show related commits"
  - "Generate a report"
- `Suggestion` `onClick` calls `handleSend(suggestion)` directly

## Files Changed

### Modified
- `frontend/src/presentation/pages/AssistantPage.tsx` — Replace render section with ai-elements components. Keep all state/WebSocket/handler logic unchanged.

### Deleted
- `frontend/src/presentation/components/chat/ChatMessage.tsx` — replaced by `Message`
- `frontend/src/presentation/components/chat/ThinkingIndicator.tsx` — replaced by `Reasoning`
- `frontend/src/presentation/components/chat/ToolStatus.tsx` — replaced by `ChainOfThought`

### Unchanged
- `frontend/src/presentation/components/chat/ChatSidebar.tsx` — already restyled
- `frontend/src/components/chat/*` — old primitives remain for potential use elsewhere
- All WebSocket, RTK Query, domain interfaces
- Top bar, layout structure, routing

## Styling Notes

- `Message` uses `is-user`/`is-assistant` CSS classes for auto-layout. User messages get `bg-secondary` bubble, assistant messages flow freely.
- For the industrial dark theme: apply `bg-[#242428]` to `Conversation`, keep sidebar `bg-[#1B1B1E]`
- `PromptInput` will need custom styling to match the docked dark input bar (`bg-[#1B1B1E] border-t border-border`)
- `Suggestion` buttons use `variant="outline"` with `rounded-full` — fits the industrial theme as-is
- `ConversationScrollButton` positioned absolutely — works within the `Conversation` container

## Success Criteria

- Messages render with Streamdown (syntax-highlighted code blocks, math, mermaid diagrams)
- Thinking state shows as collapsible `Reasoning` block with shimmer animation
- Tool execution shows as `ChainOfThought` steps with active/complete status
- Empty state shows starter suggestion buttons
- After each assistant response (when not streaming), follow-up suggestions appear
- Clicking a suggestion sends it as a message
- Scroll-to-bottom button appears when scrolled up
- All existing chat functionality preserved (send, stream, conversations, delete)
- Copy button on assistant messages
