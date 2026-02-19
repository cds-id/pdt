# PDT Dashboard Overhaul Design

**Date**: 2026-02-19
**Goal**: Implement all backend APIs in the dashboard, add charts/analytics, refine UI with yellow-dominant industrial theme

## 1. Theme Overhaul — Yellow-Dominant Industrial

**Current**: Maroon (#96031A) as primary accent, yellow (#F8C630) as secondary
**New**: Golden yellow (#F8C630) becomes the primary accent everywhere

### Color Changes

| Token | Current | New |
|-------|---------|-----|
| `pdt.accent.DEFAULT` | `#96031A` (maroon) | `#F8C630` (yellow) |
| `pdt.accent.hover` | `#b30422` | `#E5A100` (amber) |
| `pdt.accent.dark` | `#6a0011` | `#CC8400` (deep gold) |
| `pdt.background` | `#F8C630` (yellow) | `#96031A` (maroon — danger only) |
| CSS `--accent` | `350 82% 35%` (maroon) | `46 94% 58%` (yellow) |
| Dark `--accent` | `350 82% 55%` | `46 94% 58%` |

### Chart Color Palette (Industrial Yellow)
- Chart 1: `#F8C630` (bright yellow)
- Chart 2: `#E5A100` (amber)
- Chart 3: `#CC8400` (deep gold)
- Chart 4: `#A66E00` (bronze)
- Chart 5: `#96031A` (maroon — contrast accent)

### Component Updates
- Buttons (`variant="pdt"`): Yellow bg, dark text
- StatsCard: Yellow icon containers
- Sidebar: Yellow left-border on active nav item
- Hazard-stripe patterns for section dividers

## 2. Dashboard Home — Charts & Analytics Hub

### Row 1: Stats Cards (3 existing + 1 new)
- **Total Commits (30d)** — sparkline trend
- **Linked to Jira** — circular progress ring
- **Active Sprint Cards** — mini status breakdown
- **NEW: Repositories** — count with provider icons

### Row 2: Charts (2-column)
- **Left: Commit Activity** — AreaChart, 30-day rolling, daily grouping, yellow gradient fill. Data from `useListCommitsQuery` grouped by date.
- **Right: Jira Card Status** — PieChart (donut), cards by status. Data from `useListCardsQuery`.

### Row 3: Charts (2-column)
- **Left: Linked vs Unlinked** — Radial progress ring, yellow=linked, dark gray=unlinked. Data from `useListCommitsQuery`.
- **Right: Sprint Velocity** — BarChart, cards completed per sprint (last 5). Data from `useListSprintsQuery` + card counts.

### Row 4: Recent Commits (existing, refined)

## 3. New Pages & API Connections

### A. Jira Card Detail Page — `/dashboard/jira/cards/:key`
- Card summary, status, assignee, sprint info
- Linked commits list (from `getCard` API)
- Subtasks display
- Clickable from Jira page and commits page
- Uses `useGetCardQuery(key)`

### B. Report Templates — `/dashboard/reports/templates`
- CRUD for templates (list, create, edit, delete)
- Markdown editor + live preview via `previewTemplate` API
- Set default template toggle
- Tab within Reports page
- Uses `useListTemplatesQuery`, `useCreateTemplateMutation`, `useUpdateTemplateMutation`, `useDeleteTemplateMutation`, `usePreviewTemplateQuery`

### C. Sync History Panel — Dashboard Home sidebar/section
- Last sync times for commits and Jira
- Status indicators (success/failed/running)
- Uses `useGetSyncStatusQuery`

## 4. Existing Page Refinements

### Jira Page
- Cards become clickable (navigate to detail page)
- Card count badges on sprint list items
- Status filter tabs (All / To Do / In Progress / Done)

### Commits Page
- Date range filter
- Jira card keys as clickable links to card detail
- Visual distinction for linked vs unlinked rows

### Reports Page
- "Templates" tab added
- Report content viewer (modal or inline expand)

### Settings Page
- Validation status per integration with colored indicators
- "Test Connection" buttons per integration

## 5. Technical Decisions

- **Chart Library**: Recharts (declarative, React-native, composable)
- **Chart Components**: `CommitActivityChart`, `CardStatusChart`, `LinkageGaugeChart`, `SprintVelocityChart`
- **Routing**: Add `/dashboard/jira/cards/:key` route
- **State**: All data via existing RTK Query hooks — no new backend APIs needed
- **Theme**: Modify CSS variables + Tailwind config, update component classes
