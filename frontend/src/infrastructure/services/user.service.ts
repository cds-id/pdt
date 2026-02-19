import { api } from './api'
import { API_CONSTANTS } from '../constants/api.constants'
import { IUser } from '@/domain/user/interfaces/user.interface'

interface ValidateResponse {
  github: { valid: boolean; error?: string }
  gitlab: { valid: boolean; error?: string }
  jira: { valid: boolean; error?: string }
}

export const userApi = api.injectEndpoints({
  endpoints: (builder) => ({
    getProfile: builder.query<IUser, void>({
      query: () => API_CONSTANTS.USER.PROFILE,
      providesTags: () => [{ type: 'User' as const, id: 'PROFILE' }]
    }),
    updateProfile: builder.mutation<IUser, Partial<IUser>>({
      query: (data) => ({
        url: API_CONSTANTS.USER.UPDATE,
        method: 'PUT',
        body: data
      }),
      invalidatesTags: () => [{ type: 'User' as const, id: 'PROFILE' }]
    }),
    validateIntegrations: builder.mutation<ValidateResponse, void>({
      query: () => ({
        url: API_CONSTANTS.USER.VALIDATE,
        method: 'POST'
      })
    })
  })
})

// Export the generated hooks
export const {
  useGetProfileQuery,
  useUpdateProfileMutation,
  useValidateIntegrationsMutation
} = userApi
