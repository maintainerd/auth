/**
 * Redux Store Configuration
 * Configures the Redux store with RTK
 */

import { configureStore } from '@reduxjs/toolkit'
import { authReducer } from './auth/reducers'
import { tenantReducer } from './tenant/reducers'

export const store = configureStore({
  reducer: {
    auth: authReducer,
    tenant: tenantReducer,
  },
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware({
      serializableCheck: {
        ignoredActions: ['persist/PERSIST', 'persist/REHYDRATE'],
      },
    }),
})

export type RootState = ReturnType<typeof store.getState>
export type AppDispatch = typeof store.dispatch
