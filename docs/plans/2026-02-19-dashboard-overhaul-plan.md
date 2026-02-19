# PDT Dashboard Overhaul Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement all backend APIs in the dashboard, add Recharts analytics, and refine UI with a yellow-dominant industrial theme.

**Architecture:** The frontend is React 18 + TypeScript with RTK Query for data fetching, Tailwind CSS + shadcn/ui for styling. All backend APIs already exist — this is purely frontend work. We add Recharts for charts, swap the color tokens from maroon-accent to yellow-accent, create new pages for Jira card details and report templates, and refine all existing pages.

**Tech Stack:** React 18, TypeScript, Recharts, RTK Query, Tailwind CSS, shadcn/ui, Vite

---

### Task 1: Install Recharts dependency

**Files:**
- Modify: `frontend/package.json`

**Step 1: Install recharts**

Run: `cd /home/nst/GolandProjects/pdt/frontend && bun add recharts`

Expected: recharts added to dependencies in package.json

**Step 2: Verify installation**

Run: `cd /home/nst/GolandProjects/pdt/frontend && bun run typecheck`

Expected: No new type errors

**Step 3: Commit**

```bash
git add frontend/package.json frontend/bun.lockb
git commit -m "feat: add recharts dependency for dashboard analytics"
```

---

### Task 2: Theme overhaul — swap accent colors to yellow-dominant

**Files:**
- Modify: `frontend/src/index.css` (CSS variables)
- Modify: `frontend/tailwind.config.mjs` (pdt color tokens)
- Modify: `frontend/src/components/ui/button.tsx` (pdt button variants)
- Modify: `frontend/src/presentation/components/dashboard/StatsCard/StatsCard.tsx` (icon color)
- Modify: `frontend/src/presentation/components/common/StatusBadge.tsx` (warning variant)

**Step 1: Update CSS variables in `frontend/src/index.css`**

Change the `:root` CSS variables:
```css
/* :root (light mode) */
--accent: 46 94% 58%;           /* was: 350 82% 35% (maroon) → now: yellow */
--accent-foreground: 0 0% 11%;  /* was: 0 0% 99% → now: dark text on yellow */

/* Chart colors - industrial yellow palette */
--chart-1: 46 94% 58%;    /* #F8C630 bright yellow */
--chart-2: 40 100% 45%;   /* #E5A100 amber */
--chart-3: 36 100% 40%;   /* #CC8400 deep gold */
--chart-4: 30 100% 33%;   /* #A66E00 bronze */
--chart-5: 350 97% 30%;   /* #96031A maroon contrast */
```

Change the `.dark` CSS variables:
```css
--accent: 46 94% 58%;          /* was: 350 82% 55% → now: yellow */
--accent-foreground: 0 0% 11%; /* dark text on yellow */

/* Same chart colors in dark mode */
--chart-1: 46 94% 58%;
--chart-2: 40 100% 45%;
--chart-3: 36 100% 40%;
--chart-4: 30 100% 33%;
--chart-5: 350 97% 30%;
```

**Step 2: Update Tailwind pdt tokens in `frontend/tailwind.config.mjs`**

Swap the pdt color tokens — yellow becomes the accent, maroon becomes danger-only:
```javascript
pdt: {
  primary: {
    DEFAULT: '#1B1B1E',    // dark steel (unchanged)
    light: '#2d2d32',      // slightly lighter (unchanged)
    dark: '#000000',       // pure black (unchanged)
  },
  accent: {
    DEFAULT: '#F8C630',    // was #96031A → now golden yellow
    hover: '#E5A100',      // was #b30422 → now amber
    dark: '#CC8400',       // was #6a0011 → now deep gold
  },
  background: '#F8C630',   // keep yellow for backward compat (used widely)
  danger: '#96031A',       // new: maroon moved here for danger/alert usage
  neutral: '#FBFFFE',      // off-white (unchanged)
},
```

**Step 3: Update button variants in `frontend/src/components/ui/button.tsx`**

Change the `pdt` and `pdtOutline` variants to use yellow accent:
```typescript
pdt: 'bg-pdt-accent text-pdt-primary font-semibold border-2 border-pdt-accent hover:bg-pdt-accent-hover hover:border-pdt-accent-hover',
pdtOutline:
  'bg-transparent text-pdt-accent border-2 border-pdt-accent hover:bg-pdt-accent hover:text-pdt-primary font-semibold'
```

