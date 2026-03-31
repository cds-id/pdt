# Agent Scheduling Frontend Implementation Plan (Plan 3 of 3)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the frontend schedule management page with CRUD forms, run history, and navigation integration.

**Architecture:** RTK Query service for API calls, React page component with card layout for schedules, inline create/edit forms, expandable run history. Follows existing project patterns (shadcn/ui, Tailwind, Redux Toolkit).

**Tech Stack:** React 18, TypeScript, RTK Query, shadcn/ui, Tailwind CSS, Lucide icons, React Router v7

**Spec:** `docs/superpowers/specs/2026-03-31-agent-scheduling-design.md`

**Depends on:** Plan 2 (REST API) — already implemented.

---

## File Structure

```
frontend/src/
├── infrastructure/
│   ├── constants/
│   │   └── api.constants.ts       # Modify: add SCHEDULES endpoints
│   └── services/
│       └── schedule.service.ts    # New: RTK Query service
├── config/
│   └── navigation.ts             # Modify: add Schedules nav item
├── presentation/
│   ├── pages/
│   │   └── SchedulesPage.tsx     # New: main schedules page
│   └── routes/
│       └── index.tsx             # Modify: add /dashboard/schedules route
```

---

### Task 1: API constants and RTK Query service

**Files:**
- Modify: `frontend/src/infrastructure/constants/api.constants.ts`
- Create: `frontend/src/infrastructure/services/schedule.service.ts`

- [ ] **Step 1: Add schedule endpoints to API constants**

In `frontend/src/infrastructure/constants/api.constants.ts`, add a `SCHEDULES` section to the `API_CONSTANTS` object:

```typescript
SCHEDULES: {
  LIST: '/schedules',
  CREATE: '/schedules',
  UPDATE: (id: string) => `/schedules/${id}`,
  DELETE: (id: string) => `/schedules/${id}`,
  TOGGLE: (id: string) => `/schedules/${id}/toggle`,
  RUN: (id: string) => `/schedules/${id}/run`,
  RUNS: (id: string) => `/schedules/${id}/runs`,
  GET_RUN: (runId: string) => `/schedules/runs/${runId}`,
},
```

- [ ] **Step 2: Add 'Schedule' to tagTypes in api.ts**

In `frontend/src/infrastructure/services/api.ts`, add `'Schedule'` to the `tagTypes` array.

- [ ] **Step 3: Create the schedule service**

```typescript
// frontend/src/infrastructure/services/schedule.service.ts
import { api } from './api'
import { API_CONSTANTS } from '../constants/api.constants'

export interface AgentSchedule {
  id: string
  user_id: number
  name: string
  agent_name: string
  prompt: string
  trigger_type: 'cron' | 'interval' | 'event'
  cron_expr: string
  interval_seconds: number
  event_name: string
  chain_config: ChainStep[] | null
  enabled: boolean
  next_run_at: string | null
  created_at: string
  updated_at: string
}

export interface ChainStep {
  agent: string
  prompt: string
  condition: string
}

export interface AgentScheduleRun {
  id: string
  schedule_id: string
  user_id: number
  conversation_id: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  trigger_type: string
  started_at: string | null
  completed_at: string | null
  result_summary: string
  error: string
  token_usage: string
  created_at: string
}

export interface AgentScheduleRunStep {
  id: string
  run_id: string
  agent_name: string
  prompt: string
  response: string
  status: 'completed' | 'failed'
  duration_ms: number
  created_at: string
}

export interface RunDetail {
  run: AgentScheduleRun
  steps: AgentScheduleRunStep[]
}

export interface CreateScheduleRequest {
  name: string
  agent_name: string
  prompt: string
  trigger_type: 'cron' | 'interval' | 'event'
  cron_expr?: string
  interval_seconds?: number
  event_name?: string
  chain_config?: ChainStep[]
  enabled?: boolean
}

export const scheduleApi = api.injectEndpoints({
  endpoints: (builder) => ({
    listSchedules: builder.query<AgentSchedule[], void>({
      query: () => API_CONSTANTS.SCHEDULES.LIST,
      providesTags: (result) =>
        result
          ? [...result.map(({ id }) => ({ type: 'Schedule' as const, id })), { type: 'Schedule', id: 'LIST' }]
          : [{ type: 'Schedule', id: 'LIST' }],
    }),

    createSchedule: builder.mutation<AgentSchedule, CreateScheduleRequest>({
      query: (body) => ({
        url: API_CONSTANTS.SCHEDULES.CREATE,
        method: 'POST',
        body,
      }),
      invalidatesTags: [{ type: 'Schedule', id: 'LIST' }],
    }),

    updateSchedule: builder.mutation<AgentSchedule, { id: string } & Partial<CreateScheduleRequest>>({
      query: ({ id, ...body }) => ({
        url: API_CONSTANTS.SCHEDULES.UPDATE(id),
        method: 'PUT',
        body,
      }),
      invalidatesTags: (_, __, { id }) => [{ type: 'Schedule', id }, { type: 'Schedule', id: 'LIST' }],
    }),

    deleteSchedule: builder.mutation<void, string>({
      query: (id) => ({
        url: API_CONSTANTS.SCHEDULES.DELETE(id),
        method: 'DELETE',
      }),
      invalidatesTags: [{ type: 'Schedule', id: 'LIST' }],
    }),

    toggleSchedule: builder.mutation<{ enabled: boolean }, string>({
      query: (id) => ({
        url: API_CONSTANTS.SCHEDULES.TOGGLE(id),
        method: 'POST',
      }),
      invalidatesTags: (_, __, id) => [{ type: 'Schedule', id }, { type: 'Schedule', id: 'LIST' }],
    }),

    runScheduleNow: builder.mutation<{ message: string }, string>({
      query: (id) => ({
        url: API_CONSTANTS.SCHEDULES.RUN(id),
        method: 'POST',
      }),
    }),

    listScheduleRuns: builder.query<AgentScheduleRun[], string>({
      query: (scheduleId) => API_CONSTANTS.SCHEDULES.RUNS(scheduleId),
    }),

    getScheduleRun: builder.query<RunDetail, string>({
      query: (runId) => API_CONSTANTS.SCHEDULES.GET_RUN(runId),
    }),
  }),
})

export const {
  useListSchedulesQuery,
  useCreateScheduleMutation,
  useUpdateScheduleMutation,
  useDeleteScheduleMutation,
  useToggleScheduleMutation,
  useRunScheduleNowMutation,
  useListScheduleRunsQuery,
  useGetScheduleRunQuery,
} = scheduleApi
```

