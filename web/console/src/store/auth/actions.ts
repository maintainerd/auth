import { createAsyncThunk } from '@reduxjs/toolkit'
import {
  fetchAccount,
  validateAuthentication as validateAuth,
} from '@/services'

export const validateAuthAsync = createAsyncThunk(
  'auth/validate',
  async (_, thunkAPI) => {
    const account = await validateAuth()
    if (account) {
      return account
    }
    return thunkAPI.rejectWithValue({ message: 'Authentication validation failed' })
  }
)

export const initializeAuthAsync = createAsyncThunk(
  'auth/initialize',
  async () => {
    const account = await validateAuth()
    return account
  }
)

export const fetchProfileAsync = createAsyncThunk(
  'auth/fetchProfile',
  async () => {
    const account = await fetchAccount()
    if (account) return account
    throw new Error('No account found')
  }
)

// refreshAccountAsync re-reads /account using the current (cookie) session and
// syncs the full auth state. Used after events that change the account but
// happen outside login (e.g. profile creation) so routing reflects the live
// email_verified + profiles state.
export const refreshAccountAsync = createAsyncThunk(
  'auth/refreshAccount',
  async () => {
    return await fetchAccount()
  }
)
