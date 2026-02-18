import { api } from './api'
import { API_CONSTANTS } from '../constants/api.constants'
import type { ILoginCredentials } from '@/domain/auth/interfaces/auth.interface'
import {
  login as loginAction,
  logout as logoutAction,
  setError
} from '../slices/auth/auth.slice'
import { setUser, clearUser } from '../slices/user/user.slice'

interface LoginResponse {
  token: string
  user: {
    id: string
    email: string
    name: string
  }
}

interface RegisterRequest {
  email: string
  password: string
  name: string
}

export const authApi = api.injectEndpoints({
  endpoints: (builder) => ({
    register: builder.mutation<LoginResponse, RegisterRequest>({
      query: (data) => ({
        url: API_CONSTANTS.AUTH.REGISTER,
        method: 'POST',
        body: data
      }),
      async onQueryStarted(_, { dispatch, queryFulfilled }) {
        try {
          const { data } = await queryFulfilled
          dispatch(loginAction({ token: data.token }))
          dispatch(
            setUser({
              id: data.user.id,
              email: data.user.email,
              name: data.user.name
            })
          )
          localStorage.setItem('auth_token', data.token)
        } catch (error) {
          dispatch(
            setError(error instanceof Error ? error.message : 'Registration failed')
          )
        }
      },
      invalidatesTags: () => [
        { type: 'Auth' as const },
        { type: 'User' as const, id: 'PROFILE' }
      ]
    }),
    login: builder.mutation<LoginResponse, ILoginCredentials>({
      query: (credentials) => ({
        url: API_CONSTANTS.AUTH.LOGIN,
        method: 'POST',
        body: credentials
      }),
      async onQueryStarted(_, { dispatch, queryFulfilled }) {
        try {
          const { data } = await queryFulfilled
          // Update auth state with token from response
          dispatch(loginAction({ token: data.token }))
          // Update user state with user data from response
          dispatch(
            setUser({
              id: data.user.id,
              email: data.user.email,
              name: data.user.name
            })
          )
          localStorage.setItem('auth_token', data.token)
        } catch (error) {
          // Handle error and update auth state
          dispatch(
            setError(error instanceof Error ? error.message : 'Login failed')
          )
        }
      },
      invalidatesTags: () => [
        { type: 'Auth' as const },
        { type: 'User' as const, id: 'PROFILE' }
      ]
    }),
    logout: builder.mutation<void, void>({
      query: () => ({
        url: API_CONSTANTS.AUTH.LOGOUT,
        method: 'POST'
      }),
      async onQueryStarted(_, { dispatch, queryFulfilled }) {
        try {
          await queryFulfilled
          // Reset auth state on logout
          dispatch(logoutAction())
          // Clear user state on logout
          dispatch(clearUser())
          localStorage.removeItem('auth_token')
        } catch (error) {
          // Handle error if logout API call fails
          dispatch(
            setError(error instanceof Error ? error.message : 'Logout failed')
          )
          localStorage.removeItem('auth_token')
        }
      },
      invalidatesTags: () => [
        { type: 'Auth' as const },
        { type: 'User' as const }
      ]
    })
  })
})

// Export the generated hooks
export const { useRegisterMutation, useLoginMutation, useLogoutMutation } = authApi