**Step 4: Update StatsCard icon styling in `frontend/src/presentation/components/dashboard/StatsCard/StatsCard.tsx`**

Change icon container from border-based to filled yellow:
```tsx
{Icon && (
  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-pdt-accent/20">
    <Icon className="size-4 text-pdt-accent" />
  </div>
)}
```

**Step 5: Update StatusBadge warning variant in `frontend/src/presentation/components/common/StatusBadge.tsx`**

The `warning` variant currently uses `pdt-background` which is yellow. After the swap it should still use yellow:
```typescript
const variantStyles: Record<BadgeVariant, string> = {
  success: 'bg-green-500/20 text-green-400',
  warning: 'bg-pdt-accent/20 text-pdt-accent',  // was pdt-background → now pdt-accent
  info: 'bg-blue-500/20 text-blue-400',
  neutral: 'bg-gray-500/20 text-gray-400',
  danger: 'bg-red-500/20 text-red-400'
}
```

**Step 6: Search and replace all `text-pdt-background` and `bg-pdt-background` references across the codebase**

These references to the yellow color need updating. Search for `pdt-background` usages in all `.tsx` files and update them to `pdt-accent` where they reference the yellow accent color (not actual background). Key files:
- `DashboardHomePage.tsx`: `text-pdt-background` on SHA codes → `text-pdt-accent`
- `CommitsPage.tsx`: `text-pdt-background` on SHA codes → `text-pdt-accent`
- `JiraPage.tsx`: `text-pdt-background` on card keys, `border-pdt-background/20` and `bg-pdt-background/5` → `border-pdt-accent/20` and `bg-pdt-accent/5`
- `CommitsPage.tsx`: `bg-pdt-background/10` table header, `border-pdt-background/20` borders → `bg-pdt-accent/10`, `border-pdt-accent/20`
- `SettingsPage.tsx`: `border-pdt-background/20` → `border-pdt-accent/20`
- `ReportsPage.tsx`: `text-pdt-background` on download links → `text-pdt-accent`
- `StatsCard.tsx`: `border-pdt-background/30` → `border-pdt-accent/30` (if not already updated in step 4)
- `SidebarNavItem.tsx`: any `pdt-background` references → `pdt-accent`

Run a project-wide search: `grep -rn "pdt-background" frontend/src/` to find all occurrences and update each one.

**Step 7: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && bun run build`

Expected: Build succeeds with no errors

**Step 8: Commit**

```bash
git add -A
git commit -m "feat: overhaul theme to yellow-dominant industrial palette"
```

---

### Task 3: Create chart components — CommitActivityChart

**Files:**
- Create: `frontend/src/presentation/components/charts/CommitActivityChart.tsx`
- Create: `frontend/src/presentation/components/charts/index.ts`

**Step 1: Create the CommitActivityChart component**

Create `frontend/src/presentation/components/charts/CommitActivityChart.tsx`:

```tsx
import { useMemo } from 'react'
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer
} from 'recharts'

import type { Commit } from '@/infrastructure/services/commit.service'

interface CommitActivityChartProps {
  commits: Commit[]
}

export function CommitActivityChart({ commits }: CommitActivityChartProps) {
  const data = useMemo(() => {
    const grouped: Record<string, number> = {}

    // Last 30 days
    for (let i = 29; i >= 0; i--) {
      const d = new Date()
      d.setDate(d.getDate() - i)
      const key = d.toISOString().split('T')[0]
      grouped[key] = 0
    }

    commits.forEach((c) => {
      const key = new Date(c.date).toISOString().split('T')[0]
      if (key in grouped) {
        grouped[key]++
      }
    })

    return Object.entries(grouped).map(([date, count]) => ({
      date,
      label: new Date(date).toLocaleDateString('en', { month: 'short', day: 'numeric' }),
      commits: count
    }))
  }, [commits])

  return (
    <ResponsiveContainer width="100%" height={280}>
      <AreaChart data={data}>
        <defs>
          <linearGradient id="commitGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#F8C630" stopOpacity={0.4} />
            <stop offset="95%" stopColor="#F8C630" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="#2d2d32" />
        <XAxis
          dataKey="label"
          stroke="#FBFFFE50"
          fontSize={11}
          tickLine={false}
          interval="preserveStartEnd"
        />
        <YAxis
          stroke="#FBFFFE50"
          fontSize={11}
          tickLine={false}
          allowDecimals={false}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: '#1B1B1E',
            border: '1px solid #F8C63040',
            borderRadius: '8px',
            color: '#FBFFFE'
          }}
        />
        <Area
          type="monotone"
          dataKey="commits"
          stroke="#F8C630"
          strokeWidth={2}
          fill="url(#commitGradient)"
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}
```

**Step 2: Create barrel export**

Create `frontend/src/presentation/components/charts/index.ts`:
```typescript
export { CommitActivityChart } from './CommitActivityChart'
```

**Step 3: Commit**

```bash
git add frontend/src/presentation/components/charts/
git commit -m "feat: add CommitActivityChart area chart component"
```

---

### Task 4: Create chart components — CardStatusChart (donut)

**Files:**
- Create: `frontend/src/presentation/components/charts/CardStatusChart.tsx`
- Modify: `frontend/src/presentation/components/charts/index.ts`

**Step 1: Create the CardStatusChart component**

Create `frontend/src/presentation/components/charts/CardStatusChart.tsx`:

```tsx
import { useMemo } from 'react'
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip, Legend } from 'recharts'

