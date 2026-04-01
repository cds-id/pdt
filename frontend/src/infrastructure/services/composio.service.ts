import { api } from './api'
import { API_CONSTANTS } from '../constants/api.constants'

interface ComposioConfigResponse {
  configured: boolean
}

interface ComposioConnection {
  id: number
  user_id: number
  toolkit: string
  integration_id: string
  account_id: string
  status: string
  created_at: string
  updated_at: string
}

interface InitiateResponse {
  redirect_url: string
  account_id: string
  status: string
}

export const composioApi = api.injectEndpoints({
  endpoints: (builder) => ({
    getComposioConfig: builder.query<ComposioConfigResponse, void>({
      query: () => API_CONSTANTS.COMPOSIO.CONFIG,
      providesTags: [{ type: 'Composio' as const, id: 'CONFIG' }]
    }),
    saveComposioConfig: builder.mutation<ComposioConfigResponse, { api_key: string }>({
      query: (data) => ({
        url: API_CONSTANTS.COMPOSIO.CONFIG,
        method: 'PUT',
        body: data
      }),
      invalidatesTags: [{ type: 'Composio' as const, id: 'CONFIG' }]
    }),
    deleteComposioConfig: builder.mutation<void, void>({
      query: () => ({
        url: API_CONSTANTS.COMPOSIO.CONFIG,
        method: 'DELETE'
      }),
      invalidatesTags: ['Composio']
    }),
    listComposioConnections: builder.query<ComposioConnection[], void>({
      query: () => API_CONSTANTS.COMPOSIO.CONNECTIONS,
      providesTags: [{ type: 'Composio' as const, id: 'CONNECTIONS' }]
    }),
    initiateComposioConnection: builder.mutation<InitiateResponse, { toolkit: string; integration_id: string; redirect_uri: string }>({
      query: ({ toolkit, ...body }) => ({
        url: API_CONSTANTS.COMPOSIO.INITIATE(toolkit),
        method: 'POST',
        body
      }),
      invalidatesTags: [{ type: 'Composio' as const, id: 'CONNECTIONS' }]
    }),
    syncComposioConnections: builder.mutation<ComposioConnection[], void>({
      query: () => ({
        url: API_CONSTANTS.COMPOSIO.SYNC,
        method: 'POST'
      }),
      invalidatesTags: [{ type: 'Composio' as const, id: 'CONNECTIONS' }]
    }),
    deleteComposioConnection: builder.mutation<void, string>({
      query: (toolkit) => ({
        url: API_CONSTANTS.COMPOSIO.DISCONNECT(toolkit),
        method: 'DELETE'
      }),
      invalidatesTags: [{ type: 'Composio' as const, id: 'CONNECTIONS' }]
    })
  })
})

export const {
  useGetComposioConfigQuery,
  useSaveComposioConfigMutation,
  useDeleteComposioConfigMutation,
  useListComposioConnectionsQuery,
  useInitiateComposioConnectionMutation,
  useSyncComposioConnectionsMutation,
  useDeleteComposioConnectionMutation
} = composioApi