- [ ] **Step 4: Verify build**

```bash
cd frontend && npm run build 2>&1 | tail -5
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/infrastructure/
git commit -m "feat(frontend): add schedule API service with RTK Query"
```

---

### Task 2: Navigation and routing

**Files:**
- Modify: `frontend/src/config/navigation.ts`
- Modify: `frontend/src/presentation/routes/index.tsx`

- [ ] **Step 1: Add nav item**

In `frontend/src/config/navigation.ts`:
- Add import: `import { Calendar } from 'lucide-react'` (or `Clock` — check what's available)
- Add to the "AI" navigation group (where AI Assistant and AI Usage are):

```typescript
{ title: 'Schedules', href: '/dashboard/schedules', icon: Calendar },
```

- [ ] **Step 2: Add route**

In `frontend/src/presentation/routes/index.tsx`:
- Add import: `import { SchedulesPage } from '../pages/SchedulesPage'`
- Add route in the dashboard children (lazy import or direct):

```typescript
{ path: 'dashboard/schedules', element: <SchedulesPage /> },
```

- [ ] **Step 3: Create placeholder page**

Create `frontend/src/presentation/pages/SchedulesPage.tsx`:

```tsx
export function SchedulesPage() {
  return <div>Schedules page placeholder</div>
}
```

- [ ] **Step 4: Verify build**

```bash
cd frontend && npm run build 2>&1 | tail -5
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/config/navigation.ts frontend/src/presentation/routes/index.tsx frontend/src/presentation/pages/SchedulesPage.tsx
git commit -m "feat(frontend): add schedules navigation and route"
```

---

### Task 3: Schedules page — list view with cards

**Files:**
- Modify: `frontend/src/presentation/pages/SchedulesPage.tsx`

- [ ] **Step 1: Implement the full schedules page**

Replace the placeholder with the full page. This is the largest file — it includes:
- Schedule list as cards
- Create/edit form (inline)
- Toggle, delete, run now actions
- Expandable run history per schedule

```tsx
// frontend/src/presentation/pages/SchedulesPage.tsx
import { useState } from 'react'
import { PageHeader } from '../components/common/PageHeader'
import { DataCard } from '../components/common/DataCard'
import { StatusBadge } from '../components/common/StatusBadge'
import { EmptyState } from '../components/common/EmptyState'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import {
  useListSchedulesQuery,
  useCreateScheduleMutation,
  useUpdateScheduleMutation,
  useDeleteScheduleMutation,
  useToggleScheduleMutation,
  useRunScheduleNowMutation,
  useListScheduleRunsQuery,
  type AgentSchedule,
  type CreateScheduleRequest,
} from '@/infrastructure/services/schedule.service'
import {
  Calendar,
  Plus,
  Trash2,
  Play,
  Pencil,
  X,
  Clock,
  Zap,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'

const AGENTS = [
  { value: '', label: 'Auto (Orchestrator)' },
  { value: 'git', label: 'Git' },
  { value: 'jira', label: 'Jira' },
  { value: 'report', label: 'Report' },
  { value: 'proof', label: 'Proof' },
  { value: 'briefing', label: 'Briefing' },
  { value: 'whatsapp', label: 'WhatsApp' },
]

const TRIGGER_TYPES = [
  { value: 'cron', label: 'Cron', icon: Calendar },
  { value: 'interval', label: 'Interval', icon: Clock },
  { value: 'event', label: 'Event', icon: Zap },
]

const CRON_PRESETS = [
  { label: 'Every weekday 8am', value: '0 8 * * 1-5' },
  { label: 'Every Monday 9am', value: '0 9 * * 1' },
  { label: 'Every hour', value: '0 * * * *' },
  { label: 'Every day at midnight', value: '0 0 * * *' },
]

const EVENTS = [
  { value: 'commit_synced', label: 'Commit Synced' },
  { value: 'jira_synced', label: 'Jira Synced' },
  { value: 'report_generated', label: 'Report Generated' },
  { value: 'schedule_completed', label: 'Schedule Completed' },
]

function triggerBadge(type: string) {
  const colors: Record<string, string> = {
    cron: 'bg-blue-500/20 text-blue-400',
    interval: 'bg-green-500/20 text-green-400',
    event: 'bg-purple-500/20 text-purple-400',
  }
  return <span className={`px-2 py-0.5 rounded text-xs font-medium ${colors[type] || ''}`}>{type}</span>
}

function formatNextRun(nextRunAt: string | null) {
  if (!nextRunAt) return 'Event-triggered'
  const d = new Date(nextRunAt)
  const now = new Date()
  const diffMs = d.getTime() - now.getTime()
  if (diffMs < 0) return 'Overdue'
  if (diffMs < 60000) return 'Less than a minute'
  if (diffMs < 3600000) return `${Math.round(diffMs / 60000)}m`
  if (diffMs < 86400000) return `${Math.round(diffMs / 3600000)}h`
  return d.toLocaleDateString()
}

const emptyForm: CreateScheduleRequest = {
  name: '',
  agent_name: '',
  prompt: '',
  trigger_type: 'cron',
  cron_expr: '0 8 * * 1-5',
  interval_seconds: 900,
  event_name: 'commit_synced',
}

export function SchedulesPage() {
  const { data: schedules = [], isLoading } = useListSchedulesQuery()
  const [createSchedule, { isLoading: isCreating }] = useCreateScheduleMutation()
  const [updateSchedule] = useUpdateScheduleMutation()
  const [deleteSchedule] = useDeleteScheduleMutation()
  const [toggleSchedule] = useToggleScheduleMutation()
  const [runNow] = useRunScheduleNowMutation()

  const [showForm, setShowForm] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [form, setForm] = useState<CreateScheduleRequest>(emptyForm)
  const [expandedRuns, setExpandedRuns] = useState<string | null>(null)

  const handleCreate = async () => {
    try {
      await createSchedule(form).unwrap()
      setShowForm(false)
      setForm(emptyForm)
    } catch (err) {
      console.error('Failed to create schedule:', err)
    }
  }

  const handleUpdate = async () => {
    if (!editingId) return
    try {
      await updateSchedule({ id: editingId, ...form }).unwrap()
      setEditingId(null)
      setForm(emptyForm)
    } catch (err) {
      console.error('Failed to update schedule:', err)
    }
  }

  const startEdit = (s: AgentSchedule) => {
    setEditingId(s.id)
    setShowForm(false)
    setForm({
      name: s.name,
      agent_name: s.agent_name,
      prompt: s.prompt,
      trigger_type: s.trigger_type,
      cron_expr: s.cron_expr,
      interval_seconds: s.interval_seconds,
      event_name: s.event_name,
      chain_config: s.chain_config || undefined,
    })
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this schedule and all its run history?')) return
    try {
      await deleteSchedule(id).unwrap()
    } catch (err) {
      console.error('Failed to delete:', err)
    }
  }

  const renderForm = (onSubmit: () => void, submitLabel: string) => (
    <div className="space-y-4 p-4 border border-pdt-accent/20 rounded-lg bg-pdt-primary/50">
      <div className="grid grid-cols-2 gap-4">
        <div>
          <Label>Name</Label>
          <Input
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            placeholder="Morning briefing"
            className="mt-1"
          />
        </div>
        <div>
          <Label>Agent</Label>
          <select
            value={form.agent_name}
            onChange={(e) => setForm({ ...form, agent_name: e.target.value })}
            className="mt-1 w-full rounded-md border border-pdt-accent/20 bg-pdt-primary-light px-3 py-2 text-sm text-pdt-neutral"
          >
            {AGENTS.map((a) => (
              <option key={a.value} value={a.value}>{a.label}</option>
            ))}
          </select>
        </div>
      </div>

      <div>
        <Label>Prompt</Label>
        <textarea
          value={form.prompt}
          onChange={(e) => setForm({ ...form, prompt: e.target.value })}
          placeholder="Generate a morning briefing with blockers and action items"
          rows={3}
          className="mt-1 w-full rounded-md border border-pdt-accent/20 bg-pdt-primary-light px-3 py-2 text-sm text-pdt-neutral resize-none"
        />
      </div>

      <div>
        <Label>Trigger Type</Label>
        <div className="mt-1 flex gap-2">
          {TRIGGER_TYPES.map((t) => (
            <button
              key={t.value}
              onClick={() => setForm({ ...form, trigger_type: t.value as CreateScheduleRequest['trigger_type'] })}
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm border ${
                form.trigger_type === t.value
                  ? 'border-pdt-accent bg-pdt-accent/10 text-pdt-accent'
                  : 'border-pdt-accent/20 text-pdt-neutral/60 hover:border-pdt-accent/40'
              }`}
            >
              <t.icon size={14} />
              {t.label}
            </button>
          ))}
        </div>
      </div>

      {form.trigger_type === 'cron' && (
        <div>
          <Label>Cron Expression</Label>
          <Input
            value={form.cron_expr}
            onChange={(e) => setForm({ ...form, cron_expr: e.target.value })}
            placeholder="0 8 * * 1-5"
            className="mt-1"
          />
          <div className="mt-1 flex gap-2 flex-wrap">
            {CRON_PRESETS.map((p) => (
              <button
                key={p.value}
                onClick={() => setForm({ ...form, cron_expr: p.value })}
                className="text-xs px-2 py-1 rounded bg-pdt-primary-light border border-pdt-accent/10 text-pdt-neutral/60 hover:text-pdt-accent"
              >
                {p.label}
              </button>
            ))}
          </div>
        </div>
      )}

      {form.trigger_type === 'interval' && (
        <div>
          <Label>Interval (minutes)</Label>
          <Input
            type="number"
            value={Math.round((form.interval_seconds || 900) / 60)}
            onChange={(e) => setForm({ ...form, interval_seconds: parseInt(e.target.value) * 60 })}
            min={1}
            className="mt-1 w-32"
          />
        </div>
      )}

      {form.trigger_type === 'event' && (
        <div>
          <Label>Event</Label>
          <select
            value={form.event_name}
            onChange={(e) => setForm({ ...form, event_name: e.target.value })}
            className="mt-1 w-full rounded-md border border-pdt-accent/20 bg-pdt-primary-light px-3 py-2 text-sm text-pdt-neutral"
          >
            {EVENTS.map((e) => (
              <option key={e.value} value={e.value}>{e.label}</option>
            ))}
          </select>
        </div>
      )}

      <div className="flex gap-2 justify-end">
        <Button
          variant="outline"
          size="sm"
          onClick={() => { setShowForm(false); setEditingId(null); setForm(emptyForm) }}
        >
          <X size={14} className="mr-1" /> Cancel
        </Button>
        <Button size="sm" onClick={onSubmit} disabled={isCreating || !form.name || !form.prompt}>
          {submitLabel}
        </Button>
      </div>
    </div>
  )

  if (isLoading) {
    return (
      <div className="space-y-4">
        <PageHeader title="Schedules" description="Manage scheduled agent tasks" />
        <div className="text-pdt-neutral/40">Loading...</div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <PageHeader
        title="Schedules"
        description="Manage scheduled agent tasks"
        action={
          !showForm && !editingId ? (
            <Button size="sm" onClick={() => setShowForm(true)}>
              <Plus size={14} className="mr-1" /> Create Schedule
            </Button>
          ) : undefined
        }
      />

      {showForm && renderForm(handleCreate, isCreating ? 'Creating...' : 'Create Schedule')}

      {schedules.length === 0 && !showForm ? (
        <EmptyState
          title="No schedules yet"
          description="Create a schedule to run agents automatically on a timer, interval, or event trigger."
        />
      ) : (
        <div className="space-y-3">
          {schedules.map((s) => (
            <div key={s.id}>
              {editingId === s.id ? (
                renderForm(handleUpdate, 'Save Changes')
              ) : (
                <DataCard
                  title={s.name}
                  action={
                    <div className="flex items-center gap-2">
                      <Switch
                        checked={s.enabled}
                        onCheckedChange={() => toggleSchedule(s.id)}
                      />
                      <Button variant="ghost" size="sm" onClick={() => runNow(s.id)} title="Run now">
                        <Play size={14} />
                      </Button>
                      <Button variant="ghost" size="sm" onClick={() => startEdit(s)}>
                        <Pencil size={14} />
                      </Button>
                      <Button variant="ghost" size="sm" onClick={() => handleDelete(s.id)}>
                        <Trash2 size={14} className="text-red-400" />
                      </Button>
                    </div>
                  }
                >
                  <div className="space-y-2">
                    <div className="flex items-center gap-2 flex-wrap">
                      {triggerBadge(s.trigger_type)}
                      <Badge variant="outline" className="text-xs">
                        {s.agent_name || 'orchestrator'}
                      </Badge>
                      {s.enabled ? (
                        <span className="text-xs text-pdt-neutral/40">
                          Next: {formatNextRun(s.next_run_at)}
                        </span>
                      ) : (
                        <span className="text-xs text-red-400">Disabled</span>
                      )}
                    </div>
                    <p className="text-sm text-pdt-neutral/60 line-clamp-2">{s.prompt}</p>

                    <button
                      onClick={() => setExpandedRuns(expandedRuns === s.id ? null : s.id)}
                      className="flex items-center gap-1 text-xs text-pdt-accent hover:text-pdt-accent-hover"
                    >
                      {expandedRuns === s.id ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
                      Run History
                    </button>

                    {expandedRuns === s.id && <RunHistory scheduleId={s.id} />}
                  </div>
                </DataCard>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function RunHistory({ scheduleId }: { scheduleId: string }) {
  const { data: runs = [], isLoading } = useListScheduleRunsQuery(scheduleId)

  if (isLoading) return <div className="text-xs text-pdt-neutral/40 py-2">Loading runs...</div>
  if (runs.length === 0) return <div className="text-xs text-pdt-neutral/40 py-2">No runs yet</div>

  return (
    <div className="border border-pdt-accent/10 rounded overflow-hidden">
      <table className="w-full text-xs">
        <thead>
          <tr className="bg-pdt-primary-light/50 text-pdt-neutral/40">
            <th className="px-3 py-1.5 text-left">Time</th>
            <th className="px-3 py-1.5 text-left">Trigger</th>
            <th className="px-3 py-1.5 text-left">Status</th>
            <th className="px-3 py-1.5 text-left">Duration</th>
            <th className="px-3 py-1.5 text-left">Summary</th>
          </tr>
        </thead>
        <tbody>
          {runs.slice(0, 10).map((run) => (
            <tr key={run.id} className="border-t border-pdt-accent/5">
              <td className="px-3 py-1.5 text-pdt-neutral/60">
                {run.started_at ? new Date(run.started_at).toLocaleString() : '—'}
              </td>
              <td className="px-3 py-1.5">{triggerBadge(run.trigger_type)}</td>
              <td className="px-3 py-1.5">
                <StatusBadge
                  status={run.status === 'completed' ? 'success' : run.status === 'failed' ? 'danger' : 'info'}
                  label={run.status}
                />
              </td>
              <td className="px-3 py-1.5 text-pdt-neutral/60">
                {run.started_at && run.completed_at
                  ? `${Math.round((new Date(run.completed_at).getTime() - new Date(run.started_at).getTime()) / 1000)}s`
                  : '—'}
              </td>
              <td className="px-3 py-1.5 text-pdt-neutral/60 max-w-xs truncate">
                {run.error || run.result_summary || '—'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
```

- [ ] **Step 2: Verify build**

```bash
cd frontend && npm run build 2>&1 | tail -10
```

Fix any TypeScript errors. Common issues:
- Import paths may need `@/` prefix adjustments
- `StatusBadge` props might differ from expected — check the actual component

- [ ] **Step 3: Commit**

```bash
git add frontend/src/presentation/pages/SchedulesPage.tsx
git commit -m "feat(frontend): add schedules page with CRUD, toggle, run now, and history"
```

---

### Task 4: Build verification

**Files:** None (verification only)

- [ ] **Step 1: Full build**

```bash
cd frontend && npm run build
```

Expected: builds successfully.

- [ ] **Step 2: Check for TypeScript errors**

```bash
cd frontend && npx tsc --noEmit 2>&1 | tail -20
```

Expected: no errors.