import type { JiraCard } from '@/infrastructure/services/jira.service'

interface CardStatusChartProps {
  cards: JiraCard[]
}

const STATUS_COLORS: Record<string, string> = {
  'Done': '#22c55e',
  'In Progress': '#F8C630',
  'To Do': '#6b7280',
  'In Review': '#3b82f6',
  'Blocked': '#96031A'
}

const DEFAULT_COLOR = '#A66E00'

export function CardStatusChart({ cards }: CardStatusChartProps) {
  const data = useMemo(() => {
    const grouped: Record<string, number> = {}
    cards.forEach((card) => {
      grouped[card.status] = (grouped[card.status] || 0) + 1
    })
    return Object.entries(grouped).map(([status, count]) => ({
      name: status,
      value: count,
      color: STATUS_COLORS[status] || DEFAULT_COLOR
    }))
  }, [cards])

  if (data.length === 0) {
    return (
      <div className="flex h-[280px] items-center justify-center text-pdt-neutral/40">
        No card data
      </div>
    )
  }

  return (
    <ResponsiveContainer width="100%" height={280}>
      <PieChart>
        <Pie
          data={data}
          cx="50%"
          cy="50%"
          innerRadius={60}
          outerRadius={100}
          paddingAngle={3}
          dataKey="value"
        >
          {data.map((entry, index) => (
            <Cell key={`cell-${index}`} fill={entry.color} />
          ))}
        </Pie>
        <Tooltip
          contentStyle={{
            backgroundColor: '#1B1B1E',
            border: '1px solid #F8C63040',
            borderRadius: '8px',
            color: '#FBFFFE'
          }}
        />
        <Legend
          wrapperStyle={{ color: '#FBFFFE', fontSize: '12px' }}
        />
      </PieChart>
    </ResponsiveContainer>
  )
}
```

**Step 2: Add to barrel export**

Add to `frontend/src/presentation/components/charts/index.ts`:
```typescript
export { CardStatusChart } from './CardStatusChart'
```

**Step 3: Commit**

```bash
git add frontend/src/presentation/components/charts/
git commit -m "feat: add CardStatusChart donut chart component"
```

---

### Task 5: Create chart components — LinkageGaugeChart (radial ring)

**Files:**
- Create: `frontend/src/presentation/components/charts/LinkageGaugeChart.tsx`
- Modify: `frontend/src/presentation/components/charts/index.ts`

**Step 1: Create the LinkageGaugeChart component**

Create `frontend/src/presentation/components/charts/LinkageGaugeChart.tsx`:

```tsx
import { RadialBarChart, RadialBar, ResponsiveContainer } from 'recharts'

interface LinkageGaugeChartProps {
  linked: number
  total: number
}

