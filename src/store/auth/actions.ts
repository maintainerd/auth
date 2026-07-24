import { createAsyncThunk } from '@reduxjs/toolkit'
import {
  login as authLogin,
  verifyMFALogin,
  register as authRegister,
  registerInvite as authRegisterInvite,
  logout as authLogout,
  fetchAccount,
  validateAuthentication as validateAuth,
  forgotPassword as authForgotPassword,
  resetPassword as authResetPassword,
 	type LoginRequest,
 	type MFALoginVerifyRequest,
 	type RegisterRequest,
  type RegisterInviteQueryParams,
 	type ForgotPasswordRequest,
 	type ResetPasswordRequest,
 	type ResetPasswordQueryParams
} from '@/services'
import type { AccountEntity } from '@/services/api/auth/types'
import { ApiError } from '@/services/api/client'

// Extended register request with optional query parameters
export interface RegisterAsyncRequest extends Omit<RegisterRequest, 'username'> {
  clientId?: string
  tenantId?: string
  registrationFlow?: string
}

// LoginThunkResult is the shared fulfilled payload for the password and MFA
// login thunks. When mfaRequired is set, the session is NOT yet established —
// the UI must complete the second step.
export interface LoginThunkResult {
  data: AccountEntity | null
  message?: string
  mfaRequired?: boolean
  mfaChallengeToken?: string
  mfaAllowedMethods?: string[]
}

export const loginAsync = createAsyncThunk<LoginThunkResult, LoginRequest>(
  'auth/login',
  async (data: LoginRequest, thunkAPI) => {
		try {
			const response = await authLogin(data)
			// MFA enrolled: password step passed but a second factor is required.
			// Surface the challenge to the UI instead of fetching a (nonexistent) session.
			if (response.data?.mfa_required) {
				return {
					data: null,
					message: response.message,
					mfaRequired: true,
					mfaChallengeToken: response.data.mfa_challenge_token,
					mfaAllowedMethods: response.data.mfa_allowed_methods ?? [],
				}
			}
			const account = await fetchAccount()
			return { data: account, message: response.message }
		} catch (error: unknown) {
			const errorMessage = error instanceof Error ? error.message : 'Login failed'
			return thunkAPI.rejectWithValue({ message: errorMessage })
		}
  }
)

// completeMFALoginAsync finishes the login MFA second step. The verify call
// Set-Cookies an acr=2 session, after which we load the profile and mark the
// user authenticated — mirroring a normal login.
export const completeMFALoginAsync = createAsyncThunk<LoginThunkResult, MFALoginVerifyRequest & { tenantId?: string; clientId?: string }>(
  'auth/completeMFALogin',
  async (data: MFALoginVerifyRequest & { tenantId?: string; clientId?: string }, thunkAPI) => {
    try {
      const { tenantId, clientId, ...request } = data
      const response = await verifyMFALogin(request, tenantId, clientId)
      const account = await fetchAccount()
      return { data: account, message: response.message }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'MFA verification failed'
      return thunkAPI.rejectWithValue({ message: errorMessage })
    }
  }
)

export const registerAsync = createAsyncThunk(
  'auth/register',
  async (data: RegisterAsyncRequest, thunkAPI) => {
    try {
      const response = await authRegister(data)
      return { data: response.data }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Registration failed'
      // Carry the status through. rejectWithValue makes unwrap() throw this plain
      // object rather than an Error, so a caller testing `instanceof Error` sees
      // nothing and falls back to a generic string — which discarded every
      // actionable backend message ("fullname is required by the registration
      // flow", "username already taken", …). The status also lets the caller tell
      // an invalid link (404) from a field problem (400).
      return thunkAPI.rejectWithValue({
        message: errorMessage,
        status: error instanceof ApiError ? error.status : undefined,
      })
    }
  }
)

export interface RegisterInviteAsyncRequest {
  username: string
  password: string
  queryParams: RegisterInviteQueryParams
}

export const registerInviteAsync = createAsyncThunk(
  'auth/registerInvite',
  async (data: RegisterInviteAsyncRequest, thunkAPI) => {
    try {
      const response = await authRegisterInvite(
        { username: data.username, password: data.password },
        data.queryParams
      )
      return { data: response.data }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Registration failed'
      return thunkAPI.rejectWithValue({ message: errorMessage })
    }
  }
)

export const logoutAsync = createAsyncThunk(
  'auth/logout',
  async () => {
    await authLogout()
  }
)

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
// syncs the full auth state. Used after register / email verification / profile
// creation — events that change the account but happen outside login — so the
// app's routing reflects the live email_verified + profiles state.
export const refreshAccountAsync = createAsyncThunk(
  'auth/refreshAccount',
  async () => {
    return await fetchAccount()
  }
)

export const forgotPasswordAsync = createAsyncThunk(
  'auth/forgotPassword',
  async (data: ForgotPasswordRequest, thunkAPI) => {
    try {
      await authForgotPassword(data)
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to send reset email'
      return thunkAPI.rejectWithValue({ message: errorMessage })
    }
  }
)

export interface ResetPasswordAsyncRequest {
  password: ResetPasswordRequest
  queryParams: ResetPasswordQueryParams
}

export const resetPasswordAsync = createAsyncThunk(
  'auth/resetPassword',
  async (data: ResetPasswordAsyncRequest, thunkAPI) => {
    try {
      await authResetPassword(data.password, data.queryParams)
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to reset password'
      return thunkAPI.rejectWithValue({ message: errorMessage })
    }
  }
)
