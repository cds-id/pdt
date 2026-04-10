import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react'
import { RootState } from '@/application/store'
import { API_CONSTANTS } from '../constants/api.constants'

export interface ExecutiveReportListItem {
  id: number
  range_start: string
  range_end: string
  status: 'generating' | 'completed' | 'failed'
  created_at: string
  completed_at?: string
}

export interface Suggestion {
  kind: 'gap' | 'stale' | 'next_step'
  title: string
  detail: string
  refs: string[]
}

export interface DailyBucket {
  day: string
  commits: number
  jira_changes: number
  wa_messages: number
}

export interface Metrics {
  commits_total: number
  commits_linked: number
  cards_active: number
  cards_with_commits: number
  wa_topics_ticketed: number
  wa_topics_orphan: number
  stale_card_count: number
  linkage_pct_commits: number
  linkage_pct_cards: number
  truncated: boolean
}

export interface JiraCardRef {
  card_key: string
  title: string
  status: string
  assignee: string
  content: string
  updated_at: string
  workspace_id?: number
}

export interface WAMessageRef {
  message_id: string
  sender_name: string
  content: string
  timestamp: string
}

export interface CommitRef {
  sha: string
  message: string
  repo_name: string
  author: string
  committed_at: string
}

export interface Topic {
  anchor: JiraCardRef
  messages: WAMessageRef[]
  commits: CommitRef[]
  stale: boolean
  days_idle: number
}

export interface WAGroup {
  summary: string
  messages: WAMessageRef[]
  started_at: string
}

export interface CorrelatedDataset {
  user_id: number
  workspace_id?: number
  range: { Start: string; End: string }
  topics: Topic[]
  orphan_wa: WAGroup[]
  orphan_commits: CommitRef[]
  metrics: Metrics
  daily_buckets: DailyBucket[]
}

export interface ExecutiveReport {
  id: number
  user_id: number
  workspace_id?: number
  range_start: string
  range_end: string
  stale_threshold_days: number
  narrative: string
  suggestions: Suggestion[]
  dataset: CorrelatedDataset
  status: 'generating' | 'completed' | 'failed'
  error_message?: string
  created_at: string
  completed_at?: string
}

export const executiveReportApi = createApi({
  reducerPath: 'executiveReportApi',
  baseQuery: fetchBaseQuery({
    baseUrl: `${API_CONSTANTS.BASE_URL}${API_CONSTANTS.API_PREFIX}`,
    prepareHeaders: (headers, { getState }) => {
      const token = (getState() as RootState).auth.token
      if (token) {
        headers.set('authorization', `Bearer ${token}`)
      }
      return headers
    }
  }),
  tagTypes: ['ExecutiveReport'],
  endpoints: (b) => ({
    listExecutiveReports: b.query<ExecutiveReportListItem[], void>({
      query: () => '/protected/reports/executive',
      providesTags: ['ExecutiveReport'],
    }),
    getExecutiveReport: b.query<ExecutiveReport, number>({
      query: (id) => `/protected/reports/executive/${id}`,
      providesTags: (_r, _e, id) => [{ type: 'ExecutiveReport', id }],
    }),
    deleteExecutiveReport: b.mutation<void, number>({
      query: (id) => ({ url: `/protected/reports/executive/${id}`, method: 'DELETE' }),
      invalidatesTags: ['ExecutiveReport'],
    }),
  }),
})

export const {
  useListExecutiveReportsQuery,
  useGetExecutiveReportQuery,
  useDeleteExecutiveReportMutation,
} = executiveReportApi
