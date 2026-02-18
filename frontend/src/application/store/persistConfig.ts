import storage from 'redux-persist/lib/storage'

const persistConfig = {
  key: 'app',
  storage,
  whitelist: ['auth'] // Only auth will be persisted
}

export default persistConfig
