import { api } from './api'
import { API_CONSTANTS } from '../constants/api.constants'

export interface SyncStatus {
  lastSyncAt?: string
  isRunning: boolean
  commitsToday: number
}

export interface TriggerSyncResponse {
  success: boolean
  message: string
}

export const syncApi = api.injectEndpoints({
  endpoints: (builder) => ({
    triggerSync: builder.mutation<TriggerSyncResponse, void>({
      query: () => ({
        url: API_CONSTANTS.SYNC.COMMITS,
        method: 'POST'
      })
    }),
    getSyncStatus: builder.query<SyncStatus, void>({
      query: () => API_CONSTANTS.SYNC.STATUS,
      providesTags: [{ type: 'Sync' as const, id: 'STATUS' }]
    })
  })
})

export const { useTriggerSyncMutation, useGetSyncStatusQuery } = syncApi
