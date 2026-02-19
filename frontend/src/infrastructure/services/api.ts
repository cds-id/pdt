import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react'
import { RootState } from '@/application/store'
import { API_CONSTANTS } from '../constants/api.constants'

export const api = createApi({
  reducerPath: 'api',
  tagTypes: [
    'User',
    'Auth',
    'Repo',
    'Sync',
    'Commit',
    'Jira',
    'Report',
    'ReportTemplate'
  ],
  baseQuery: fetchBaseQuery({
    baseUrl: `${API_CONSTANTS.BASE_URL}${API_CONSTANTS.API_PREFIX}`,
    prepareHeaders: (headers, { getState }) => {
      // Get the token from the auth state
      const token = (getState() as RootState).auth.token

      // If we have a token, include it in the headers
      if (token) {
        headers.set('authorization', `Bearer ${token}`)
      }

      return headers
    }
  }),
  endpoints: () => ({})
})
