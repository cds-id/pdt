import { useState } from 'react'
import {
  Calendar,
  Plus,
  Play,
  Trash2,
  ChevronDown,
  ChevronUp,
  Clock,
  Zap,
  Radio,
  RefreshCw,
} from 'lucide-react'
import {
  useListSchedulesQuery,
  useCreateScheduleMutation,
  useDeleteScheduleMutation,
  useToggleScheduleMutation,
  useRunScheduleNowMutation,
  useListScheduleRunsQuery,
  type AgentSchedule,
  type CreateScheduleRequest,
} from '@/infrastructure/services/schedule.service'
import { PageHeader, DataCard, EmptyState, StatusBadge } from '@/presentation/components/common'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

// --- Helpers ---

function formatDate(s: string | null): string {
  if (!s) return '—'
  return new Date(s).toLocaleString()
}

function triggerLabel(s: AgentSchedule): string {
  if (s.trigger_type === 'cron') return s.cron_expr || 'cron'
  if (s.trigger_type === 'interval') return `every ${s.interval_seconds}s`
  return s.event_name || 'event'
}

function triggerIcon(type: AgentSchedule['trigger_type']) {
  if (type === 'cron') return <Clock className="size-3" />
  if (type === 'interval') return <RefreshCw className="size-3" />
  return <Radio className="size-3" />
}

type RunStatus = 'pending' | 'running' | 'completed' | 'failed'

function runStatusVariant(status: RunStatus): 'info' | 'warning' | 'success' | 'danger' {
  switch (status) {
    case 'pending': return 'info'
    case 'running': return 'warning'
    case 'completed': return 'success'
    case 'failed': return 'danger'
  }
}

// --- Run history sub-panel ---

function RunHistory({ scheduleId }: { scheduleId: string }) {
  const { data: runs, isLoading } = useListScheduleRunsQuery(scheduleId)

  if (isLoading) {
    return <p className="text-xs text-pdt-neutral/40 py-2">Loading runs…</p>
  }

  if (!runs?.length) {
    return <p className="text-xs text-pdt-neutral/40 py-2">No runs yet.</p>
  }

  return (
    <div className="mt-3 space-y-1.5">
      {runs.map((run) => (
        <div
          key={run.id}
          className="flex items-center justify-between rounded border border-pdt-accent/10 bg-pdt-primary/40 px-3 py-2 text-xs"
        >
          <div className="flex items-center gap-2">
            <StatusBadge variant={runStatusVariant(run.status as RunStatus)}>
              {run.status}
            </StatusBadge>
            <span className="text-pdt-neutral/60">{formatDate(run.created_at)}</span>
          </div>
          {run.result_summary && (
            <span className="max-w-xs truncate text-pdt-neutral/50">{run.result_summary}</span>
          )}
          {run.error && (
            <span className="max-w-xs truncate text-red-400">{run.error}</span>
          )}
        </div>
      ))}
    </div>
  )
}

// --- Schedule row ---

