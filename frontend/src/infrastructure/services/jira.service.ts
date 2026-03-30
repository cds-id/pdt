import { api } from './api'
import { API_CONSTANTS } from '../constants/api.constants'

export interface JiraWorkspace {
  id: number
  workspace: string
  name: string
  project_keys: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface JiraSprint {
  id: number
  jira_sprint_id: string
  workspace_id?: number
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
  workspace_id?: number
  summary: string
  status: string
  assignee?: string
  sprint_id?: number
  details_json?: string
  created_at: string
}

export interface JiraComment {
  id: number
  card_key: string
  comment_id: string
  author: string
  author_email: string
  body: string
  commented_at: string
}

export const jiraApi = api.injectEndpoints({
  endpoints: (builder) => ({
    // Workspace CRUD
    listWorkspaces: builder.query<JiraWorkspace[], void>({
      query: () => API_CONSTANTS.JIRA.WORKSPACES,
      providesTags: [{ type: 'Jira' as const, id: 'WORKSPACES' }]
    }),
    addWorkspace: builder.mutation<JiraWorkspace, { workspace: string; name?: string; project_keys?: string }>({
      query: (body) => ({
        url: API_CONSTANTS.JIRA.WORKSPACES,
        method: 'POST',
        body
      }),
      invalidatesTags: [{ type: 'Jira' as const, id: 'WORKSPACES' }]
    }),
    updateWorkspace: builder.mutation<JiraWorkspace, { id: number } & Partial<JiraWorkspace>>({
      query: ({ id, ...body }) => ({
        url: API_CONSTANTS.JIRA.WORKSPACE(id),
        method: 'PATCH',
        body
      }),
      invalidatesTags: [{ type: 'Jira' as const, id: 'WORKSPACES' }]
    }),
    deleteWorkspace: builder.mutation<void, number>({
      query: (id) => ({
        url: API_CONSTANTS.JIRA.WORKSPACE(id),
        method: 'DELETE'
      }),
      invalidatesTags: [{ type: 'Jira' as const, id: 'WORKSPACES' }]
    }),

    // Sprint / Card / Comment
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
    }),
    getCardComments: builder.query<JiraComment[], string>({
      query: (key) => `/jira/cards/${key}/comments`,
      providesTags: (_result, _error, key) => [{ type: 'Jira' as const, id: `comments-${key}` }]
    })
  })
})

export const {
  useListWorkspacesQuery,
  useAddWorkspaceMutation,
  useUpdateWorkspaceMutation,
  useDeleteWorkspaceMutation,
  useListSprintsQuery,
  useGetSprintQuery,
  useGetActiveSprintQuery,
  useListCardsQuery,
  useGetCardQuery,
  useGetCardCommentsQuery
} = jiraApi
