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
      query: (body) => ({ url: API_CONSTANTS.SCHEDULES.CREATE, method: 'POST', body }),
      invalidatesTags: [{ type: 'Schedule', id: 'LIST' }],
    }),
    updateSchedule: builder.mutation<AgentSchedule, { id: string } & Partial<CreateScheduleRequest>>({
      query: ({ id, ...body }) => ({ url: API_CONSTANTS.SCHEDULES.UPDATE(id), method: 'PUT', body }),
      invalidatesTags: (_, __, { id }) => [{ type: 'Schedule', id }, { type: 'Schedule', id: 'LIST' }],
    }),
    deleteSchedule: builder.mutation<void, string>({
      query: (id) => ({ url: API_CONSTANTS.SCHEDULES.DELETE(id), method: 'DELETE' }),
      invalidatesTags: [{ type: 'Schedule', id: 'LIST' }],
    }),
    toggleSchedule: builder.mutation<{ enabled: boolean }, string>({
      query: (id) => ({ url: API_CONSTANTS.SCHEDULES.TOGGLE(id), method: 'POST' }),
      invalidatesTags: (_, __, id) => [{ type: 'Schedule', id }, { type: 'Schedule', id: 'LIST' }],
    }),
    runScheduleNow: builder.mutation<{ message: string }, string>({
      query: (id) => ({ url: API_CONSTANTS.SCHEDULES.RUN(id), method: 'POST' }),
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
