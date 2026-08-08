/**
 * Profile Hook
 * Custom hook for profile operations (setup flow and register flow)
 * - createProfile: For setup/profile (no authentication required, one-time only)
 * - createProfileForRegister: For register/profile (requires authentication cookies)
 */

import { useCallback, useState } from 'react'
import { useToast } from '@/hooks/useToast'
import { useAppDispatch } from '@/store/hooks'
import { setProfile } from '@/store/auth/reducers'
import { createUserProfile, createRegisterProfile } from '@/services'
import type { CreateProfileRequest } from '@/services/api/auth/types'

/**
 * Hook for profile operations (setup flow and register flow)
 */
export function useProfile() {
  const { showError, parseError } = useToast()
  const dispatch = useAppDispatch()
  const [isLoading, setIsLoading] = useState(false)

  /**
   * Create profile for setup flow
   * Used in setup/profile - does not require authentication and can only be triggered once
   */
  const createProfile = useCallback(async (data: CreateProfileRequest) => {
    setIsLoading(true)
    try {
      const response = await createUserProfile(data)
      // Update Redux with the new profile
      if (response.success && response.data) {
        dispatch(setProfile(response.data))
      }
      return { success: true, data: response }
    } catch (error: unknown) {
      const parsedError = parseError(error)
      if (parsedError.isValidationError && parsedError.fieldErrors) {
        const fieldErrorMessages = Object.entries(parsedError.fieldErrors)
          .map(([field, message]) => `${field}: ${message}`)
          .join('\n')
        showError(`${parsedError.message}\n\nField errors:\n${fieldErrorMessages}`, "Validation Failed")
      } else {
        showError(error, "Failed to create profile")
      }
      return { success: false, message: parsedError.message, fieldErrors: parsedError.fieldErrors }
    } finally {
      setIsLoading(false)
    }
  }, [showError, parseError, dispatch])

  /**
   * Create profile for register flow
   * Used in register/profile - requires authentication cookies from successful registration
   */
  const createProfileForRegister = useCallback(async (data: CreateProfileRequest) => {
    setIsLoading(true)
    try {
      const response = await createRegisterProfile(data)
      // Update Redux with the new profile
      if (response.success && response.data) {
        dispatch(setProfile(response.data))
      }
      return { success: true, data: response }
    } catch (error: unknown) {
      const parsedError = parseError(error)
      if (parsedError.isValidationError && parsedError.fieldErrors) {
        const fieldErrorMessages = Object.entries(parsedError.fieldErrors)
          .map(([field, message]) => `${field}: ${message}`)
          .join('\n')
        showError(`${parsedError.message}\n\nField errors:\n${fieldErrorMessages}`, "Validation Failed")
      } else {
        showError(error, "Failed to create profile")
      }

      return { success: false, message: parsedError.message, fieldErrors: parsedError.fieldErrors }
    } finally {
      setIsLoading(false)
    }
  }, [showError, parseError, dispatch])

  return {
    isLoading,
    createProfile,
    createProfileForRegister
  }
}
