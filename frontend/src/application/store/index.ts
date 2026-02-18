import { configureStore, combineReducers } from '@reduxjs/toolkit'
import { persistStore, persistReducer } from 'redux-persist'
import { api } from '@/infrastructure/services/api'
import authReducer from '@/infrastructure/slices/auth/auth.slice'
import userReducer from '@/infrastructure/slices/user/user.slice'
import persistConfig from './persistConfig'

const rootReducer = combineReducers({
  auth: authReducer,
  user: userReducer,
  [api.reducerPath]: api.reducer
})

const persistedReducer = persistReducer(persistConfig, rootReducer)

export const store = configureStore({
  reducer: persistedReducer,
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware({
      serializableCheck: {
        ignoredActions: [
          'persist/PERSIST',
          'persist/REHYDRATE',
          'persist/REGISTER'
        ]
      }
    }).concat(api.middleware)
})

export const persistor = persistStore(store)

export type RootState = ReturnType<typeof store.getState>
export type AppDispatch = typeof store.dispatch
