import { api } from './api'
import { API_CONSTANTS } from '../constants/api.constants'

export interface JiraSprint {
  id: number
  jira_sprint_id: string
  name: string
  state: 'active' | 'closed' | 'future'
  start_date?: string
  end_date?: string
  created_at: string
  cards?: JiraCard[]
}

export interface JiraCard {
  id: number
  key: string
  summary: string
  status: string
  assignee?: string
  sprint_id?: number
  details_json?: string
  created_at: string
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
    getActiveSprint: builder.query<JiraSprint | null, void>({
      async queryFn(_arg, _queryApi, _extraOptions, fetchWithBQ) {
        const result = await fetchWithBQ(API_CONSTANTS.JIRA.ACTIVE_SPRINT)
        if (result.error) {
          if (result.error.status === 404) return { data: null }
          return { error: result.error }
        }
        return { data: result.data as JiraSprint }
      },
      providesTags: [{ type: 'Jira' as const, id: 'ACTIVE_SPRINT' }]
    }),
    listCards: builder.query<JiraCard[], number | void>({
      query: (sprintId) =>
        sprintId
          ? `${API_CONSTANTS.JIRA.CARDS}?sprint_id=${sprintId}`
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
