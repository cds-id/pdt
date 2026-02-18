import { api } from './api'
import { API_CONSTANTS } from '../constants/api.constants'

export interface JiraSprint {
  id: string
  name: string
  state: 'active' | 'closed' | 'future'
  startDate?: string
  endDate?: string
  completeDate?: string
}

export interface JiraCard {
  key: string
  summary: string
  status: string
  type: 'story' | 'bug' | 'task' | 'subtask'
  assignee?: string
  priority: string
  created: string
  updated: string
  commits: string[]
}

export interface JiraCardsResponse {
  cards: JiraCard[]
  total: number
}

export const jiraApi = api.injectEndpoints({
  endpoints: (builder) => ({
    listSprints: builder.query<JiraSprint[], void>({
      query: () => API_CONSTANTS.JIRA.SPRINTS,
      providesTags: [{ type: 'Jira' as const, id: 'SPRINTS' }]
    }),
    getSprint: builder.query<JiraSprint, string>({
      query: (id) => API_CONSTANTS.JIRA.SPRINT(id),
      providesTags: (_, __, id) => [{ type: 'Jira' as const, id }]
    }),
    getActiveSprint: builder.query<JiraSprint, void>({
      query: () => API_CONSTANTS.JIRA.ACTIVE_SPRINT,
      providesTags: [{ type: 'Jira' as const, id: 'ACTIVE_SPRINT' }]
    }),
    listCards: builder.query<JiraCardsResponse, string | void>({
      query: (sprintId) =>
        sprintId
          ? `${API_CONSTANTS.JIRA.CARDS}?sprintId=${sprintId}`
          : API_CONSTANTS.JIRA.CARDS,
      providesTags: [{ type: 'Jira' as const, id: 'CARDS' }]
    }),
    getCard: builder.query<JiraCard, string>({
      query: (key) => API_CONSTANTS.JIRA.CARD(key),
      providesTags: (_, __, key) => [{ type: 'Jira' as const, id: key }]
    })
  })
})

export const {
  useListSprintsQuery,
  useGetSprintQuery,
  useGetActiveSprintQuery,
  useListCardsQuery,
  useGetCardQuery
} = jiraApi