export function LinkageGaugeChart({ linked, total }: LinkageGaugeChartProps) {
  const percent = total > 0 ? Math.round((linked / total) * 100) : 0

  const data = [
    { name: 'bg', value: 100, fill: '#2d2d32' },
    { name: 'linked', value: percent, fill: '#F8C630' }
  ]

  return (
    <div className="relative">
      <ResponsiveContainer width="100%" height={280}>
        <RadialBarChart
          cx="50%"
          cy="50%"
          innerRadius="60%"
          outerRadius="90%"
          startAngle={90}
          endAngle={-270}
          data={data}
          barSize={20}
        >
          <RadialBar
            dataKey="value"
            cornerRadius={10}
            background={false}
          />
        </RadialBarChart>
      </ResponsiveContainer>
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <span className="text-3xl font-bold text-pdt-accent">{percent}%</span>
        <span className="text-xs text-pdt-neutral/60">
          {linked}/{total} linked
        </span>
      </div>
    </div>
  )
}
```

**Step 2: Add to barrel export**

Add to `frontend/src/presentation/components/charts/index.ts`:
```typescript
export { LinkageGaugeChart } from './LinkageGaugeChart'
```

**Step 3: Commit**

```bash
git add frontend/src/presentation/components/charts/
git commit -m "feat: add LinkageGaugeChart radial progress component"
```

---

### Task 6: Create chart components — SprintVelocityChart (bar)

**Files:**
- Create: `frontend/src/presentation/components/charts/SprintVelocityChart.tsx`
- Modify: `frontend/src/presentation/components/charts/index.ts`

**Step 1: Create the SprintVelocityChart component**

Create `frontend/src/presentation/components/charts/SprintVelocityChart.tsx`:

```tsx
import { useMemo } from 'react'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer
} from 'recharts'

import type { JiraSprint } from '@/infrastructure/services/jira.service'

interface SprintVelocityChartProps {
  sprints: JiraSprint[]
}

export function SprintVelocityChart({ sprints }: SprintVelocityChartProps) {
  const data = useMemo(() => {
    // Take last 5 sprints that have cards, sorted by start_date
    return sprints
      .filter((s) => s.cards && s.cards.length > 0)
      .sort((a, b) => (a.start_date || '').localeCompare(b.start_date || ''))
      .slice(-5)
      .map((sprint) => {
        const total = sprint.cards?.length || 0
        const done = sprint.cards?.filter((c) => c.status === 'Done').length || 0
        return {
          name: sprint.name.length > 15 ? sprint.name.slice(0, 15) + '...' : sprint.name,
          total,
          done
        }
      })
  }, [sprints])

  if (data.length === 0) {
    return (
      <div className="flex h-[280px] items-center justify-center text-pdt-neutral/40">
        No sprint data
      </div>
    )
  }

  return (
    <ResponsiveContainer width="100%" height={280}>
      <BarChart data={data}>
        <CartesianGrid strokeDasharray="3 3" stroke="#2d2d32" />
        <XAxis
          dataKey="name"
          stroke="#FBFFFE50"
          fontSize={11}
          tickLine={false}
        />
        <YAxis
          stroke="#FBFFFE50"
          fontSize={11}
          tickLine={false}
          allowDecimals={false}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: '#1B1B1E',
            border: '1px solid #F8C63040',
            borderRadius: '8px',
            color: '#FBFFFE'
          }}
        />
        <Bar dataKey="total" fill="#A66E00" radius={[4, 4, 0, 0]} name="Total Cards" />
        <Bar dataKey="done" fill="#F8C630" radius={[4, 4, 0, 0]} name="Completed" />
      </BarChart>
    </ResponsiveContainer>
  )
}
```

**Step 2: Add to barrel export**

Add to `frontend/src/presentation/components/charts/index.ts`:
```typescript
export { SprintVelocityChart } from './SprintVelocityChart'
```

**Step 3: Commit**

```bash
git add frontend/src/presentation/components/charts/
git commit -m "feat: add SprintVelocityChart bar chart component"
```

---

### Task 7: Revamp DashboardHomePage with charts and enhanced stats

**Files:**
- Modify: `frontend/src/presentation/pages/DashboardHomePage.tsx`

**Step 1: Rewrite DashboardHomePage**

Replace the contents of `frontend/src/presentation/pages/DashboardHomePage.tsx` with a version that includes:

1. **4 stats cards** (add repos count) — using existing `useListReposQuery`
2. **Row 2**: CommitActivityChart (left) + CardStatusChart (right)
3. **Row 3**: LinkageGaugeChart (left) + SprintVelocityChart (right)
4. **Sync Status panel** — shows last sync times from `useGetSyncStatusQuery`
5. **Recent Commits** (refined) at bottom

New imports to add:
```tsx
import { useListReposQuery } from '@/infrastructure/services/repo.service'
import { useListCardsQuery, useListSprintsQuery } from '@/infrastructure/services/jira.service'
import { CommitActivityChart, CardStatusChart, LinkageGaugeChart, SprintVelocityChart } from '@/presentation/components/charts'
import { GitBranch, Clock } from 'lucide-react'
```

New data hooks to add:
```tsx
const { data: repos } = useListReposQuery()
const { data: cards = [] } = useListCardsQuery()
const { data: sprints = [] } = useListSprintsQuery()
```

Add a 4th stats card:
```tsx
{
  title: 'Repositories',
  value: repos?.length || 0,
  description: 'Tracked repos',
  icon: GitBranch
}
```

Charts section layout — two 2-column grid rows:
```tsx
{/* Charts Row 1 */}
<div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
  <DataCard title="Commit Activity (30 days)">
    <CommitActivityChart commits={commits} />
  </DataCard>
  <DataCard title="Card Status Breakdown">
    <CardStatusChart cards={cards} />
  </DataCard>
