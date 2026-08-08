/**
 * Auth Store Types
 * Redux-specific types for auth state management
 */

import type { ProfileEntity, AccountEntity } from '@/services/api/auth/types'

/**
 * Where the session stands.
 *
 *  - `unknown`       boot has not resolved yet. NEVER treat as anonymous.
 *  - `authenticated` the backend confirmed a session.
 *  - `anonymous`     the backend said there is no session (401/403).
 *  - `error`         we could not reach the backend, so we do not know. Also
 *                    not anonymous — bouncing the user to SSO here logs out a
 *                    perfectly good session over a transient blip.
 *
 * `isAuthenticated` remains as a convenience for the many consumers that only
 * care about the positive case, but it is derived from this and is FALSE for
 * both `unknown` and `error`. Anything making a redirect decision must read
 * `status`, not `isAuthenticated`.
 */
export type AuthStatus = 'unknown' | 'authenticated' | 'anonymous' | 'error'

export interface AuthState {
  status: AuthStatus
  account: AccountEntity | null
  profile: ProfileEntity | null
  roles: string[]
  permissions: string[]
  isAuthenticated: boolean
  isLoading: boolean
  isInitialized: boolean
  error: string | null
}
