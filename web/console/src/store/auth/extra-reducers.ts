/**
 * Auth Extra Reducers
 * Async thunk reducers for auth slice
 */

import type { ActionReducerMapBuilder } from '@reduxjs/toolkit'
import { validateAuthAsync, initializeAuthAsync, fetchProfileAsync, refreshAccountAsync } from './actions'
import type { AuthState } from './types'
import type { AccountEntity, ProfileEntity } from '@/services/api/auth/types'

function populateAccount(state: AuthState, account: AccountEntity | null) {
  state.account = account
  state.profile = account?.profiles?.[0] ? ({
    profile_id: account.profiles[0].profile_id,
    email: account.email,
    first_name: account.profiles[0].first_name,
    last_name: account.profiles[0].last_name || '',
    display_name: account.profiles[0].display_name || '',
    phone: account.phone,
    created_at: '',
    updated_at: '',
  } as ProfileEntity) : null
  state.roles = account?.roles ?? []
  state.permissions = account?.permissions ?? []
}

/**
 * Record the session verdict. `isAuthenticated` and `status` must never
 * disagree, so nothing writes them separately — the two used to drift, which is
 * how "not yet known" started reading as "definitely anonymous".
 */
function setAuthed(state: AuthState, authed: boolean) {
  state.isAuthenticated = authed
  state.status = authed ? 'authenticated' : 'anonymous'
}

export const authExtraReducers = (builder: ActionReducerMapBuilder<AuthState>) => {
  builder
    // Validate
    .addCase(validateAuthAsync.fulfilled, (state, action) => {
      populateAccount(state, action.payload)
      setAuthed(state, true)
    })
    .addCase(validateAuthAsync.rejected, (state) => {
      populateAccount(state, null)
      setAuthed(state, false)
    })
    // Initialize
    .addCase(initializeAuthAsync.pending, (state) => {
      state.isLoading = true
    })
    .addCase(initializeAuthAsync.fulfilled, (state, action) => {
      state.isLoading = false
      state.isInitialized = true
      populateAccount(state, action.payload)
      setAuthed(state, !!action.payload)
    })
    .addCase(initializeAuthAsync.rejected, (state) => {
      state.isLoading = false
      state.isInitialized = true
      // We could not reach the backend, so the session is UNKNOWN, not absent.
      // Clear the account too — leaving it populated alongside
      // isAuthenticated=false lets guards read a stale tenant off it.
      populateAccount(state, null)
      setAuthed(state, false)
      state.status = 'error'
    })
    // Fetch Profile
    .addCase(fetchProfileAsync.fulfilled, (state, action) => {
      populateAccount(state, action.payload)
      setAuthed(state, true)
    })
    .addCase(fetchProfileAsync.rejected, (state) => {
      populateAccount(state, null)
      setAuthed(state, false)
    })
    // Refresh Account (sync auth state with the live cookie session)
    .addCase(refreshAccountAsync.fulfilled, (state, action) => {
      state.isInitialized = true
      populateAccount(state, action.payload)
      setAuthed(state, !!action.payload)
    })
    .addCase(refreshAccountAsync.rejected, (state) => {
      populateAccount(state, null)
      setAuthed(state, false)
    })
}