</div>

{/* Charts Row 2 */}
<div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
  <DataCard title="Jira Linkage">
    <LinkageGaugeChart linked={linkedCommits} total={totalCommits} />
  </DataCard>
  <DataCard title="Sprint Velocity">
    <SprintVelocityChart sprints={sprints} />
  </DataCard>
</div>
```

Sync status section:
```tsx
{/* Sync Status */}
<DataCard title="Sync Status">
  <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
    <div className="flex items-center gap-3">
      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-pdt-accent/20">
        <GitCommit className="h-5 w-5 text-pdt-accent" />
      </div>
      <div>
        <p className="text-sm font-medium text-pdt-neutral">Commit Sync</p>
        <p className="text-xs text-pdt-neutral/50">
          {syncStatus?.commits?.last_sync
            ? `Last: ${new Date(syncStatus.commits.last_sync).toLocaleString()}`
            : 'Never synced'}
        </p>
      </div>
    </div>
    <div className="flex items-center gap-3">
      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-pdt-accent/20">
        <Trello className="h-5 w-5 text-pdt-accent" />
      </div>
      <div>
        <p className="text-sm font-medium text-pdt-neutral">Jira Sync</p>
        <p className="text-xs text-pdt-neutral/50">
          {syncStatus?.jira?.last_sync
            ? `Last: ${new Date(syncStatus.jira.last_sync).toLocaleString()}`
            : 'Never synced'}
        </p>
      </div>
    </div>
  </div>
</DataCard>
```

Change stats grid to 4 columns:
```tsx
<div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
```

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && bun run build`

Expected: Build succeeds

**Step 3: Commit**

```bash
git add frontend/src/presentation/pages/DashboardHomePage.tsx
git commit -m "feat: revamp dashboard home with charts, stats, and sync status"
```

---

### Task 8: Create Jira Card Detail Page

**Files:**
- Create: `frontend/src/presentation/pages/JiraCardDetailPage.tsx`
- Modify: `frontend/src/presentation/routes/index.tsx` (add route)
- Modify: `frontend/src/config/navigation.ts` (add breadcrumb support)

**Step 1: Create JiraCardDetailPage**

Create `frontend/src/presentation/pages/JiraCardDetailPage.tsx`:

