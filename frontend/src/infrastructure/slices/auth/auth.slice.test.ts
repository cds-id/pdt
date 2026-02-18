import { describe, it, expect } from 'vitest'
import authReducer, { login } from './auth.slice'
import type { IAuthState } from '@/domain/auth/interfaces/auth.interface'

describe('auth slice', () => {
  const initialState: IAuthState = {
    token: null,
    isAuthenticated: false,
    error: null
  }

  it('should handle initial state', () => {
    expect(authReducer(undefined, { type: 'unknown' })).toEqual(initialState)
  })

  it('should handle login', () => {
    const actual = authReducer(initialState, login({ token: 'fake-token' }))
    expect(actual).toEqual({
      token: 'fake-token',
      isAuthenticated: true,
      error: null
    })
  })
})
