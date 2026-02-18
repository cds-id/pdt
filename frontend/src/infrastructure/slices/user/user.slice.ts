import { createSlice, PayloadAction } from '@reduxjs/toolkit'

interface UserState {
  id: string | null
  email: string | null
  name: string | null
}

const initialState: UserState = {
  id: null,
  email: null,
  name: null
}

const userSlice = createSlice({
  name: 'user',
  initialState,
  reducers: {
    setUser: (state, action: PayloadAction<UserState>) => {
      state.id = action.payload.id
      state.email = action.payload.email
      state.name = action.payload.name
    },
    clearUser: (state) => {
      state.id = null
      state.email = null
      state.name = null
    }
  }
})

export const { setUser, clearUser } = userSlice.actions
export default userSlice.reducer
