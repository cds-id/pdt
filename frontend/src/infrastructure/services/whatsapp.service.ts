import { api } from './api'
import { API_CONSTANTS } from '../constants/api.constants'
import {
  IWaNumber,
  IWaListener,
  IWaMessage,
  IWaOutbox,
  IWaMessagePage
} from '@/domain/whatsapp/interfaces/whatsapp.interface'

export const whatsappApi = api.injectEndpoints({
  endpoints: (builder) => ({
    // Numbers
    listNumbers: builder.query<IWaNumber[], void>({
      query: () => API_CONSTANTS.WA.NUMBERS,
      providesTags: [{ type: 'WhatsApp' as const, id: 'NUMBERS' }]
    }),
    addNumber: builder.mutation<IWaNumber, { phone_number: string; display_name: string }>({
      query: (body) => ({
        url: API_CONSTANTS.WA.NUMBERS,
        method: 'POST',
        body
      }),
      invalidatesTags: [{ type: 'WhatsApp' as const, id: 'NUMBERS' }]
    }),
    updateNumber: builder.mutation<IWaNumber, { id: number; display_name?: string }>({
      query: ({ id, ...body }) => ({
        url: API_CONSTANTS.WA.NUMBER(id),
        method: 'PUT',
        body
      }),
      invalidatesTags: [{ type: 'WhatsApp' as const, id: 'NUMBERS' }]
    }),
    deleteNumber: builder.mutation<void, number>({
      query: (id) => ({
        url: API_CONSTANTS.WA.NUMBER(id),
        method: 'DELETE'
      }),
      invalidatesTags: [{ type: 'WhatsApp' as const, id: 'NUMBERS' }]
    }),
    disconnectNumber: builder.mutation<void, number>({
      query: (id) => ({
        url: `${API_CONSTANTS.WA.NUMBER(id)}/disconnect`,
        method: 'POST'
      }),
      invalidatesTags: [{ type: 'WhatsApp' as const, id: 'NUMBERS' }]
    }),

    // Groups & Contacts (for JID discovery)
    getGroups: builder.query<{ jid: string; name: string; topic?: string; participant_count: number }[], number>({
      query: (numberId) => `${API_CONSTANTS.WA.NUMBER(numberId)}/groups`
    }),
    getContacts: builder.query<{ jid: string; name: string; push_name?: string }[], number>({
      query: (numberId) => `${API_CONSTANTS.WA.NUMBER(numberId)}/contacts`
    }),

    // Listeners
    listListeners: builder.query<IWaListener[], number>({
      query: (numberId) => API_CONSTANTS.WA.LISTENERS(numberId),
      providesTags: (_result, _error, numberId) => [
        { type: 'WhatsApp' as const, id: `LISTENERS_${numberId}` }
      ]
    }),
    addListener: builder.mutation<
      IWaListener,
      { numberId: number; jid: string; name: string; type: 'group' | 'personal' }
    >({
      query: ({ numberId, ...body }) => ({
        url: API_CONSTANTS.WA.LISTENERS(numberId),
        method: 'POST',
        body
      }),
      invalidatesTags: (_result, _error, { numberId }) => [
        { type: 'WhatsApp' as const, id: `LISTENERS_${numberId}` }
      ]
    }),
    updateListener: builder.mutation<
      IWaListener,
      { id: number; numberId: number; is_active?: boolean; name?: string }
    >({
      query: ({ id, numberId: _numberId, ...body }) => ({
        url: API_CONSTANTS.WA.LISTENER(id),
        method: 'PUT',
        body
      }),
      invalidatesTags: (_result, _error, { numberId }) => [
        { type: 'WhatsApp' as const, id: `LISTENERS_${numberId}` }
      ]
    }),
    deleteListener: builder.mutation<void, { id: number; numberId: number }>({
      query: ({ id }) => ({
        url: API_CONSTANTS.WA.LISTENER(id),
        method: 'DELETE'
      }),
      invalidatesTags: (_result, _error, { numberId }) => [
        { type: 'WhatsApp' as const, id: `LISTENERS_${numberId}` }
      ]
    }),

    // Messages
    listMessages: builder.query<
      IWaMessagePage,
      { listenerId: number; page?: number; limit?: number }
    >({
      query: ({ listenerId, page = 1, limit = 50 }) =>
        `${API_CONSTANTS.WA.MESSAGES(listenerId)}?page=${page}&limit=${limit}`,
      providesTags: (_result, _error, { listenerId }) => [
        { type: 'WhatsApp' as const, id: `MESSAGES_${listenerId}` }
      ]
    }),
    searchMessages: builder.query<IWaMessage[], { q: string; listenerId?: number }>({
      query: ({ q, listenerId }) => {
        const params = new URLSearchParams({ q })
        if (listenerId !== undefined) params.set('listener_id', String(listenerId))
        return `${API_CONSTANTS.WA.SEARCH_MESSAGES}?${params.toString()}`
      },
      providesTags: [{ type: 'WhatsApp' as const, id: 'SEARCH' }]
    }),

    // Outbox
    listOutbox: builder.query<IWaOutbox[], { status?: string } | void>({
      query: (params) => {
        if (params?.status && params.status !== 'all') {
          return `${API_CONSTANTS.WA.OUTBOX}?status=${params.status}`
        }
        return API_CONSTANTS.WA.OUTBOX
      },
      providesTags: [{ type: 'WhatsApp' as const, id: 'OUTBOX' }]
    }),
    updateOutbox: builder.mutation<
      IWaOutbox,
      { id: number; status?: string; content?: string }
    >({
      query: ({ id, ...body }) => ({
        url: API_CONSTANTS.WA.OUTBOX_ITEM(id),
        method: 'PUT',
        body
      }),
      invalidatesTags: [{ type: 'WhatsApp' as const, id: 'OUTBOX' }]
    }),
    deleteOutbox: builder.mutation<void, number>({
      query: (id) => ({
        url: API_CONSTANTS.WA.OUTBOX_ITEM(id),
        method: 'DELETE'
      }),
      invalidatesTags: [{ type: 'WhatsApp' as const, id: 'OUTBOX' }]
    })
  })
})

export const {
  useListNumbersQuery,
  useAddNumberMutation,
  useUpdateNumberMutation,
  useDeleteNumberMutation,
  useDisconnectNumberMutation,
  useGetGroupsQuery,
  useGetContactsQuery,
  useListListenersQuery,
  useAddListenerMutation,
  useUpdateListenerMutation,
  useDeleteListenerMutation,
  useListMessagesQuery,
  useSearchMessagesQuery,
  useListOutboxQuery,
  useUpdateOutboxMutation,
  useDeleteOutboxMutation
} = whatsappApi