```tsx
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, GitCommit, ListTree } from 'lucide-react'

import { useGetCardQuery } from '@/infrastructure/services/jira.service'
import { useListCommitsQuery } from '@/infrastructure/services/commit.service'
import { Button } from '@/components/ui/button'
import { PageHeader, DataCard, StatusBadge, EmptyState } from '@/presentation/components/common'

export function JiraCardDetailPage() {
  const { key } = useParams<{ key: string }>()
  const { data: card, isLoading, error } = useGetCardQuery(key!, { skip: !key })
  const { data: allCommits = [] } = useListCommitsQuery({ jira_card_key: key })

  if (isLoading) {
    return <p className="text-pdt-neutral/60">Loading card details...</p>
  }

  if (error || !card) {
    return (
      <EmptyState
        title="Card not found"
        description={`Could not find Jira card ${key}`}
      />
    )
  }

  // Parse subtasks from details_json if available
  let subtasks: { key: string; summary: string; status: string }[] = []
  if (card.details_json) {
    try {
      const details = JSON.parse(card.details_json)
      subtasks = details.subtasks || []
    } catch {
      // ignore parse errors
    }
  }

  return (
    <div className="min-w-0 space-y-4 md:space-y-6">
      <div className="flex items-center gap-3">
        <Link to="/dashboard/jira">
          <Button variant="pdtOutline" size="sm">
            <ArrowLeft className="mr-1 h-4 w-4" /> Back
          </Button>
        </Link>
      </div>

      <PageHeader
        title={card.key}
        description={card.summary}
      />

      {/* Card Info */}
      <DataCard>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div>
            <p className="text-xs text-pdt-neutral/50">Status</p>
            <StatusBadge
              variant={
                card.status === 'Done'
                  ? 'success'
                  : card.status === 'In Progress'
                  ? 'warning'
                  : 'neutral'
              }
            >
              {card.status}
            </StatusBadge>
          </div>
          <div>
            <p className="text-xs text-pdt-neutral/50">Assignee</p>
            <p className="text-sm text-pdt-neutral">{card.assignee || 'Unassigned'}</p>
          </div>
          <div>
            <p className="text-xs text-pdt-neutral/50">Sprint</p>
            <p className="text-sm text-pdt-neutral">{card.sprint_id ? `Sprint #${card.sprint_id}` : 'No sprint'}</p>
          </div>
        </div>
      </DataCard>

      {/* Linked Commits */}
      <DataCard title="Linked Commits">
        {allCommits.length === 0 ? (
          <EmptyState title="No linked commits" description="No commits reference this card." />
        ) : (
          <div className="space-y-0">
            {allCommits.map((commit) => (
              <div
                key={commit.id}
                className="flex items-center gap-3 border-b border-pdt-neutral/10 py-3 last:border-0"
              >
                <GitCommit className="h-4 w-4 shrink-0 text-pdt-accent" />
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm text-pdt-neutral">{commit.message}</p>
                  <p className="text-xs text-pdt-neutral/50">
                    <code className="text-pdt-accent">{commit.sha.slice(0, 7)}</code>
                    {' '}&middot; {commit.author} &middot; {new Date(commit.date).toLocaleDateString()}
                  </p>
                </div>
              </div>
            ))}
          </div>
        )}
      </DataCard>

      {/* Subtasks */}
      {subtasks.length > 0 && (
        <DataCard title="Subtasks">
          <div className="space-y-2">
            {subtasks.map((sub) => (
              <div
                key={sub.key}
                className="flex items-center justify-between rounded-lg border border-pdt-neutral/10 bg-pdt-primary-light p-3"
              >
                <div className="flex items-center gap-2">
                  <ListTree className="h-4 w-4 text-pdt-accent" />
                  <span className="text-sm font-medium text-pdt-accent">{sub.key}</span>
                  <span className="text-sm text-pdt-neutral">{sub.summary}</span>
                </div>
                <StatusBadge
                  variant={sub.status === 'Done' ? 'success' : sub.status === 'In Progress' ? 'warning' : 'neutral'}
                >
                  {sub.status}
                </StatusBadge>
              </div>
            ))}
          </div>
        </DataCard>
      )}
    </div>
  )
}

export default JiraCardDetailPage
```

**Step 2: Add route in `frontend/src/presentation/routes/index.tsx`**

Add import:
```tsx
import { JiraCardDetailPage } from '../pages/JiraCardDetailPage'
```

Add route in the dashboard children array, after the jira route:
```tsx
{ path: 'dashboard/jira/cards/:key', element: <JiraCardDetailPage /> },
```

**Step 3: Update breadcrumbs in `frontend/src/config/navigation.ts`**

Update `getBreadcrumbsForPath` to handle the dynamic card detail path:
```typescript
export const getBreadcrumbsForPath = (
  pathname: string
): { title: string; href: string }[] => {
  const breadcrumbs: { title: string; href: string }[] = [
    { title: 'Home', href: '/dashboard' }
  ]

  // Handle Jira card detail page
  const cardMatch = pathname.match(/^\/dashboard\/jira\/cards\/(.+)$/)
  if (cardMatch) {
    breadcrumbs.push({ title: 'Jira', href: '/dashboard/jira' })
    breadcrumbs.push({ title: cardMatch[1], href: pathname })
    return breadcrumbs
  }

  const item = getNavItemByHref(pathname)
  if (item && item.href !== '/dashboard') {
    breadcrumbs.push({ title: item.title, href: item.href })
  }

  return breadcrumbs
}
```

**Step 4: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && bun run build`

Expected: Build succeeds

**Step 5: Commit**

```bash
git add frontend/src/presentation/pages/JiraCardDetailPage.tsx frontend/src/presentation/routes/index.tsx frontend/src/config/navigation.ts
git commit -m "feat: add Jira card detail page with commits and subtasks"
```

---

### Task 9: Refine JiraPage — clickable cards, filters, card counts

**Files:**
- Modify: `frontend/src/presentation/pages/JiraPage.tsx`