function ScheduleRow({ schedule }: { schedule: AgentSchedule }) {
  const [expanded, setExpanded] = useState(false)
  const [toggle] = useToggleScheduleMutation()
  const [runNow, { isLoading: isRunning }] = useRunScheduleNowMutation()
  const [deleteSchedule, { isLoading: isDeleting }] = useDeleteScheduleMutation()

  return (
    <div className="rounded-lg border border-pdt-accent/20 bg-pdt-primary-light">
      <div className="flex items-center gap-3 p-4">
        {/* Toggle */}
        <Switch
          checked={schedule.enabled}
          onCheckedChange={() => toggle(schedule.id)}
        />

        {/* Info */}
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-medium text-pdt-neutral">{schedule.name}</span>
            <StatusBadge variant={schedule.enabled ? 'success' : 'neutral'}>
              {schedule.enabled ? 'enabled' : 'disabled'}
            </StatusBadge>
            <span className="flex items-center gap-1 rounded bg-pdt-accent/10 px-1.5 py-0.5 text-xs text-pdt-accent">
              {triggerIcon(schedule.trigger_type)}
              {triggerLabel(schedule)}
            </span>
          </div>
          <p className="mt-0.5 text-xs text-pdt-neutral/50">
            Agent: <span className="text-pdt-neutral/70">{schedule.agent_name}</span>
            {schedule.next_run_at && (
              <> &middot; Next: {formatDate(schedule.next_run_at)}</>
            )}
          </p>
        </div>

        {/* Actions */}
        <div className="flex shrink-0 items-center gap-1">
          <Button
            size="sm"
            variant="ghost"
            disabled={isRunning}
            onClick={() => runNow(schedule.id)}
            title="Run now"
          >
            <Play className="size-3.5" />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            disabled={isDeleting}
            onClick={() => deleteSchedule(schedule.id)}
            title="Delete"
            className="text-red-400 hover:text-red-300"
          >
            <Trash2 className="size-3.5" />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => setExpanded((v) => !v)}
            title={expanded ? 'Hide runs' : 'Show runs'}
          >
            {expanded ? <ChevronUp className="size-3.5" /> : <ChevronDown className="size-3.5" />}
          </Button>
        </div>
      </div>

      {expanded && (
        <div className="border-t border-pdt-accent/10 px-4 pb-4">
          <p className="mt-3 text-xs text-pdt-neutral/40 uppercase tracking-wide">Prompt</p>
          <p className="mt-1 text-sm text-pdt-neutral/70 line-clamp-3">{schedule.prompt}</p>
          <p className="mt-3 text-xs text-pdt-neutral/40 uppercase tracking-wide">Run History</p>
          <RunHistory scheduleId={schedule.id} />
        </div>
      )}
    </div>
  )
}

// --- Create dialog ---

const DEFAULT_FORM: CreateScheduleRequest = {
  name: '',
  agent_name: '',
  prompt: '',
  trigger_type: 'cron',
  cron_expr: '0 9 * * *',
  interval_seconds: 3600,
  event_name: '',
  enabled: true,
}

