import { createSlice, PayloadAction } from '@reduxjs/toolkit'
import { IAuthState } from '@/domain/auth/interfaces/auth.interface'

const initialState: IAuthState = {
  token: null,
  isAuthenticated: false,
  error: null
}

const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    login: (state, action: PayloadAction<{ token: string }>) => {
      state.token = action.payload.token
      state.isAuthenticated = true
      state.error = null
    },
    logout: (state) => {
      state.token = null
      state.isAuthenticated = false
      state.error = null
    },
    setError: (state, action: PayloadAction<string>) => {
      state.error = action.payload
    }
  }
})

export const { login, logout, setError } = authSlice.actions
export default authSlice.reducer
