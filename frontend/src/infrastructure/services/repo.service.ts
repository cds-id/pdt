import { api } from './api'
import { API_CONSTANTS } from '../constants/api.constants'

export interface Repository {
  id: string
  url: string
  name: string
  provider: 'github' | 'gitlab'
  isValid: boolean
  lastSyncedAt?: string
  createdAt: string
}

export interface AddRepoRequest {
  url: string
}

export const repoApi = api.injectEndpoints({
  endpoints: (builder) => ({
    listRepos: builder.query<Repository[], void>({
      query: () => API_CONSTANTS.REPOS.LIST,
      providesTags: (result) =>
        result
          ? [
              ...result.map(({ id }) => ({ type: 'Repo' as const, id })),
              { type: 'Repo', id: 'LIST' }
            ]
          : [{ type: 'Repo', id: 'LIST' }]
    }),
    addRepo: builder.mutation<Repository, AddRepoRequest>({
      query: (data) => ({
        url: API_CONSTANTS.REPOS.ADD,
        method: 'POST',
        body: data
      }),
      invalidatesTags: [{ type: 'Repo', id: 'LIST' }]
    }),
    deleteRepo: builder.mutation<void, string>({
      query: (id) => ({
        url: API_CONSTANTS.REPOS.DELETE(id),
        method: 'DELETE'
      }),
      invalidatesTags: [{ type: 'Repo', id: 'LIST' }]
    }),
    validateRepo: builder.mutation<{ valid: boolean; error?: string }, string>({
      query: (id) => ({
        url: API_CONSTANTS.REPOS.VALIDATE(id),
        method: 'POST'
      })
    })
  })
})

export const {
  useListReposQuery,
  useAddRepoMutation,
  useDeleteRepoMutation,
  useValidateRepoMutation
} = repoApi
