import { api } from './api'
import { API_CONSTANTS } from '../constants/api.constants'

export interface Commit {
  id: number
  sha: string
  message: string
  author: string
  author_email: string
  branch: string
  date: string
  repo_id: number
  jira_card_key: string
  has_link: boolean
  created_at: string
  url: string
}

export interface CommitFilters {
  repo_id?: string
  jira_card_key?: string
}

export const commitApi = api.injectEndpoints({
  endpoints: (builder) => ({
    listCommits: builder.query<Commit[], CommitFilters | void>({
      query: (filters) => {
        const params = new URLSearchParams()
        if (filters?.repo_id) params.append('repo_id', filters.repo_id)
        if (filters?.jira_card_key)
          params.append('jira_card_key', filters.jira_card_key)
        const query = params.toString()
        return `${API_CONSTANTS.COMMITS.LIST}${query ? `?${query}` : ''}`
      },
      providesTags: (result) =>
        result
          ? [
              ...result.map(({ id }) => ({ type: 'Commit' as const, id })),
              { type: 'Commit', id: 'LIST' }
            ]
          : [{ type: 'Commit', id: 'LIST' }]
    }),
    getMissingCommits: builder.query<Commit[], void>({
      query: () => API_CONSTANTS.COMMITS.MISSING,
      providesTags: [{ type: 'Commit', id: 'MISSING' }]
    }),
    linkToJira: builder.mutation<
      Commit,
      { sha: string; jira_card_key: string }
    >({
      query: ({ sha, jira_card_key }) => ({
        url: API_CONSTANTS.COMMITS.LINK(sha),
        method: 'POST',
        body: { jira_card_key }
      }),
      invalidatesTags: [
        { type: 'Commit', id: 'LIST' },
        { type: 'Commit', id: 'MISSING' }
      ]
    })
  })
})

export const {
  useListCommitsQuery,
  useGetMissingCommitsQuery,
  useLinkToJiraMutation
} = commitApi
