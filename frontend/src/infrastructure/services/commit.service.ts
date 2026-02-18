import { api } from './api'
import { API_CONSTANTS } from '../constants/api.constants'

export interface Commit {
  id: string
  sha: string
  message: string
  author: string
  authorEmail: string
  date: string
  repoId: string
  jiraKey?: string
  url: string
}

export interface CommitFilters {
  repoId?: string
  jiraKey?: string
  fromDate?: string
  toDate?: string
  page?: number
  limit?: number
}

export interface CommitListResponse {
  commits: Commit[]
  total: number
  page: number
  limit: number
}

export interface LinkToJiraRequest {
  jiraKey: string
}

export const commitApi = api.injectEndpoints({
  endpoints: (builder) => ({
    listCommits: builder.query<CommitListResponse, CommitFilters | void>({
      query: (filters) => {
        const params = new URLSearchParams()
        if (filters?.repoId) params.append('repoId', filters.repoId)
        if (filters?.jiraKey) params.append('jiraKey', filters.jiraKey)
        if (filters?.fromDate) params.append('fromDate', filters.fromDate)
        if (filters?.toDate) params.append('toDate', filters.toDate)
        if (filters?.page) params.append('page', String(filters.page))
        if (filters?.limit) params.append('limit', String(filters.limit))
        const query = params.toString()
        return `${API_CONSTANTS.COMMITS.LIST}${query ? `?${query}` : ''}`
      },
      providesTags: (result) =>
        result
          ? [
              ...result.commits.map(({ id }) => ({ type: 'Commit' as const, id })),
              { type: 'Commit', id: 'LIST' }
            ]
          : [{ type: 'Commit', id: 'LIST' }]
    }),
    getMissingCommits: builder.query<Commit[], void>({
      query: () => API_CONSTANTS.COMMITS.MISSING,
      providesTags: [{ type: 'Commit', id: 'MISSING' }]
    }),
    linkToJira: builder.mutation<Commit, { sha: string; jiraKey: string }>({
      query: ({ sha, jiraKey }) => ({
        url: API_CONSTANTS.COMMITS.LINK(sha),
        method: 'POST',
        body: { jiraKey }
      }),
      invalidatesTags: [{ type: 'Commit', id: 'LIST' }, { type: 'Commit', id: 'MISSING' }]
    })
  })
})

export const { useListCommitsQuery, useGetMissingCommitsQuery, useLinkToJiraMutation } = commitApi