function CreateScheduleDialog({
  open,
  onClose,
}: {
  open: boolean
  onClose: () => void
}) {
  const [form, setForm] = useState<CreateScheduleRequest>(DEFAULT_FORM)
  const [createSchedule, { isLoading }] = useCreateScheduleMutation()

  function set<K extends keyof CreateScheduleRequest>(key: K, value: CreateScheduleRequest[K]) {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  async function handleSubmit() {
    if (!form.name || !form.agent_name || !form.prompt) return
    await createSchedule(form)
    setForm(DEFAULT_FORM)
    onClose()
  }

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>New Schedule</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="sched-name">Name</Label>
              <Input
                id="sched-name"
                placeholder="Daily standup"
                value={form.name}
                onChange={(e) => set('name', e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="sched-agent">Agent</Label>
              <Input
                id="sched-agent"
                placeholder="orchestrator"
                value={form.agent_name}
                onChange={(e) => set('agent_name', e.target.value)}
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="sched-prompt">Prompt</Label>
            <Textarea
              id="sched-prompt"
              placeholder="What should the agent do?"
              rows={3}
              value={form.prompt}
              onChange={(e) => set('prompt', e.target.value)}
            />
          </div>

          <div className="space-y-1.5">
            <Label>Trigger type</Label>
            <Select
              value={form.trigger_type}
              onValueChange={(v) => set('trigger_type', v as CreateScheduleRequest['trigger_type'])}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="cron">
                  <span className="flex items-center gap-2"><Clock className="size-3.5" /> Cron</span>
                </SelectItem>
                <SelectItem value="interval">
                  <span className="flex items-center gap-2"><RefreshCw className="size-3.5" /> Interval</span>
                </SelectItem>
                <SelectItem value="event">
                  <span className="flex items-center gap-2"><Radio className="size-3.5" /> Event</span>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          {form.trigger_type === 'cron' && (
            <div className="space-y-1.5">
              <Label htmlFor="sched-cron">Cron expression</Label>
              <Input
                id="sched-cron"
                placeholder="0 9 * * *"
                value={form.cron_expr}
                onChange={(e) => set('cron_expr', e.target.value)}
              />
              <p className="text-xs text-pdt-neutral/40">Standard 5-field cron (e.g. "0 9 * * 1-5" = weekdays at 9am)</p>
            </div>
          )}

          {form.trigger_type === 'interval' && (
            <div className="space-y-1.5">
              <Label htmlFor="sched-interval">Interval (seconds)</Label>
              <Input
                id="sched-interval"
                type="number"
                min={60}
                value={form.interval_seconds}
                onChange={(e) => set('interval_seconds', Number(e.target.value))}
              />
            </div>
          )}

          {form.trigger_type === 'event' && (
            <div className="space-y-1.5">
              <Label htmlFor="sched-event">Event name</Label>
              <Input
                id="sched-event"
                placeholder="commit.pushed"
                value={form.event_name}
                onChange={(e) => set('event_name', e.target.value)}
              />
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="ghost" onClick={onClose}>Cancel</Button>
          <Button onClick={handleSubmit} disabled={isLoading || !form.name || !form.agent_name || !form.prompt}>
            {isLoading ? 'Creating…' : 'Create schedule'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// --- Page ---

export function SchedulesPage() {
  const { data: schedules, isLoading } = useListSchedulesQuery()
  const [dialogOpen, setDialogOpen] = useState(false)

  const enabled = schedules?.filter((s) => s.enabled) ?? []
  const disabled = schedules?.filter((s) => !s.enabled) ?? []

  return (
    <div className="min-w-0 space-y-4 md:space-y-6">
      <PageHeader
        title="Schedules"
        description="Automate agents with cron, interval, or event triggers"
        action={
          <Button onClick={() => setDialogOpen(true)}>
            <Plus className="mr-1.5 size-4" />
            New Schedule
          </Button>
        }
      />

      {/* Stats row */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <DataCard>
          <div className="flex items-center gap-3">
            <div className="flex size-9 items-center justify-center rounded-lg bg-pdt-accent/10">
              <Calendar className="size-5 text-pdt-accent" />
            </div>
            <div>
              <p className="text-2xl font-bold text-pdt-neutral">{schedules?.length ?? 0}</p>
              <p className="text-xs text-pdt-neutral/50">Total</p>
            </div>
          </div>
        </DataCard>
        <DataCard>
          <div className="flex items-center gap-3">
            <div className="flex size-9 items-center justify-center rounded-lg bg-green-500/10">
              <Zap className="size-5 text-green-400" />
            </div>
            <div>
              <p className="text-2xl font-bold text-pdt-neutral">{enabled.length}</p>
              <p className="text-xs text-pdt-neutral/50">Active</p>
            </div>
          </div>
        </DataCard>
        <DataCard>
          <div className="flex items-center gap-3">
            <div className="flex size-9 items-center justify-center rounded-lg bg-blue-500/10">
              <Clock className="size-5 text-blue-400" />
            </div>
            <div>
              <p className="text-2xl font-bold text-pdt-neutral">
                {schedules?.filter((s) => s.trigger_type === 'cron').length ?? 0}
              </p>
              <p className="text-xs text-pdt-neutral/50">Cron</p>
            </div>
          </div>
        </DataCard>
        <DataCard>
          <div className="flex items-center gap-3">
            <div className="flex size-9 items-center justify-center rounded-lg bg-gray-500/10">
              <RefreshCw className="size-5 text-gray-400" />
            </div>
            <div>
              <p className="text-2xl font-bold text-pdt-neutral">
                {schedules?.filter((s) => s.trigger_type === 'interval').length ?? 0}
              </p>
              <p className="text-xs text-pdt-neutral/50">Interval</p>
            </div>
          </div>
        </DataCard>
      </div>

      {/* Schedule list */}
      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="h-20 animate-pulse rounded-lg bg-pdt-primary-light" />
          ))}
        </div>
      ) : !schedules?.length ? (
        <EmptyState
          icon={Calendar}
          title="No schedules yet"
          description="Create a schedule to automate your agents on a recurring basis."
          action={
            <Button onClick={() => setDialogOpen(true)}>
              <Plus className="mr-1.5 size-4" />
              New Schedule
            </Button>
          }
        />
      ) : (
        <div className="space-y-3">
          {enabled.map((s) => (
            <ScheduleRow key={s.id} schedule={s} />
          ))}
          {disabled.length > 0 && (
            <>
              {enabled.length > 0 && (
                <p className="pt-2 text-xs text-pdt-neutral/40 uppercase tracking-wide">Disabled</p>
              )}
              {disabled.map((s) => (
                <ScheduleRow key={s.id} schedule={s} />
              ))}
            </>
          )}
        </div>
      )}

      <CreateScheduleDialog open={dialogOpen} onClose={() => setDialogOpen(false)} />
    </div>
  )
}

export default SchedulesPage
