/**
 * The session verdict must distinguish "not yet known" from "known negative".
 *
 * Conflating them is what made an authenticated user briefly see the login /
 * no-access page: a boot `/account` call that failed for ANY reason resolved to
 * isAuthenticated=false, which every guard read as "anonymous" and acted on.
 */
import { describe, expect, it } from 'vitest'
import reducer from './slice'
import { initializeAuthAsync } from './actions'
import type { AuthState } from './types'
import type { AccountEntity } from '@/services/api/auth/types'

const account = { profiles: [], roles: [], permissions: [] } as unknown as AccountEntity

function boot(): AuthState {
  return reducer(undefined, { type: '@@INIT' })
}

describe('auth status', () => {
  it('starts unknown, not anonymous', () => {
    const state = boot()
    expect(state.status).toBe('unknown')
    expect(state.isInitialized).toBe(false)
    // The convenience flag is false here too — which is exactly why nothing may
    // make a redirect decision from it alone.
    expect(state.isAuthenticated).toBe(false)
  })

  it('records a confirmed session as authenticated', () => {
    const state = reducer(boot(), initializeAuthAsync.fulfilled(account, '', undefined))
    expect(state.status).toBe('authenticated')
    expect(state.isAuthenticated).toBe(true)
  })

  it("records the backend's no-session verdict as anonymous", () => {
    const state = reducer(boot(), initializeAuthAsync.fulfilled(null, '', undefined))
    expect(state.status).toBe('anonymous')
    expect(state.isAuthenticated).toBe(false)
  })

  it('records an unreachable backend as error, never anonymous', () => {
    const state = reducer(boot(), initializeAuthAsync.rejected(new Error('network down'), '', undefined))
    expect(state.status).toBe('error')
    expect(state.isAuthenticated).toBe(false)
    // Stale account must not survive: guards read tenant off it.
    expect(state.account).toBeNull()
  })

  it('always resolves out of unknown so the boot gate cannot hang', () => {
    for (const action of [
      initializeAuthAsync.fulfilled(account, '', undefined),
      initializeAuthAsync.fulfilled(null, '', undefined),
      initializeAuthAsync.rejected(new Error('x'), '', undefined),
    ]) {
      const state = reducer(boot(), action)
      expect(state.status).not.toBe('unknown')
      expect(state.isInitialized).toBe(true)
    }
  })
})
