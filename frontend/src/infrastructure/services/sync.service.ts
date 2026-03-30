import { api } from './api'
import { API_CONSTANTS } from '../constants/api.constants'

export interface SyncInfo {
  last_sync?: string
  next_sync?: string
  status: 'idle' | 'syncing'
  last_error?: string
}

export interface SyncStatus {
  commits: SyncInfo
  jira: SyncInfo
}

export const syncApi = api.injectEndpoints({
  endpoints: (builder) => ({
    triggerSync: builder.mutation<{ results: unknown[] }, void>({
      query: () => ({
        url: API_CONSTANTS.SYNC.COMMITS,
        method: 'POST'
      })
    }),
    triggerJiraSync: builder.mutation<{ status: string }, number | void>({
      query: (workspaceId) => ({
        url: workspaceId
          ? `${API_CONSTANTS.SYNC.JIRA}?workspace_id=${workspaceId}`
          : API_CONSTANTS.SYNC.JIRA,
        method: 'POST'
      }),
      invalidatesTags: [{ type: 'Jira' as const, id: 'SPRINTS' }, { type: 'Jira' as const, id: 'CARDS' }]
    }),
    getSyncStatus: builder.query<SyncStatus, void>({
      query: () => API_CONSTANTS.SYNC.STATUS,
      providesTags: [{ type: 'Sync' as const, id: 'STATUS' }]
    })
  })
})

export const { useTriggerSyncMutation, useTriggerJiraSyncMutation, useGetSyncStatusQuery } = syncApi