**Step 1: Update JiraPage**

Add imports:
```tsx
import { useState } from 'react'
import { Link } from 'react-router-dom'
```

Add state for filter tabs:
```tsx
const [statusFilter, setStatusFilter] = useState<string>('all')
```

Make cards clickable by wrapping with `<Link>`:
```tsx
<Link to={`/dashboard/jira/cards/${card.key}`} key={card.key}>
  <div className="rounded-lg border border-pdt-accent/20 bg-pdt-accent/5 p-4 transition-colors hover:border-pdt-accent/40 hover:bg-pdt-accent/10">
    ...card content...
  </div>
</Link>
```

Add status filter tabs above the cards grid:
```tsx
<div className="mb-4 flex gap-2">
  {['all', 'To Do', 'In Progress', 'Done'].map((status) => (
    <Button
      key={status}
      variant={statusFilter === status ? 'pdt' : 'pdtOutline'}
      size="sm"
      onClick={() => setStatusFilter(status)}
    >
      {status === 'all' ? 'All' : status}
    </Button>
  ))}
</div>
```

Filter cards by status:
```tsx
const filteredCards = statusFilter === 'all'
  ? cards
  : cards.filter((c) => c.status === statusFilter)
```

Add card count to sprint list items:
```tsx
{sprint.cards && (
  <span className="text-xs text-pdt-accent">{sprint.cards.length} cards</span>
)}
```

To get sprint cards data, use `useGetSprintQuery` for each sprint, OR add the card count to the sprint list response. Since the backend returns sprints with cards when using `getSprint`, the simplest approach is to show the count only for the active sprint and keep the sprint list as-is, adding a note about card counts from the data we already have.

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && bun run build`

**Step 3: Commit**

```bash
git add frontend/src/presentation/pages/JiraPage.tsx
git commit -m "feat: add clickable cards, status filters, and refinements to Jira page"
```

---

### Task 10: Create Report Templates tab in ReportsPage

**Files:**
- Modify: `frontend/src/presentation/pages/ReportsPage.tsx`

**Step 1: Rewrite ReportsPage with tabs**

Add imports:
```tsx
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  useListTemplatesQuery,
  useCreateTemplateMutation,
  useUpdateTemplateMutation,
  useDeleteTemplateMutation
} from '@/infrastructure/services/report.service'
import { FileCode, Edit, Trash2 as Trash } from 'lucide-react'
```

Wrap existing content in a `<TabsContent value="reports">` and add a new `<TabsContent value="templates">` with:

Templates tab UI:
- List all templates from `useListTemplatesQuery()`
- "Create Template" button that opens a form with name + content (textarea) fields
- Each template row has Edit and Delete buttons
- Edit opens inline form with name + content
- Delete with confirmation
- Uses `useCreateTemplateMutation`, `useUpdateTemplateMutation`, `useDeleteTemplateMutation`

Add template state management:
```tsx
const { data: templates = [] } = useListTemplatesQuery()
const [createTemplate] = useCreateTemplateMutation()
const [updateTemplate] = useUpdateTemplateMutation()
const [deleteTemplate] = useDeleteTemplateMutation()
const [showTemplateForm, setShowTemplateForm] = useState(false)
const [editingTemplate, setEditingTemplate] = useState<{ id: number; name: string; content: string } | null>(null)
const [templateForm, setTemplateForm] = useState({ name: '', content: '' })
```

Also add a report content expand/collapse to each report row — clicking a report shows its content inline below the row.

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && bun run build`

**Step 3: Commit**

```bash
git add frontend/src/presentation/pages/ReportsPage.tsx
git commit -m "feat: add report templates tab with CRUD and report content viewer"
```

---

### Task 11: Refine CommitsPage — clickable Jira keys, visual distinction

**Files:**
- Modify: `frontend/src/presentation/pages/CommitsPage.tsx`

**Step 1: Update CommitsPage**

Add `Link` import:
```tsx
import { Link } from 'react-router-dom'
```

Make Jira card keys clickable — in the Jira column:
```tsx
{commit.jira_card_key ? (
  <Link to={`/dashboard/jira/cards/${commit.jira_card_key}`}>
    <StatusBadge variant="warning" className="cursor-pointer hover:opacity-80">
      {commit.jira_card_key}
    </StatusBadge>
  </Link>
) : (
  <span className="text-sm text-pdt-neutral/40">-</span>
)}
```

