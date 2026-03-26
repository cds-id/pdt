# Assistant Page Redesign — Full-Screen Industrial Chat

## Overview

Redesign the AI Assistant page from a poorly styled dashboard-embedded view into a full-screen dedicated chat experience with an industrial dark theme. The page will have its own layout — no dashboard sidebar, no navbar — with a slim top bar for navigation back to the dashboard.

## Current State

- Assistant page is embedded inside `DashboardLayout` at `/dashboard/assistant`
- Uses golden yellow (`#F8C630`) as background color — overwhelming and visually poor
- Generic layout with no visual distinction from other dashboard pages
- Messages lack visual hierarchy and polish
- **Root cause:** `index.html` has no `class="dark"` on `<html>`, so the app uses `:root` CSS variables where `--background` is golden yellow (`46 90% 56%`). The `.dark` variables (proper dark palette) are never activated. Components using hardcoded `pdt-*` hex colors look dark, but anything using CSS variable tokens (`bg-background`, `bg-primary`, `text-primary-foreground`) renders with the wrong light-mode palette — e.g., default shadcn buttons appear as invisible black buttons with no border

## Design Goals

- Full-screen chat experience (no dashboard chrome)
- Industrial dark aesthetic using existing PDT brand colors
- Golden accent used sparingly — buttons, active states, links only
- Claude-style layout with always-visible conversation sidebar
- Polished, modern feel

## Layout

```
┌─────────────────────────────────────────────────────┐
│ [← Back to Dashboard]              PDT Assistant    │  Top bar (h-12)
├──────────────┬──────────────────────────────────────┤
│              │                                      │
│  Conversation│     Chat Messages Area               │
│  Sidebar     │     (centered max-w-3xl)             │
│  (w-64)      │                                      │
│              │     ┌─────────────────────┐          │
│  + New Conv  │     │ Message card        │          │
│  ● Conv 1    │     └─────────────────────┘          │
│  ● Conv 2    │     ┌─────────────────────┐          │
│              │     │ Message card        │          │
│              │     └─────────────────────┘          │
│              │                                      │
│              ├──────────────────────────────────────┤
│              │ [Input area....................] [▶]  │  Docked input bar
└──────────────┴──────────────────────────────────────┘
```

### Top Bar
- Height: `h-12`
- Background: `#1B1B1E` with bottom border
- Left: Back button (← icon + "Back to Dashboard") linking to `/dashboard/home`
- Right or center: "PDT Assistant" title text

### Conversation Sidebar
- Width: `w-64`, always visible
- Background: `#1B1B1E` with right border (`border-border`)
- Top: "+ New Conversation" button
- Below: scrollable list of conversations
- Active conversation: golden accent highlight (`bg-pdt-accent/10 text-pdt-accent`)
- Hover: subtle lighter background
- Delete button appears on hover (existing behavior, keep it)

### Chat Area
- Background: `#242428` (slightly lighter than sidebar for visual separation)
- Messages in a centered container (`max-w-3xl mx-auto`)
- Full height between top bar and input bar, scrollable

### Message Cards
- Background: `#2d2d32` (`pdt-primary-light`)
- Border: thin `border-border`
- Rounded: `rounded-lg`
- Padding: `p-4`
- Avatar on the left, message content on the right
- User messages and assistant messages use the same card style (differentiated by avatar)
- Assistant avatar: golden accent ring/background
- User avatar: neutral dark background

### Input Bar
- Docked to bottom of chat area
- Background: `#1B1B1E` with top border
- Contains auto-growing textarea and send button
- Send button: golden accent (`bg-pdt-accent text-pdt-primary`)
- Placeholder text in `muted-foreground`

## Color Palette

| Element | Color | Token |
|---------|-------|-------|
| Page/sidebar background | `#1B1B1E` | `pdt-primary` |
| Chat area background | `#242428` | New: `pdt-primary-chat` or inline |
| Message cards | `#2d2d32` | `pdt-primary-light` |
| Primary text | `#FBFFFE` | `pdt-neutral` |
| Muted text | existing | `muted-foreground` |
| Golden accent | `#F8C630` | `pdt-accent` |
| Borders | existing | `border` CSS variable |
| Danger/delete | `#96031A` | `pdt-danger` |

## Routing Changes

Move the assistant page out of `DashboardLayout` into its own top-level route:

```tsx
// Before: nested inside DashboardLayout
{ path: 'dashboard/assistant', element: <AssistantPage /> }

// After: top-level route with no layout wrapper (or its own minimal layout)
{ path: 'assistant', element: <AssistantPage /> }
```

The assistant page will manage its own full-screen layout internally — no external layout wrapper needed.

Update any navigation links pointing to `/dashboard/assistant` to point to `/assistant`.

## Component Changes

### AssistantPage.tsx
- Remove dependency on `DashboardLayout`
- Implement full-screen layout: `h-screen flex flex-col`
- Add top bar with back navigation
- Restructure into: top bar + (sidebar | chat area + input)

### ChatSidebar.tsx
- Restyle with dark industrial theme
- Remove yellow/golden backgrounds
- Slim down, cleaner typography
- Keep existing functionality (new conversation, list, delete)

### ChatMessage.tsx
- Restyle as subtle dark cards
- Remove golden background usage
- Keep markdown rendering with updated prose styling (golden only for links/code)

### ThinkingIndicator.tsx
- Keep animation, update colors to match new theme

### ToolStatus.tsx
- Update colors to match new theme

### Global Dark Mode Fix
- Add `class="dark"` to `<html>` in `index.html` — this activates the `.dark` CSS variables which already have the correct dark palette
- This fixes shadcn components globally (buttons, cards, inputs, etc.) across ALL pages, not just the assistant page
- The `:root` (light mode) variables can remain as-is since they won't be used

### CSS Variables (index.css)
- Review `.dark` variables to ensure they align with the industrial theme
- No changes expected — the existing `.dark` palette is already correct

## What NOT to Change

- WebSocket connection logic
- RTK Query services
- Conversation state management
- Message streaming/buffering logic
- Domain interfaces

## Success Criteria

- `class="dark"` on `<html>` — all shadcn components render correctly across all pages
- Assistant page is full-screen with no dashboard sidebar/navbar
- Dark industrial look with golden accent used sparingly
- Back button navigates to dashboard
- All existing chat functionality works (send, receive, stream, conversations, delete)
- Messages display in subtle dark cards
- Input is docked at bottom
- Conversation sidebar always visible on the left
- Existing dashboard pages (Jira, Commits, Reports, etc.) still look correct after the dark mode fix
