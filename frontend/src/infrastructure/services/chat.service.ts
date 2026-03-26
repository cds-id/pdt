import { api } from './api'
import { API_CONSTANTS } from '../constants/api.constants'
import type { IConversation } from '../../domain/chat/interfaces/chat.interface'

export const chatApi = api.injectEndpoints({
  endpoints: (builder) => ({
    listConversations: builder.query<IConversation[], void>({
      query: () => API_CONSTANTS.CONVERSATIONS.LIST,
      providesTags: (result) =>
        result
          ? [
              ...result.map(({ id }) => ({ type: 'Conversation' as const, id })),
              { type: 'Conversation', id: 'LIST' },
            ]
          : [{ type: 'Conversation', id: 'LIST' }],
    }),
    getConversation: builder.query<IConversation, string>({
      query: (id) => API_CONSTANTS.CONVERSATIONS.GET(id),
      providesTags: (_result, _error, id) => [{ type: 'Conversation', id }],
    }),
    deleteConversation: builder.mutation<void, string>({
      query: (id) => ({
        url: API_CONSTANTS.CONVERSATIONS.DELETE(id),
        method: 'DELETE',
      }),
      invalidatesTags: [{ type: 'Conversation', id: 'LIST' }],
    }),
  }),
})

export const {
  useListConversationsQuery,
  useGetConversationQuery,
  useDeleteConversationMutation,
} = chatApi