Add visual distinction for linked vs unlinked rows — add a left border accent:
```tsx
<tr
  key={commit.id}
  className={cn(
    'border-t border-pdt-neutral/10',
    commit.jira_card_key
      ? 'border-l-2 border-l-pdt-accent'
      : 'border-l-2 border-l-transparent'
  )}
>
```

Add `cn` import if not already present:
```tsx
import { cn } from '@/lib/utils'
```

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && bun run build`

**Step 3: Commit**

```bash
git add frontend/src/presentation/pages/CommitsPage.tsx
git commit -m "feat: add clickable Jira keys and visual distinction for linked commits"
```

---

### Task 12: Refine SettingsPage — test connection buttons, validation indicators

**Files:**
- Modify: `frontend/src/presentation/pages/SettingsPage.tsx`

**Step 1: Update SettingsPage**

Add per-integration "Test Connection" buttons by splitting the integrations into separate `<DataCard>` sections for GitHub, GitLab, and Jira. Each gets its own Save and Test button.

Use the existing `useValidateIntegrationsMutation` for the test button — it validates all integrations at once. After validation, show the result using colored indicators (green check / red x) next to each integration section header.

Replace `alert()` calls with inline success/error messages using state:
```tsx
const [saveMessage, setSaveMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)
const [validateResult, setValidateResult] = useState<{ type: 'success' | 'error'; text: string } | null>(null)
```

**Step 2: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && bun run build`

**Step 3: Commit**

```bash
git add frontend/src/presentation/pages/SettingsPage.tsx
git commit -m "feat: refine settings page with test connection and inline feedback"
```

---

### Task 13: Add industrial design flourishes — hazard stripes, refined DataCard

**Files:**
- Modify: `frontend/src/index.css` (add hazard stripe utility)
- Modify: `frontend/src/presentation/components/common/DataCard.tsx` (add top accent stripe)
- Modify: `frontend/src/presentation/components/dashboard/Sidebar/SidebarNavItem.tsx` (yellow active indicator)

**Step 1: Add hazard stripe CSS utility in `frontend/src/index.css`**

Add at the end of the file:
```css
@layer utilities {
  .hazard-stripe {
    background: repeating-linear-gradient(
      -45deg,
      #F8C630,
      #F8C630 10px,
      #1B1B1E 10px,
      #1B1B1E 20px
    );
  }
  .hazard-stripe-subtle {
    background: repeating-linear-gradient(
      -45deg,
      #F8C63015,
      #F8C63015 10px,
      transparent 10px,
      transparent 20px
    );
  }
}
```

**Step 2: Add a yellow top accent line to DataCard in `frontend/src/presentation/components/common/DataCard.tsx`**

Read the current DataCard and add a thin yellow top border:
```tsx
<div className={cn('rounded-lg border border-pdt-neutral/10 bg-pdt-primary-light p-4 border-t-2 border-t-pdt-accent', className)}>
```

**Step 3: Update SidebarNavItem active state**

Add a yellow left border to the active nav item. Find the active state styles and add:
```tsx
className={cn(
  // existing classes,
  isActive && 'border-l-2 border-l-pdt-accent bg-pdt-accent/10 text-pdt-accent'
)}
```

**Step 4: Verify build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && bun run build`

**Step 5: Commit**

```bash
git add frontend/src/index.css frontend/src/presentation/components/common/DataCard.tsx frontend/src/presentation/components/dashboard/Sidebar/SidebarNavItem.tsx
git commit -m "feat: add industrial design flourishes — hazard stripes, accent borders"
```

---

### Task 14: Final build verification and cleanup

**Files:**
- All modified files

**Step 1: Run full typecheck**

Run: `cd /home/nst/GolandProjects/pdt/frontend && bun run typecheck`

Expected: No type errors

**Step 2: Run linter**

Run: `cd /home/nst/GolandProjects/pdt/frontend && bun run lint`

Expected: No lint errors (fix any that appear)

**Step 3: Run build**

Run: `cd /home/nst/GolandProjects/pdt/frontend && bun run build`

Expected: Build succeeds

**Step 4: Run tests**

Run: `cd /home/nst/GolandProjects/pdt/frontend && bun run test`

Expected: All tests pass

**Step 5: Fix any issues found in steps 1-4**

Address any type errors, lint issues, or test failures.

**Step 6: Commit any fixes**

```bash
git add -A
git commit -m "fix: address build/lint/test issues from dashboard overhaul"
```
