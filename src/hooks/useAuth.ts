import { useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { useToast } from '@/hooks/useToast'
import type { ProfileEntity } from '@/services/api/auth/types'
import {
  loginAsync,
  completeMFALoginAsync,
  registerAsync,
  registerInviteAsync,
  logoutAsync,
  validateAuthAsync,
  initializeAuthAsync,
  fetchProfileAsync,
  refreshAccountAsync,
  forgotPasswordAsync,
  resetPasswordAsync,
  type ResetPasswordAsyncRequest
} from '@/store/auth/actions'
import {
  clearError,
  setProfile
} from '@/store/auth/reducers'
import { clearTenant } from '@/store/tenant/reducers'

// RegisterInput is the signup payload the form collects. fullname and phone are
// present only when the resolved registration context requires them.
export interface RegisterInput {
  email: string
  password: string
  fullname?: string
  phone?: string
}

export function useAuth() {
  const dispatch = useAppDispatch()
  const queryClient = useQueryClient()
  const [searchParams] = useSearchParams()
  const { showError } = useToast()
  const { profile, account, isAuthenticated, isLoading, isInitialized, error, status } = useAppSelector((state) => state.auth)
  // Tenant slug resolved by the domain bootstrap (never parsed from the host).
  const tenantSlug = useAppSelector((state) => state.tenant.currentTenant?.name)

  const login = useCallback(async (email: string, password: string) => {
    const result = await dispatch(loginAsync({ username: email, password })).unwrap()

    // MFA enrolled: a second factor is required before the session is issued.
    if (result.mfaRequired) {
      return {
        requiresProfileSetup: false,
        mfaRequired: true,
        challengeToken: result.mfaChallengeToken ?? '',
        allowedMethods: result.mfaAllowedMethods ?? [],
        totpDigits: result.mfaTotpDigits ?? 6,
      }
    }

    const account = result.data
    return { account, mfaRequired: false }
  }, [dispatch])

  // completeMFALogin finishes the login MFA second step (acr=2 session) and
  // returns whether the user still needs to set up a profile.
  const completeMFALogin = useCallback(async (
    challengeToken: string,
    method: string,
    proof: { code?: string; assertion?: unknown },
    tenantId?: string,
    clientId?: string,
    rememberDevice?: boolean,
  ) => {
    const result = await dispatch(completeMFALoginAsync({
      mfa_challenge_token: challengeToken,
      method,
      code: proof.code,
      assertion: proof.assertion,
      remember_device: rememberDevice,
      tenantId,
      clientId,
    })).unwrap()
    return { account: result.data }
  }, [dispatch])

  // An object parameter rather than positional args: fullname and phone are
  // conditional on the registration flow, and two optional trailing strings would
  // be trivial to transpose.
  const register = useCallback(async (input: RegisterInput) => {
    const clientId = searchParams.get('client_id')
    // Tenant comes from the domain bootstrap (its slug), never from a query param.
    const tenantId = tenantSlug ?? undefined
    const registrationFlow = searchParams.get('registration_flow')

    const result = await dispatch(registerAsync({
      email: input.email,
      password: input.password,
      fullname: input.fullname,
      phone: input.phone,
      clientId: clientId || undefined,
      tenantId,
      registrationFlow: registrationFlow || undefined,
    })).unwrap()

    return { data: result.data }
  }, [dispatch, searchParams, tenantSlug])

  const registerInvite = useCallback(async (
    email: string,
    password: string,
  ) => {
    const inviteToken = searchParams.get('invite_token') || ''
    const expires = searchParams.get('expires') || ''
    const sig = searchParams.get('sig') || ''
    const inviteClientId = searchParams.get('client_id') || undefined
    const inviteTenantId = searchParams.get('tenant_id') || undefined
    // email + callback_url are signed into the invite URL; forward them verbatim
    // so the backend signature check passes.
    const inviteEmail = searchParams.get('email') || email
    const inviteCallbackUrl = searchParams.get('callback_url') || undefined

    const result = await dispatch(registerInviteAsync({
      username: email,
      password,
      queryParams: {
        invite_token: inviteToken,
        expires,
        sig,
        email: inviteEmail,
        callback_url: inviteCallbackUrl,
        client_id: inviteClientId,
        tenant_id: inviteTenantId,
      }
    })).unwrap()

    return { data: result.data }
  }, [dispatch, searchParams])

  const logout = useCallback(async () => {
    try {
      await dispatch(logoutAsync()).unwrap()
      dispatch(clearTenant())
      queryClient.clear()
    } catch (error) {
      showError('Logout failed')
      dispatch(clearTenant())
      queryClient.clear()
      throw error
    }
  }, [dispatch, queryClient, showError])

  const checkAuth = useCallback(async () => {
    try {
      await dispatch(validateAuthAsync()).unwrap()
      return true
    } catch {
      // Silent failure for auth check - don't show error toast
      return false
    }
  }, [dispatch])

  const initializeAuth = useCallback(async () => {
    try {
      queryClient.clear()
      await dispatch(initializeAuthAsync()).unwrap()
    } catch {
      // Silent failure for auth initialization
    }
  }, [dispatch, queryClient])

  const clearAuthError = useCallback(() => {
    dispatch(clearError())
  }, [dispatch])

  const getProfile = useCallback(() => {
    return profile
  }, [profile])

  const fetchProfile = useCallback(async () => {
    try {
      const result = await dispatch(fetchProfileAsync()).unwrap()
      return result
    } catch {
      // Silent failure - profile might not exist yet
      return null
    }
  }, [dispatch])

  // refreshAccount re-syncs auth state with the live cookie session and returns
  // the fresh account (or null). Use after register / verify email / create
  // profile so routing reflects the current email_verified + profiles state.
  const refreshAccount = useCallback(async () => {
    try {
      return await dispatch(refreshAccountAsync()).unwrap()
    } catch {
      return null
    }
  }, [dispatch])

  const updateProfile = useCallback((profileData: ProfileEntity | null) => {
    dispatch(setProfile(profileData))
  }, [dispatch])

  const forgotPassword = useCallback(async (email: string) => {
    await dispatch(forgotPasswordAsync({ email })).unwrap()
  }, [dispatch])

  const resetPassword = useCallback(async (data: ResetPasswordAsyncRequest) => {
    await dispatch(resetPasswordAsync(data)).unwrap()
  }, [dispatch])

  return {
    // State
    profile,
    account,
    isAuthenticated,
    isLoading,
    isInitialized,
    status,
    error,
    // Actions
    login,
    completeMFALogin,
    register,
    registerInvite,
    logout,
    checkAuth,
    initializeAuth,
    clearAuthError,
    getProfile,
    fetchProfile,
    refreshAccount,
    updateProfile,
    forgotPassword,
    resetPassword
  }
}
